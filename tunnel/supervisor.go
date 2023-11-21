package tunnel

import (
	"context"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/stats"
	"go.uber.org/zap"
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

		for {
			select {
			case <-ctx.Done():
				return

			default:
				select {
				case <-ticker.C:
				case <-initialRun:
				}

				func() {
					// Build visibility interfaces
					st := s.Stats.
						WithPrefix("tunnel").
						WithTags(stats.Tags{
							"tunnel_id": s.Tunnel.GetID().String(),
						})

					logger := log.Get().Named("Tunnel").With(
						zap.String("tunnel_id", s.Tunnel.GetID().String()),
					)

					lifecycle := lifecycleLogger{logger}
					lifecycle.Start()
					defer lifecycle.Stop()

					// Inject visibility interfaces into context
					ctx, cancel := context.WithCancel(ctx)
					defer cancel()

					ctx = stats.InjectContext(ctx, st)
					ctx = injectCtxLifecycle(ctx, lifecycle)
					ctx = log.Context(ctx, logger)

					if err := s.Tunnel.Start(ctx, s.TunnelOptions, newTunnelStatusUpdater(logger)); err != nil {
						switch err.(type) {
						case bootError:
							logger.Errorw("Boot error", zap.Error(err))
						default:
							logger.Errorw("Error", zap.Error(err))
						}
					}
				}()
			}
		}
	}()

	initialRun <- true
	<-s.stop
}

func (s *Supervisor) Stop() {
	close(s.stop)
}
