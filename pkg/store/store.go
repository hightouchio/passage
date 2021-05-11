package store

import (
	"context"
	"errors"
	"github.com/hightouchio/passage/pkg/models"
)

var ErrTunnelNotFound = errors.New("tunnel not found")

type Tunnels interface {
	Create(
		ctx context.Context,
		tunnel models.Tunnel,
	) (*models.Tunnel, error)
	Get(
		ctx context.Context,
		id string,
	) (*models.Tunnel, error)
	List(
		ctx context.Context,
	) ([]models.Tunnel, error)
}

var ErrReverseTunnelNotFound = errors.New("reverse tunnel not found")

type ReverseTunnels interface {
	Create(
		ctx context.Context,
		reverseTunnel models.ReverseTunnel,
	) (*models.ReverseTunnel, error)
	Get(
		ctx context.Context,
		id int,
	) (*models.ReverseTunnel, error)
	List(
		ctx context.Context,
	) ([]models.ReverseTunnel, error)
}
