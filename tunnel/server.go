package tunnel

import (
	"context"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/pkg/errors"
	"time"
)

type Server struct {
	SQL sqlClient

	normalTunnels  *Manager
	reverseTunnels *Manager
}

type sqlClient interface {
	CreateReverseTunnel(ctx context.Context, data postgres.ReverseTunnel) (postgres.ReverseTunnel, error)
	GetReverseTunnel(ctx context.Context, id uuid.UUID) (postgres.ReverseTunnel, error)
	ListReverseTunnels(ctx context.Context) ([]postgres.ReverseTunnel, error)
	GetReverseTunnelAuthorizedKeys(ctx context.Context, tunnelID uuid.UUID) ([]postgres.Key, error)

	CreateNormalTunnel(ctx context.Context, data postgres.NormalTunnel) (postgres.NormalTunnel, error)
	GetNormalTunnel(ctx context.Context, id uuid.UUID) (postgres.NormalTunnel, error)
	ListNormalTunnels(ctx context.Context) ([]postgres.NormalTunnel, error)
	GetNormalTunnelPrivateKeys(ctx context.Context, tunnelID uuid.UUID) ([]postgres.Key, error)
}

const managerRefreshDuration = 1 * time.Second
const supervisorRetryDuration = 1 * time.Second

func NewServer(sql sqlClient, options SSHOptions) Server {
	return Server{
		SQL: sql,

		reverseTunnels: newManager(
			createReverseTunnelListFunc(sql.ListReverseTunnels, reverseTunnelServices{sql}),
			options, managerRefreshDuration, supervisorRetryDuration,
		),
		normalTunnels: newManager(
			createNormalTunnelListFunc(sql.ListNormalTunnels, normalTunnelServices{sql}),
			options, managerRefreshDuration, supervisorRetryDuration,
		),
	}
}

func (s Server) StartNormalTunnels() {
	s.normalTunnels.Start()
}

func (s Server) StopNormalTunnels() {
}

func (s Server) StartReverseTunnels() {
	s.reverseTunnels.Start()
}

func (s Server) StopReverseTunnels() {
}

type GetConnectionDetailsRequest struct {
	ID uuid.UUID
}

type GetConnectionDetailsResponse struct {
	ConnectionDetails
	TunnelType `json:"type"`
}

// GetConnectionDetails returns the connection details for the tunnel, so Hightouch can connect using it
func (s Server) GetConnectionDetails(ctx context.Context, req GetConnectionDetailsRequest) (*GetConnectionDetailsResponse, error) {
	// identify tunnel
	tunnel, tunnelType, err := s.getTunnel(ctx, req.ID)
	if err == postgres.ErrTunnelNotFound {
		return nil, postgres.ErrTunnelNotFound
	} else if err != nil {
		return nil, errors.Wrap(err, "error fetching tunnel")
	}

	return &GetConnectionDetailsResponse{
		TunnelType:        tunnelType,
		ConnectionDetails: tunnel.GetConnectionDetails(),
	}, nil
}

type NewReverseTunnelRequest struct {
	PublicKey string `json:"publicKey"`
}
type NewReverseTunnelResponse struct {
	ID         uuid.UUID `json:"id"`
	PrivateKey *string   `json:"privateKeyBase64,omitempty"`
}

func (s Server) NewReverseTunnel(ctx context.Context, req NewReverseTunnelRequest) (*NewReverseTunnelResponse, error) {
	// check if we need to generate a new keypair or can just use what the customer provided
	var keys KeyPair
	if req.PublicKey != "" {
		if !IsValidPublicKey(req.PublicKey) {
			return nil, errors.New("invalid public key")
		}
		keys = KeyPair{PublicKey: req.PublicKey}
	} else {
		var err error
		keys, err = GenerateKeyPair()
		if err != nil {
			return nil, errors.New("could not generate key pair")
		}
	}

	tunnel, err := s.SQL.CreateReverseTunnel(ctx, postgres.ReverseTunnel{})
	if err != nil {
		return nil, errors.Wrap(err, "could not create reverse tunnel")
	}

	response := &NewReverseTunnelResponse{ID: tunnel.ID}
	// attach private key if necessary
	if keys.PrivateKey != "" {
		b64 := keys.Base64PrivateKey()
		response.PrivateKey = &b64
	}
	return response, nil
}
