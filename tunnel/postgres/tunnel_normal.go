package postgres

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

//func (t NormalTunnels) Create(ctx context.Context, tunnel tunnel.NormalTunnel) (*tunnel.NormalTunnel, error) {
//	if _, err := t.db.ExecContext(
//		ctx,
//		createTunnel,
//		tunnel.ID,
//		tunnel.PublicKey,
//		tunnel.PrivateKey,
//		tunnel.Port,
//		tunnel.ServiceEndpoint,
//		tunnel.ServicePort,
//	); err != nil {
//		return nil, err
//	}
//
//	return t.Get(ctx, tunnel.ID)
//}
//
//func (t *Tunnels) Get(
//	ctx context.Context,
//	id string,
//) (*tunnel.NormalTunnel, error) {
//	row := t.db.QueryRowContext(ctx, getTunnel, id)
//
//	tunnel, err := t.scanTunnel(row)
//	if err == sql.ErrNoRows {
//		return nil, ErrTunnelNotFound
//	} else if err != nil {
//		return nil, err
//	}
//
//	return tunnel, nil
//}

type NormalTunnel struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"createdAt"`

	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`

	Port            uint32 `json:"port"`
	ServerEndpoint  string `json:"serverEndpoint"`
	ServerPort      uint32 `json:"serverPort"`
	ServiceEndpoint string `json:"serviceEndpoint"`
	ServicePort     uint32 `json:"servicePort"`
}

func (c Client) ListNormalTunnels(ctx context.Context) ([]NormalTunnel, error) {
	rows, err := c.db.QueryContext(ctx, listNormalTunnels)
	if err != nil {
		return nil, errors.Wrap(err, "query tunnel")
	}
	defer rows.Close()

	tunnels := make([]NormalTunnel, 0)
	for rows.Next() {
		tunnel, err := scanNormalTunnel(rows)
		if err != nil {
			return nil, err
		}
		tunnels = append(tunnels, tunnel)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tunnels, nil
}

func scanNormalTunnel(scanner scanner) (NormalTunnel, error) {
	var tunnel NormalTunnel
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
		return NormalTunnel{}, err
	}
	return tunnel, nil
}

var ErrNormalTunnelNotFound = errors.New("tunnel not found")

const createTunnel = `
INSERT INTO tunnel (id, public_key, private_key, port, server_endpoint, server_port, service_endpoint, service_port)
VALUES ($1, $2, $3, $4, $5)
`

const getTunnel = `
SELECT id, created_at, public_key, private_key, port, server_endpoint, server_port, service_endpoint, service_port
FROM tunnel
WHERE id = $1
`

const listNormalTunnels = `
SELECT id, created_at, public_key, private_key, port, server_endpoint, server_port, service_endpoint, service_port
FROM tunnel
`
