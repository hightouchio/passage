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
	Port uint32 `json:"port"`
}

//goland:noinspection GoNameStartsWithPackageName
type TunnelType string

// getTunnel finds whichever tunnel type matches the UUID
func getTunnel(ctx context.Context, sql sqlClient, id uuid.UUID) (Tunnel, TunnelType, error) {
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
