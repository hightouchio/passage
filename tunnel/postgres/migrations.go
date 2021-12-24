package postgres

import (
	"database/sql"
	"embed"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/pkg/errors"
)

//go:embed migrations/*.sql
var static embed.FS

// ApplyMigrations runs the embedded migrations against the Postgres instance.
func ApplyMigrations(db *sql.DB) (bool, error) {
	// Ensure the passage schema exists first, so the migrations table can be placed in it.
	if _, err := db.Exec("CREATE SCHEMA IF NOT EXISTS passage;"); err != nil {
		return false, errors.Wrap(err, "could not create passage schema")
	}

	// Register "schema_migrations" table.
	pgDriver, err := postgres.WithInstance(db, &postgres.Config{
		SchemaName:      "passage",
		MigrationsTable: "schema_migrations",
	})
	if err != nil {
		return false, errors.Wrap(err, "could not init postgres driver")
	}

	sourceDriver, err := iofs.New(static, "migrations")
	if err != nil {
		return false, errors.Wrap(err, "could not init embedded iofs")
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", pgDriver)
	if err != nil {
		return false, errors.Wrap(err, "could not initialize migrator")
	}

	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			return false, nil
		}
		return false, errors.Wrap(err, "could not run up migrations")
	}

	return true, nil
}
