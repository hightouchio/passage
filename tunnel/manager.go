package tunnel

import (
	"context"
	"github.com/apex/log"
	"github.com/pkg/errors"
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

	tunnels     map[int]Tunnel
	supervisors map[int]Supervisor

	lock sync.Mutex
	once sync.Once
}

func newManager(listFunc ListFunc, sshOptions SSHOptions, refreshDuration, supervisorRetryDuration time.Duration) *Manager {
	return &Manager{
		ListFunc: listFunc,

		SSHOptions:              sshOptions,
		RefreshDuration:         refreshDuration,
		SupervisorRetryDuration: supervisorRetryDuration,

		tunnels:     make(map[int]Tunnel),
		supervisors: make(map[int]Supervisor),
	}
}

func (m *Manager) startSupervisorWorker() {
	ticker := time.NewTicker(m.RefreshDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.refreshSupervisors()
		}
	}
}

func (m *Manager) refreshSupervisors() {
	m.lock.Lock()
	defer m.lock.Unlock()

	for tunnelID, tunnel := range m.tunnels {
		if _, ok := m.supervisors[tunnelID]; !ok {
			s := Supervisor{
				Tunnel:        tunnel,
				SSHOptions:    m.SSHOptions,
				RetryDuration: m.SupervisorRetryDuration,
			}
			go s.Start()
			// TODO: Implement cancellation

			m.supervisors[tunnelID] = s
		}
	}
}

func (m *Manager) startDatabaseWorker() {
	ticker := time.NewTicker(m.RefreshDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := m.refreshTunnels(context.Background()); err != nil {
				log.WithError(err).Error("could not refresh tunnels from DB")
			}
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

	m.tunnels = make(map[int]Tunnel)
	for i, tunnel := range tunnels {
		m.tunnels[tunnel.GetID()] = tunnels[i]
	}

	return nil
}
