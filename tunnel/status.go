package tunnel

import (
	"context"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/tunnel/discovery"
	"go.uber.org/zap"
	"io"
	"net"
	"time"
)

type Status string

const (
	StatusBooting Status = "booting"
	StatusReady   Status = "ready"
	StatusError   Status = "error"
)

type StatusUpdate struct {
	Status  Status
	Message string
}

const (
	statusHealthcheckID       = "tunnel"
	statusHealthcheckName     = "Tunnel"
	statusHealthcheckTTL      = 180 * time.Second
	statusHealthcheckInterval = 20 * time.Second
)

// intervalStatusReporter sends regular status updates to a StatusUpdate channel
func intervalStatusReporter(ctx context.Context, ch chan<- StatusUpdate, getStatus func() StatusUpdate) {
	// Send one update immediately
	ch <- getStatus()

	ticker := time.NewTicker(statusHealthcheckInterval)
	defer ticker.Stop()

	// Report regularly
	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			ch <- getStatus()
		}
	}
}

// statusHealthcheck reports the self-reported tunnel status into a healthcheck
func statusHealthcheck(ctx context.Context, tunnel Tunnel, log *log.Logger, serviceDiscovery discovery.Service, statusUpdates <-chan StatusUpdate) {
	options := discovery.HealthcheckOptions{
		ID:   statusHealthcheckID,
		Name: statusHealthcheckName,
		TTL:  statusHealthcheckTTL,
	}
	withTunnelHealthcheck(
		tunnel.GetID(),
		log,
		serviceDiscovery,
		options,
		func(updateHealthcheck func(status discovery.HealthcheckStatus, message string)) {
			for update := range statusUpdates {
				// If the context has been cancelled (it has an error), continue to drain the status updates
				//	, but do no more work.
				if ctx.Err() != nil {
					continue
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
		},
	)
}

const (
	upstreamHealthcheckID       = "upstream"
	upstreamHealthcheckName     = "Upstream reachability"
	upstreamHealthcheckTTL      = 180 * time.Second
	upstreamHealthcheckInterval = 60 * time.Second
)

// upstreamHealthcheck reports the health of the upstream service to service discovery
func upstreamHealthcheck(
	ctx context.Context,
	tunnel Tunnel,
	log *log.Logger,
	serviceDiscovery discovery.Service,
	fn GetUpstreamFn,
) {
	options := discovery.HealthcheckOptions{
		ID:   upstreamHealthcheckID,
		Name: upstreamHealthcheckName,
		TTL:  upstreamHealthcheckTTL,
	}

	ticker := time.NewTicker(upstreamHealthcheckInterval)
	defer ticker.Stop()

	// Register the healthcheck
	withTunnelHealthcheck(
		tunnel.GetID(),
		log,
		serviceDiscovery,
		options,
		func(updateHealthcheck func(status discovery.HealthcheckStatus, message string)) {
			runOnceAndTick(ctx, upstreamHealthcheckInterval, func() {
				if err := testUpstream(ctx, fn); err != nil {
					updateHealthcheck(discovery.HealthcheckCritical, err.Error())
				} else {
					updateHealthcheck(discovery.HealthcheckPassing, "Upstream is reachable")
				}
			})
		},
	)
}

type GetUpstreamFn func() (io.ReadWriteCloser, error)

// Test upstream reachability
func testUpstream(ctx context.Context, fn GetUpstreamFn) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	errchan := make(chan error)
	go func() {
		upstream, err := fn()

		if upstream != nil {
			upstream.Close()
		}

		// If the context has closed before we receive an error, short circuit
		if ctx.Err() != nil {
			return
		}

		errchan <- err
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errchan:
		return err
	}
}

const (
	listenerHealthcheckID       = "listener"
	listenerHealthcheckName     = "Listener reachability"
	listenerHealthcheckTTL      = 180 * time.Second
	listenerHealthcheckInterval = 60 * time.Second
)

// listenerHealthcheck continuously checks the status of the tunnel listener
func listenerHealthcheck(
	ctx context.Context,
	tunnel Tunnel,
	log *log.Logger,
	serviceDiscovery discovery.Service,
	addr net.Addr,
) {
	options := discovery.HealthcheckOptions{
		ID:   listenerHealthcheckID,
		Name: listenerHealthcheckName,
		TTL:  listenerHealthcheckTTL,
	}

	// Register the healthcheck
	withTunnelHealthcheck(
		tunnel.GetID(),
		log,
		serviceDiscovery,
		options,
		func(updateHealthcheck func(status discovery.HealthcheckStatus, message string)) {
			runOnceAndTick(ctx, listenerHealthcheckInterval, func() {
				if err := testListener(ctx, addr); err != nil {
					updateHealthcheck(discovery.HealthcheckCritical, err.Error())
				} else {
					updateHealthcheck(discovery.HealthcheckPassing, "Listener is reachable")
				}
			})
		},
	)
}

const (
	healthcheckDialTimeout = 5 * time.Second
)

// testListener dials the listener to confirm that its open
func testListener(ctx context.Context, addr net.Addr) error {
	dialer := &net.Dialer{
		Timeout: healthcheckDialTimeout,
	}
	conn, err := dialer.DialContext(ctx, "tcp", addr.String())
	if err != nil {
		return err
	}
	defer conn.Close()

	return nil
}

// withTunnelHealthcheck registers a healthcheck with service discovery, calls the given function, and deregisters the healthcheck when the function exits
func withTunnelHealthcheck(
	tunnelId uuid.UUID,
	log *log.Logger,
	serviceDiscovery discovery.Service,
	options discovery.HealthcheckOptions,
	fn func(update func(status discovery.HealthcheckStatus, message string)),
) {
	if err := serviceDiscovery.RegisterHealthcheck(tunnelId, options); err != nil {
		log.Errorw("Failed to register healthcheck", zap.Error(err))
		return
	}

	// Deregister the healthcheck when the function exits
	defer func() {
		if err := serviceDiscovery.DeregisterHealthcheck(tunnelId, options.ID); err != nil {
			// It's OK if we fail to deregister the healthcheck
			log.Errorw("Failed to deregister healthcheck", zap.Error(err))
		}
	}()

	// Call the function add pass it a function which it can use to update the healthcheck status
	fn(func(status discovery.HealthcheckStatus, message string) {
		if err := serviceDiscovery.UpdateHealthcheck(tunnelId, options.ID, status, message); err != nil {
			log.Errorw("Failed to update healthcheck", zap.Error(err))
		}
	})
}
