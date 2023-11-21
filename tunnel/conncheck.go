package tunnel

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/tunnel/discovery"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"net"
	"strconv"
	"time"
)

const (
	conncheckTimeout          = 10 * time.Second
	conncheckDialTimeout      = 5 * time.Second
	conncheckErrorWaitTimeout = 1 * time.Second
	conncheckReadMaxBytes     = 256
)

type CheckTunnelRequest struct {
	ID uuid.UUID
}

type CheckTunnelResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// runTunnelConnectivityCheck continuously checks the status of a tunnel, independent of the tunnel-handling code itself.
func runTunnelConnectivityCheck(ctx context.Context, tunnelID uuid.UUID, logger *log.Logger, serviceDiscovery discovery.DiscoveryService) {
	logger = logger.Named("ConnectivityCheck").With(zap.String("tunnel_id", tunnelID.String()))

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Check the connectivity every 15 seconds
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	logger.Debug("Start")

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			logger.Debug("Checking tunnel")

			// Resolve the tunnel connection details from service discovery
			tunnelDetails, err := serviceDiscovery.GetTunnel(tunnelID)
			if err != nil {
				logger.Error("could not get tunnel details for connectivity check", zap.Error(err))
				break
			}

			// Check connectivity to the tunnel.
			if err := checkConnectivity(ctx, tunnelDetails.Host, tunnelDetails.Port); err != nil {
				logger.Warnw("Tunnel is unhealthy", zap.Error(err))

				// Report Unhealthy status
				if err := serviceDiscovery.UpdateHealth(tunnelID, discovery.TunnelUnhealthy, fmt.Sprintf("Tunnel connectivity check failed: %s", err.Error())); err != nil {
					logger.Errorw("Could not update service discovery with tunnel status", zap.Error(err))
				}
			} else {
				logger.Info("Tunnel is healthy")
				// Report Healthy status
				if err := serviceDiscovery.UpdateHealth(tunnelID, discovery.TunnelHealthy, "Tunnel connectivity check successful"); err != nil {
					logger.Errorw("Could not update service discovery with tunnel status", zap.Error(err))
				}
			}
		}
	}
}

// checkConnectivity
func checkConnectivity(ctx context.Context, host string, port int) error {
	dialer := &net.Dialer{Timeout: conncheckDialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf(net.JoinHostPort(host, strconv.Itoa(port))))
	if err != nil {
		return errors.Wrap(err, "dial error")
	}
	defer conn.Close()

	c := make(chan error)
	go func() { c <- waitForTunnelError(ctx, conn, conncheckErrorWaitTimeout) }()

	// wait for waitForTunnelError to complete
	select {
	case <-ctx.Done():
		return ctx.Err()

	case err := <-c:
		if err != nil {
			return err
		} else {
			return nil
		}
	}
}

// waitForTunnelError
func waitForTunnelError(ctx context.Context, reader io.ReadCloser, waitDuration time.Duration) error {
	done := make(chan error)

	// read in a context-aware fashion
	go func() {
		_, err := ioutil.ReadAll(io.LimitReader(reader, conncheckReadMaxBytes))
		if err != nil {
			done <- errors.Wrap(err, "read error")
			return
		}
		done <- nil
	}()

	select {
	case <-time.After(waitDuration):
		return nil // success

	case err := <-done:
		return err

	case <-ctx.Done(): // if an error does not occur before the context times out, we're OK
		return ctx.Err()
	}
}
