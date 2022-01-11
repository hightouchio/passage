package tunnel

import (
	"context"
	"fmt"
	"github.com/hightouchio/passage/stats"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"strconv"
)

// handleTunnelConnection handles incoming TCP connections on the tunnel listen port, dials the tunneled upstream, and copies bytes bidirectionally
func (t NormalTunnel) handleTunnelConnection(ctx context.Context, sshClient *ssh.Client, tunnelConn net.Conn) (r, w int64, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	st := stats.GetStats(ctx)

	// Dial upstream service.
	upstreamAddr := net.JoinHostPort(t.ServiceHost, strconv.Itoa(t.ServicePort))
	st.WithEventTags(stats.Tags{"upstream_addr": upstreamAddr}).SimpleEvent("upstream.dial")
	serviceConn, err := sshClient.Dial("tcp", upstreamAddr)
	if err != nil {
		// Return upstreamConnectionError to indicate that the connection should be forcibly closed.
		return 0, 0, upstreamConnectionError{err: err}
	}
	defer serviceConn.Close()

	// Pass data bidirectionally from tunnel to service net.Conn, wait for errors, and record read/write
	done := make(chan struct{})
	go func() {
		r, w, err = bidirectionalReadWrite(tunnelConn, serviceConn)
		close(done)
	}()

	// Wait for completion or for context cancellation
	select {
	case <-done:
		return
	case <-ctx.Done():
		return
	}
}

// upstreamConnectionError wraps an error connecting to an upstream service
type upstreamConnectionError struct {
	err error
}

func (e upstreamConnectionError) Error() string {
	return fmt.Sprintf("could not connect to upstream: %s", e.err.Error())
}

func bidirectionalReadWrite(a, b io.ReadWriter) (ra, rb int64, err error) {
	copyConn := func(src io.Reader, dst io.Writer, written *int64, errors chan<- error) {
		byteCount, err := io.Copy(dst, src)
		*written = byteCount // Record number of bytes written in this direction
		errors <- err
	}

	// Copy data bidirectionally.
	errA := make(chan error)
	go copyConn(a, b, &ra, errA)
	errB := make(chan error)
	go copyConn(b, a, &rb, errB)

	// Wait for either side A or B to close, and return ero
	select {
	case err = <-errA:
		if err != nil {
			err = errors.Wrap(err, "read A")
		}
	case err = <-errB:
		if err != nil {
			err = errors.Wrap(err, "read B")
		}
	}

	return
}
