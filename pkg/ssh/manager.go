package ssh

import (
	"github.com/hightouchio/passage/pkg/models"
	"github.com/hightouchio/passage/pkg/ssh/clients"
	"github.com/hightouchio/passage/pkg/ssh/server"
)

type Manager struct {
	clientsManager *clients.Manager
	server         *server.Server
}

func NewManager() *Manager {
	return &Manager{
		clientsManager: clients.NewManager(),
		server:         server.NewServer(),
	}
}

func (m *Manager) SetTunnels(tunnels []models.Tunnel) {
	var normalTunnels []models.Tunnel
	var reverseTunnels []models.Tunnel
	for _, tunnel := range tunnels {
		switch tunnel.Type {
		case models.TunnelTypeNormal:
			normalTunnels = append(normalTunnels, tunnel)
		case models.TunnelTypeReverse:
			reverseTunnels = append(reverseTunnels, tunnel)
		}
	}
	m.clientsManager.SetTunnels(normalTunnels)
	m.server.SetTunnels(reverseTunnels)
}
