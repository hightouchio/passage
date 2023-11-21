package tunnel

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/stats"
	"github.com/pkg/errors"
	"io"
	"net"
	"sync"
	"time"
)

type TCPForwarder struct {
	Listener *net.TCPListener

	// GetUpstreamConn is a function thats job is to initiate a connection to the upstream service.
	// It is called once for each incoming TunnelConnection.
	// It should return a dedicated io.ReadWriteCloser for each incoming TunnelConnection.
	GetUpstreamConn func(net.Conn) (io.ReadWriteCloser, error)

	// KeepaliveInterval is the interval between OS level TCP keepalive handshakes
	KeepaliveInterval time.Duration

	Lifecycle Lifecycle
	Stats     stats.Stats
	conns     map[string]net.Conn
	close     chan struct{}

	sync.RWMutex
}

type TCPSession struct {
	*net.TCPConn
	id string

	bytesSent, bytesReceived uint64
}

func (s *TCPSession) ID() string {
	return s.id
}

func (f *TCPForwarder) Serve() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if f.Listener == nil {
		return fmt.Errorf("must set listener")
	}

	if f.close == nil {
		f.close = make(chan struct{})
	}

	for {
		select {
		case <-f.close:
			return nil

		default:
			// Accept incoming tunnel connections
			conn, err := f.Listener.AcceptTCP()
			if err != nil {
				return errors.Wrap(err, "accept tcp")
			}

			// Configure keepalive
			if err := conn.SetKeepAlive(true); err != nil {
				return errors.Wrap(err, "set keepalive")
			}
			if err := conn.SetKeepAlivePeriod(f.KeepaliveInterval); err != nil {
				return errors.Wrap(err, "set keepalive period")
			}

			// Pass connections off to tunnel connection handler.
			go func() {
				session := &TCPSession{
					TCPConn: conn,
					id:      uuid.New().String(),
				}
				defer session.Close()

				f.handleSession(ctx, session)
			}()
		}
	}
}

func (f *TCPForwarder) Close() error {
	if f.close != nil {
		close(f.close)
	}
	if f.Listener != nil {
		return f.Listener.Close()
	}
	return nil
}

// handleSession takes a TCPSession (backed by a net.TCPConn), then initiates an upstream connection to our forwarding backend
// and forwards packets between the two.
func (f *TCPForwarder) handleSession(ctx context.Context, session *TCPSession) {
	f.Lifecycle.SessionEvent(session.ID(), "open", stats.Tags{"remote_addr": session.RemoteAddr().String()})

	defer func() {
		session.Close()
		// Record pipeline metrics to logs and statsd
		f.Stats.Count("bytes_rcvd", int64(session.bytesReceived), nil, 1)
		f.Stats.Count("bytes_sent", int64(session.bytesSent), nil, 1)
		f.Lifecycle.SessionEvent(session.ID(), "close", stats.Tags{
			"bytes_rcvd": session.bytesReceived,
			"bytes_sent": session.bytesSent,
		})
	}()

	// Get upstream connection.
	upstream, err := f.GetUpstreamConn(session)
	if err != nil {
		// Set SO_LINGER=0 so the tunnel net.TCPConn does not perform a graceful shutdown, indicating that the upstream couldn't be reached.
		if err := session.SetLinger(0); err != nil {
			f.Lifecycle.SessionError(session.ID(), errors.Wrap(err, "set linger"))
		}

		f.Lifecycle.SessionError(session.ID(), errors.Wrap(err, "dial upstream"))
		return
	}
	defer upstream.Close()

	// Initialize pipeline, and point the byte counters to bytesReceived and bytesSent on the TCPSession
	pipeline := NewBidirectionalPipeline(session, upstream)

	done := make(chan struct{})
	go func() {
		// Tally up bytes
		go writeCountTo(pipeline.writeCounterA, &session.bytesReceived)
		go writeCountTo(pipeline.writeCounterB, &session.bytesSent)

		// Forward bytes.
		if err := pipeline.Run(); err != nil {
			f.Lifecycle.SessionError(session.ID(), errors.Wrap(err, "pipeline"))
		}

		close(done)
	}()

	select {
	case <-ctx.Done():
	case <-done: // Finished pipeline
	}
}

// BidirectionalPipeline passes bytes bidirectionally from io.ReadWriters a and b, and records the number of bytes written to each.
type BidirectionalPipeline struct {
	a, b io.ReadWriteCloser

	writeCounterA, writeCounterB chan uint64
}

func NewBidirectionalPipeline(a, b io.ReadWriteCloser) *BidirectionalPipeline {
	return &BidirectionalPipeline{
		a:             a,
		writeCounterA: make(chan uint64),

		b:             b,
		writeCounterB: make(chan uint64),
	}
}

// Run starts the bidirectional copying of bytes, and blocks until completion.
func (p *BidirectionalPipeline) Run() error {
	// Buffered error channel to allow both sides to send an error without blocking and leaking goroutines.
	cerr := make(chan error, 1)
	// Copy data bidirectionally.
	go func() {
		defer p.a.Close()
		defer p.b.Close()
		defer close(p.writeCounterB)

		cerr <- copyWithCounter(p.a, p.b, p.writeCounterB)
	}()
	go func() {
		defer p.b.Close()
		defer p.a.Close()
		defer close(p.writeCounterA)

		cerr <- copyWithCounter(p.b, p.a, p.writeCounterA)
	}()

	// Wait for either side A or B to close, and return err
	return <-cerr
}

// copyWithCounter performs an io.Copy from src to dst, and keeps track of the number of bytes written by writing to the *written pointer.
func copyWithCounter(src io.Reader, dst io.Writer, writeCounter chan<- uint64) error {
	_, err := io.Copy(io.MultiWriter(dst, CounterWriter{writeCounter}), src)
	return err
}

// CounterWriter is a no-op Writer that records how many bytes have been written to it
type CounterWriter struct {
	writeCounter chan<- uint64
}

// Write does nothing with the input byte slice but records the length to the WriteCounter
func (b CounterWriter) Write(p []byte) (n int, err error) {
	count := len(p)
	b.writeCounter <- uint64(count)
	return count, nil
}

func writeCountTo(counter <-chan uint64, n *uint64) {
	for {
		v, ok := <-counter
		if !ok {
			return
		}
		*n += v
	}
}
