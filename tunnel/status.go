package tunnel

import (
	"github.com/hightouchio/passage/log"
	"go.uber.org/zap"
)

type StatusUpdateFn func(status Status, message string)

type Status string

const (
	StatusBooting Status = "booting"
	StatusOnline  Status = "healthy"
	StatusError   Status = "error"
)

func newTunnelStatusUpdater(log *log.Logger) StatusUpdateFn {
	logger := log.Named("StatusUpdater")
	return func(status Status, message string) {
		logger.With(
			zap.String("status", string(status)),
			zap.String("message", message),
		).Debugf("Status update: %s", status)
	}
}
