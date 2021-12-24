package postgres

import (
	"context"
	"database/sql"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type Key struct {
	ID   uuid.UUID
	Type string
}

const getStandardTunnelPrivateKeys = `
SELECT key_id FROM passage.key_authorizations WHERE tunnel_id=$1 AND tunnel_type='standard';
`

func (c Client) GetStandardTunnelPrivateKeys(ctx context.Context, tunnelID uuid.UUID) ([]Key, error) {
	return c.getKeysForTunnel(ctx, getStandardTunnelPrivateKeys, tunnelID)
}

const getReverseTunnelAuthorizedKeys = `
SELECT key_id FROM passage.key_authorizations WHERE tunnel_id=$1 AND tunnel_type='reverse';
`

func (c Client) GetReverseTunnelAuthorizedKeys(ctx context.Context, tunnelID uuid.UUID) ([]Key, error) {
	return c.getKeysForTunnel(ctx, getReverseTunnelAuthorizedKeys, tunnelID)
}

func (c Client) getKeysForTunnel(ctx context.Context, query string, tunnelID uuid.UUID) ([]Key, error) {
	rows, err := c.db.QueryContext(ctx, query, tunnelID)
	if err != nil {
		if err == sql.ErrNoRows {
			return []Key{}, nil
		}

		return []Key{}, err
	}

	results := make([]Key, 0)
	for rows.Next() {
		key, err := scanKey(rows)
		if err != nil {
			return []Key{}, errors.Wrap(err, "could not scan")
		}
		results = append(results, key)
	}

	return results, nil
}

func scanKey(scan scanner) (Key, error) {
	var key Key
	if err := scan.Scan(
		&key.ID,
	); err != nil {
		return Key{}, err
	}
	return key, nil
}

const authorizeKeyForTunnel = `
INSERT INTO passage.key_authorizations (tunnel_type, tunnel_id, key_id) VALUES ($1, $2, $3);
`

func (c Client) AuthorizeKeyForTunnel(ctx context.Context, tunnelType string, tunnelID uuid.UUID, keyID uuid.UUID) error {
	if _, err := c.db.ExecContext(ctx, authorizeKeyForTunnel, tunnelType, tunnelID, keyID); err != nil {
		return err
	}
	return nil
}
