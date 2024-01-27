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
	"sync/atomic"
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
	Stats  forwarderStats

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

	// Create a channel to receive stat delta reports from connections
	statReports := make(chan forwarderStatsPayload)

	// Aggregate stat delta reports on a per-tunnel basis and report them to the stats client
	aggregateTicker := time.NewTicker(metricReportInterval)
	defer aggregateTicker.Stop()
	go connectionStatAggregator(ctx, statReports, func(report forwarderStatsPayload) {
		reportForwarderStats(f.Stats, report)
	}, aggregateTicker.C)

	// TODO: Can't do this until we wait for connections to exit.
	//defer close(statReports)

	// Keep track of the number of active connections and report metrics
	var connectionCount atomic.Int32
	go intervalMetricReporter(ctx, func() {
		// Report the current client connection count
		f.Stats.Gauge(StatTunnelClientActiveConnectionCount, float64(connectionCount.Load()), stats.Tags{}, 1)
	})

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

				connectionCount.Add(1)
				defer connectionCount.Add(-1)

				f.handleSession(ctx, session, statReports)
			}()
		}
	}
}

// handleSession takes a TCPSession (backed by a net.TCPConn), then initiates an upstream connection to our forwarding backend
// and forwards packets between the two.
func (f *TCPForwarder) handleSession(ctx context.Context, session *TCPSession, statReports chan<- forwarderStatsPayload) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sessionLogger := f.logger.With(zap.String("session_id", session.ID()))

	defer func() {
		if err := session.Close(); err != nil {
			// If the session is already closed, we can ignore this error.
			if !errors.Is(err, net.ErrClosed) {
				sessionLogger.Warnw("Could not close session", zap.Error(err))
			}
		}
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

	// Wrap the session and upstream in a CountedReadWriteCloser to count bytes sent and received
	upstreamRwc := NewCountedReadWriteCloser(upstream)
	sessionRwc := NewCountedReadWriteCloser(session)

	// Stream connection stats to the aggregator
	go func() {
		// Record connection stats every second and report deltas to the aggregator
		ticker := time.NewTicker(metricReportInterval)
		defer ticker.Stop()
		connectionStatProducer(ctx, sessionRwc, upstreamRwc, statReports, ticker.C)
	}()

	// Run bidirectional pipeline
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := runPipeline(sessionRwc, upstreamRwc); err != nil {
			sessionLogger.Errorw("Pipeline", zap.Error(err))
		}
	}()

	select {
	case <-ctx.Done():
	case <-done: // Finished pipeline
	}
}

func (f *TCPForwarder) Close() error {
	if f.close != nil {
		close(f.close)
	}
	return nil
}

// runPipeline passes bytes bidirectionally from io.ReadWriterClosers a and b, and blocks until completion
func runPipeline(a, b io.ReadWriteCloser) error {
	// Buffered error channel to allow both sides to send an error without blocking and leaking goroutines.
	cerr := make(chan error, 1)
	// Copy data bidirectionally.
	go func() {
		defer a.Close()
		defer b.Close()

		_, err := io.Copy(b, a)
		cerr <- err
	}()
	go func() {
		defer b.Close()
		defer a.Close()

		_, err := io.Copy(a, b)
		cerr <- err
	}()

	// Wait for either side A or B to close, and return err
	return <-cerr
}

func NewCountedReadWriteCloser(rwc io.ReadWriteCloser) *CountedReadWriteCloser {
	return &CountedReadWriteCloser{ReadWriteCloser: rwc}
}

// CountedReadWriteCloser is a wrapper around an io.ReadWriteCloser that counts the number of bytes read and written
type CountedReadWriteCloser struct {
	io.ReadWriteCloser

	bytesWritten uint64
	bytesRead    uint64
}

func (c *CountedReadWriteCloser) Read(p []byte) (n int, err error) {
	bytesRead, err := c.ReadWriteCloser.Read(p)
	c.bytesRead += uint64(bytesRead)
	return bytesRead, err
}

func (c *CountedReadWriteCloser) Write(p []byte) (n int, err error) {
	c.bytesWritten += uint64(len(p))
	return c.ReadWriteCloser.Write(p)
}

func (c *CountedReadWriteCloser) GetBytesWritten() uint64 {
	return c.bytesWritten
}

func (c *CountedReadWriteCloser) GetBytesRead() uint64 {
	return c.bytesRead
}

type forwarderStats interface {
	Count(name string, value int64, tags stats.Tags, rate float64)
	Gauge(name string, value float64, tags stats.Tags, rate float64)
}

type forwarderStatsPayload struct {
	ClientBytesSent       uint64
	ClientBytesReceived   uint64
	UpstreamBytesSent     uint64
	UpstreamBytesReceived uint64
}

// reportForwarderStats reports the number of bytes read and written to the statsd client
func reportForwarderStats(st forwarderStats, payload forwarderStatsPayload) {
	st.Count(StatTunnelClientBytesSent, int64(payload.ClientBytesSent), stats.Tags{}, 1)
	st.Count(StatTunnelClientBytesReceived, int64(payload.ClientBytesReceived), stats.Tags{}, 1)
	st.Count(StatTunnelUpstreamBytesSent, int64(payload.UpstreamBytesSent), stats.Tags{}, 1)
	st.Count(StatTunnelUpstreamBytesReceived, int64(payload.UpstreamBytesReceived), stats.Tags{}, 1)
}

type trackedConnection interface {
	GetBytesWritten() uint64
	GetBytesRead() uint64
}

func connectionStatProducer(
	ctx context.Context,
	client, upstream trackedConnection,
	deltas chan<- forwarderStatsPayload,
	tick <-chan time.Time,
) {
	var last forwarderStatsPayload
	doTick := func() {
		current := forwarderStatsPayload{
			ClientBytesSent:       client.GetBytesWritten(),
			ClientBytesReceived:   client.GetBytesRead(),
			UpstreamBytesSent:     upstream.GetBytesWritten(),
			UpstreamBytesReceived: upstream.GetBytesRead(),
		}

		// Report the delta between the last and current stats
		deltas <- forwarderStatsPayload{
			ClientBytesSent:       max(current.ClientBytesSent-last.ClientBytesSent, 0),
			ClientBytesReceived:   max(current.ClientBytesReceived-last.ClientBytesReceived, 0),
			UpstreamBytesSent:     max(current.UpstreamBytesSent-last.UpstreamBytesSent, 0),
			UpstreamBytesReceived: max(current.UpstreamBytesReceived-last.UpstreamBytesReceived, 0),
		}

		last = current
	}

	// Report the final stats before exiting
	defer doTick()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick:
			doTick()
		}
	}
}

func connectionStatAggregator(
	ctx context.Context,
	deltas <-chan forwarderStatsPayload,
	reportFunc func(forwarderStatsPayload),
	tick <-chan time.Time,
) {
	var agg forwarderStatsPayload

	for {
		select {
		case <-ctx.Done():
			return

		case delta := <-deltas:
			// Aggregate deltas and report the aggregated value
			//	(bundling / batching happens in the stat client)
			agg.ClientBytesSent += delta.ClientBytesSent
			agg.ClientBytesReceived += delta.ClientBytesReceived
			agg.UpstreamBytesSent += delta.UpstreamBytesSent
			agg.UpstreamBytesReceived += delta.UpstreamBytesReceived

		case <-tick:
			// Report the aggregated stats to the reporter
			reportFunc(agg)
		}
	}
}
