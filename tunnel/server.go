package tunnel

import (
	"context"
	"github.com/DataDog/datadog-go/statsd"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/pkg/errors"
	"time"
)

type Server struct {
	SQL sqlClient

	Stats statsd.ClientInterface

	normalTunnels  *Manager
	reverseTunnels *Manager
}

type sqlClient interface {
	CreateReverseTunnel(ctx context.Context, data postgres.ReverseTunnel) (postgres.ReverseTunnel, error)
	GetReverseTunnel(ctx context.Context, id uuid.UUID) (postgres.ReverseTunnel, error)
	ListReverseActiveTunnels(ctx context.Context) ([]postgres.ReverseTunnel, error)
	GetReverseTunnelAuthorizedKeys(ctx context.Context, tunnelID uuid.UUID) ([]postgres.Key, error)

	CreateNormalTunnel(ctx context.Context, data postgres.NormalTunnel) (postgres.NormalTunnel, error)
	GetNormalTunnel(ctx context.Context, id uuid.UUID) (postgres.NormalTunnel, error)
	ListNormalActiveTunnels(ctx context.Context) ([]postgres.NormalTunnel, error)
	GetNormalTunnelPrivateKeys(ctx context.Context, tunnelID uuid.UUID) ([]postgres.Key, error)

	AddKeyAndAttachToTunnel(ctx context.Context, tunnelType string, tunnelID uuid.UUID, keyType string, contents string) error
	AuthorizeKeyForTunnel(ctx context.Context, tunnelType string, tunnelID uuid.UUID, keyID uuid.UUID) error
}

const managerRefreshDuration = 1 * time.Second
const tunnelRestartInterval = 15 * time.Second // how long to wait after a tunnel crashes

func NewServer(sql sqlClient, stats statsd.ClientInterface, options SSHOptions) Server {
	return Server{
		SQL:   sql,
		Stats: stats,

		reverseTunnels: newManager(
			createReverseTunnelListFunc(sql.ListReverseActiveTunnels, reverseTunnelServices{sql}),
			options, managerRefreshDuration, tunnelRestartInterval,
		),
		normalTunnels: newManager(
			createNormalTunnelListFunc(sql.ListNormalActiveTunnels, normalTunnelServices{sql}),
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

	return &GetTunnelResponse{
		TunnelType:        tunnelType,
		Tunnel:            tunnel,
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
