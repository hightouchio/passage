package postgres

//
//import (
//	"context"
//	"database/sql"
//	"github.com/hightouchio/passage/tunnel"
//
//	"github.com/pkg/errors"
//)
//
//type Tunnels struct {
//	db *sql.DB
//}
//
//func NewTunnels(db *sql.DB) *Tunnels {
//	return &Tunnels{db: db}
//}
//
//func (t *Tunnels) Create(ctx context.Context, tunnel tunnel.NormalTunnel) (*tunnel.NormalTunnel, error) {
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
//
//func (t *Tunnels) List(
//	ctx context.Context,
//) ([]tunnel.NormalTunnel, error) {
//	rows, err := t.db.QueryContext(ctx, listTunnels)
//	if err != nil {
//		return nil, errors.Wrap(err, "query tunnel")
//	}
//	defer rows.Close()
//
//	tunnels := make([]tunnel.NormalTunnel, 0)
//	for rows.Next() {
//		tunnel, err := t.scanTunnel(rows)
//		if err != nil {
//			return nil, err
//		}
//		tunnels = append(tunnels, *tunnel)
//	}
//
//	if err := rows.Err(); err != nil {
//		return nil, err
//	}
//
//	return tunnels, nil
//}
//
//func (t *Tunnels) scanTunnel(scanner scanner) (*tunnel.NormalTunnel, error) {
//	var tunnel tunnel.NormalTunnel
//	if err := scanner.Scan(
//		&tunnel.ID,
//		&tunnel.CreatedAt,
//		&tunnel.PublicKey,
//		&tunnel.PrivateKey,
//		&tunnel.Port,
//		&tunnel.ServerEndpoint,
//		&tunnel.ServerPort,
//		&tunnel.ServiceEndpoint,
//		&tunnel.ServicePort,
//	); err != nil {
//		return nil, err
//	}
//	return &tunnel, nil
//}
//
//var ErrTunnelNotFound = errors.New("tunnel not found")
//
//const createTunnel = `
//INSERT INTO tunnel (id, public_key, private_key, port, server_endpoint, server_port, service_endpoint, service_port)
//VALUES ($1, $2, $3, $4, $5)
//`
//
//const getTunnel = `
//SELECT id, created_at, public_key, private_key, port, server_endpoint, server_port, service_endpoint, service_port
//FROM tunnel
//WHERE id = $1
//`
//
//const listTunnels = `
//SELECT id, created_at, public_key, private_key, port, server_endpoint, server_port, service_endpoint, service_port
//FROM tunnel
//`
//
