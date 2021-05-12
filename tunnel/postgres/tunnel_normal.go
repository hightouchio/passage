package postgres

import (
	"context"
	"github.com/google/uuid"
	"time"

	"github.com/pkg/errors"
)

type NormalTunnel struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"createdAt"`

	TunnelPort uint32 `json:"tunnelPort"`

	SSHUser         string `json:"sshUser"`
	SSHHostname     string `json:"sshHostname"`
	SSHPort         uint32 `json:"sshPort"`
	ServiceHostname string `json:"serviceHostname"`
	ServicePort     uint32 `json:"servicePort"`
}

func (c Client) ListNormalTunnels(ctx context.Context) ([]NormalTunnel, error) {
	rows, err := c.db.QueryContext(ctx, listNormalTunnels)
	if err != nil {
		return nil, errors.Wrap(err, "query tunnel")
	}
	defer rows.Close()

	tunnels := make([]NormalTunnel, 0)
	for rows.Next() {
		tunnel, err := scanNormalTunnel(rows)
		if err != nil {
			return nil, err
		}
		tunnels = append(tunnels, tunnel)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tunnels, nil
}

func scanNormalTunnel(scanner scanner) (NormalTunnel, error) {
	var tunnel NormalTunnel
	if err := scanner.Scan(
		&tunnel.ID,
		&tunnel.CreatedAt,
		&tunnel.TunnelPort,
		&tunnel.SSHUser,
		&tunnel.SSHHostname,
		&tunnel.SSHPort,
		&tunnel.ServiceHostname,
		&tunnel.ServicePort,
	); err != nil {
		return NormalTunnel{}, err
	}
	return tunnel, nil
}

var ErrNormalTunnelNotFound = errors.New("tunnel not found")

const createTunnel = `
INSERT INTO passage.tunnels (ssh_hostname, ssh_port, service_hostname, service_port)
VALUES ($1, $2, $3, $4)
RETURNING id
`

const getTunnel = `
SELECT id, created_at, tunnel_port, ssh_user, ssh_hostname, ssh_port, service_hostname, service_port
FROM passage.tunnels
WHERE id=$1
`

const listNormalTunnels = `
SELECT id, created_at, tunnel_port, ssh_user, ssh_hostname, ssh_port, service_hostname, service_port
FROM passage.tunnels
`
