package postgres

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/tunnel/keystore"
	"github.com/jmoiron/sqlx"
)

type Keystore struct {
	db        *sqlx.DB
	tableName string
}

func New(db *sqlx.DB, tableName string) Keystore {
	return Keystore{
		db:        db,
		tableName: tableName,
	}
}

func (p Keystore) Get(ctx context.Context, keyType keystore.KeyType, id uuid.UUID) (keystore.Key, error) {
	var contents string
	row := p.db.QueryRowxContext(ctx, fmt.Sprintf(`SELECT id, contents FROM %s WHERE key_type=$1 AND id=$2;`, p.tableName), keyType, id)
	if err := row.Scan(&contents); err != nil {
		return keystore.Key{}, err
	}
	return keystore.Key{ID: id, Contents: contents}, nil
}

func (p Keystore) Set(ctx context.Context, keyType keystore.KeyType, key keystore.Key) error {
	_, err := p.db.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET contents=$3::text WHERE key_type=$1::text AND id=$2::int;`, p.tableName), keyType, key.ID, key.Contents)
	return err
}
