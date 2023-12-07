package tunnel

import (
	"context"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/tunnel/discovery"
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
	// Define the frequency at which Passage reports tunnel status to service discovery.
	tunnelStatusReportInterval = 15 * time.Second
)

// tunnelStatusReporter reports the Ready status regularly until the context is cancelled
func tunnelStatusReporter(ctx context.Context, statusUpdate chan<- StatusUpdate) {
	// Send one update immediately
	statusUpdate <- StatusUpdate{StatusReady, "Tunnel is online"}
	defer func() {
		statusUpdate <- StatusUpdate{StatusError, "Tunnel is offline"}
	}()

	// Report regularly
	ticker := time.NewTicker(tunnelStatusReportInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			statusUpdate <- StatusUpdate{StatusReady, "Tunnel is online"}
		}
	}
}

type GetUpstreamFn func() (io.ReadWriteCloser, error)

// upstreamHealthcheck reports the health of the upstream service to service discovery
func upstreamHealthcheck(
	ctx context.Context,
	tunnel Tunnel,
	log *log.Logger,
	serviceDiscovery discovery.DiscoveryService,
	fn GetUpstreamFn,
) {
	options := discovery.HealthcheckOptions{
		ID:   HealthcheckUpstream,
		Name: "Upstream reachability",
		TTL:  60 * time.Second,
	}
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		// Register the healthcheck
		withTunnelHealthcheck(
			tunnel.GetID(),
			log,
			serviceDiscovery,
			options,
			func(updateHealthcheck func(status discovery.HealthcheckStatus, message string)) {
				for {
					select {
					case <-ctx.Done():
						return

					case <-ticker.C:
						err := testUpstream(ctx, fn)

						if err == nil {
							updateHealthcheck(discovery.HealthcheckPassing, "Upstream is reachable")
						} else {
							updateHealthcheck(discovery.HealthcheckCritical, err.Error())
						}
					}
				}
			},
		)
	}()
}

// Test upstream reachability
func testUpstream(ctx context.Context, fn GetUpstreamFn) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	errchan := make(chan error)
	go func() {
		upstream, err := fn()
		errchan <- err
		if upstream != nil {
			defer upstream.Close()
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errchan:
		return err
	}
}
