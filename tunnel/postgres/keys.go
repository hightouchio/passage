package postgres

import (
	"context"
	"database/sql"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type Key struct {
	ID       uuid.UUID
	Type     string
	Contents string
}

const getStandardTunnelPrivateKeys = `
SELECT passage.keys.id, passage.keys.key_type, passage.keys.contents FROM passage.keys
JOIN passage.key_authorizations ON passage.keys.id=passage.key_authorizations.key_id
JOIN passage.tunnels ON passage.key_authorizations.tunnel_id=passage.tunnels.id AND passage.key_authorizations.tunnel_type IN ('standard', 'normal')
WHERE passage.keys.key_type='private' AND passage.tunnels.id=$1;
`

func (c Client) GetStandardTunnelPrivateKeys(ctx context.Context, tunnelID uuid.UUID) ([]Key, error) {
	return c.getKeysForTunnel(ctx, getStandardTunnelPrivateKeys, tunnelID)
}

const getReverseTunnelAuthorizedKeys = `
SELECT passage.keys.id, passage.keys.key_type, passage.keys.contents FROM passage.keys
JOIN passage.key_authorizations ON passage.keys.id=passage.key_authorizations.key_id
JOIN passage.reverse_tunnels ON passage.key_authorizations.tunnel_id=passage.reverse_tunnels.id AND passage.key_authorizations.tunnel_type='reverse'
WHERE passage.keys.key_type='public' AND passage.reverse_tunnels.id=$1;
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
		&key.Type,
		&key.Contents,
	); err != nil {
		return Key{}, err
	}
	return key, nil
}

const insertKey = `
INSERT INTO passage.keys (key_type, contents) VALUES ($1, $2) RETURNING id;
`

const attachKey = `
INSERT INTO passage.key_authorizations (tunnel_type, tunnel_id, key_id) VALUES ($1, $2, $3);
`

func (c Client) AddKeyAndAttachToTunnel(ctx context.Context, tunnelType string, tunnelID uuid.UUID, keyType string, contents string) error {
	var tx *sql.Tx
	var err error
	if tx, err = c.db.BeginTx(ctx, nil); err != nil {
		return err
	}

	// insert key, returns ID
	var keyID uuid.UUID
	if err := tx.QueryRowContext(ctx, insertKey, keyType, contents).Scan(&keyID); err != nil {
		return err
	}

	// insert key authorization
	if _, err := tx.ExecContext(ctx, attachKey, tunnelType, tunnelID, keyID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
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
