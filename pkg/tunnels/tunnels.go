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
	tunnelType models.TunnelType,
) (*models.Tunnel, error) {
	public, private, err := generateKeyPair()
	if err != nil {
		return nil, err
	}
	return t.tunnels.Create(ctx, id, tunnelType, public, private, 1)
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
