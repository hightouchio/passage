package ssh

import (
	"sync"
	"time"

	"github.com/hightouchio/passage/pkg/models"
)

const refreshDuration = time.Second

type Manager struct {
	tunnels     map[string]models.Tunnel
	supervisors map[string]supervisor
	lock        sync.Mutex
	once        sync.Once
}

func NewManager() *Manager {
	return &Manager{
		tunnels:     make(map[string]models.Tunnel),
		supervisors: make(map[string]supervisor),
		lock:        sync.Mutex{},
		once:        sync.Once{},
	}
}

func (m *Manager) SetTunnels(tunnels []models.Tunnel) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.tunnels = make(map[string]models.Tunnel)
	for _, tunnel := range tunnels {
		m.tunnels[tunnel.ID] = tunnel
	}

	m.once.Do(func() {
		go m.start()
	})
}

func (m *Manager) start() {
	ticker := time.NewTicker(refreshDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.refresh()
		}
	}
}

func (m *Manager) refresh() {
	m.lock.Lock()
	defer m.lock.Unlock()

	for tunnelID, tunnel := range m.tunnels {
		if _, ok := m.supervisors[tunnelID]; !ok {
			s := newSupervisor(tunnel)
			s.Start()
			m.supervisors[tunnelID] = *s
		}
	}
}
