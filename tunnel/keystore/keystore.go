package keystore

import (
	"context"
	"github.com/google/uuid"
)

type Keystore interface {
	Get(ctx context.Context, id uuid.UUID) ([]byte, error)
	Set(ctx context.Context, id uuid.UUID, contents []byte) error
	Delete(ctx context.Context, id uuid.UUID) error
}
