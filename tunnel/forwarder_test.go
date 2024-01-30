package tunnel

import (
	"fmt"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/stats"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

type upstreamService struct {
	listener *net.TCPListener
}

func (u *upstreamService) Serve() error {
	for {
		conn, err := u.listener.Accept()
		if err != nil {
			return err
		}

		go func() {
			_, err := io.Copy(conn, conn)
			if err != nil {
				fmt.Printf("error copying: %s", err)
			}
		}()
	}
}

type dummyStats struct {
	mux      sync.RWMutex
	recorded map[string]int64
}

func newDummyStats() *dummyStats {
	return &dummyStats{
		recorded: make(map[string]int64),
	}
}

func (d *dummyStats) Count(name string, value int64, tags stats.Tags, rate float64) {
	d.mux.Lock()
	defer d.mux.Unlock()
	d.recorded[name] = value
}

func (d *dummyStats) Gauge(name string, value float64, tags stats.Tags, rate float64) {
	d.mux.Lock()
	defer d.mux.Unlock()
	// convert it to int64 because w/ the forwarder, we're not actually recording floats
	d.recorded[name] = int64(value)
}

func (d *dummyStats) Get() map[string]int64 {
	d.mux.RLock()
	defer d.mux.RUnlock()
	return d.recorded
}

// Test basic bidirectional forwarding of data, and recording of stats
func Test_Forwarder(t *testing.T) {
	// Upstream listener
	upstreamListener, err := newEphemeralTCPListener(":0")
	if err != nil {
		t.Error(errors.Wrap(err, "open upstream listener"))
	}
	defer upstreamListener.Close()

	// Upstream service
	go func() {
		service := upstreamService{listener: upstreamListener}
		if err := service.Serve(); err != nil {
			if !errors.Is(err, net.ErrClosed) {
				t.Error(err)
			}
		}
	}()

	// Client side listener
	clientListener, err := newEphemeralTCPListener(":0")
	if err != nil {
		t.Error(errors.Wrap(err, "open client listener"))
	}
	defer clientListener.Close()

	// TCP Forwarder
	forwarderStats := newDummyStats()
	forwarder := &TCPForwarder{
		Listener: clientListener,
		GetUpstreamConn: func() (io.ReadWriteCloser, error) {
			return net.Dial("tcp", fmt.Sprintf("localhost:%d", portFromNetAddr(upstreamListener.Addr())))
		},
		KeepaliveInterval: 5 * time.Second,
		Stats:             forwarderStats,
		logger:            log.Get().Named("Forwarder"),
	}
	defer forwarder.Close()
	go func() {
		if err := forwarder.Serve(); err != nil {
			if !errors.Is(err, net.ErrClosed) {
				t.Error(err)
			}
		}
	}()

	clientConn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", portFromNetAddr(clientListener.Addr())))
	if err != nil {
		t.Error(err, "could not initiate client conn")
		return
	}

	// Write some data
	if _, err := clientConn.Write([]byte("Hello, world!")); err != nil {
		t.Error(err, "could not write to client conn")
		return
	}

	// Read some data
	buf := make([]byte, 1024)
	bytesRead, err := clientConn.Read(buf)
	if err != nil {
		t.Error(err, "could not read from client conn")
		return
	}
	if string(buf[:bytesRead]) != "Hello, world!" {
		t.Errorf("read unexpected data from client conn: %s", string(buf[:bytesRead]))
		return
	}

	clientConn.Close()

	// Wait for metrics to fully report
	//	Kinda hacky, but it is what it is
	time.Sleep(1500 * time.Millisecond)

	// Check stats
	finalStats := forwarderStats.Get()
	assert.Equal(t, int64(0), finalStats[StatTunnelClientActiveConnectionCount])
	assert.Equal(t, int64(13), finalStats[StatTunnelClientBytesSent])
	assert.Equal(t, int64(13), finalStats[StatTunnelClientBytesReceived])
	assert.Equal(t, int64(13), finalStats[StatTunnelUpstreamBytesSent])
	assert.Equal(t, int64(13), finalStats[StatTunnelUpstreamBytesReceived])
}
