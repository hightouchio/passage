package tunnel

import (
	"context"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/tunnel/discovery"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/pkg/errors"
)

type Tunnel interface {
	Start(context.Context, TunnelOptions) error
	GetConnectionDetails(discovery.DiscoveryService) (ConnectionDetails, error)

	GetID() uuid.UUID
	Equal(any) bool
}

type TunnelOptions struct {
	BindHost string
}

// ConnectionDetails describes how the SaaS will use the tunnel
type ConnectionDetails struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

//goland:noinspection GoNameStartsWithPackageName
type TunnelType string

const (
	Normal  = "normal"
	Reverse = "reverse"
)

type CreateNormalTunnelRequest struct {
	NormalTunnel

	CreateKeyPair bool        `json:"createKeyPair"`
	Keys          []uuid.UUID `json:"keys"`
}

func (r CreateNormalTunnelRequest) Validate() error {
	re := newRequestErrors()
	if r.SSHHost == "" {
		re.addError("sshHost is required")
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

	PublicKey         *string `json:"publicKey,omitempty"`
	ConnectionDetails `json:"connection,omitempty"`
}

const defaultSSHPort = 22

func (s API) CreateNormalTunnel(ctx context.Context, request CreateNormalTunnelRequest) (*CreateNormalTunnelResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	// set default SSH port
	if request.SSHPort == 0 {
		request.SSHPort = defaultSSHPort
	}

	// insert into DB
	record, err := s.SQL.CreateNormalTunnel(ctx, sqlFromNormalTunnel(request.NormalTunnel))
	if err != nil {
		return nil, errors.Wrap(err, "could not insert")
	}

	// add keys
	for _, keyID := range request.Keys {
		if err := s.SQL.AuthorizeKeyForTunnel(ctx, Normal, record.ID, keyID); err != nil {
			return nil, errors.Wrapf(err, "could not add key %d", keyID)
		}
	}

	tunnel := normalTunnelFromSQL(record)
	connectionDetails, err := tunnel.GetConnectionDetails(s.DiscoveryService)
	if err != nil {
		return nil, errors.Wrap(err, "could not get connection details")
	}
	response := &CreateNormalTunnelResponse{Tunnel: tunnel, ConnectionDetails: connectionDetails}

	// if requested, we will generate a keypair and return the public key to the user
	if request.CreateKeyPair {
		keyId := uuid.New()
		keyPair, err := GenerateKeyPair()
		if err != nil {
			return nil, errors.Wrap(err, "could not generate keypair")
		}

		// insert into Keystore
		if err := s.Keystore.Set(ctx, keyId, keyPair.PrivateKey); err != nil {
			return nil, errors.Wrap(err, "could not set key")
		}

		// add to DB and attach to tunnel
		if err := s.SQL.AuthorizeKeyForTunnel(ctx, Normal, record.ID, keyId); err != nil {
			return nil, errors.Wrap(err, "could not auth key for tunnel")
		}

		// return the public key to the user
		keyString := string(keyPair.PublicKey)
		response.PublicKey = &keyString
	}

	return response, nil
}

type CreateReverseTunnelRequest struct {
	ReverseTunnel

	Keys          []uuid.UUID `json:"keys"`
	CreateKeyPair bool        `json:"createKeyPair"`
}

type CreateReverseTunnelResponse struct {
	Tunnel `json:"tunnel"`

	PrivateKey        *string `json:"privateKeyBase64,omitempty"`
	ConnectionDetails `json:"connection,omitempty"`
}

func (s API) CreateReverseTunnel(ctx context.Context, request CreateReverseTunnelRequest) (*CreateReverseTunnelResponse, error) {
	var tunnelData postgres.ReverseTunnel

	record, err := s.SQL.CreateReverseTunnel(ctx, tunnelData)
	if err != nil {
		return nil, errors.Wrap(err, "could not insert")
	}

	// add keys
	for _, keyID := range request.Keys {
		if err := s.SQL.AuthorizeKeyForTunnel(ctx, Reverse, record.ID, keyID); err != nil {
			return nil, errors.Wrapf(err, "could not add key %d", keyID)
		}
	}

	tunnel := reverseTunnelFromSQL(record)
	connectionDetails, err := tunnel.GetConnectionDetails(s.DiscoveryService)
	if err != nil {
		return nil, errors.Wrap(err, "could not get connection details")
	}
	response := &CreateReverseTunnelResponse{Tunnel: tunnel, ConnectionDetails: connectionDetails}

	// if requested, we will generate a keypair and return the public key to the user
	if request.CreateKeyPair {
		keyId := uuid.New()
		keyPair, err := GenerateKeyPair()
		if err != nil {
			return nil, errors.Wrap(err, "could not generate keypair")
		}

		// insert into Keystore
		if err := s.Keystore.Set(ctx, keyId, keyPair.PublicKey); err != nil {
			return nil, errors.Wrap(err, "could not set key")
		}

		// add to DB and attach to tunnel
		if err := s.SQL.AuthorizeKeyForTunnel(ctx, Reverse, record.ID, keyId); err != nil {
			return nil, errors.Wrap(err, "could not auth key for tunnel")
		}

		// return the public key to the user
		b64 := keyPair.Base64PrivateKey()
		response.PrivateKey = &b64
	}

	return response, nil
}

// findTunnel finds whichever tunnel type matches the UUID
func findTunnel(ctx context.Context, sql sqlClient, id uuid.UUID) (Tunnel, TunnelType, error) {
	// Reverse funnel first
	reverseTunnel, err := sql.GetReverseTunnel(ctx, id)
	if err == nil {
		return reverseTunnelFromSQL(reverseTunnel), Reverse, nil
	} else if err != postgres.ErrTunnelNotFound {
		// internal server error
		return nil, "", errors.Wrap(err, "could not fetch from database")
	}

	// Normal tunnel next
	normalTunnel, err := sql.GetNormalTunnel(ctx, id)
	if err == nil {
		return normalTunnelFromSQL(normalTunnel), Normal, nil
	} else if err != postgres.ErrTunnelNotFound {
		// internal server error
		return nil, "", errors.Wrap(err, "could not fetch from database")
	}

	return nil, "", postgres.ErrTunnelNotFound
}
