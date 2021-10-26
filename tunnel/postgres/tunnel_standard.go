package postgres

import (
	"context"
	"database/sql"
	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"time"

	"github.com/pkg/errors"
)

type StandardTunnel struct {
	ID        uuid.UUID `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	Enabled   bool      `db:"enabled"`

	TunnelPort  int    `db:"tunnel_port"`
	SSHUser     string `db:"ssh_user"`
	SSHHost     string `db:"ssh_host"`
	SSHPort     int    `db:"ssh_port"`
	ServiceHost string `db:"service_host"`
	ServicePort int    `db:"service_port"`
}

func (c Client) CreateStandardTunnel(ctx context.Context, input StandardTunnel) (StandardTunnel, error) {
	var tunnel StandardTunnel
	query, args, err := psql.Insert("passage.tunnels").SetMap(map[string]interface{}{
		"ssh_host":     input.SSHHost,
		"ssh_port":     input.SSHPort,
		"service_host": input.ServiceHost,
		"service_port": input.ServicePort,
	}).Suffix("RETURNING *").ToSql()
	if err != nil {
		return StandardTunnel{}, errors.Wrap(err, "could not generate SQL")
	}
	result := c.db.QueryRowxContext(ctx, query, args...)
	if err = result.StructScan(&tunnel); err != nil {
		return StandardTunnel{}, errors.Wrap(err, "could not scan")
	}
	return tunnel, nil
}

func (c Client) UpdateStandardTunnel(ctx context.Context, id uuid.UUID, fields map[string]interface{}) (StandardTunnel, error) {
	var tunnel StandardTunnel
	query, args, err := psql.Update("passage.tunnels").SetMap(filterAllowedFields(fields, standardTunnelAllowedFields)).Where(sq.Eq{"id": id}).Suffix("RETURNING *").ToSql()
	if err != nil {
		return StandardTunnel{}, errors.Wrap(err, "could not generate SQL")
	}
	result := c.db.QueryRowxContext(ctx, query, args...)
	if err := result.StructScan(&tunnel); err != nil {
		return StandardTunnel{}, errors.Wrap(err, "could not scan")
	}
	return tunnel, nil
}

func (c Client) GetStandardTunnel(ctx context.Context, id uuid.UUID) (StandardTunnel, error) {
	var tunnel StandardTunnel
	result := c.db.QueryRowxContext(ctx, `SELECT * FROM passage.tunnels WHERE id=$1`, id)

	switch err := result.StructScan(&tunnel); err {
	case nil:
		return tunnel, nil
	case sql.ErrNoRows:
		return StandardTunnel{}, ErrTunnelNotFound
	default:
		return StandardTunnel{}, err
	}
}

func (c Client) ListStandardActiveTunnels(ctx context.Context) ([]StandardTunnel, error) {
	rows, err := c.db.QueryxContext(ctx, `SELECT * FROM passage.tunnels WHERE enabled=true;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tunnels := make([]StandardTunnel, 0)
	for rows.Next() {
		var tunnel StandardTunnel
		if err := rows.StructScan(&tunnel); err != nil {
			return nil, err
		}
		tunnels = append(tunnels, tunnel)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tunnels, nil
}

var standardTunnelAllowedFields = []string{"enabled", "service_host", "service_port", "ssh_host", "ssh_port", "ssh_user"}
