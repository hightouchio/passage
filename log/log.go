package log

import (
	"context"
	"github.com/sirupsen/logrus"
)

const contextLoggerKey = "_logger"

func WithLogger(ctx context.Context, entry logrus.FieldLogger) context.Context {
	return context.WithValue(ctx, contextLoggerKey, entry)
}

func GetLogger(ctx context.Context) logrus.FieldLogger {
	entry, ok := ctx.Value(contextLoggerKey).(logrus.FieldLogger)
	if !ok {
		return logrus.NewEntry(logrus.StandardLogger())
	}
	return entry
}
