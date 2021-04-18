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
	_ store.Tunnels        = &Tunnels{}
	_ store.ReverseTunnels = &ReverseTunnels{}
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
	tunnel models.Tunnel,
) (*models.Tunnel, error) {
	if _, err := t.db.ExecContext(
		ctx,
		createTunnel,
		tunnel.ID,
		tunnel.PublicKey,
		tunnel.PrivateKey,
		tunnel.Port,
		tunnel.ServiceEndpoint,
		tunnel.ServicePort,
	); err != nil {
		return nil, err
	}

	return t.Get(ctx, tunnel.ID)
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
		&tunnel.PublicKey,
		&tunnel.PrivateKey,
		&tunnel.Port,
		&tunnel.ServerEndpoint,
		&tunnel.ServerPort,
		&tunnel.ServiceEndpoint,
		&tunnel.ServicePort,
	); err != nil {
		return nil, err
	}
	return &tunnel, nil
}

type ReverseTunnels struct {
	db *sql.DB
}

func NewReverseTunnels(db *sql.DB) *ReverseTunnels {
	return &ReverseTunnels{
		db: db,
	}
}

func (t *ReverseTunnels) Create(
	ctx context.Context,
	reverseTunnel models.ReverseTunnel,
) (*models.ReverseTunnel, error) {
	if _, err := t.db.ExecContext(
		ctx,
		createReverseTunnel,
		reverseTunnel.ID,
		reverseTunnel.PublicKey,
		reverseTunnel.PrivateKey,
		reverseTunnel.Port,
		reverseTunnel.SSHPort,
	); err != nil {
		return nil, err
	}

	return t.Get(ctx, reverseTunnel.ID)
}

func (t *ReverseTunnels) Get(
	ctx context.Context,
	id string,
) (*models.ReverseTunnel, error) {
	row := t.db.QueryRowContext(ctx, getReverseTunnel, id)

	reverseTunnel, err := t.scanReverseTunnel(row)
	if err == sql.ErrNoRows {
		return nil, store.ErrReverseTunnelNotFound
	} else if err != nil {
		return nil, err
	}

	return reverseTunnel, nil
}

func (t *ReverseTunnels) List(
	ctx context.Context,
) ([]models.ReverseTunnel, error) {
	rows, err := t.db.QueryContext(ctx, listReverseTunnels)
	if err != nil {
		return nil, errors.Wrap(err, "query reverse tunnels")
	}
	defer rows.Close()

	reverseTunnels := make([]models.ReverseTunnel, 0)
	for rows.Next() {
		reverseTunnel, err := t.scanReverseTunnel(rows)
		if err != nil {
			return nil, err
		}
		reverseTunnels = append(reverseTunnels, *reverseTunnel)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return reverseTunnels, nil
}

func (t *ReverseTunnels) scanReverseTunnel(scanner scanner) (*models.ReverseTunnel, error) {
	var reverseTunnel models.ReverseTunnel
	if err := scanner.Scan(
		&reverseTunnel.ID,
		&reverseTunnel.CreatedAt,
		&reverseTunnel.PublicKey,
		&reverseTunnel.PrivateKey,
		&reverseTunnel.Port,
		&reverseTunnel.SSHPort,
	); err != nil {
		return nil, err
	}
	return &reverseTunnel, nil
}
