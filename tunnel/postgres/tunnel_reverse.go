package postgres

import (
	"context"
	"database/sql"
	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"time"
)

type ReverseTunnel struct {
	ID                 uuid.UUID      `db:"id"`
	CreatedAt          time.Time      `db:"created_at"`
	Enabled            bool           `db:"enabled"`
	SSHDPort           int            `db:"sshd_port"`
	AuthorizedKeysHash string         `db:"authorized_keys_hash"`
	TunnelPort         int            `db:"tunnel_port"`
	HealthcheckConfig  sql.NullString `db:"healthcheck_config"`

	// Deprecated
	HttpProxy  bool           `db:"http_proxy"`
	Error      sql.NullString `db:"error"`
	LastUsedAt sql.NullTime   `db:"last_used_at"`
}

func (c Client) CreateReverseTunnel(ctx context.Context, input ReverseTunnel, authorizedKeys []uuid.UUID) (ReverseTunnel, error) {
	query, args, err := psql.Insert("passage.reverse_tunnels").Values(sq.Expr("DEFAULT")).Suffix("RETURNING *").ToSql()
	if err != nil {
		return ReverseTunnel{}, errors.Wrap(err, "could not generate sql")
	}

	var tunnel ReverseTunnel
	if err := withTx(ctx, c.db, func(tx *sqlx.Tx) error {
		// Insert tunnel record
		if err = tx.QueryRowxContext(ctx, query, args...).StructScan(&tunnel); err != nil {
			return errors.Wrap(err, "could not scan tunnel record")
		}

		// Insert authorized keys
		for _, keyId := range authorizedKeys {
			if err := authorizeKeyForTunnel(ctx, tx, "reverse", tunnel.ID, keyId); err != nil {
				return errors.Wrapf(err, "could not authorize key %s", keyId.String())
			}
		}

		return nil
	}); err != nil {
		return ReverseTunnel{}, errors.Wrap(err, "could not create reverse tunnel")
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

// withTx is a helper function to wrap a function in a transaction, and commit or rollback depending on if the fn
//
//	returned with an error
func withTx(ctx context.Context, db *sqlx.DB, fn func(tx *sqlx.Tx) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "could not open tx")
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	} else {
		if err := tx.Commit(); err != nil {
			return errors.Wrap(err, "could not commit")
		}
	}

	return nil
}
