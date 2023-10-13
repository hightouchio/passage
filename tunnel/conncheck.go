package tunnel

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
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
