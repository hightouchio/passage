package tunnel

import (
	"context"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/stats"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"sync"
	"time"
)

type normalTunnelInstance struct {
	upstreamAddr     string
	sshClient        *ssh.Client
	sshClientOptions SSHClientOptions

	conns map[uuid.UUID]tunnelConnection
	mux   sync.RWMutex
}

// tunnelConnection is a representation of an active connection for visibility
type tunnelConnection struct {
	startAt time.Time
}

func (i *normalTunnelInstance) HandleConnection(ctx context.Context, conn *net.TCPConn) {
	sessionId := uuid.New()
	st := stats.GetStats(ctx).WithEventTags(stats.Tags{"session_id": sessionId}).WithPrefix("conn")
	ctx = stats.InjectContext(ctx, st)
	st.WithEventTags(stats.Tags{"remote_addr": conn.RemoteAddr().String()}).SimpleEvent("accept")
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
		st.WithEventTags(stats.Tags{"read_bytes": *bytesRead, "write_bytes": *bytesWritten}).SimpleEvent("close")
	}()

	// Register connection for visibility.
	i.registerConnection(sessionId, time.Now())
	defer i.deregisterConnection(sessionId)

	// Configure networking parameters.
	conn.SetKeepAlive(true)
	conn.SetKeepAlivePeriod(i.sshClientOptions.KeepaliveInterval)

	// Connect upstream.
	upstream, err := i.dialUpstream(ctx)
	if err != nil {
		// Set SO_LINGER=0 so the TCP connection does not perform a graceful shutdown, indicating that the upstream couldn't be reached.
		if err := conn.SetLinger(0); err != nil {
			st.ErrorEvent("error", errors.Wrap(err, "error SetLinger"))
		}
		st.ErrorEvent("error", errors.Wrap(err, "upstream connection error"))
		return
	}
	defer upstream.Close()

	// Forward bytes.
	done := make(chan struct{})
	pipeline := NewBidirectionalPipeline(conn, upstream)
	go func() {
		if err := pipeline.Run(); err != nil {
			st.ErrorEvent("error", errors.Wrap(err, "pipeline error"))
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
	stats.GetStats(ctx).WithEventTags(stats.Tags{"upstream_addr": i.upstreamAddr}).SimpleEvent("upstream.dial")
	serviceConn, err := i.sshClient.Dial("tcp", i.upstreamAddr)
	if err != nil {
		return nil, err
	}
	return serviceConn, err
}

// handleTunnelConnection handles incoming TCP connections on the tunnel listen port, dials the tunneled upstream, and copies bytes bidirectionally
func (i *normalTunnelInstance) registerConnection(sessionId uuid.UUID, startAt time.Time) {
	i.mux.Lock()
	defer i.mux.Unlock()
	i.conns[sessionId] = tunnelConnection{
		startAt: startAt,
	}
}

func (i *normalTunnelInstance) deregisterConnection(sessionId uuid.UUID) {
	i.mux.Lock()
	defer i.mux.Unlock()
	delete(i.conns, sessionId)
}

// logNormalTunnelInstanceState is a helper function for recording the state of a normalTunnelInstance to logger and statsd
func logNormalTunnelInstanceState(i *normalTunnelInstance, st stats.Stats, logger *logrus.Entry) {
	i.mux.RLock()
	defer i.mux.RUnlock()

	ids := make([]uuid.UUID, 0)
	for id, _ := range i.conns {
		ids = append(ids, id)
	}

	st.Gauge("active_connections", float64(len(i.conns)), nil, 1)
	if len(i.conns) > 0 {
		logger.WithField("active_connections", ids).Trace("tunnel active connections")
	}
}

// BidirectionalPipeline passes bytes bidirectionally from io.ReadWriters a and b, and records the number of bytes written to each.
type BidirectionalPipeline struct {
	a, b               io.ReadWriter
	writtenA, writtenB int64
}

func NewBidirectionalPipeline(a, b io.ReadWriter) *BidirectionalPipeline {
	return &BidirectionalPipeline{a: a, b: b}
}

// Run starts the bidirectional copying of bytes, and blocks until completion.
func (p *BidirectionalPipeline) Run() error {
	// Buffered error channel to allow both sides to send an error without blocking and leaking goroutines.
	cerr := make(chan error, 1)
	// Copy data bidirectionally.
	go func() {
		cerr <- copyWithCounter(p.a, p.b, &p.writtenB)
	}()
	go func() {
		cerr <- copyWithCounter(p.b, p.a, &p.writtenA)
	}()

	// Wait for either side A or B to close, and return err
	return <-cerr
}

// copyWithCounter performs an io.Copy from src to dst, and keeps track of the number of bytes written by writing to the *written pointer.
func copyWithCounter(src io.Reader, dst io.Writer, written *int64) error {
	count, err := io.Copy(io.MultiWriter(dst, CounterWriter{written}), src)
	*written = count
	return err
}

// CounterWriter is a no-op Writer that records how many bytes have been written to it
type CounterWriter struct {
	written *int64
}

// Write does nothing with the input byte slice but records the length
func (b CounterWriter) Write(p []byte) (n int, err error) {
	count := len(p)
	*b.written += int64(count)
	return count, nil
}
