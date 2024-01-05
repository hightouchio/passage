package log

import (
	"context"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
)

type Logger = zap.SugaredLogger

// Get returns the global SugaredLogger
func Get() *Logger {
	return zap.S()
}

// Init initializes the global logger
func Init(level, format string) {
	logLevel, err := zap.ParseAtomicLevel(level)
	if err != nil {
		log.Fatalf("Invalid log level: %s", level)
		return
	}

	// Global config
	config := zap.Config{
		Level:       logLevel,
		Development: false,

		// Disable sampling
		Sampling: nil,
		// Disable caller reporting
		DisableCaller: true,
		// Disable stacktraces on errors
		DisableStacktrace: true,

		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	switch format {
	case "text":
		// This is the "development" logger
		config.Encoding = "console"
		config.EncoderConfig = zap.NewDevelopmentEncoderConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	case "json":
		// This is the "production" logger
		config.Encoding = "json"
		config.EncoderConfig = zap.NewProductionEncoderConfig()
		config.EncoderConfig.EncodeDuration = zapcore.MillisDurationEncoder

	default:
		log.Fatalf("Invalid log format: %s", format)
		return
	}

	config.EncoderConfig.TimeKey = "time"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := config.Build()
	if err != nil {
		log.Fatalf("Could not initialize logger: %s", err)
		return
	}

	// Set the logger as the Zap global logger
	zap.ReplaceGlobals(logger)
}

func Context(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, "_logger", logger)
}

func FromContext(ctx context.Context) *Logger {
	logger, ok := ctx.Value("_logger").(*Logger)
	if ok {
		return logger
	} else {
		// Default to the global logger
		return Get()
	}
}
