package tunnel

import (
	"context"
	"fmt"
	"github.com/hightouchio/passage/log"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"net"
	"strconv"
	"time"
)

const (
	// TODO: Adjust this interval to align with discovery TTLs
	conncheckInterval = 30 * time.Second

	conncheckDialTimeout      = 5 * time.Second
	conncheckErrorWaitTimeout = 1 * time.Second
	conncheckReadMaxBytes     = 256
)

// tunnelConnectivityCheck continuously checks the status of a tunnel, independent of the tunnel-handling code itself.
func tunnelConnectivityCheck(ctx context.Context, log *log.Logger, host string, port int, updates chan<- error) {
	logger := log.Named("ConnectivityCheck")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create a ticker to run the check on an interval
	ticker := time.NewTicker(conncheckInterval)
	defer ticker.Stop()

	tick := func() {
		// Check connectivity to the tunnel.
		err := checkConnectivity(ctx, host, port)
		updates <- err
		if err == nil {
			logger.Debug("Tunnel is online")
		} else {
			logger.Debug("Tunnel is offline")
		}
	}

	// Initial check
	logger.Debug("Start")
	defer logger.Debug("Stop")
	tick()
	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			// Subsequent checks
			tick()
		}
	}
}

// checkConnectivity
func checkConnectivity(ctx context.Context, host string, port int) error {
	dialer := &net.Dialer{Timeout: conncheckDialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf(net.JoinHostPort(host, strconv.Itoa(port))))
	if err != nil {
		return err
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
