package tunnel

import (
	"context"
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
