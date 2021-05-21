package postgres

import (
	"context"
	"database/sql"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"time"
)

type ReverseTunnel struct {
	ID         uuid.UUID
	CreatedAt  time.Time
	Enabled    bool
	TunnelPort int
	SSHDPort   int
}

func (c Client) CreateReverseTunnel(ctx context.Context, tunnel ReverseTunnel) (ReverseTunnel, error) {
	result, err := c.db.QueryContext(ctx, createReverseTunnel)
	if err != nil {
		return ReverseTunnel{}, errors.Wrap(err, "could not insert")
	}
	result.Next()

	var recordID uuid.UUID
	if err = result.Scan(&recordID); err != nil {
		return ReverseTunnel{}, errors.Wrap(err, "could not scan id")
	}

	return c.GetReverseTunnel(ctx, recordID)
}

func (c Client) GetReverseTunnel(ctx context.Context, id uuid.UUID) (ReverseTunnel, error) {
	row := c.db.QueryRowContext(ctx, getReverseTunnel, id)

	reverseTunnel, err := scanReverseTunnel(row)
	if err == sql.ErrNoRows {
		return ReverseTunnel{}, ErrTunnelNotFound
	} else if err != nil {
		return ReverseTunnel{}, errors.Wrap(err, "could not fetch")
	}

	return reverseTunnel, nil
}

func (c Client) ListReverseActiveTunnels(ctx context.Context) ([]ReverseTunnel, error) {
	rows, err := c.db.QueryContext(ctx, listReverseActiveTunnels)
	if err != nil {
		return nil, errors.Wrap(err, "could not list")
	}
	defer rows.Close()

	reverseTunnels := make([]ReverseTunnel, 0)
	for rows.Next() {
		reverseTunnel, err := scanReverseTunnel(rows)
		if err != nil {
			return nil, err
		}
		reverseTunnels = append(reverseTunnels, reverseTunnel)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return reverseTunnels, nil
}

func scanReverseTunnel(scanner scanner) (ReverseTunnel, error) {
	var reverseTunnel ReverseTunnel
	if err := scanner.Scan(
		&reverseTunnel.ID,
		&reverseTunnel.CreatedAt,
		&reverseTunnel.Enabled,
		&reverseTunnel.TunnelPort,
		&reverseTunnel.SSHDPort,
	); err != nil {
		return ReverseTunnel{}, err
	}
	return reverseTunnel, nil
}

const createReverseTunnel = `
INSERT INTO passage.reverse_tunnels DEFAULT VALUES
RETURNING id
`

const getReverseTunnel = `
SELECT id, created_at, enabled, tunnel_port, sshd_port
FROM passage.reverse_tunnels
WHERE id=$1
`

const listReverseActiveTunnels = `
SELECT id, created_at, enabled, tunnel_port, sshd_port
FROM passage.reverse_tunnels WHERE enabled=true
`
