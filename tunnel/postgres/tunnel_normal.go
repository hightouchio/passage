package postgres

import (
	"context"
	"database/sql"
	"github.com/google/uuid"
	"time"

	"github.com/pkg/errors"
)

type NormalTunnel struct {
	ID        uuid.UUID
	CreatedAt time.Time
	Enabled   bool

	TunnelPort  int
	SSHUser     string
	SSHHost     string
	SSHPort     int
	ServiceHost string
	ServicePort int
}

func (c Client) CreateNormalTunnel(ctx context.Context, tunnel NormalTunnel) (NormalTunnel, error) {
	result, err := c.db.QueryContext(ctx, createNormalTunnel, tunnel.SSHHost, tunnel.SSHPort, tunnel.ServiceHost, tunnel.ServicePort)
	if err != nil {
		return NormalTunnel{}, errors.Wrap(err, "could not insert")
	}
	result.Next()

	var recordID uuid.UUID
	if err = result.Scan(&recordID); err != nil {
		return NormalTunnel{}, errors.Wrap(err, "could not scan id")
	}

	return c.GetNormalTunnel(ctx, recordID)
}

func (c Client) GetNormalTunnel(ctx context.Context, id uuid.UUID) (NormalTunnel, error) {
	row := c.db.QueryRowContext(ctx, getNormalTunnel, id)

	normalTunnel, err := scanNormalTunnel(row)
	if err == sql.ErrNoRows {
		return NormalTunnel{}, ErrTunnelNotFound
	} else if err != nil {
		return NormalTunnel{}, errors.Wrap(err, "could not fetch")
	}

	return normalTunnel, nil
}

func (c Client) ListNormalActiveTunnels(ctx context.Context) ([]NormalTunnel, error) {
	rows, err := c.db.QueryContext(ctx, listNormalActiveTunnels)
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
		&tunnel.Enabled,
		&tunnel.TunnelPort,
		&tunnel.SSHUser,
		&tunnel.SSHHost,
		&tunnel.SSHPort,
		&tunnel.ServiceHost,
		&tunnel.ServicePort,
	); err != nil {
		return NormalTunnel{}, err
	}
	return tunnel, nil
}

const createNormalTunnel = `
INSERT INTO passage.tunnels (ssh_host, ssh_port, service_host, service_port)
VALUES ($1, $2, $3, $4)
RETURNING id
`

const getNormalTunnel = `
SELECT id, created_at, enabled, tunnel_port, ssh_user, ssh_host, ssh_port, service_host, service_port
FROM passage.tunnels
WHERE id=$1
`

const listNormalActiveTunnels = `
SELECT id, created_at, enabled, tunnel_port, ssh_user, ssh_host, ssh_port, service_host, service_port
FROM passage.tunnels
WHERE enabled=true
;
`
