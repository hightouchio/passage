package postgres

import (
	"context"
	"errors"
	"github.com/google/uuid"
)

func (q *Queries) DeleteTunnel(ctx context.Context, id uuid.UUID) error {
	return errors.Join(
		q.DeleteNormalTunnel(ctx, id),
		q.DeleteReverseTunnel(ctx, id),
	)
}
