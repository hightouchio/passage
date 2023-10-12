package discovery

import "github.com/google/uuid"

// DiscoveryService represents a service that can tell Passage where a tunnel is located.
type DiscoveryService interface {
	RegisterTunnel(id uuid.UUID, port int) error
	DeregisterTunnel(id uuid.UUID) error

	GetTunnel(id uuid.UUID) (TunnelDetails, error)

	MarkHealthy(id uuid.UUID, reason string) error
	MarkUnhealthy(id uuid.UUID, reason string) error
}

type TunnelDetails struct {
	Host         string
	Port         int
	Status       string
	StatusReason string
}
