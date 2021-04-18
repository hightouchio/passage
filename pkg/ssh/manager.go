package ssh

import (
	"sync"
	"time"

	"github.com/hightouchio/passage/pkg/models"
	"github.com/hightouchio/passage/pkg/ssh/supervisor"
)

const refreshDuration = time.Second

type Manager struct {
	bindHost           string
	hostKey            *string
	user               string
	tunnels            map[string]models.Tunnel
	reverseTunnels     map[string]models.ReverseTunnel
	normalSupervisors  map[string]supervisor.NormalSupervisor
	reverseSupervisors map[string]supervisor.ReverseSupervisor
	lock               sync.Mutex
	once               sync.Once
}

func NewManager(
	bindHost string,
	hostKey *string,
	user string,
) *Manager {
	return &Manager{
		bindHost:           bindHost,
		hostKey:            hostKey,
		user:               user,
		tunnels:            make(map[string]models.Tunnel),
		reverseTunnels:     make(map[string]models.ReverseTunnel),
		normalSupervisors:  make(map[string]supervisor.NormalSupervisor),
		reverseSupervisors: make(map[string]supervisor.ReverseSupervisor),
		lock:               sync.Mutex{},
		once:               sync.Once{},
	}
}

func (m *Manager) SetTunnels(
	tunnels []models.Tunnel,
	reverseTunnels []models.ReverseTunnel,
) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.tunnels = make(map[string]models.Tunnel)
	for i, tunnel := range tunnels {
		m.tunnels[tunnel.ID] = tunnels[i]
	}

	m.reverseTunnels = make(map[string]models.ReverseTunnel)
	for i, reverseTunnel := range reverseTunnels {
		m.reverseTunnels[reverseTunnel.ID] = reverseTunnels[i]
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
		if _, ok := m.normalSupervisors[tunnelID]; !ok {
			s := supervisor.NewNormalSupervisor(m.bindHost, m.user, tunnel)
			s.Start()
			m.normalSupervisors[tunnelID] = *s
		}
	}

	for reverseTunnelID, reverseTunnel := range m.reverseTunnels {
		if _, ok := m.reverseSupervisors[reverseTunnelID]; !ok {
			s := supervisor.NewReverseSupervisor(m.bindHost, m.hostKey, reverseTunnel)
			s.Start()
			m.reverseSupervisors[reverseTunnelID] = *s
		}
	}
}
