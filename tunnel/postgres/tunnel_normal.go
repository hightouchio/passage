package postgres

import (
	"context"
	"database/sql"
	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"time"

	"github.com/pkg/errors"
)

type NormalTunnel struct {
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

func (c Client) CreateNormalTunnel(ctx context.Context, input NormalTunnel) (NormalTunnel, error) {
	var tunnel NormalTunnel
	query, args, err := psql.Insert("passage.tunnels").SetMap(map[string]interface{}{
		"ssh_host":     input.SSHHost,
		"ssh_port":     input.SSHPort,
		"service_host": input.ServiceHost,
		"service_port": input.ServicePort,
	}).Suffix("RETURNING *").ToSql()
	if err != nil {
		return NormalTunnel{}, errors.Wrap(err, "could not generate SQL")
	}
	result := c.db.QueryRowxContext(ctx, query, args...)
	if err = result.StructScan(&tunnel); err != nil {
		return NormalTunnel{}, errors.Wrap(err, "could not scan")
	}
	return tunnel, nil
}

func (c Client) UpdateNormalTunnel(ctx context.Context, id uuid.UUID, fields map[string]interface{}) (NormalTunnel, error) {
	var tunnel NormalTunnel
	query, args, err := psql.Update("passage.tunnels").SetMap(filterAllowedFields(fields, normalTunnelAllowedFields)).Where(sq.Eq{"id": id}).Suffix("RETURNING *").ToSql()
	if err != nil {
		return NormalTunnel{}, errors.Wrap(err, "could not generate SQL")
	}
	result := c.db.QueryRowxContext(ctx, query, args...)
	if err := result.StructScan(&tunnel); err != nil {
		return NormalTunnel{}, errors.Wrap(err, "could not scan")
	}
	return tunnel, nil
}

func (c Client) GetNormalTunnel(ctx context.Context, id uuid.UUID) (NormalTunnel, error) {
	var tunnel NormalTunnel
	result := c.db.QueryRowxContext(ctx, `SELECT * FROM passage.tunnels WHERE id=$1`, id)

	switch err := result.StructScan(&tunnel); err {
	case nil:
		return tunnel, nil
	case sql.ErrNoRows:
		return NormalTunnel{}, ErrTunnelNotFound
	default:
		return NormalTunnel{}, err
	}
}

func (c Client) ListNormalActiveTunnels(ctx context.Context) ([]NormalTunnel, error) {
	rows, err := c.db.QueryxContext(ctx, `SELECT * FROM passage.tunnels WHERE enabled=true;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tunnels := make([]NormalTunnel, 0)
	for rows.Next() {
		var tunnel NormalTunnel
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

var normalTunnelAllowedFields = []string{"enabled", "service_host", "service_port", "ssh_host", "ssh_port", "ssh_user"}
