package tunnels

import (
	"context"
	"github.com/hightouchio/passage/pkg/models"
	"github.com/hightouchio/passage/pkg/ssh"
	"github.com/hightouchio/passage/pkg/store"
)

type Tunnels struct {
	tunnels store.Tunnels
}

func NewTunnels(tunnels store.Tunnels) *Tunnels {
	return &Tunnels{
		tunnels: tunnels,
	}
}

func (t *Tunnels) Create(ctx context.Context, id string, serviceEndpoint string, servicePort uint32, keys ssh.KeyPair) (*models.Tunnel, error) {
	return t.tunnels.Create(ctx, models.Tunnel{
		ID:              id,
		PublicKey:       keys.PublicKey,
		PrivateKey:      keys.PrivateKey,
		Port:            1,
		ServiceEndpoint: serviceEndpoint,
		ServicePort:     servicePort,
	})
}

func (t *Tunnels) Get(ctx context.Context, id string) (*models.Tunnel, error) {
	return t.tunnels.Get(ctx, id)
}

func (t *Tunnels) List(ctx context.Context) ([]models.Tunnel, error) {
	return t.tunnels.List(ctx)
}

type ReverseTunnels struct {
	reverseTunnels store.ReverseTunnels
}

func NewReverseTunnels(reverseTunnels store.ReverseTunnels) *ReverseTunnels {
	return &ReverseTunnels{
		reverseTunnels: reverseTunnels,
	}
}

func (t *ReverseTunnels) Create(ctx context.Context, keys ssh.KeyPair) (*models.ReverseTunnel, error) {
	record, err := t.reverseTunnels.Create(ctx, models.ReverseTunnel{
		PublicKey: keys.PublicKey,
	})
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (t *ReverseTunnels) Get(ctx context.Context, id int) (*models.ReverseTunnel, error) {
	return t.reverseTunnels.Get(ctx, id)
}

func (t *ReverseTunnels) List(ctx context.Context) ([]models.ReverseTunnel, error) {
	return t.reverseTunnels.List(ctx)
}