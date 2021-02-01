package postgres

import (
	"context"
	"database/sql"

	"github.com/hightouchio/passage/pkg/models"
	"github.com/hightouchio/passage/pkg/store"
	"github.com/pkg/errors"
)

type scanner interface {
	Scan(...interface{}) error
}

var (
	_ store.Tunnels = &Tunnels{}
)

type Tunnels struct {
	db *sql.DB
}

func NewTunnels(db *sql.DB) *Tunnels {
	return &Tunnels{
		db: db,
	}
}

func (t *Tunnels) Create(
	ctx context.Context,
	id string,
	tunnelType models.TunnelType,
	publicKey string,
	privateKey string,
	port uint32,
) (*models.Tunnel, error) {
	if _, err := t.db.ExecContext(
		ctx,
		createTunnel,
		id,
		tunnelType,
		publicKey,
		privateKey,
		port,
	); err != nil {
		return nil, err
	}

	return t.Get(ctx, id)
}

func (t *Tunnels) Get(
	ctx context.Context,
	id string,
) (*models.Tunnel, error) {
	row := t.db.QueryRowContext(ctx, getTunnel, id)

	tunnel, err := t.scanTunnel(row)
	if err == sql.ErrNoRows {
		return nil, store.ErrTunnelNotFound
	} else if err != nil {
		return nil, err
	}

	return tunnel, nil
}

func (t *Tunnels) List(
	ctx context.Context,
) ([]models.Tunnel, error) {
	rows, err := t.db.QueryContext(ctx, listTunnels)
	if err != nil {
		return nil, errors.Wrap(err, "query tunnels")
	}
	defer rows.Close()

	tunnels := make([]models.Tunnel, 0)
	for rows.Next() {
		tunnel, err := t.scanTunnel(rows)
		if err != nil {
			return nil, err
		}
		tunnels = append(tunnels, *tunnel)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tunnels, nil
}

func (t *Tunnels) scanTunnel(scanner scanner) (*models.Tunnel, error) {
	var tunnel models.Tunnel
	if err := scanner.Scan(
		&tunnel.ID,
		&tunnel.CreatedAt,
		&tunnel.Type,
		&tunnel.PublicKey,
		&tunnel.PrivateKey,
		&tunnel.Port,
	); err != nil {
		return nil, err
	}
	return &tunnel, nil
}
