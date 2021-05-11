package tunnel

import (
	"context"
	"github.com/hightouchio/passage/pkg/ssh"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/pkg/errors"
)

type Server struct {
	HostKey []byte

	SQL sqlClient
}

type sqlClient interface {
	CreateReverseTunnel(ctx context.Context, data postgres.ReverseTunnel) (postgres.ReverseTunnel, error)
	GetReverseTunnel(ctx context.Context, id int) (postgres.ReverseTunnel, error)
	ListReverseTunnels(ctx context.Context) ([]postgres.ReverseTunnel, error)
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
	var keys ssh.KeyPair
	if req.PublicKey != "" {
		if !ssh.IsValidPublicKey(req.PublicKey) {
			return nil, errors.New("invalid public key")
		}

		keys = ssh.KeyPair{PublicKey: req.PublicKey}
	} else {
		var err error
		keys, err = ssh.GenerateKeyPair()
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