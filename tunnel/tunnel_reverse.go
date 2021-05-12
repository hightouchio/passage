package tunnel

import (
	"context"
	"fmt"
	"github.com/gliderlabs/ssh"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	gossh "golang.org/x/crypto/ssh"
	"time"
)

type ReverseTunnel struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	SSHDPort  uint32    `json:"sshPort"`
	Port      uint32    `json:"port"`

	services reverseTunnelServices
}

// reverseTunnelServices are the external dependencies that ReverseTunnel needs to do its job
type reverseTunnelServices struct {
	sql interface {
		GetReverseTunnelAuthorizedKeys(ctx context.Context, tunnelID int) ([]string, error)
	}
}

func (t ReverseTunnel) Start(ctx context.Context, options SSHOptions) error {
	var hostSigners []ssh.Signer

	if len(options.HostKey) != 0 {
		hostSigner, err := gossh.ParsePrivateKey(options.HostKey)
		if err != nil {
			return err
		}
		hostSigners = []ssh.Signer{hostSigner}
	}

	forwardHandler := &ssh.ForwardedTCPHandler{}
	sshServer := &ssh.Server{
		Addr: fmt.Sprintf(":%d", t.SSHDPort),
		Handler: func(s ssh.Session) {
			t.Logger().Info("new session")
			select {}
		},
		RequestHandlers: map[string]ssh.RequestHandler{
			"tcpip-forward":        forwardHandler.HandleSSHRequest,
			"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
		},
		ChannelHandlers: map[string]ssh.ChannelHandler{
			"session":      ssh.DefaultSessionHandler,
			"direct-tcpip": ssh.DirectTCPIPHandler,
		},
		HostSigners: hostSigners,
		ReversePortForwardingCallback: func(ctx ssh.Context, bindHost string, bindPort uint32) bool {
			return bindHost == options.BindHost && bindPort == t.Port
		},
	}

	// integrate public key auth
	if err := sshServer.SetOption(ssh.PublicKeyAuth(func(ctx ssh.Context, incomingKey ssh.PublicKey) bool {
		log := t.Logger().WithField("key_type", incomingKey.Type())

		ok, err := t.isAuthorizedKey(ctx, incomingKey)
		if err != nil {
			log.WithError(err).Error("could not authorize key")
			return false
		}

		if ok {
			log.Debug("accepted public key")
		} else {
			log.Debug("rejected public key")
		}

		return ok
	})); err != nil {
		return err
	}

	t.Logger().WithField("ssh_port", t.SSHDPort).Info("started tunnel")

	return sshServer.ListenAndServe()
}

// compare incoming connection key to the key authorized for this tunnel configuration
func (t ReverseTunnel) isAuthorizedKey(ctx context.Context, testKey ssh.PublicKey) (bool, error) {
	authorizedKeys, err := t.services.sql.GetReverseTunnelAuthorizedKeys(ctx, t.ID)
	if err != nil {
		return false, errors.Wrap(err, "could not get keys from db")
	}

	// check all authorized keys configured for this tunnel
	for i, key := range authorizedKeys {
		authorizedKey, comment, _, _, err := gossh.ParseAuthorizedKey([]byte(key))
		if err != nil {
			return false, errors.Errorf("could not parse key %d with comment %s", i, comment)
		}

		if ssh.KeysEqual(testKey, authorizedKey) {
			return true, nil
		}
	}

	return false, nil
}

func (t ReverseTunnel) Logger() *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		"tunnel_type": "reverse",
		"tunnel_id":   t.ID,
	})
}

func (t ReverseTunnel) GetID() int {
	return t.ID
}

// createReverseTunnelListFunc wraps our Postgres list function in something that converts the records into ReverseTunnel structs so they can be passed to Manager which accepts the Tunnel interface
func createReverseTunnelListFunc(postgresList func(ctx context.Context) ([]postgres.ReverseTunnel, error), services reverseTunnelServices) ListFunc {
	return func(ctx context.Context) ([]Tunnel, error) {
		reverseTunnels, err := postgresList(ctx)
		if err != nil {
			return []Tunnel{}, err
		}

		// convert all the SQL records to our primary struct
		tunnels := make([]Tunnel, len(reverseTunnels))
		for i, record := range reverseTunnels {
			tunnel := reverseTunnelFromSQL(record)
			tunnel.services = services // inject dependencies
			tunnels[i] = tunnel
		}

		return tunnels, nil
	}
}

// convert a SQL DB representation of a postgres.ReverseTunnel into the primary ReverseTunnel struct
func reverseTunnelFromSQL(record postgres.ReverseTunnel) ReverseTunnel {
	return ReverseTunnel{
		ID:        record.ID,
		CreatedAt: record.CreatedAt,
		Port:      record.TunnelPort,
		SSHDPort:  record.SSHDPort,
	}
}
