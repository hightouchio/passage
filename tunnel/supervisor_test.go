package tunnel

import (
	"context"
	"fmt"
	"github.com/DataDog/datadog-go/statsd"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/stats"
	"github.com/hightouchio/passage/tunnel/discovery"
	"github.com/sirupsen/logrus"
	"testing"
	"time"
)

type dummyTunnel struct {
}

func (d dummyTunnel) Start(ctx context.Context, options TunnelOptions) error {
	time.Sleep(10 * time.Millisecond)
	return fmt.Errorf("bad tunnel")
}

func (d dummyTunnel) GetConnectionDetails(service discovery.DiscoveryService) (ConnectionDetails, error) {
	return ConnectionDetails{Host: "", Port: 0}, nil
}

func (d dummyTunnel) GetID() uuid.UUID {
	return uuid.New()
}

func (d dummyTunnel) Equal(i interface{}) bool {
	return true
}

func TestSupervisor_Profile(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	st := stats.New(&statsd.NoOpClient{}, logger)
	tunnel := dummyTunnel{}
	supervisor := NewSupervisor(tunnel, st, TunnelOptions{BindHost: "0.0.0.0"}, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timer := time.NewTimer(60 * time.Second)
	go func() {
		defer timer.Stop()
		<-timer.C
		logger.Info("timer done")
		supervisor.Stop()
		cancel()
	}()

	t.Logf("start")
	supervisor.Start(ctx)
	t.Logf("stop")
}
