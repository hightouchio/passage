package discovery

import "github.com/google/uuid"

// DiscoveryService represents a service that can tell Passage where a tunnel is located.
type DiscoveryService interface {
	RegisterTunnel(id uuid.UUID, port int) error
	DeregisterTunnel(id uuid.UUID) error

	GetTunnel(id uuid.UUID) (TunnelDetails, error)

	UpdateHealth(id uuid.UUID, status, message string) error
}

type TunnelDetails struct {
	Host         string
	Port         int
	Status       string
	StatusReason string
}

// Consul healthcheck status codes
const (
	TunnelUnhealthy = "critical"
	TunnelWarning   = "warning"
	TunnelHealthy   = "passing"
)
