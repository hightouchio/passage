package tunnel

import (
	"context"
	"github.com/hightouchio/passage/stats"
	"github.com/sirupsen/logrus"
	"time"
)

// Supervisor is responsible for a single tunnel. Supervisor monitors Tunnel status and restarts it if it crashes.
type Supervisor struct {
	Tunnel        Tunnel
	TunnelOptions TunnelOptions
	Retry         time.Duration
	Stats         stats.Stats

	stop chan bool
}

func NewSupervisor(tunnel Tunnel, st stats.Stats, options TunnelOptions, retry time.Duration) *Supervisor {
	return &Supervisor{
		Tunnel:        tunnel,
		TunnelOptions: options,
		Retry:         retry,
		Stats:         st,

		stop: make(chan bool),
	}
}

func (s *Supervisor) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	initialRun := make(chan bool)
	go func() {
		ticker := time.NewTicker(s.Retry)
		defer ticker.Stop()

		s.Stats.SimpleEvent("supervisor.start")
		for {
			select {
			case <-ctx.Done():
				s.Stats.SimpleEvent("supervisor.stop")
				return

			default:
				select {
				case <-ticker.C:
				case <-initialRun:
				}

				s.Stats.SimpleEvent("start")
				if err := s.Tunnel.Start(ctx, s.TunnelOptions); err != nil {
					s.Stats.ErrorEvent("error", err)
				} else {
					s.Stats.SimpleEvent("stop")
				}
			}
		}
	}()

	initialRun <- true
	<-s.stop
}

func (s *Supervisor) Stop() {
	close(s.stop)
}

func (s Supervisor) logger() *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		"tunnel_id": s.Tunnel.GetID().String(),
	})
}
