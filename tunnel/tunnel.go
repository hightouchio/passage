package tunnel

import (
	"context"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/pkg/errors"
)

type Tunnel interface {
	Start(context.Context, SSHOptions) error

	GetID() uuid.UUID
	GetConnectionDetails() ConnectionDetails
}

type SSHOptions struct {
	BindHost string
	HostKey  []byte
}

// ConnectionDetails describes how the SaaS will use the tunnel
type ConnectionDetails struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

//goland:noinspection GoNameStartsWithPackageName
type TunnelType string

type CreateNormalTunnelRequest struct {
	NormalTunnel `json:"tunnel"`

	CreateKeyPair bool        `json:"createKeyPair"`
	Keys          []uuid.UUID `json:"keys"`
}

func (r CreateNormalTunnelRequest) Validate() error {
	re := newRequestErrors()
	if r.SSHHost == "" {
		re.addError("sshHost is required")
	}
	if r.SSHPort == 0 {
		re.addError("sshPort is required")
	}
	if r.ServiceHost == "" {
		re.addError("serviceHost is required")
	}
	if r.ServicePort == 0 {
		re.addError("servicePort is required")
	}
	if re.IsEmpty() {
		return nil
	}
	return re
}

type CreateNormalTunnelResponse struct {
	Tunnel `json:"tunnel"`

	PublicKey *string `json:"publicKey,omitempty"`
}

func (s Server) CreateNormalTunnel(ctx context.Context, request CreateNormalTunnelRequest) (*CreateNormalTunnelResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	// insert into DB
	record, err := s.SQL.CreateNormalTunnel(ctx, sqlFromNormalTunnel(request.NormalTunnel))
	if err != nil {
		return nil, errors.Wrap(err, "could not insert")
	}

	// add keys
	for _, keyID := range request.Keys {
		if err := s.SQL.AuthorizeKeyForTunnel(ctx, "normal", record.ID, keyID); err != nil {
			return nil, errors.Wrapf(err, "could not add key %d", keyID)
		}
	}

	response := &CreateNormalTunnelResponse{Tunnel: normalTunnelFromSQL(record)}

	// if requested, we will generate a keypair and return the public key to the user
	if request.CreateKeyPair {
		keyPair, err := GenerateKeyPair()
		if err != nil {
			return nil, errors.Wrap(err, "could not generate keypair")
		}

		// add to DB and attach to tunnel
		if err := s.SQL.AddKeyAndAttachToTunnel(ctx, "normal", record.ID, "private", keyPair.PrivateKey); err != nil {
			return nil, errors.Wrap(err, "could not add private key to tunnel")
		}

		// return the public key to the user
		response.PublicKey = &keyPair.PublicKey
	}

	return response, nil
}

type CreateReverseTunnelRequest struct {
	ReverseTunnel `json:"tunnel"`

	Keys          []uuid.UUID `json:"keys"`
	CreateKeyPair bool        `json:"createKeyPair"`
}

type CreateReverseTunnelResponse struct {
	Tunnel `json:"tunnel"`

	PrivateKey *string `json:"privateKeyBase64,omitempty"`
}

func (s Server) CreateReverseTunnel(ctx context.Context, request CreateReverseTunnelRequest) (*CreateReverseTunnelResponse, error) {
	var tunnelData postgres.ReverseTunnel

	record, err := s.SQL.CreateReverseTunnel(ctx, tunnelData)
	if err != nil {
		return nil, errors.Wrap(err, "could not insert")
	}

	// add keys
	for _, keyID := range request.Keys {
		if err := s.SQL.AuthorizeKeyForTunnel(ctx, "reverse", record.ID, keyID); err != nil {
			return nil, errors.Wrapf(err, "could not add key %d", keyID)
		}
	}

	response := &CreateReverseTunnelResponse{Tunnel: reverseTunnelFromSQL(record)}

	// if requested, we will generate a keypair and return the public key to the user
	if request.CreateKeyPair {
		keyPair, err := GenerateKeyPair()
		if err != nil {
			return nil, errors.Wrap(err, "could not generate keypair")
		}

		// add to DB and attach to tunnel
		if err := s.SQL.AddKeyAndAttachToTunnel(ctx, "reverse", record.ID, "public", keyPair.PublicKey); err != nil {
			return nil, errors.Wrap(err, "could not add public key to tunnel")
		}

		// return the public key to the user
		b64 := keyPair.Base64PrivateKey()
		response.PrivateKey = &b64
	}

	return response, nil
}

// findTunnel finds whichever tunnel type matches the UUID
func findTunnel(ctx context.Context, sql sqlClient, id uuid.UUID) (Tunnel, TunnelType, error) {
	// reverse funnel first
	reverseTunnel, err := sql.GetReverseTunnel(ctx, id)
	if err == nil {
		return reverseTunnelFromSQL(reverseTunnel), "reverse", nil
	} else if err != postgres.ErrTunnelNotFound {
		// internal server error
		return nil, "", errors.Wrap(err, "could not fetch from database")
	}

	// normal tunnel next
	normalTunnel, err := sql.GetNormalTunnel(ctx, id)
	if err == nil {
		return normalTunnelFromSQL(normalTunnel), "normal", nil
	} else if err != postgres.ErrTunnelNotFound {
		// internal server error
		return nil, "", errors.Wrap(err, "could not fetch from database")
	}

	return nil, "", postgres.ErrTunnelNotFound
}
