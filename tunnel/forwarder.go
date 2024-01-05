package tunnel

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/stats"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io"
	"net"
	"sync"
	"time"
)

type TCPForwarder struct {
	Listener *net.TCPListener

	// GetUpstreamConn is a function that's job is to initiate a connection to the upstream service.
	// It is called once for each incoming TunnelConnection.
	// It should return a dedicated io.ReadWriteCloser for each incoming TunnelConnection.
	GetUpstreamConn func() (io.ReadWriteCloser, error)

	// KeepaliveInterval is the interval between OS level TCP keepalive handshakes
	KeepaliveInterval time.Duration

	logger *log.Logger
	Stats  stats.Stats

	conns map[string]net.Conn
	close chan struct{}

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
			return net.ErrClosed

		default:
			// Accept incoming tunnel connections
			conn, err := f.Listener.AcceptTCP()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return err
				}

				return errors.Wrap(err, "accept tcp")
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
	return nil
}

// handleSession takes a TCPSession (backed by a net.TCPConn), then initiates an upstream connection to our forwarding backend
// and forwards packets between the two.
func (f *TCPForwarder) handleSession(ctx context.Context, session *TCPSession) {
	sessionLogger := f.logger.With(zap.String("session_id", session.ID()))

	defer func() {
		if err := session.Close(); err != nil {
			// If the session is already closed, we can ignore this error.
			if !errors.Is(err, net.ErrClosed) {
				sessionLogger.Warnw("Could not close session", zap.Error(err))
			}
		}

		// Record pipeline metrics to logs and statsd
		f.Stats.Count(StatTunnelBytesReceived, int64(session.bytesReceived), nil, 1)
		f.Stats.Count(StatTunnelBytesSent, int64(session.bytesSent), nil, 1)
	}()

	// Get upstream connection.
	upstream, err := f.GetUpstreamConn()
	if err != nil {
		// Set SO_LINGER=0 so the tunnel net.TCPConn does not perform a graceful shutdown, indicating that the upstream couldn't be reached.
		if err := session.SetLinger(0); err != nil {
			sessionLogger.Warnw("Could not set error", zap.Error(err))
		}

		sessionLogger.Errorw("Could not dial upstream", zap.Error(err))
		return
	}
	defer upstream.Close()

	// Configure keepalive
	if err := session.SetKeepAlive(true); err != nil {
		sessionLogger.Errorw("Set keepalive", zap.Error(err))
		return
	}
	if err := session.SetKeepAlivePeriod(f.KeepaliveInterval); err != nil {
		sessionLogger.Errorw("Set keepalive period", zap.Error(err))
		return
	}

	// Initialize pipeline, and point the byte counters to bytesReceived and bytesSent on the TCPSession
	pipeline := NewBidirectionalPipeline(session, upstream)

	done := make(chan struct{})
	go func() {
		// Tally up bytes
		go writeCountTo(pipeline.writeCounterA, &session.bytesReceived)
		go writeCountTo(pipeline.writeCounterB, &session.bytesSent)

		// Forward bytes.
		if err := pipeline.Run(); err != nil {
			sessionLogger.Errorw("Pipeline", zap.Error(err))
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
