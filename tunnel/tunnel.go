package tunnel

import (
	"context"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/pkg/errors"
)

// ConnectionDetails describes how the SaaS will use the tunnel
type ConnectionDetails struct {
	Host string `json:"host"`
	Port uint32 `json:"port"`
}

type Tunnel interface {
	GetID() uuid.UUID
	Start(context.Context, SSHOptions) error
	GetConnectionDetails() ConnectionDetails
}

type SSHOptions struct {
	BindHost string
	HostKey  []byte
}

//goland:noinspection GoNameStartsWithPackageName
type TunnelType string

// getTunnel finds whichever tunnel type matches the UUID
func (s Server) getTunnel(ctx context.Context, id uuid.UUID) (Tunnel, TunnelType, error) {
	// reverse funnel first
	reverseTunnel, err := s.SQL.GetReverseTunnel(ctx, id)
	if err == nil {
		return reverseTunnelFromSQL(reverseTunnel), "reverse", nil
	} else if err != postgres.ErrTunnelNotFound {
		// internal server error
		return nil, "", errors.Wrap(err, "could not fetch from database")
	}

	// normal tunnel next
	normalTunnel, err := s.SQL.GetNormalTunnel(ctx, id)
	if err == nil {
		return normalTunnelFromSQL(normalTunnel), "normal", nil
	} else if err != postgres.ErrTunnelNotFound {
		// internal server error
		return nil, "", errors.Wrap(err, "could not fetch from database")
	}

	return nil, "", postgres.ErrTunnelNotFound
}
