package discovery

import (
	"context"
	"github.com/google/uuid"
	"time"
)

// Service represents a service that can tell Passage where a tunnel is located.
type Service interface {
	Wait(ctx context.Context) error

	RegisterTunnel(ctx context.Context, id uuid.UUID, port int) error
	DeregisterTunnel(ctx context.Context, id uuid.UUID) error
	GetTunnel(ctx context.Context, id uuid.UUID) (TunnelDetails, error)

	RegisterHealthcheck(ctx context.Context, tunnelId uuid.UUID, options HealthcheckOptions) error
	DeregisterHealthcheck(ctx context.Context, tunnelId uuid.UUID, id string) error
	UpdateHealthcheck(ctx context.Context, tunnelId uuid.UUID, id string, status HealthcheckStatus, message string) error
}

type HealthcheckDetails struct {
	ID      string
	Status  string
	Message string
}

type TunnelDetails struct {
	Host string
	Port int

	Checks []HealthcheckDetails
}

type HealthcheckOptions struct {
	ID   string
	Name string
	TTL  time.Duration
}

type HealthcheckStatus string

// Healthcheck status codes
const (
	HealthcheckCritical HealthcheckStatus = "critical"
	HealthcheckWarning  HealthcheckStatus = "warning"
	HealthcheckPassing  HealthcheckStatus = "passing"
)
