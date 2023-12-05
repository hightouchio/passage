package tunnel

import (
	"context"
	"github.com/DataDog/datadog-go/statsd"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/stats"
	"github.com/hightouchio/passage/tunnel/discovery"
	"github.com/hightouchio/passage/tunnel/discovery/static"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

func Test_Manager_restartTunnel(t *testing.T) {
	dbTunnels := []*mockTunnel{
		newMockTunnel(23456),
		newMockTunnel(34567),
		newMockTunnel(45678),
		newMockTunnel(56789),
	}

	listFunc := func(ctx context.Context) ([]Tunnel, error) {
		t := make([]Tunnel, len(dbTunnels))
		for i, dbt := range dbTunnels {
			t[i] = dbt
		}
		return t, nil
	}

	manager := NewManager(log.Get(), stats.New(&statsd.NoOpClient{}), listFunc, TunnelOptions{}, 50*time.Millisecond, 50*time.Millisecond, static.Discovery{})

	baseCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tunnelCtx, stopServers := context.WithCancel(baseCtx)
	defer stopServers()

	manager.refreshTunnels(baseCtx)
	manager.refreshSupervisors(tunnelCtx)

	assert.Len(t, manager.tunnels, 4, "tunnel length")
	assert.Len(t, manager.supervisors, 4, "supervisor length")

	// wait for tunnels to start
	for _, tunnel := range dbTunnels {
		ctx, cancel := context.WithTimeout(tunnelCtx, 1*time.Second)
		assert.Truef(t, tunnel.WaitForStart(ctx), "tunnel %d started", tunnel.id)
		cancel()
	}

	// adjust one tunnel to restart
	tunnel1 := dbTunnels[0]
	tunnel1.port = 23457 // changing port should trigger a restart
	manager.refreshTunnels(baseCtx)
	manager.refreshSupervisors(tunnelCtx)
	waitCtx, cancel := context.WithTimeout(tunnelCtx, 1*time.Second)
	defer cancel()
	assert.Truef(t, tunnel1.WaitForStop(waitCtx), "tunnel %d stopped", tunnel1.id)
}

type mockTunnel struct {
	id      uuid.UUID
	started bool
	stopped bool
	port    int
}

func newMockTunnel(port int) *mockTunnel {
	return &mockTunnel{
		id:      uuid.New(),
		started: false,
		port:    port,
	}
}

func (m *mockTunnel) Start(ctx context.Context, listener *net.TCPListener, fn StatusUpdateFn) error {
	m.started = true
	<-ctx.Done()
	m.stopped = true

	return nil
}

func (m *mockTunnel) GetID() uuid.UUID {
	return m.id
}

func (m *mockTunnel) WaitForStart(ctx context.Context) bool {
	for {
		select {
		case <-ctx.Done():
			return false

		default:
			if m.started {
				return true
			}
		}
	}
}

func (m *mockTunnel) WaitForStop(ctx context.Context) bool {
	for {
		select {
		case <-ctx.Done():

			return false
		default:
			if m.stopped {
				return true
			}
		}
	}
}

func (m *mockTunnel) GetConnectionDetails(discovery.DiscoveryService) (ConnectionDetails, error) {
	return ConnectionDetails{
		Host: "127.0.0.1",
		Port: m.port,
	}, nil
}

func (m *mockTunnel) Equal(i interface{}) bool {
	v, ok := i.(mockTunnel)
	if !ok {
		return false
	}
	return m.id == v.id && m.port == v.port
}
