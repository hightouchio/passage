package tunnel

import (
	"context"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/tunnel/discovery"
	"go.uber.org/zap"
	"io"
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
	statusHealthcheckInterval = 30 * time.Second
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
	upstreamHealthcheckInterval = 65 * time.Second
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
