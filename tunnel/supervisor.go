package tunnel

import (
	"context"
	"github.com/hightouchio/passage/stats"
	"github.com/sirupsen/logrus"
	"time"
)

type Supervisor struct {
	Tunnel     Tunnel
	SSHOptions SSHOptions
	Retry      time.Duration
	Stats      stats.Stats

	stop chan bool
}

func NewSupervisor(tunnel Tunnel, st stats.Stats, options SSHOptions, retry time.Duration) *Supervisor {
	return &Supervisor{
		Tunnel:     tunnel,
		SSHOptions: options,
		Retry:      retry,
		Stats:      st,

		stop: make(chan bool),
	}
}

func (s *Supervisor) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	s.Stats.SimpleEvent("supervisor.start")
	go func() {
		ticker := time.NewTicker(s.Retry)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				s.Stats.SimpleEvent("supervisor.stop")
				return

			default:
				<-ticker.C
				s.Stats.SimpleEvent("start")

				if err := s.Tunnel.Start(ctx, s.SSHOptions); err != nil {
					s.Stats.ErrorEvent("error", err)
				} else {
					s.Stats.SimpleEvent("stop")
				}
			}
		}
	}()

	<-s.stop
}

func (s *Supervisor) Stop() {
	s.Stats.SimpleEvent("stop")
	close(s.stop)
}

func (s Supervisor) logger() *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		"tunnel_id": s.Tunnel.GetID().String(),
	})
}
