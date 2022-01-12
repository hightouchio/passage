package static

import (
	"github.com/google/uuid"
	"github.com/hightouchio/passage/tunnel/discovery"
)

// Discovery is a tunnel discovery provider that returns a static host
type Discovery struct {
	Host string
}

func (d Discovery) RegisterTunnel(tunnelID uuid.UUID, connectionDetails discovery.LocalConnectionDetails) error {
	// no-op
	return nil
}

func (d Discovery) DeregisterTunnel(tunnelID uuid.UUID) error {
	// no-op
	return nil
}

func (d Discovery) ResolveTunnelHost(_ string, _ uuid.UUID) (string, error) {
	return d.Host, nil
}
