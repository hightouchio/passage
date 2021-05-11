package store

import (
	"context"
	"errors"
	"github.com/hightouchio/passage/tunnel"
)

var ErrTunnelNotFound = errors.New("tunnel not found")

type Tunnels interface {
	Create(
		ctx context.Context,
		tunnel tunnel.NormalTunnel,
	) (*tunnel.NormalTunnel, error)
	Get(
		ctx context.Context,
		id string,
	) (*tunnel.NormalTunnel, error)
	List(
		ctx context.Context,
	) ([]tunnel.NormalTunnel, error)
}

var ErrReverseTunnelNotFound = errors.New("reverse tunnel not found")

type ReverseTunnels interface {
	Create(
		ctx context.Context,
		reverseTunnel tunnel.ReverseTunnel,
	) (*tunnel.ReverseTunnel, error)
	Get(
		ctx context.Context,
		id int,
	) (*tunnel.ReverseTunnel, error)
	List(
		ctx context.Context,
	) ([]tunnel.ReverseTunnel, error)
}
