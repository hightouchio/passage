package discovery

import "github.com/google/uuid"

// DiscoveryService represents a service that can tell Passage where a tunnel is located.
type DiscoveryService interface {
	RegisterTunnel(id uuid.UUID, port int) error
	DeregisterTunnel(id uuid.UUID) error

	GetTunnel(id uuid.UUID) (TunnelDetails, error)

	UpdateHealth(id uuid.UUID, status HealthcheckStatus, message string) error
}

type TunnelDetails struct {
	Host         string
	Port         int
	Status       string
	StatusReason string
}

type HealthcheckStatus string

// Healthcheck status codes
const (
	HealthcheckCritical HealthcheckStatus = "critical"
	HealthcheckWarning                    = "warning"
	HealthcheckPassing                    = "passing"
)
