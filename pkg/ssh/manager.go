package ssh

import (
	"github.com/hightouchio/passage/pkg/models"
	"sync"
	"time"

	"github.com/hightouchio/passage/pkg/ssh/supervisor"
)

const refreshDuration = time.Second

type NormalRegistry map[string]models.Tunnel
type NormalSupervisor map[string]supervisor.NormalSupervisor

type ReverseRegistry map[int]models.ReverseTunnel
type ReverseSupervisor map[int]supervisor.ReverseSupervisor

type Manager struct {
	bindHost           string
	hostKey            []byte
	user               string
	tunnels            NormalRegistry
	reverseTunnels     ReverseRegistry
	normalSupervisors  NormalSupervisor
	reverseSupervisors ReverseSupervisor
	lock               sync.Mutex
	once               sync.Once
}

func NewManager(
	bindHost string,
	hostKey []byte,
	user string,
) *Manager {
	return &Manager{
		bindHost:           bindHost,
		hostKey:            hostKey,
		user:               user,
		tunnels:            make(NormalRegistry),
		reverseTunnels:     make(ReverseRegistry),
		normalSupervisors:  make(NormalSupervisor),
		reverseSupervisors: make(ReverseSupervisor),
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

	m.tunnels = make(NormalRegistry)
	for i, tunnel := range tunnels {
		m.tunnels[tunnel.ID] = tunnels[i]
	}

	m.reverseTunnels = make(ReverseRegistry)
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
