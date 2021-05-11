package postgres

import (
	"context"
	"database/sql"
	"github.com/pkg/errors"
	"time"
)

type ReverseTunnel struct {
	ID         int
	CreatedAt  time.Time
	TunnelPort uint32
	SSHDPort   uint32
}

func (c Client) CreateReverseTunnel(ctx context.Context, tunnel ReverseTunnel) (ReverseTunnel, error) {
	result, err := c.db.QueryContext(ctx, createReverseTunnel)
	if err != nil {
		return ReverseTunnel{}, errors.Wrap(err, "could not insert reverse tunnel")
	}
	result.Next()

	var recordID int
	if err = result.Scan(&recordID); err != nil {
		return ReverseTunnel{}, errors.Wrap(err, "could not scan id")
	}

	return c.GetReverseTunnel(ctx, recordID)
}

func (c Client) ListReverseTunnels(ctx context.Context) ([]ReverseTunnel, error) {
	rows, err := c.db.QueryContext(ctx, listReverseTunnels)
	if err != nil {
		return nil, errors.Wrap(err, "query reverse tunnel")
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

func (c Client) GetReverseTunnel(ctx context.Context, id int) (ReverseTunnel, error) {
	row := c.db.QueryRowContext(ctx, getReverseTunnel, id)

	reverseTunnel, err := scanReverseTunnel(row)
	if err == sql.ErrNoRows {
		return ReverseTunnel{}, ErrReverseTunnelNotFound
	} else if err != nil {
		return ReverseTunnel{}, errors.Wrap(err, "could not get reverse tunnel")
	}

	return reverseTunnel, nil
}

func scanReverseTunnel(scanner scanner) (ReverseTunnel, error) {
	var reverseTunnel ReverseTunnel
	if err := scanner.Scan(
		&reverseTunnel.ID,
		&reverseTunnel.CreatedAt,
		&reverseTunnel.TunnelPort,
		&reverseTunnel.SSHDPort,
	); err != nil {
		return ReverseTunnel{}, err
	}
	return reverseTunnel, nil
}

var ErrReverseTunnelNotFound = errors.New("reverse tunnel not found")

const createReverseTunnel = `
INSERT INTO passage.reverse_tunnels DEFAULT VALUES
RETURNING id
`

const getReverseTunnel = `
SELECT id, created_at, tunnel_port, sshd_port
FROM passage.reverse_tunnels
WHERE id=$1
`

const listReverseTunnels = `
SELECT id, created_at, tunnel_port, sshd_port
FROM passage.reverse_tunnels
`