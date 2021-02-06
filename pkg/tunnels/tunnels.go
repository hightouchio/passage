package tunnels

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"github.com/hightouchio/passage/pkg/models"
	"github.com/hightouchio/passage/pkg/store"
	"golang.org/x/crypto/ssh"
)

type Tunnels struct {
	tunnels store.Tunnels
}

func NewTunnels(tunnels store.Tunnels) *Tunnels {
	return &Tunnels{
		tunnels: tunnels,
	}
}

func (t *Tunnels) Create(
	ctx context.Context,
	id string,
	serviceEndpoint string,
	servicePort uint32,
) (*models.Tunnel, error) {
	public, private, err := generateKeyPair()
	if err != nil {
		return nil, err
	}
	return t.tunnels.Create(ctx, models.Tunnel{
		ID:             id,
		PublicKey:      public,
		PrivateKey:     private,
		Port:           1,
		ServiceEndpont: serviceEndpoint,
		ServicePort:    servicePort,
	})
}

func (t *Tunnels) Get(
	ctx context.Context,
	id string,
) (*models.Tunnel, error) {
	return t.tunnels.Get(ctx, id)
}

func (t *Tunnels) List(
	ctx context.Context,
) ([]models.Tunnel, error) {
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

func (t *ReverseTunnels) Create(
	ctx context.Context,
	id string,
) (*models.ReverseTunnel, error) {
	public, private, err := generateKeyPair()
	if err != nil {
		return nil, err
	}
	return t.reverseTunnels.Create(ctx, models.ReverseTunnel{
		ID:         id,
		PublicKey:  public,
		PrivateKey: private,
		Port:       1,
		SSHPort:    1,
	})
}

func (t *ReverseTunnels) Get(
	ctx context.Context,
	id string,
) (*models.ReverseTunnel, error) {
	return t.reverseTunnels.Get(ctx, id)
}

func (t *ReverseTunnels) List(
	ctx context.Context,
) ([]models.ReverseTunnel, error) {
	return t.reverseTunnels.List(ctx)
}

func generateKeyPair() (string, string, error) {
	return "", "", nil

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	if err = privateKey.Validate(); err != nil {
		return "", "", err
	}

	publicKey, err := ssh.NewPublicKey(privateKey)
	if err != nil {
		return "", "", err
	}

	return string(ssh.MarshalAuthorizedKey(publicKey)), string(pem.EncodeToMemory(&pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   x509.MarshalPKCS1PrivateKey(privateKey),
	})), nil
}
