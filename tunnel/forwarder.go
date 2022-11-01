package tunnel

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/stats"
	"github.com/pkg/errors"
	"io"
	"net"
)

type TCPForwarder struct {
	// Listener provides a flow of net.Conns
	Listener net.Listener

	// GetUpstreamConn is a function that's job is to initiate a connection to the upstream service.
	// It is called once for each incoming TunnelConnection.
	// It should return a dedicated io.ReadWriteCloser for each incoming TunnelConnection.
	GetUpstreamConn func(net.Conn) (io.ReadWriteCloser, error)

	Lifecycle Lifecycle
	Stats     stats.Stats
}

type TCPSession struct {
	net.Conn

	id                       string
	bytesSent, bytesReceived uint64
}

func (s *TCPSession) ID() string {
	return s.id
}

func (f *TCPForwarder) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	f.Lifecycle.BootEvent("forwarder_start", stats.Tags{})

	for {
		select {
		case <-ctx.Done():
			return nil

		default:
			if f.Listener == nil {
				return fmt.Errorf("cannot serve without first starting listener")
			}

			// Accept incoming tunnel connections
			conn, err := f.Listener.Accept()
			if err != nil {
				return errors.Wrap(err, "accept tcp")
			}

			// Pass connections off to tunnel connection handler.
			go func() {
				session := &TCPSession{
					Conn: conn,
					id:   uuid.New().String(),
				}

				f.handleSession(ctx, session)
			}()
		}
	}
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
		// Cast back to a TCP conn
		tcpConn := session.Conn.(*net.TCPConn)

		// Set SO_LINGER=0 so the tunnel net.TCPConn does not perform a graceful shutdown, indicating that the upstream couldn't be reached.
		if err := tcpConn.SetLinger(0); err != nil {
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
	case <-done: // Finished pipeline
	case <-ctx.Done():
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
