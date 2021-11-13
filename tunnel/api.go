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
	SQL sqlClient

	DiscoveryService discovery.DiscoveryService
	Keystore         keystore.Keystore
	Stats            stats.Stats

	SSHServerOptions SSHServerOptions
	SSHClientOptions SSHClientOptions
}

// GetStandardTunnels is a ListFunc which returns the set of StandardTunnel[] that should be run.
func (s API) GetStandardTunnels(ctx context.Context) ([]Tunnel, error) {
	standardTunnels, err := s.SQL.ListStandardActiveTunnels(ctx)
	if err != nil {
		return []Tunnel{}, err
	}

	services := standardTunnelServices{sql: s.SQL, keystore: s.Keystore}
	// convert all the SQL records to our primary struct
	tunnels := make([]Tunnel, len(standardTunnels))
	for i, record := range standardTunnels {
		tunnel := standardTunnelFromSQL(record)
		tunnel.services = services // inject dependencies
		tunnels[i] = tunnel
	}

	return tunnels, nil
}

// GetReverseTunnels is a ListFunc which returns the set of ReverseTunnel[] that should be run.
func (s API) GetReverseTunnels(ctx context.Context) ([]Tunnel, error) {
	reverseTunnels, err := s.SQL.ListReverseActiveTunnels(ctx)
	if err != nil {
		return []Tunnel{}, err
	}

	services := reverseTunnelServices{sql: s.SQL, keystore: s.Keystore}
	// convert all the SQL records to our primary struct
	tunnels := make([]Tunnel, len(reverseTunnels))
	for i, record := range reverseTunnels {
		tunnel := reverseTunnelFromSQL(record)
		tunnel.services = services // inject dependencies
		tunnel.serverOptions = s.SSHServerOptions
		tunnels[i] = tunnel
	}

	return tunnels, nil
}

type GetTunnelRequest struct {
	ID uuid.UUID
}

type GetTunnelResponse struct {
	TunnelType        `json:"type"`
	Tunnel            `json:"tunnel"`
	ConnectionDetails `json:"connection"`
}

// GetTunnel returns the connection details for the tunnel, so Hightouch can connect using it
func (s API) GetTunnel(ctx context.Context, req GetTunnelRequest) (*GetTunnelResponse, error) {
	tunnel, tunnelType, err := findTunnel(ctx, s.SQL, req.ID)
	if err == postgres.ErrTunnelNotFound {
		return nil, postgres.ErrTunnelNotFound
	} else if err != nil {
		return nil, errors.Wrap(err, "error fetching tunnel")
	}

	connectionDetails, err := tunnel.GetConnectionDetails(s.DiscoveryService)
	if err != nil {
		return nil, errors.Wrap(err, "could not get connection details")
	}

	return &GetTunnelResponse{
		TunnelType:        tunnelType,
		Tunnel:            tunnel,
		ConnectionDetails: connectionDetails,
	}, nil
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
	case "standard":
		var newTunnel postgres.StandardTunnel
		newTunnel, err = s.SQL.UpdateStandardTunnel(ctx, req.ID, mapUpdateFields(req.UpdateFields, map[string]string{
			"enabled":     "enabled",
			"serviceHost": "service_host",
			"servicePort": "service_port",
			"sshHost":     "ssh_host",
			"sshPort":     "ssh_port",
			"sshUser":     "ssh_user",
		}))
		tunnel = standardTunnelFromSQL(newTunnel)
	case "reverse":
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

type sqlClient interface {
	CreateReverseTunnel(ctx context.Context, data postgres.ReverseTunnel) (postgres.ReverseTunnel, error)
	GetReverseTunnel(ctx context.Context, id uuid.UUID) (postgres.ReverseTunnel, error)
	UpdateReverseTunnel(ctx context.Context, id uuid.UUID, data map[string]interface{}) (postgres.ReverseTunnel, error)
	ListReverseActiveTunnels(ctx context.Context) ([]postgres.ReverseTunnel, error)
	GetReverseTunnelAuthorizedKeys(ctx context.Context, tunnelID uuid.UUID) ([]postgres.Key, error)

	CreateStandardTunnel(ctx context.Context, data postgres.StandardTunnel) (postgres.StandardTunnel, error)
	GetStandardTunnel(ctx context.Context, id uuid.UUID) (postgres.StandardTunnel, error)
	UpdateStandardTunnel(ctx context.Context, id uuid.UUID, data map[string]interface{}) (postgres.StandardTunnel, error)
	ListStandardActiveTunnels(ctx context.Context) ([]postgres.StandardTunnel, error)
	GetStandardTunnelPrivateKeys(ctx context.Context, tunnelID uuid.UUID) ([]postgres.Key, error)

	DeleteTunnel(ctx context.Context, tunnelID uuid.UUID) error

	AuthorizeKeyForTunnel(ctx context.Context, tunnelType string, tunnelID uuid.UUID, keyID uuid.UUID) error
}
