package tunnel

import (
	"context"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/tunnel/discovery"
	"github.com/pkg/errors"
	"go.uber.org/zap"
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
		defer tunnelListener.Close()
		logger.Infow("Open tunnel listener", "listen_addr", tunnelListener.Addr().String())

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

		// Create a channel to receive connectivity check updates
		connCheckUpdates := make(chan error)
		defer close(connCheckUpdates)

		// Consume from the status update channel
		go func() {
			// TODO: Do we really want to shutdown the tunnel if healthchecks fail?
			defer cancel()

			withTunnelHealthcheck(tunnel.GetID(), logger, serviceDiscovery, discovery.HealthcheckOptions{
				ID:   StatusHealthcheck,
				Name: "Tunnel Self-reported Status",
				TTL:  30 * time.Second,
			}, func(updateHealthcheck func(status discovery.HealthcheckStatus, message string)) {
				var connCheckStarted bool
				for update := range statusUpdates {
					logStatus(logger, update)

					// Now that the tunnel is online, start running the connectivity check
					//	Only do this once
					if update.Status == StatusReady && !connCheckStarted {
						connCheckStarted = true
						go tunnelConnectivityCheck(ctx, logger, "localhost", portFromNetAddr(tunnelListener.Addr()), connCheckUpdates)
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

					// Update service discovery with tunnel health status
					if err := serviceDiscovery.UpdateHealthcheck(tunnel.GetID(), StatusHealthcheck, status, update.Message); err != nil {
						logger.Errorw("Failed to update tunnel health", zap.Error(err))
					}
				}
			})
		}()

		// Consume from the connectivity check update channel
		go func() {
			// TODO: Do we really want to shutdown the tunnel if healthchecks fail?
			defer cancel()

			withTunnelHealthcheck(tunnel.GetID(), logger, serviceDiscovery, discovery.HealthcheckOptions{
				ID:   ConnectivityHealthcheck,
				Name: "Tunnel Network Connectivity",
				TTL:  30 * time.Second,
			}, func(updateHealthcheck func(status discovery.HealthcheckStatus, message string)) {
				// Listen for connection check updates
				for connErr := range connCheckUpdates {
					// Update service discovery with tunnel health status
					if connErr == nil {
						updateHealthcheck(discovery.HealthcheckPassing, "Tunnel is reachable")
					} else {
						updateHealthcheck(discovery.HealthcheckCritical, connErr.Error())
					}
				}
			})
		}()

		// Run the tunnel
		return tunnel.Start(ctx, tunnelListener, statusUpdates)
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
			log.Errorw("Failed to deregister healthcheck", zap.Error(err))
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
