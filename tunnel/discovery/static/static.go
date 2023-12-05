package static

import (
	"github.com/google/uuid"
	"github.com/hightouchio/passage/tunnel/discovery"
)

// StaticDiscovery is a tunnel discovery provider that returns a static host
type Discovery struct {
	Host string
}

func (d Discovery) RegisterTunnel(id uuid.UUID, port int) error {
	return nil
}

func (d Discovery) DeregisterTunnel(id uuid.UUID) error {
	return nil
}

func (d Discovery) GetTunnel(id uuid.UUID) (discovery.TunnelDetails, error) {
	return discovery.TunnelDetails{}, nil
}

func (d Discovery) UpdateHealth(id uuid.UUID, status, message string) error {
	return nil
}

func (d Discovery) ResolveTunnelHost(_ string, _ uuid.UUID) (string, error) {
	return d.Host, nil
}
