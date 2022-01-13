package tunnel

import (
	"github.com/google/uuid"
	"github.com/hightouchio/passage/stats"
	"github.com/pkg/errors"
	"io"
	"net"
	"sync"
	"time"
)

type TCPForwarder struct {
	BindAddr string

	// KeepaliveInterval is the interval between OS level TCP keepalive handshakes
	KeepaliveInterval time.Duration

	// GetUpstreamConn is a function thats job is to initiate a connection to the upstream service.
	// It is called once for each incoming TunnelConnection.
	// It should return a dedicated io.ReadWriteCloser for each incoming TunnelConnection.
	GetUpstreamConn func(net.Conn) (io.ReadWriteCloser, error)

	Lifecycle Lifecycle
	Stats     stats.Stats

	conns     map[string]net.Conn
	close     chan struct{}
	closeOnce sync.Once

	sync.RWMutex
}

func (f *TCPForwarder) Listen() error {
	f.close = make(chan struct{})

	// Open tunnel TCP listener
	f.Lifecycle.BootEvent("listener_start", stats.Tags{"listen_addr": f.BindAddr})
	listenTcpAddr, err := net.ResolveTCPAddr("tcp", f.BindAddr)
	if err != nil {
		return bootError{event: "listener_start", err: errors.Wrap(err, "resolve addr")}
	}
	listener, err := net.ListenTCP("tcp", listenTcpAddr)
	if err != nil {
		return bootError{event: "listener_start", err: err}
	}
	defer listener.Close()

	f.Lifecycle.Open()
	defer f.Lifecycle.Close()

	go func() {
		for {
			// Accept incoming tunnel connections
			conn, err := listener.AcceptTCP()
			if err != nil {
				// TODO: Make sure that accept errors are logged.
				break
			}
			// Pass connections off to tunnel connection handler.
			go f.handleConnection(conn)
		}
	}()

	<-f.close
	return nil
}

func (f *TCPForwarder) Close() error {
	f.closeOnce.Do(func() { close(f.close) })
	return nil
}

// handleConnection takes a net.TCPConn, representing a tunnel connection, then initiates an upstream connection to our forwarding backend
// and forwards packets between the two.
func (f *TCPForwarder) handleConnection(conn *net.TCPConn) {
	sessionId := uuid.New().String()
	f.Lifecycle.SessionEvent(sessionId, "open", stats.Tags{"remote_addr": conn.RemoteAddr().String()})

	// Establish pointers to store read and written bytes
	var bytesRead, bytesWritten *int64
	// Set defaults so we never nil dereference.
	bytesRead = new(int64)
	bytesWritten = new(int64)

	defer func() {
		conn.Close()
		// Record pipeline metrics to logs and statsd
		f.Stats.Count("read_bytes", *bytesRead, nil, 1)
		f.Stats.Count("write_bytes", *bytesWritten, nil, 1)
		f.Lifecycle.SessionEvent(sessionId, "close", stats.Tags{
			"read_bytes":  *bytesRead,
			"write_bytes": *bytesWritten,
		})
	}()

	// Configure keepalive
	conn.SetKeepAlive(true)
	conn.SetKeepAlivePeriod(f.KeepaliveInterval)

	// Get upstream connection.
	upstream, err := f.GetUpstreamConn(conn)
	if err != nil {
		// Set SO_LINGER=0 so the tunnel net.TCPConn does not perform a graceful shutdown, indicating that the upstream couldn't be reached.
		if err := conn.SetLinger(0); err != nil {
			f.Lifecycle.SessionError(sessionId, errors.Wrap(err, "set linger"))
		}

		f.Lifecycle.SessionError(sessionId, errors.Wrap(err, "dial upstream"))
		return
	}
	defer upstream.Close()

	// Forward bytes.
	done := make(chan struct{})
	pipeline := NewBidirectionalPipeline(conn, upstream)
	go func() {
		if err := pipeline.Run(); err != nil {
			f.Lifecycle.SessionError(sessionId, errors.Wrap(err, "pipeline"))
		}
		close(done)
	}()
	bytesRead = &pipeline.writtenA
	bytesWritten = &pipeline.writtenB

	select {
	case <-f.close: // Forwarder close
	case <-done: // Finished pipeline
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
