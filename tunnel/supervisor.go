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

	// stop is the signal to stop the tunnel
	stop chan any

	// isStopped is the signal when the tunnel is stopped
	doneStopping chan any
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

		stop:         make(chan any),
		doneStopping: make(chan any),
	}
}

func (s *Supervisor) Start() {
	ctx, cancel := context.WithCancel(context.Background())

	logger := loggerForTunnel(s.Tunnel, log.FromContext(ctx))
	ctx = log.Context(
		stats.InjectContext(ctx, statsForTunnel(s.Tunnel, s.Stats)),
		logger,
	)

	// Serve the tunnel via a TCP socket
	serveStrategy := TCPServeStrategy(s.TunnelOptions.BindHost, s.ServiceDiscovery)

	// Once the stop channel is closed (by the Stop() function), cancel the context
	//	We propagate the cancellation to the tunnel's context, which will cause the tunnel to shut down internally
	go func() {
		<-s.stop

		logger.Info("Shutting down tunnel")
		cancel()
	}()

	go func() {
		// Signal that the tunnel is stopped once this function exits
		defer close(s.doneStopping)

		for {
			select {
			case <-ctx.Done():
				return

			default:
				if err := serveStrategy(ctx, s.Tunnel); err != nil {
					logger.With(zap.Error(err)).Errorf("Error: %s", err.Error())
				}

				// If the context is cancelled, immediately stop
				if ctx.Err() != nil {
					return
				}

				time.Sleep(s.Retry)
			}
		}
	}()
}

func (s *Supervisor) Stop() {
	// Trigger the tunnel to shut down
	close(s.stop)

	// Wait for the tunnel to completely shut down
	<-s.doneStopping
}

func loggerForTunnel(tunnel Tunnel, log *log.Logger) *log.Logger {
	return log.Named("Tunnel").With(
		zap.String("tunnel_id", tunnel.GetID().String()),
	)
}

func statsForTunnel(tunnel Tunnel, st stats.Stats) stats.Stats {
	return st.WithTags(stats.Tags{"tunnel_id": tunnel.GetID().String()})
}
