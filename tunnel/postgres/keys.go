package postgres

import (
	"context"
	"database/sql"
	"github.com/pkg/errors"
)

type Key struct {
	ID       int
	Type     string
	Contents string
}

func (c Client) GetNormalTunnelPrivateKeys(ctx context.Context, tunnelID int) ([]Key, error) {
	return c.getKeysForTunnel(ctx, getNormalTunnelPrivateKeys, tunnelID)
}

func (c Client) GetReverseTunnelAuthorizedKeys(ctx context.Context, tunnelID int) ([]Key, error) {
	return c.getKeysForTunnel(ctx, getReverseTunnelAuthorizedKeys, tunnelID)
}

func (c Client) getKeysForTunnel(ctx context.Context, query string, tunnelID int) ([]Key, error) {
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

const getNormalTunnelPrivateKeys = `
SELECT passage.keys.id, passage.keys.key_type, passage.keys.contents FROM passage.keys
JOIN passage.key_authorizations ON passage.keys.id=passage.key_authorizations.key_id
JOIN passage.tunnels ON passage.key_authorizations.tunnel_id=passage.tunnels.id AND passage.key_authorizations.tunnel_type='normal'
WHERE passage.keys.key_type='private' AND passage.tunnels.id=$1;
`

const getReverseTunnelAuthorizedKeys = `
SELECT passage.keys.id, passage.keys.key_type, passage.keys.contents FROM passage.keys
JOIN passage.key_authorizations ON passage.keys.id=passage.key_authorizations.key_id
JOIN passage.reverse_tunnels ON passage.key_authorizations.tunnel_id=passage.reverse_tunnels.id AND passage.key_authorizations.tunnel_type='reverse'
WHERE passage.keys.key_type='public' AND passage.reverse_tunnels.id=$1;
`