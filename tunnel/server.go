package tunnel

import (
	"context"
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
	GetReverseTunnel(ctx context.Context, id int) (postgres.ReverseTunnel, error)
	ListReverseTunnels(ctx context.Context) ([]postgres.ReverseTunnel, error)

	ListNormalTunnels(ctx context.Context) ([]postgres.NormalTunnel, error)
}

const managerRefreshDuration = 1 * time.Second
const supervisorRetryDuration = 1 * time.Second

func NewServer(sql sqlClient, options SSHOptions) Server {
	return Server{
		SQL: sql,

		reverseTunnels: newManager(createReverseTunnelListFunc(sql.ListReverseTunnels), options, managerRefreshDuration, supervisorRetryDuration),
		normalTunnels:  newManager(createNormalTunnelListFunc(sql.ListNormalTunnels), options, managerRefreshDuration, supervisorRetryDuration),
	}
}

// StartWorkers kicks off internal worker processes
func (s Server) StartWorkers() {
	s.reverseTunnels.Start()
	s.normalTunnels.Start()
}

type NewReverseTunnelRequest struct {
	PublicKey string `json:"publicKey"`
}
type NewReverseTunnelResponse struct {
	ID         int     `json:"id"`
	PrivateKey *string `json:"privateKeyBase64,omitempty"`
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

	tunnel, err := s.SQL.CreateReverseTunnel(ctx, postgres.ReverseTunnel{PublicKey: keys.PublicKey})
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
