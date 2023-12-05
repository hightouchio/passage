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
				logger.Errorw("Failed to deregister tunnel from service discovery", zap.Error(err))
			}
		}()

		return tunnel.Start(ctx, tunnelListener, newTunnelStatusUpdater(logger))
	}
}
