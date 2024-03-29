package tunnel

import (
	"context"
	"github.com/cenkalti/backoff/v4"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/tunnel/discovery"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"sync"
	"time"
)

type ServeStrategy func(ctx context.Context, tunnel Tunnel) error

// TCPServeStrategy opens a tunnel listener on an ephemeral TCP port, registers the tunnel with service discovery,
//
//	manages healthchecks, and serves the tunnel itself.
func TCPServeStrategy(bindHost string, serviceDiscovery discovery.Service, retryInterval time.Duration) ServeStrategy {
	return func(ctx context.Context, tunnel Tunnel) error {
		logger := log.FromContext(ctx)

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Run listening on a random, unused local port
		tunnelListener, err := newEphemeralTCPListener(bindHost)
		if err != nil {
			return errors.Wrap(err, "open listener")
		}
		logger.Debugw("Start listener", "listen_addr", tunnelListener.Addr().String())
		defer tunnelListener.Close()

		// Register tunnel with service discovery.
		if err := serviceDiscovery.RegisterTunnel(tunnel.GetID(), portFromNetAddr(tunnelListener.Addr())); err != nil {
			return errors.Wrap(err, "register with service discovery")
		}
		defer func() {
			if err := serviceDiscovery.DeregisterTunnel(tunnel.GetID()); err != nil {
				logger.Errorw("deregister tunnel from service discovery", zap.Error(err))
			}
		}()

		// Track all goroutines
		wg := sync.WaitGroup{}

		// Create a channel to receive tunnel status updates
		statusUpdates := make(chan StatusUpdate)
		closeStatusUpdates := sync.OnceFunc(func() {
			close(statusUpdates)
		})
		defer closeStatusUpdates()

		// Start status healthcheck
		wg.Add(1)
		go func() {
			defer wg.Done()
			statusHealthcheck(ctx, tunnel, logger, serviceDiscovery, statusUpdates)
		}()

		// Run the tunnel, and restart it if it crashes
		err = retryTunnel(ctx, func() error {
			logger.Info("Starting tunnel")
			if err := tunnel.Start(ctx, tunnelListener, statusUpdates); err != nil {
				logger.Errorw("Error", zap.Error(err))

				// Record a healthcheck status update
				statusUpdates <- StatusUpdate{StatusError, errors.Cause(err).Error()}

				return err
			}

			logger.Info("Stopped tunnel")
			statusUpdates <- StatusUpdate{StatusError, "Tunnel offline"}

			return nil
		})

		// Status updates are done, we don't need them anymore
		closeStatusUpdates()

		// Wait for all goroutines to exit
		wg.Wait()

		// Wait for tunnel to completely shut down
		return err
	}
}

func retryTunnel(ctx context.Context, runTunnel func() error) error {
	return backoff.Retry(runTunnel, backoff.WithContext(newTunnelBackoff(), ctx))
}

func newTunnelBackoff() backoff.BackOff {
	return &backoff.ExponentialBackOff{
		InitialInterval:     5 * time.Second,
		MaxInterval:         120 * time.Second,
		RandomizationFactor: 0.6,
		Multiplier:          1.5,

		// If MaxElapsedTime is 0, the exponential backoff will never stop.
		MaxElapsedTime: 0,

		// This is basically just a wrapper around time.Now
		Clock: backoff.SystemClock,

		// Stop signal
		Stop: backoff.Stop,
	}
}
