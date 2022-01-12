package tunnel

import (
	"context"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/stats"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"net"
	"sync"
	"time"
)

type normalTunnelInstance struct {
	upstreamAddr     string
	sshClient        *ssh.Client
	sshClientOptions SSHClientOptions

	conns map[string]tunnelConnection
	mux   sync.RWMutex
}

// tunnelConnection is a representation of an active connection for visibility
type tunnelConnection struct {
	startAt time.Time
}

func (i *normalTunnelInstance) HandleConnection(ctx context.Context, conn *net.TCPConn) {
	lifecycle := getCtxLifecycle(ctx)
	st := stats.GetStats(ctx)

	sessionId := uuid.New().String()
	lifecycle.SessionEvent(sessionId, "open", stats.Tags{"remote_addr": conn.RemoteAddr().String()})
	defer conn.Close()

	// Establish pointers to store read and written bytes
	var bytesRead, bytesWritten *int64
	// Set defaults so we never nil dereference.
	bytesRead = new(int64)
	bytesWritten = new(int64)
	defer func() {
		// Record pipeline metrics to logs and statsd
		st.Count("read_bytes", *bytesRead, nil, 1)
		st.Count("write_bytes", *bytesWritten, nil, 1)
		lifecycle.SessionEvent(sessionId, "close", stats.Tags{
			"read_bytes":  *bytesRead,
			"write_bytes": *bytesWritten,
		})
	}()

	// Register connection for visibility.
	i.registerConnection(sessionId, time.Now())
	defer i.deregisterConnection(sessionId)

	// Configure networking parameters.
	conn.SetKeepAlive(true)
	conn.SetKeepAlivePeriod(i.sshClientOptions.KeepaliveInterval)

	// Connect upstream.
	lifecycle.SessionEvent(sessionId, "dial_upstream", stats.Tags{})
	upstream, err := i.dialUpstream(ctx)
	if err != nil {
		// Set SO_LINGER=0 so the TCP connection does not perform a graceful shutdown, indicating that the upstream couldn't be reached.
		if err := conn.SetLinger(0); err != nil {
			lifecycle.SessionError(sessionId, errors.Wrap(err, "set linger"))
		}

		lifecycle.SessionError(sessionId, errors.Wrap(err, "dial upstream"))
		return
	}
	defer upstream.Close()

	// Forward bytes.
	done := make(chan struct{})
	pipeline := NewBidirectionalPipeline(conn, upstream)
	go func() {
		if err := pipeline.Run(); err != nil {
			lifecycle.SessionError(sessionId, errors.Wrap(err, "pipeline"))
		}
		close(done)
	}()
	bytesRead = &pipeline.writtenA
	bytesWritten = &pipeline.writtenB

	// Block on context cancellation or on pipeline completion.
	select {
	case <-ctx.Done():
	case <-done:
	}
}

// dialUpstream connects to the upstream service
func (i *normalTunnelInstance) dialUpstream(ctx context.Context) (net.Conn, error) {
	// Dial upstream service.
	serviceConn, err := i.sshClient.Dial("tcp", i.upstreamAddr)
	if err != nil {
		return nil, err
	}
	return serviceConn, err
}

// handleTunnelConnection handles incoming TCP connections on the tunnel listen port, dials the tunneled upstream, and copies bytes bidirectionally
func (i *normalTunnelInstance) registerConnection(sessionId string, startAt time.Time) {
	i.mux.Lock()
	defer i.mux.Unlock()
	i.conns[sessionId] = tunnelConnection{
		startAt: startAt,
	}
}

func (i *normalTunnelInstance) deregisterConnection(sessionId string) {
	i.mux.Lock()
	defer i.mux.Unlock()
	delete(i.conns, sessionId)
}

// logNormalTunnelInstanceState is a helper function for recording the state of a normalTunnelInstance to logger and statsd
func logNormalTunnelInstanceState(ctx context.Context, i *normalTunnelInstance) {
	st := stats.GetStats(ctx)

	i.mux.RLock()
	defer i.mux.RUnlock()

	ids := make([]string, 0)
	for id, _ := range i.conns {
		ids = append(ids, id)
	}

	st.Gauge("active_connections", float64(len(i.conns)), nil, 1)
}
