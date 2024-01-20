package tunnel

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/stats"
	"github.com/hightouchio/passage/tunnel/discovery"
	"github.com/hightouchio/passage/tunnel/keystore"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/pkg/errors"
)

// API provides a source of truth for Tunnel configuration. It serves remote clients via HTTP APIs, as well as Manager instances via an exported ListFunc
type API struct {
	SQL              sqlClient
	DiscoveryService discovery.Service
	Keystore         keystore.Keystore
	Stats            stats.Stats
}

// GetNormalTunnels is a ListFunc which returns the set of NormalTunnel[] that should be run.
func (s API) GetNormalTunnels(ctx context.Context) ([]NormalTunnel, error) {
	normalTunnels, err := s.SQL.ListNormalActiveTunnels(ctx)
	if err != nil {
		return []NormalTunnel{}, err
	}

	// convert all the SQL records to our primary struct
	tunnels := make([]NormalTunnel, len(normalTunnels))
	for i, record := range normalTunnels {
		tunnel := normalTunnelFromSQL(record)
		tunnels[i] = tunnel
	}

	return tunnels, nil
}

// GetReverseTunnels is a ListFunc which returns the set of ReverseTunnel[] that should be run.
func (s API) GetReverseTunnels(ctx context.Context) ([]ReverseTunnel, error) {
	reverseTunnels, err := s.SQL.ListReverseActiveTunnels(ctx)
	if err != nil {
		return []ReverseTunnel{}, err
	}

	// convert all the SQL records to our primary struct
	tunnels := make([]ReverseTunnel, len(reverseTunnels))
	for i, record := range reverseTunnels {
		tunnel := reverseTunnelFromSQL(record)
		tunnels[i] = tunnel
	}

	return tunnels, nil
}

type GetTunnelRequest struct {
	ID uuid.UUID
}

type GetTunnelResponse struct {
	TunnelType `json:"type"`
	Tunnel     `json:"tunnel"`

	Instances []discovery.TunnelInstance `json:"-"`

	// DEPRECATED
	*ConnectionDetails `json:"connection"`
	// DEPRECATED
	Healthchecks []HealthcheckDetails `json:"healthchecks"`
}

type ConnectionDetails struct {
	Host string `json:"host"`
	Port uint32 `json:"port"`
}

type HealthcheckDetails struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"reason"`
}

// GetTunnel returns the connection details for the tunnel, so Hightouch can connect using it
func (s API) GetTunnel(ctx context.Context, req GetTunnelRequest) (*GetTunnelResponse, error) {
	tunnel, tunnelType, err := findTunnel(ctx, s.SQL, req.ID)
	if errors.Is(err, postgres.ErrTunnelNotFound) {
		return nil, postgres.ErrTunnelNotFound
	} else if err != nil {
		return nil, errors.Wrap(err, "error fetching tunnel")
	}

	response := GetTunnelResponse{
		TunnelType: tunnelType,
		Tunnel:     tunnel,
	}

	tunnelDetails, err := s.DiscoveryService.GetTunnel(ctx, req.ID)
	// If we couldn't get the tunnel from service discovery, just return early
	if err != nil {
		return &response, nil
	}

	response.Instances = tunnelDetails.Instances

	// Find the most healthy tunnel instance
	bestInstance := tunnelDetails.Instances[0]
	for _, instance := range tunnelDetails.Instances {
		if instance.Status == discovery.HealthcheckPassing {
			bestInstance = instance
			break
		}
	}

	// Populate connection details
	response.ConnectionDetails = &ConnectionDetails{
		Host: bestInstance.Host,
		Port: bestInstance.Port,
	}

	// Populate healthchecks
	response.Healthchecks = make([]HealthcheckDetails, len(bestInstance.Checks))
	for i, check := range bestInstance.Checks {
		response.Healthchecks[i] = HealthcheckDetails{
			ID:      check.ID,
			Status:  string(check.Status),
			Message: check.Message,
		}
	}

	return &response, nil
}

type UpdateTunnelRequest struct {
	ID           uuid.UUID              `json:"id"`
	UpdateFields map[string]interface{} `json:"-"`
}

type UpdateTunnelResponse struct {
	ID     uuid.UUID `json:"id"`
	Tunnel `json:"tunnel"`
}

func (s API) UpdateTunnel(ctx context.Context, req UpdateTunnelRequest) (*UpdateTunnelResponse, error) {
	// Get initial tunnel to determine type.
	tunnel, tunnelType, err := findTunnel(ctx, s.SQL, req.ID)
	if err == postgres.ErrTunnelNotFound {
		return nil, postgres.ErrTunnelNotFound
	} else if err != nil {
		return nil, errors.Wrap(err, "error fetching tunnel")
	}

	// Map the input fields.
	mapUpdateFields := func(fields map[string]interface{}, mapping map[string]string) map[string]interface{} {
		output := make(map[string]interface{})
		for inputKey, outputKey := range mapping {
			if inputVal, ok := fields[inputKey]; ok { // input has key
				output[outputKey] = inputVal
			}
		}
		return output
	}

	// Update tunnel
	switch tunnelType {
	case Normal:
		var newTunnel postgres.NormalTunnel
		newTunnel, err = s.SQL.UpdateNormalTunnel(ctx, req.ID, mapUpdateFields(req.UpdateFields, map[string]string{
			"enabled":     "enabled",
			"serviceHost": "service_host",
			"servicePort": "service_port",
			"sshHost":     "ssh_host",
			"sshPort":     "ssh_port",
			"sshUser":     "ssh_user",
		}))
		tunnel = normalTunnelFromSQL(newTunnel)
	case Reverse:
		var newTunnel postgres.ReverseTunnel
		newTunnel, err = s.SQL.UpdateReverseTunnel(ctx, req.ID, mapUpdateFields(req.UpdateFields, map[string]string{
			"enabled": "enabled",
		}))
		tunnel = reverseTunnelFromSQL(newTunnel)
	default:
		return nil, fmt.Errorf("invalid tunnel type %s", tunnelType)
	}
	if err != nil {
		return nil, errors.Wrap(err, "could not update postgres")
	}

	return &UpdateTunnelResponse{ID: req.ID, Tunnel: tunnel}, nil
}

type DeleteTunnelRequest struct {
	ID uuid.UUID
}

type DeleteTunnelResponse struct{}

// DeleteTunnel returns the connection details for the tunnel, so Hightouch can connect using it
func (s API) DeleteTunnel(ctx context.Context, req DeleteTunnelRequest) (*DeleteTunnelResponse, error) {
	err := s.SQL.DeleteTunnel(ctx, req.ID)
	if err == postgres.ErrTunnelNotFound {
		return nil, postgres.ErrTunnelNotFound
	} else if err != nil {
		return nil, errors.Wrap(err, "error deleting tunnel")
	}

	return &DeleteTunnelResponse{}, nil
}

type CheckTunnelRequest struct {
	ID uuid.UUID
}

type CheckTunnelResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// CheckTunnel identifies a currently running tunnel, gets connection details, and attempts a connection
func (s API) CheckTunnel(ctx context.Context, req CheckTunnelRequest) (*CheckTunnelResponse, error) {
	details, err := s.GetTunnel(ctx, GetTunnelRequest{ID: req.ID})
	if err != nil {
		return nil, errors.Wrap(err, "could not get connection details")
	}

	if len(details.Healthchecks) == 0 {
		return &CheckTunnelResponse{Success: false, Error: "Tunnel is not online"}, nil
	}

	for _, check := range details.Healthchecks {
		if discovery.HealthcheckStatus(check.Status) != discovery.HealthcheckPassing {
			return &CheckTunnelResponse{Success: false, Error: check.Message}, nil
		}
	}

	return &CheckTunnelResponse{Success: true}, nil
}

type sqlClient interface {
	CreateReverseTunnel(ctx context.Context, data postgres.ReverseTunnel, authorizedKeys []uuid.UUID) (postgres.ReverseTunnel, error)
	GetReverseTunnel(ctx context.Context, id uuid.UUID) (postgres.ReverseTunnel, error)
	UpdateReverseTunnel(ctx context.Context, id uuid.UUID, data map[string]interface{}) (postgres.ReverseTunnel, error)
	ListReverseActiveTunnels(ctx context.Context) ([]postgres.ReverseTunnel, error)

	CreateNormalTunnel(ctx context.Context, data postgres.NormalTunnel) (postgres.NormalTunnel, error)
	GetNormalTunnel(ctx context.Context, id uuid.UUID) (postgres.NormalTunnel, error)
	UpdateNormalTunnel(ctx context.Context, id uuid.UUID, data map[string]interface{}) (postgres.NormalTunnel, error)
	ListNormalActiveTunnels(ctx context.Context) ([]postgres.NormalTunnel, error)

	DeleteTunnel(ctx context.Context, tunnelID uuid.UUID) error

	AuthorizeKeyForTunnel(ctx context.Context, tunnelType string, tunnelID uuid.UUID, keyID uuid.UUID) error
}
