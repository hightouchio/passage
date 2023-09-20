package tunnel

import (
	"context"
	"github.com/gliderlabs/ssh"
	"github.com/hightouchio/passage/stats"
	"github.com/hightouchio/passage/tunnel/discovery"
	"github.com/hightouchio/passage/tunnel/keystore"
	"time"

	"github.com/google/uuid"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type ReverseTunnel struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Enabled   bool      `json:"enabled"`

	SSHDPort   int  `json:"sshdPort"`
	TunnelPort int  `json:"tunnelPort"`
	HTTPProxy  bool `json:"httpProxy"`

	services ReverseTunnelServices

	Error *string `json:"error"`
}

func (t ReverseTunnel) Start(ctx context.Context, tunnelOptions TunnelOptions) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	authorizedKeys, err := t.getAuthorizedKeys(ctx)
	if err != nil {
		return bootError{event: "get_authorized_keys", err: err}
	}

	t.services.SSHServer.RegisterTunnel(t.ID, t.TunnelPort, authorizedKeys, getCtxLifecycle(ctx), stats.GetStats(ctx))
	defer t.services.SSHServer.DeregisterTunnel(t.ID)

	// Wait for tunnel closure
	<-ctx.Done()

	return nil
}

func (t ReverseTunnel) getAuthorizedKeys(ctx context.Context) ([]ssh.PublicKey, error) {
	registeredKeys, err := t.services.SQL.GetReverseTunnelAuthorizedKeys(ctx, t.ID)
	if err != nil {
		return []ssh.PublicKey{}, errors.Wrap(err, "could not read keys from database")
	}

	authorizedKeys := make([]ssh.PublicKey, len(registeredKeys))
	for i, registeredKey := range registeredKeys {
		keyBytes, err := t.services.Keystore.Get(ctx, registeredKey.ID)
		if err != nil {
			return authorizedKeys, errors.Wrapf(err, "could not read key %s from keystore", registeredKey.ID.String())
		}

		key, _, _, _, err := ssh.ParseAuthorizedKey(keyBytes)
		if err != nil {
			return authorizedKeys, errors.Wrapf(err, "could not parse key %s", registeredKey.ID.String())
		}

		authorizedKeys[i] = key
	}

	return authorizedKeys, nil
}

func (t ReverseTunnel) GetConnectionDetails(discovery discovery.DiscoveryService) (ConnectionDetails, error) {
	tunnelHost, err := discovery.ResolveTunnelHost(Reverse, t.ID)
	if err != nil {
		return ConnectionDetails{}, errors.Wrap(err, "could not resolve tunnel host")
	}

	return ConnectionDetails{
		Host: tunnelHost,
		Port: t.TunnelPort,
	}, nil
}

// ReverseTunnelServices are the external dependencies that ReverseTunnel needs to do its job
type ReverseTunnelServices struct {
	*SSHServer
	SQL interface {
		GetReverseTunnelAuthorizedKeys(ctx context.Context, tunnelID uuid.UUID) ([]postgres.Key, error)
	}
	Keystore keystore.Keystore
	Logger   *logrus.Logger
}

func InjectReverseTunnelDependencies(f func(ctx context.Context) ([]ReverseTunnel, error), services ReverseTunnelServices) ListFunc {
	return func(ctx context.Context) ([]Tunnel, error) {
		sts, err := f(ctx)
		if err != nil {
			return []Tunnel{}, err
		}
		// Inject dependencies
		tunnels := make([]Tunnel, len(sts))
		for i, st := range sts {
			st.services = services
			tunnels[i] = st
		}
		return tunnels, nil
	}
}

func (t ReverseTunnel) Equal(v interface{}) bool {
	t2, ok := v.(ReverseTunnel)
	if !ok {
		return false
	}

	return t.ID == t2.ID && t.TunnelPort == t2.TunnelPort && t.SSHDPort == t2.SSHDPort && t.HTTPProxy == t2.HTTPProxy
}

// convert a SQL DB representation of a postgres.ReverseTunnel into the primary ReverseTunnel struct
func reverseTunnelFromSQL(record postgres.ReverseTunnel) ReverseTunnel {
	return ReverseTunnel{
		ID:         record.ID,
		CreatedAt:  record.CreatedAt,
		Enabled:    record.Enabled,
		TunnelPort: record.TunnelPort,
		SSHDPort:   record.SSHDPort,
		HTTPProxy:  record.HTTPProxy,
	}
}

func (t ReverseTunnel) GetID() uuid.UUID {
	return t.ID
}

func (t ReverseTunnel) GetError() *string {
	return t.Error
}
