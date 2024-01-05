package tunnel

import (
	"context"
	"fmt"
	"github.com/DataDog/datadog-go/statsd"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/stats"
	"github.com/hightouchio/passage/tunnel/discovery"
	"github.com/hightouchio/passage/tunnel/discovery/static"
	"net"
	"testing"
	"time"
)

type dummyTunnel struct {
}

func (d dummyTunnel) Start(ctx context.Context, listener *net.TCPListener, statusUpdate chan<- StatusUpdate) error {
	time.Sleep(10 * time.Millisecond)
	return fmt.Errorf("bad tunnel")
}

func (d dummyTunnel) GetConnectionDetails(service discovery.Service) (ConnectionDetails, error) {
	return ConnectionDetails{Host: "", Port: 0}, nil
}

func (d dummyTunnel) GetID() uuid.UUID {
	return uuid.New()
}

func (d dummyTunnel) Equal(i interface{}) bool {
	return true
}

func TestSupervisor_Profile(t *testing.T) {
	st := stats.New(&statsd.NoOpClient{})
	tunnel := dummyTunnel{}
	supervisor := NewSupervisor(tunnel, st, TunnelOptions{BindHost: "0.0.0.0"}, 50*time.Millisecond, static.Discovery{})

	timer := time.NewTimer(60 * time.Second)
	go func() {
		defer timer.Stop()
		<-timer.C
		t.Logf("timer done\n")
		supervisor.Stop()
	}()

	t.Logf("start")
	supervisor.Start()
	t.Logf("stop")
}
