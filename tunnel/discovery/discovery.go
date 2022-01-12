package discovery

import "github.com/google/uuid"

// DiscoveryService represents a service that can tell Passage where a tunnel is located.
type DiscoveryService interface {
	ResolveTunnelHost(tunnelType string, tunnelID uuid.UUID) (string, error)

	RegisterTunnel(tunnelID uuid.UUID, connectionDetails LocalConnectionDetails) error
	DeregisterTunnel(tunnelID uuid.UUID) error
}

type LocalConnectionDetails struct {
	Port       int
	SocketPath string
}
