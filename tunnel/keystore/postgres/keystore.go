package postgres

import (
	"context"
	"fmt"
	"github.com/google/uuid"
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

func (p Keystore) Get(ctx context.Context, id uuid.UUID) ([]byte, error) {
	var contents []byte
	row := p.db.QueryRowxContext(ctx, fmt.Sprintf(`SELECT contents FROM %s WHERE id=$1;`, p.tableName), id)
	if err := row.Scan(&contents); err != nil {
		return []byte{}, err
	}
	return contents, nil
}

func (p Keystore) Set(ctx context.Context, id uuid.UUID, contents []byte) error {
	_, err := p.db.ExecContext(ctx, fmt.Sprintf(`
	INSERT INTO %s(id, contents)
	VALUES($1::uuid, $2::text)
	ON CONFLICT(id) DO UPDATE SET contents=$2::text;
	`, p.tableName), id, string(contents))
	return err
}

func (p Keystore) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := p.db.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE id=$1;`, p.tableName), id)
	return err
}
