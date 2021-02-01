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
		id string,
		tunnelType models.TunnelType,
		publicKey string,
		privateKey string,
		port uint32,
	) (*models.Tunnel, error)
	Get(
		ctx context.Context,
		id string,
	) (*models.Tunnel, error)
	List(
		ctx context.Context,
	) ([]models.Tunnel, error)
}
