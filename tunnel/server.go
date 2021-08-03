package tunnel

import (
	"context"
	"fmt"
	"github.com/hightouchio/passage/stats"
	"time"

	"github.com/google/uuid"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/pkg/errors"
)

type Server struct {
	SQL sqlClient

	Stats stats.Stats

	normalTunnels  *Manager
	reverseTunnels *Manager
}

type sqlClient interface {
	CreateReverseTunnel(ctx context.Context, data postgres.ReverseTunnel) (postgres.ReverseTunnel, error)
	GetReverseTunnel(ctx context.Context, id uuid.UUID) (postgres.ReverseTunnel, error)
	UpdateReverseTunnel(ctx context.Context, id uuid.UUID, data map[string]interface{}) (postgres.ReverseTunnel, error)
	ListReverseActiveTunnels(ctx context.Context) ([]postgres.ReverseTunnel, error)
	GetReverseTunnelAuthorizedKeys(ctx context.Context, tunnelID uuid.UUID) ([]postgres.Key, error)

	CreateNormalTunnel(ctx context.Context, data postgres.NormalTunnel) (postgres.NormalTunnel, error)
	GetNormalTunnel(ctx context.Context, id uuid.UUID) (postgres.NormalTunnel, error)
	UpdateNormalTunnel(ctx context.Context, id uuid.UUID, data map[string]interface{}) (postgres.NormalTunnel, error)
	ListNormalActiveTunnels(ctx context.Context) ([]postgres.NormalTunnel, error)
	GetNormalTunnelPrivateKeys(ctx context.Context, tunnelID uuid.UUID) ([]postgres.Key, error)

	DeleteTunnel(ctx context.Context, tunnelID uuid.UUID) error

	AddKeyAndAttachToTunnel(ctx context.Context, tunnelType string, tunnelID uuid.UUID, keyType string, contents string) error
	AuthorizeKeyForTunnel(ctx context.Context, tunnelType string, tunnelID uuid.UUID, keyID uuid.UUID) error
}

const managerRefreshDuration = 1 * time.Second
const tunnelRestartInterval = 15 * time.Second // how long to wait after a tunnel crashes

func NewServer(sql sqlClient, st stats.Stats, options SSHOptions) Server {
	return Server{
		SQL:   sql,
		Stats: st,

		normalTunnels: newManager(
			st.WithTags(stats.Tags{"tunnelType": "normal"}),
			createNormalTunnelListFunc(sql.ListNormalActiveTunnels, normalTunnelServices{sql}),
			options, managerRefreshDuration, tunnelRestartInterval,
		),
		reverseTunnels: newManager(
			st.WithTags(stats.Tags{"tunnelType": "reverse"}),
			createReverseTunnelListFunc(sql.ListReverseActiveTunnels, reverseTunnelServices{sql}),
			options, managerRefreshDuration, tunnelRestartInterval,
		),
	}
}

func (s Server) StartNormalTunnels(ctx context.Context) {
	s.normalTunnels.Start(ctx)
}

func (s Server) StartReverseTunnels(ctx context.Context) {
	s.reverseTunnels.Start(ctx)
}

func (s Server) CheckNormalTunnels(ctx context.Context) error {
	return s.normalTunnels.Check(stats.InjectContext(ctx, s.Stats.WithTags(stats.Tags{"tunnelType": "normal"})))
}

func (s Server) CheckReverseTunnels(ctx context.Context) error {
	return s.reverseTunnels.Check(ctx)
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
func (s Server) GetTunnel(ctx context.Context, req GetTunnelRequest) (*GetTunnelResponse, error) {
	tunnel, tunnelType, err := findTunnel(ctx, s.SQL, req.ID)
	if err == postgres.ErrTunnelNotFound {
		return nil, postgres.ErrTunnelNotFound
	} else if err != nil {
		return nil, errors.Wrap(err, "error fetching tunnel")
	}

	connectionDetails, err := tunnel.GetConnectionDetails()
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

func (s Server) UpdateTunnel(ctx context.Context, req UpdateTunnelRequest) (*UpdateTunnelResponse, error) {
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
	case TunnelType("normal"):
		var newTunnel postgres.NormalTunnel
		newTunnel, err = s.SQL.UpdateNormalTunnel(ctx, req.ID, mapUpdateFields(req.UpdateFields, map[string]string{
			"serviceHost": "service_host",
			"servicePort": "service_port",
			"sshHost":     "ssh_host",
			"sshPort":     "ssh_port",
			"sshUser":     "ssh_user",
		}))
		tunnel = normalTunnelFromSQL(newTunnel)
	case TunnelType("reverse"):
		var newTunnel postgres.ReverseTunnel
		newTunnel, err = s.SQL.UpdateReverseTunnel(ctx, req.ID, mapUpdateFields(req.UpdateFields, map[string]string{}))
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
func (s Server) DeleteTunnel(ctx context.Context, req DeleteTunnelRequest) (*DeleteTunnelResponse, error) {
	err := s.SQL.DeleteTunnel(ctx, req.ID)
	if err == postgres.ErrTunnelNotFound {
		return nil, postgres.ErrTunnelNotFound
	} else if err != nil {
		return nil, errors.Wrap(err, "error deleting tunnel")
	}

	return &DeleteTunnelResponse{}, nil
}
