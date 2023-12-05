package tunnel

import (
	"context"
	"github.com/gliderlabs/ssh"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/stats"
	"github.com/hightouchio/passage/tunnel/discovery"
	"github.com/hightouchio/passage/tunnel/keystore"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/pkg/errors"
)

type ReverseTunnel struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Enabled   bool      `json:"enabled"`

	SSHDPort           int `json:"sshdPort"`
	AuthorizedKeysHash string

	services ReverseTunnelServices
}

func (t ReverseTunnel) Start(ctx context.Context, listener *net.TCPListener, statusUpdate chan<- StatusUpdate) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	authorizedKeys, err := t.getAuthorizedKeys(ctx)
	if err != nil {
		return errors.Wrap(err, "get authorized keys")
	}

	// Register this tunnel with the global reverse SSH server
	if t.services.GlobalSSHServer != nil {
		statusUpdate <- StatusUpdate{StatusBooting, "SSHD server is online. Waiting for connections"}

		t.services.GlobalSSHServer.RegisterTunnel(SSHServerRegisteredTunnel{
			ID:             t.ID,
			AuthorizedKeys: authorizedKeys,
			Listener:       listener,

			// This is not actually the port that the tunnel is listening on,
			//	but the port that the tunnel is *registered* on, which is how we uniquely identify incoming requests
			//	for tunnels.
			RegisteredPort: t.SSHDPort,

			StatusUpdate: statusUpdate,
			Logger:       log.FromContext(ctx),
			Stats:        stats.GetStats(ctx),
		})
		defer t.services.GlobalSSHServer.DeregisterTunnel(t.ID)
	}

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

// ReverseTunnelServices are the external dependencies that ReverseTunnel needs to do its job
type ReverseTunnelServices struct {
	SQL interface {
		GetReverseTunnelAuthorizedKeys(ctx context.Context, tunnelID uuid.UUID) ([]postgres.Key, error)
	}
	GlobalSSHServer *SSHServer
	Keystore        keystore.Keystore
	Discovery       discovery.DiscoveryService

	EnableIndividualSSHD bool
	GetIndividualSSHD    func(sshdPort int) *SSHServer
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

	return t.ID == t2.ID && t.SSHDPort == t2.SSHDPort && t.AuthorizedKeysHash == t2.AuthorizedKeysHash
}

// convert a SQL DB representation of a postgres.ReverseTunnel into the primary ReverseTunnel struct
func reverseTunnelFromSQL(record postgres.ReverseTunnel) ReverseTunnel {
	return ReverseTunnel{
		ID:                 record.ID,
		CreatedAt:          record.CreatedAt,
		Enabled:            record.Enabled,
		SSHDPort:           record.SSHDPort,
		AuthorizedKeysHash: record.AuthorizedKeysHash,
	}
}

func (t ReverseTunnel) GetID() uuid.UUID {
	return t.ID
}
