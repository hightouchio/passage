package tunnel

import (
	"context"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/pkg/errors"
)

type Tunnel interface {
	GetID() uuid.UUID
	Start(context.Context, SSHOptions) error
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

type NewTunnelResponse struct {
	Tunnel Tunnel `json:"tunnel"`
}

type CreateNormalTunnelRequest struct {
	NormalTunnel `json:"tunnel"`
	Keys []string `json:"keys"`
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

func (s Server) CreateNormalTunnel(ctx context.Context, request CreateNormalTunnelRequest) (*NewTunnelResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	// insert into DB
	record, err := s.SQL.CreateNormalTunnel(ctx, sqlFromNormalTunnel(request.NormalTunnel))
	if err != nil {
		return nil, errors.Wrap(err, "could not insert")
	}

	// TODO: add keys

	return &NewTunnelResponse{normalTunnelFromSQL(record)}, nil
}

type CreateReverseTunnelRequest struct {
	NormalTunnel `json:"tunnel"`
	Keys []string `json:"keys"`
}

func (s Server) CreateReverseTunnel(ctx context.Context, request CreateReverseTunnelRequest) (*NewTunnelResponse, error) {
	var tunnelData postgres.ReverseTunnel

	record, err := s.SQL.CreateReverseTunnel(ctx, tunnelData)
	if err != nil {
		return nil, errors.Wrap(err, "could not insert")
	}

	return &NewTunnelResponse{reverseTunnelFromSQL(record)}, nil
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

