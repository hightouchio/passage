package tunnel

import (
	"context"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/stats"
	"github.com/hightouchio/passage/tunnel/discovery"
	"go.uber.org/zap"
	"time"
)

// Supervisor is responsible for a single tunnel. Supervisor monitors Tunnel status and restarts it if it crashes.
type Supervisor struct {
	Tunnel        Tunnel
	TunnelOptions TunnelOptions

	Retry            time.Duration
	Stats            stats.Stats
	ServiceDiscovery discovery.DiscoveryService

	stop chan bool
}

func NewSupervisor(
	tunnel Tunnel,
	st stats.Stats,
	options TunnelOptions,
	retry time.Duration,
	serviceDiscovery discovery.DiscoveryService,
) *Supervisor {
	return &Supervisor{
		Tunnel:           tunnel,
		TunnelOptions:    options,
		Retry:            retry,
		Stats:            st,
		ServiceDiscovery: serviceDiscovery,

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
					ctx, cancel := context.WithCancel(ctx)
					defer cancel()

					st := s.Stats.WithTags(stats.Tags{"tunnel_id": s.Tunnel.GetID().String()})

					logger := log.Get().Named("Tunnel").With(
						zap.String("tunnel_id", s.Tunnel.GetID().String()),
					)

					logger.Info("Start supervisor")
					defer logger.Info("Stop supervisor")

					ctx = log.Context(stats.InjectContext(ctx, st), logger)

					// Serve the tunnel over TCP
					if err := TCPServeStrategy(s.TunnelOptions.BindHost, s.ServiceDiscovery)(ctx, s.Tunnel); err != nil {
						logger.With(zap.Error(err)).Errorf("Error: %s", err.Error())
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
