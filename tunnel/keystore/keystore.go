package keystore

import (
	"context"
	"github.com/google/uuid"
)

type KeyType string

type Key struct {
	ID       uuid.UUID
	Contents string
}

type Keystore interface {
	Get(ctx context.Context, keyType KeyType, id uuid.UUID) (Key, error)
	Set(ctx context.Context, keyType KeyType, key Key) error
}
