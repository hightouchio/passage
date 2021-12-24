package sqlite3

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

type Sqlite3 struct {
	db        *sql.DB
	tableName string
}

func New(path string, tableName string) (*Sqlite3, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, errors.Wrap(err, "could not open sqlite3 file")
	}

	sqlite3 := &Sqlite3{db: db, tableName: tableName}
	if err := sqlite3.initTables(db); err != nil {
		return nil, errors.Wrap(err, "could not init tables")
	}

	return sqlite3, nil
}

// init the internal schema
func (k Sqlite3) initTables(db *sql.DB) error {
	_, err := db.Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s(
		   id		VARCHAR(255) NOT NULL PRIMARY KEY,
		   contents	TEXT
		);
	`, k.tableName))
	return err
}

func (k Sqlite3) Get(ctx context.Context, id uuid.UUID) ([]byte, error) {
	var contents []byte
	row := k.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT contents FROM %s WHERE id=$1;`, k.tableName), id)
	if err := row.Scan(&contents); err != nil {
		return []byte{}, err
	}
	return contents, nil
}

func (k Sqlite3) Set(ctx context.Context, id uuid.UUID, contents []byte) error {
	_, err := k.db.ExecContext(ctx, fmt.Sprintf(`
	INSERT INTO %s(id, contents)
	VALUES($1::uuid, $2::text);
	`, k.tableName), id, string(contents))
	return err
}

func (k Sqlite3) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := k.db.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE id=$1;`, k.tableName), id)
	return err
}

func (k Sqlite3) Close() error {
	return k.db.Close()
}
