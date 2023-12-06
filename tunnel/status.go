package tunnel

import (
	"github.com/hightouchio/passage/log"
	"go.uber.org/zap"
)

type Status string

const (
	StatusBooting Status = "booting"
	StatusReady   Status = "ready"
	StatusError   Status = "error"
)

type StatusUpdate struct {
	Status  Status
	Message string
}

func logStatus(log *log.Logger, status StatusUpdate) {
	log.With(
		zap.String("status", string(status.Status)),
		zap.String("message", status.Message),
	).Debugf("[%s] %s", status.Status, status.Message)
}
