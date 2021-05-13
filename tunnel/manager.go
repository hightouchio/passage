package tunnel

import (
	"context"
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

	RefreshDuration         time.Duration
	SupervisorRetryDuration time.Duration

	tunnels     map[uuid.UUID]Tunnel
	supervisors map[uuid.UUID]*Supervisor

	lock sync.Mutex
	once sync.Once
}

func newManager(listFunc ListFunc, sshOptions SSHOptions, refreshDuration, supervisorRetryDuration time.Duration) *Manager {
	return &Manager{
		ListFunc: listFunc,

		SSHOptions:              sshOptions,
		RefreshDuration:         refreshDuration,
		SupervisorRetryDuration: supervisorRetryDuration,

		tunnels:     make(map[uuid.UUID]Tunnel),
		supervisors: make(map[uuid.UUID]*Supervisor),
	}
}

func (m *Manager) Start(ctx context.Context) {
	go m.startDatabaseWorker(ctx)
	go m.startSupervisorWorker(ctx)
}

func (m *Manager) startSupervisorWorker(ctx context.Context) {
	ticker := time.NewTicker(m.RefreshDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.refreshSupervisors(ctx)

		case <-ctx.Done():
			return
		}
	}
}

func (m *Manager) refreshSupervisors(ctx context.Context) {
	m.lock.Lock()
	defer m.lock.Unlock()

	// start new supervisors
	for tunnelID, tunnel := range m.tunnels {
		if _, ok := m.supervisors[tunnelID]; !ok {
			supervisor := NewSupervisor(tunnel, m.SSHOptions, m.SupervisorRetryDuration)
			go supervisor.Start(ctx)
			m.supervisors[tunnelID] = supervisor
		}
	}

	// shut down old supervisors
	for tunnelID, supervisor := range m.supervisors {
		// if this supervisor's tunnel ID no longer appears in the list of tunnels
		if _, ok := m.tunnels[tunnelID]; !ok {
			supervisor.Stop()
			delete(m.supervisors, tunnelID)
		}
	}
}

func (m *Manager) startDatabaseWorker(ctx context.Context) {
	ticker := time.NewTicker(m.RefreshDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := m.refreshTunnels(ctx); err != nil {
				logrus.WithError(err).Error("could not refresh tunnels from DB")
			}

		case <-ctx.Done():
			return
		}
	}
}

// refreshTunnels calls out to the list func and swaps out the in-memory tunnel representation with the new data that we received
func (m *Manager) refreshTunnels(ctx context.Context) error {
	tunnels, err := m.ListFunc(ctx)
	if err != nil {
		return errors.Wrap(err, "could not list tunnels")
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	// write new tunnels
	m.tunnels = make(map[uuid.UUID]Tunnel)
	for i, tunnel := range tunnels {
		m.tunnels[tunnel.GetID()] = tunnels[i]
	}

	return nil
}
