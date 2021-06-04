package tunnel

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type ListFunc func(ctx context.Context) ([]Tunnel, error)

// Manager keeps track of the tunnels that need to be loaded in from the database, and the tunnels that need to be started up with a supervisor
type Manager struct {
	// ListFunc is the function that will list all tunnels that should be running
	ListFunc

	// SSHOptions are the config options for the SSH server that we start up
	SSHOptions

	RefreshDuration       time.Duration
	TunnelRestartInterval time.Duration

	tunnels     map[uuid.UUID]runningTunnel
	supervisors map[uuid.UUID]*Supervisor

	// lastRefresh records when the last successful refresh took place. indicating that nothing is frozen
	lastRefresh time.Time

	lock sync.Mutex
	once sync.Once
}

func newManager(listFunc ListFunc, sshOptions SSHOptions, refreshDuration, tunnelRestartInterval time.Duration) *Manager {
	return &Manager{
		ListFunc: listFunc,

		SSHOptions:            sshOptions,
		RefreshDuration:       refreshDuration,
		TunnelRestartInterval: tunnelRestartInterval,

		tunnels:     make(map[uuid.UUID]runningTunnel),
		supervisors: make(map[uuid.UUID]*Supervisor),
	}
}

func (m *Manager) Start(ctx context.Context) {
	m.startWorker(ctx)
}

func (m *Manager) startWorker(ctx context.Context) {
	ticker := time.NewTicker(m.RefreshDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := m.refreshTunnels(ctx); err != nil {
				logrus.WithError(err).Error("could not refresh tunnels from DB")
				continue
			}
			m.refreshSupervisors(ctx)
			m.lastRefresh = time.Now()

		case <-ctx.Done():
			return
		}
	}
}

func (m *Manager) refreshSupervisors(ctx context.Context) {
	m.lock.Lock()
	defer m.lock.Unlock()

	// shut down supervisors or supervisors that need a restart
	for tunnelID, supervisor := range m.supervisors {
		tunnel, stillExists := m.tunnels[tunnelID]
		if !stillExists || tunnel.needsRestart {
			supervisor.Stop()
			delete(m.supervisors, tunnelID)
		}
	}

	// start new supervisors
	for tunnelID, tunnel := range m.tunnels {
		if _, alreadyRunning := m.supervisors[tunnelID]; !alreadyRunning {
			supervisor := NewSupervisor(tunnel, m.SSHOptions, m.TunnelRestartInterval)
			go supervisor.Start(ctx)
			m.supervisors[tunnelID] = supervisor
		}
	}
}

// runningTunnel is useful because it has more stateful information about the tunnel such as whether or not it needs to restart
type runningTunnel struct {
	Tunnel
	needsRestart bool
}

// refreshTunnels calls out to the list func and swaps out the in-memory tunnel representation with the new data that we received
func (m *Manager) refreshTunnels(ctx context.Context) error {
	tunnels, err := m.ListFunc(ctx)
	if err != nil {
		return errors.Wrap(err, "could not list tunnels")
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	oldTunnels := m.tunnels
	newTunnels := make(map[uuid.UUID]runningTunnel, len(tunnels))

	for _, newTunnel := range tunnels {
		oldTunnel, oldExists := oldTunnels[newTunnel.GetID()]

		newTunnels[newTunnel.GetID()] = runningTunnel{
			Tunnel:       newTunnel,
			needsRestart: oldExists && !newTunnel.Equal(oldTunnel.Tunnel), // see if the critical fields have changed
		}
	}

	m.tunnels = newTunnels

	return nil
}

// Check performs a healthcheck on the manager
func (m *Manager) Check(ctx context.Context) error {
	maxDelay := m.RefreshDuration * 2
	secondsSinceLastRefresh := time.Now().Sub(m.lastRefresh)
	if maxDelay > secondsSinceLastRefresh {
		return fmt.Errorf("manager has not refreshed in %0.2f seconds", secondsSinceLastRefresh.Seconds())
	}

	return nil
}
