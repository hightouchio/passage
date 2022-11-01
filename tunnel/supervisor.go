package tunnel

import (
	"context"
	consul "github.com/hashicorp/consul/api"
	"github.com/hightouchio/passage/stats"
	"io"
	"net"
	"time"
)

// Supervisor is responsible for a single tunnel. Supervisor monitors Tunnel status and restarts it if it crashes.
type Supervisor struct {
	Tunnel        Tunnel
	TunnelOptions TunnelOptions
	Retry         time.Duration
	Consul        *consul.Client
	Stats         stats.Stats

	stop chan bool
}

func NewSupervisor(tunnel Tunnel, consul *consul.Client, st stats.Stats, options TunnelOptions, retry time.Duration) *Supervisor {
	return &Supervisor{
		Tunnel:        tunnel,
		TunnelOptions: options,
		Retry:         retry,
		Consul:        consul,
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

				// Build visibility interfaces
				st := s.Stats.
					WithPrefix("tunnel").
					WithTags(stats.Tags{
						"tunnel_id": s.Tunnel.GetID().String(),
					})
				lifecycle := lifecycleLogger{st}

				// Inject visibility interfaces into context
				ctx = stats.InjectContext(ctx, st)
				ctx = injectCtxLifecycle(ctx, lifecycle)

				lifecycle.Start()

				errs := make(chan error)

				// Start SSH tunnel
				go func() {
					if err := s.Tunnel.Start(ctx, s.TunnelOptions); err != nil {
						errs <- err
					}
				}()

				// Start tunnel client listener
				listener := &TCPListener{
					BindHost:          s.TunnelOptions.BindHost,
					KeepaliveInterval: 5 * time.Second,
					Lifecycle:         lifecycle,
					conns:             make(chan net.Conn),
				}
				go func() {
					if err := listener.Start(ctx); err != nil {
						errs <- err
					}
				}()

				// Create a TCP forwarder between the two
				forwarder := &TCPForwarder{
					Listener: listener,
					GetUpstreamConn: func(c net.Conn) (io.ReadWriteCloser, error) {
						return s.Tunnel.Dial(c, "localhost:3000")
					},
					Lifecycle: lifecycle,
					Stats:     st,
				}
				go func() {
					if err := forwarder.Start(ctx); err != nil {
						errs <- err
					}
				}()

				select {
				case err := <-errs:
					lifecycle.BootError(err)
					continue
				case <-ctx.Done():
					continue
				}

				lifecycle.Stop()
			}
		}
	}()

	initialRun <- true
	<-s.stop
}

func (s *Supervisor) Stop() {
	close(s.stop)
}
