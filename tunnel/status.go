package tunnel

import (
	"github.com/hightouchio/passage/log"
	"go.uber.org/zap"
)

type Status string

const (
	StatusBooting Status = "booting"
	StatusOnline  Status = "healthy"
	StatusError   Status = "error"
)

type StatusUpdate struct {
	Status  Status
	Message string
}

func statusLogger(log *log.Logger, statuses <-chan StatusUpdate) {
	for status := range statuses {
		log.With(
			zap.String("status", string(status.Status)),
			zap.String("message", string(status.Status)),
		).Debugf("[%s] %s", status.Status, status.Message)
	}
}
