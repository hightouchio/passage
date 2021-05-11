package postgres

import (
	"context"
	"database/sql"
	"github.com/pkg/errors"
)

func (c Client) GetReverseTunnelAuthorizedKeys(ctx context.Context, tunnelID int) ([]string, error) {
	rows, err := c.db.QueryContext(ctx, getReverseTunnelAuthorizedKeys, tunnelID)
	if err != nil {
		if err == sql.ErrNoRows {
			return []string{}, nil
		}

		return []string{}, err
	}

	results := make([]string, 0)
	for rows.Next() {
		var row string
		if err := rows.Scan(&row); err != nil {
			return []string{}, errors.Wrap(err, "could not scan")
		}
		results = append(results, row)
	}

	return results, nil
}

const getReverseTunnelAuthorizedKeys = `
SELECT passage.keys.public_key FROM passage.keys
JOIN passage.key_authorizations ON passage.keys.id=passage.key_authorizations.key_id 
JOIN passage.reverse_tunnels ON passage.key_authorizations.tunnel_id=passage.reverse_tunnels.id AND passage.key_authorizations.tunnel_type='reverse'
WHERE passage.keys.key_type='public' AND passage.reverse_tunnels.id=$1;
`