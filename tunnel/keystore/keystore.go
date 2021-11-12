package keystore

import (
	"context"
	"github.com/google/uuid"
)

type Keystore interface {
	Get(ctx context.Context, id uuid.UUID) (string, error)
	Set(ctx context.Context, id uuid.UUID, contents string) error
	Delete(ctx context.Context, id uuid.UUID) error
}
