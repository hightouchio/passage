package grpc

import (
	"context"
	"fmt"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/hightouchio/passage/log"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func NewServer(logger *log.Logger) *grpc.Server {
	logger = logger.Named("gRPC")

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			// Set up OpenTelemetry tracing
			otelgrpc.UnaryServerInterceptor(),

			// Set up logging
			logging.UnaryServerInterceptor(newZapGrpcLogger(logger),
				logging.WithLogOnEvents(logging.StartCall, logging.FinishCall),

				// Extract trace ID from context
				logging.WithFieldsFromContext(func(ctx context.Context) logging.Fields {
					if span := trace.SpanContextFromContext(ctx); span.IsSampled() {
						return logging.Fields{"traceID", span.TraceID().String()}
					}
					return nil
				}),
			),
		),
	)

	return srv
}

func newZapGrpcLogger(l *log.Logger) logging.LoggerFunc {
	return func(ctx context.Context, level logging.Level, msg string, fields ...any) {
		f := make([]zap.Field, 0, len(fields)/2)

		for i := 0; i < len(fields); i += 2 {
			key := fields[i]
			value := fields[i+1]

			switch v := value.(type) {
			case string:
				f = append(f, zap.String(key.(string), v))
			case int:
				f = append(f, zap.Int(key.(string), v))
			case bool:
				f = append(f, zap.Bool(key.(string), v))
			default:
				f = append(f, zap.Any(key.(string), v))
			}
		}

		logger := l.WithOptions(zap.AddCallerSkip(1)).Desugar().With(f...)

		switch level {
		case logging.LevelDebug:
			logger.Debug(msg)
		case logging.LevelInfo:
			logger.Info(msg)
		case logging.LevelWarn:
			logger.Warn(msg)
		case logging.LevelError:
			logger.Error(msg)
		default:
			panic(fmt.Sprintf("unknown level %v", level))
		}
	}
}
