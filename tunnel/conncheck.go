package tunnel

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/log"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"net"
	"strings"
	"time"
)

const (
	conncheckTimeout          = 10 * time.Second
	conncheckDialTimeout      = 1 * time.Second
	conncheckErrorWaitTimeout = 1 * time.Second
	conncheckReadMaxBytes     = 256

	conncheckErrorPrefix = "passage-error"
)

type CheckTunnelRequest struct {
	ID uuid.UUID
}

type CheckTunnelResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// CheckTunnel identifies a currently running tunnel, gets connection details, and attempts a connection
func (s Server) CheckTunnel(ctx context.Context, req CheckTunnelRequest) (*CheckTunnelResponse, error) {
	s.Stats.Incr("passage.tunnelStatusChecks", nil, 1)

	details, err := s.GetTunnel(ctx, GetTunnelRequest{ID: req.ID})
	if err != nil {
		return nil, errors.Wrap(err, "could not get connection details")
	}

	log.GetLogger(ctx).WithField("tunnel_id", req.ID).Info("checking tunnel")

	ctx, cancel := context.WithTimeout(ctx, conncheckTimeout)
	defer cancel()

	if err := checkConnectivity(ctx, details.ConnectionDetails); err != nil {
		return &CheckTunnelResponse{Success: false, Error: err.Error()}, nil
	}

	return &CheckTunnelResponse{Success: true}, nil
}

// checkConnectivity
func checkConnectivity(ctx context.Context, details ConnectionDetails) error {
	dialer := &net.Dialer{Timeout: conncheckDialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", details.Host, details.Port))
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
func waitForTunnelError(ctx context.Context, reader io.ReadCloser, duration time.Duration) error {
	done := make(chan error)

	// read in a context-aware fashion
	go func() {
		data, err := ioutil.ReadAll(io.LimitReader(reader, conncheckReadMaxBytes))
		if err != nil {
			done <- errors.Wrap(err, "read error")
			return
		}

		// check if the bytes we read were an error
		message := string(data)
		if strings.HasPrefix(message, conncheckErrorPrefix) {
			done <- errors.New(message)
			return
		}

		done <- nil
	}()

	select {
	case <-time.After(duration):
		return nil // success

	case err := <-done:
		return err

	case <-ctx.Done(): // if an error does not occur before the context times out, we're OK
		return ctx.Err()
	}
}
