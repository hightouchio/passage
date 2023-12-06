package tunnel

import (
	"context"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/tunnel/discovery"
	"github.com/pkg/errors"
	"go.uber.org/zap"
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
			var connCheckStarted bool
			for update := range statusUpdates {
				logStatus(logger, update)

				// Now that the tunnel is online, start running the connectivity check
				//	Only do this once
				if update.Status == StatusReady && !connCheckStarted {
					connCheckStarted = true
					go tunnelConnectivityCheck(ctx, logger, "localhost", portFromNetAddr(tunnelListener.Addr()), connCheckUpdates)
				}
			}
		}()

		// Consume from the connectivity check update channel
		go func() {
			for connErr := range connCheckUpdates {
				// Update service discovery with tunnel health status
				var status discovery.HealthcheckStatus
				var message string

				if connErr == nil {
					status = discovery.HealthcheckPassing
					message = "Tunnel is online"
				} else {
					status = discovery.HealthcheckCritical
					message = connErr.Error()
				}

				logger.Infof("Update tunnel health: %v, %s", status, message)

				// TODO: Write this to a different healthcheck?
				if err := serviceDiscovery.UpdateHealth(tunnel.GetID(), status, message); err != nil {
					logger.Errorw("Failed to update tunnel health", zap.Error(err))
				}
			}
		}()

		// Run the tunnel
		return tunnel.Start(ctx, tunnelListener, statusUpdates)
	}
}
