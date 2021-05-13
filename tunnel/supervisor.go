package tunnel

import (
	"context"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"time"
)

type Supervisor struct {
	Tunnel     Tunnel
	SSHOptions SSHOptions
	Retry      time.Duration

	stop chan bool
}

func NewSupervisor(tunnel Tunnel, options SSHOptions, retry time.Duration) *Supervisor {
	return &Supervisor{
		Tunnel:     tunnel,
		SSHOptions: options,
		Retry:      retry,

		stop: make(chan bool),
	}
}

func (s *Supervisor) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		ticker := time.NewTicker(s.Retry)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			default:
				<-ticker.C
				s.logger().Info("starting tunnel'")
				if err := s.Tunnel.Start(ctx, s.SSHOptions); err != nil {
					s.logger().Error(errors.Wrap(err, "start tunnel"))
				}
			}
		}
	}()

	<-s.stop
}

func (s *Supervisor) Stop() {
	s.logger().Info("stopping tunnel")
	close(s.stop)
}

func (s Supervisor) logger() *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		"tunnel_id": s.Tunnel,
	})
}
