package tunnel

import (
	"context"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/tunnel/discovery"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"sync"
	"time"
)

type ServeStrategy func(ctx context.Context, tunnel Tunnel) error

func TCPServeStrategy(bindHost string, serviceDiscovery discovery.DiscoveryService) ServeStrategy {
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

		// Create a channel to receive tunnel status updates
		statusUpdates := make(chan StatusUpdate)
		defer close(statusUpdates)

		// Create a channel to signal that the connection check should begin
		connCheckStartSignal := make(chan any)
		// Ensure we only close the channel once
		signalConnCheck := sync.OnceFunc(func() { close(connCheckStartSignal) })

		// Start tunnel status update routine
		go withTunnelHealthcheck(tunnel.GetID(), logger, serviceDiscovery, discovery.HealthcheckOptions{
			ID:   StatusHealthcheck,
			Name: "Tunnel Self-reported Status",
			TTL:  30 * time.Second,
		}, func(updateHealthcheck func(status discovery.HealthcheckStatus, message string)) {
			for update := range statusUpdates {
				// If the context has been cancelled (it has an error), continue to drain the status updates
				//	, but do no more work.
				if ctx.Err() != nil {
					continue
				}

				// Now that the tunnel is online, start running the connectivity check
				//	Only do this once
				if update.Status == StatusReady {
					signalConnCheck()
				}

				// Map tunnel status to healthcheck status
				var status discovery.HealthcheckStatus
				switch update.Status {
				case StatusBooting:
					status = discovery.HealthcheckWarning
				case StatusReady:
					status = discovery.HealthcheckPassing
				case StatusError:
					status = discovery.HealthcheckCritical
				}

				updateHealthcheck(status, update.Message)
			}
		})

		// Start tunnel connectivity check routine
		go func() {
			select {
			// If the context is cancelled before the start signal is received, we should just exit early.
			case <-ctx.Done():
				return

			// Wait for the start signal to be received.
			case <-connCheckStartSignal:
			}

			// If the context has already been cancelled, short circuit.
			if ctx.Err() != nil {
				return
			}

			// Create a channel to receive connectivity check updates
			connCheckUpdates := make(chan error)

			// Start the tunnel connectivity check
			go func() {
				// Close the update channel as soon as the connectivity check has completed
				defer close(connCheckUpdates)

				tunnelConnectivityCheck(ctx, logger, "localhost", portFromNetAddr(tunnelListener.Addr()), connCheckUpdates)
			}()

			// Register a tunnel connectivity healthcheck, and update it with the connectivity check updates
			withTunnelHealthcheck(tunnel.GetID(), logger, serviceDiscovery, discovery.HealthcheckOptions{
				ID:   ConnectivityHealthcheck,
				Name: "Tunnel Network Connectivity",
				TTL:  30 * time.Second,
			}, func(updateHealthcheck func(status discovery.HealthcheckStatus, message string)) {
				for connErr := range connCheckUpdates {
					// If the context has been cancelled (it has an error), continue to drain the updates
					//	, but do no more work.
					if ctx.Err() != nil {
						continue
					}

					// Update service discovery with tunnel health status
					if connErr == nil {
						updateHealthcheck(discovery.HealthcheckPassing, "Tunnel is reachable")
					} else {
						updateHealthcheck(discovery.HealthcheckCritical, connErr.Error())
					}
				}
			})
		}()

		// Start the tunnel, and retry if it fails.
		return retry(ctx, 30*time.Second, func() error {
			logger.Info("Starting tunnel")
			if err := tunnel.Start(ctx, tunnelListener, statusUpdates); err != nil {
				logger.Errorw("Error", zap.Error(err))

				// Record a healthcheck status update
				statusUpdates <- StatusUpdate{StatusError, errors.Cause(err).Error()}

				return err
			}
			logger.Info("Stopped tunnel")

			return nil
		})
	}
}

// retry the given function until it succeeds
func retry(ctx context.Context, interval time.Duration, fn func() error) error {
	for {
		select {
		case <-ctx.Done():
			return nil

		default:
			if err := fn(); err != nil {
				time.Sleep(interval)
				continue
			}

			return nil
		}
	}
}

func withTunnelHealthcheck(
	tunnelId uuid.UUID,
	log *log.Logger,
	serviceDiscovery discovery.DiscoveryService,
	options discovery.HealthcheckOptions,
	fn func(update func(status discovery.HealthcheckStatus, message string)),
) {
	if err := serviceDiscovery.RegisterHealthcheck(tunnelId, options); err != nil {
		log.Errorw("Failed to register healthcheck", zap.Error(err))
		return
	}
	defer func() {
		if err := serviceDiscovery.DeregisterHealthcheck(tunnelId, options.ID); err != nil {
			// It's OK if we fail to deregister the healthcheck
		}
	}()

	updateFunc := func(status discovery.HealthcheckStatus, message string) {
		if err := serviceDiscovery.UpdateHealthcheck(tunnelId, options.ID, status, message); err != nil {
			log.Errorw("Failed to update healthcheck", zap.Error(err))
		}
	}

	fn(updateFunc)
}

const (
	ConnectivityHealthcheck = "connectivity"
	StatusHealthcheck       = "status"
)
