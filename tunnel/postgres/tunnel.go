package postgres

import (
	"context"
	"github.com/google/uuid"
)

func (c Client) DeleteTunnel(ctx context.Context, id uuid.UUID) error {
	if _, err := c.db.ExecContext(ctx, `DELETE FROM passage.tunnels WHERE id=$1;`, id); err != nil {
		return err
	}
	if _, err := c.db.ExecContext(ctx, `DELETE FROM passage.reverse_tunnels WHERE id=$1;`, id); err != nil {
		return err
	}
	return nil
}
