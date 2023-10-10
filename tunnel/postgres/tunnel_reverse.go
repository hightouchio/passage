package postgres

import (
	"context"
	"database/sql"
	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"time"
)

type ReverseTunnel struct {
	ID                 uuid.UUID      `db:"id"`
	CreatedAt          time.Time      `db:"created_at"`
	Enabled            bool           `db:"enabled"`
	TunnelPort         int            `db:"tunnel_port"`
	SSHDPort           int            `db:"sshd_port"`
	HTTPProxy          bool           `db:"http_proxy"`
	Error              sql.NullString `db:"error"`
	LastUsedAt         sql.NullTime   `db:"last_used_at"`
	AuthorizedKeysHash string         `db:"authorized_keys_hash"`
}

func (c Client) CreateReverseTunnel(ctx context.Context, input ReverseTunnel) (ReverseTunnel, error) {
	var tunnel ReverseTunnel
	query, args, err := psql.Insert("passage.reverse_tunnels").Values(sq.Expr("DEFAULT")).Suffix("RETURNING *").ToSql()
	if err != nil {
		return ReverseTunnel{}, errors.Wrap(err, "could not generate SQL")
	}
	result := c.db.QueryRowxContext(ctx, query, args...)
	if err = result.StructScan(&tunnel); err != nil {
		return ReverseTunnel{}, errors.Wrap(err, "could not scan")
	}
	return tunnel, nil
}

func (c Client) UpdateReverseTunnel(ctx context.Context, id uuid.UUID, fields map[string]interface{}) (ReverseTunnel, error) {
	var tunnel ReverseTunnel
	query, args, err := psql.Update("passage.reverse_tunnels").SetMap(filterAllowedFields(fields, reverseTunnelAllowedFields)).Where(sq.Eq{"id": id}).Suffix("RETURNING *").ToSql()
	if err != nil {
		return ReverseTunnel{}, errors.Wrap(err, "could not generate SQL")
	}
	result := c.db.QueryRowxContext(ctx, query, args...)
	if err := result.StructScan(&tunnel); err != nil {
		return ReverseTunnel{}, errors.Wrap(err, "could not scan")
	}
	return tunnel, nil
}

func (c Client) GetReverseTunnel(ctx context.Context, id uuid.UUID) (ReverseTunnel, error) {
	var tunnel ReverseTunnel
	result := c.db.QueryRowxContext(ctx, `SELECT * FROM passage.reverse_tunnels WHERE id=$1`, id)

	switch err := result.StructScan(&tunnel); err {
	case nil:
		return tunnel, nil
	case sql.ErrNoRows:
		return ReverseTunnel{}, ErrTunnelNotFound
	default:
		return ReverseTunnel{}, err
	}
}

func (c Client) ListReverseActiveTunnels(ctx context.Context) ([]ReverseTunnel, error) {
	rows, err := c.db.QueryxContext(ctx, `
		SELECT rt.*, encode(sha256(array_to_string(array_agg(ka.key_id), ',')::bytea), 'hex') AS authorized_keys_hash
		FROM passage.reverse_tunnels rt
				  LEFT JOIN passage.key_authorizations ka ON ka.tunnel_id = rt.id
		WHERE rt.enabled = true
		GROUP BY rt.id;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tunnels := make([]ReverseTunnel, 0)
	for rows.Next() {
		var tunnel ReverseTunnel
		if err := rows.StructScan(&tunnel); err != nil {
			return nil, err
		}
		tunnels = append(tunnels, tunnel)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tunnels, nil
}

var reverseTunnelAllowedFields = []string{"enabled"}
