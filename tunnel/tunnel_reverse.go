package tunnel

import (
	"context"
	"fmt"
	"github.com/gliderlabs/ssh"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	gossh "golang.org/x/crypto/ssh"
	"time"
)

type ReverseTunnel struct {
	ID         uuid.UUID `json:"id"`
	CreatedAt  time.Time `json:"createdAt"`
	SSHDPort   uint32    `json:"sshPort"`
	TunnelPort uint32    `json:"tunnelPort"`

	services reverseTunnelServices
}

// reverseTunnelServices are the external dependencies that ReverseTunnel needs to do its job
type reverseTunnelServices struct {
	sql interface {
		GetReverseTunnelAuthorizedKeys(ctx context.Context, tunnelID uuid.UUID) ([]postgres.Key, error)
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
			t.logger().Info("new session")
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
			return bindHost == options.BindHost && bindPort == t.TunnelPort
		},
	}

	// integrate public key auth
	if err := sshServer.SetOption(ssh.PublicKeyAuth(func(ctx ssh.Context, incomingKey ssh.PublicKey) bool {
		log := t.logger().WithField("key_type", incomingKey.Type())

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

	t.logger().WithField("ssh_port", t.SSHDPort).Info("started tunnel")

	return sshServer.ListenAndServe()
}

// compare incoming connection key to the key authorized for this tunnel configuration
func (t ReverseTunnel) isAuthorizedKey(ctx context.Context, testKey ssh.PublicKey) (bool, error) {
	authorizedKeys, err := t.services.sql.GetReverseTunnelAuthorizedKeys(ctx, t.ID)
	if err != nil {
		return false, errors.Wrap(err, "could not get keys from db")
	}

	// check all authorized keys configured for this tunnel
	for _, key := range authorizedKeys {
		authorizedKey, _, _, _, err := gossh.ParseAuthorizedKey([]byte(key.Contents))
		if err != nil {
			return false, errors.Wrapf(err, "could not parse key %d", key.ID)
		}

		if ssh.KeysEqual(testKey, authorizedKey) {
			return true, nil
		}
	}

	return false, nil
}

func (t ReverseTunnel) GetConnectionDetails() ConnectionDetails {
	return ConnectionDetails{
		Host: "localhost", // TODO: need to get current server public IP
		Port: t.TunnelPort,
	}
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
		ID:         record.ID,
		CreatedAt:  record.CreatedAt,
		TunnelPort: record.TunnelPort,
		SSHDPort:   record.SSHDPort,
	}
}

func (t ReverseTunnel) GetID() uuid.UUID {
	return t.ID
}

func (t ReverseTunnel) logger() *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		"tunnel_type": "reverse",
		"tunnel_id":   t.ID.String(),
	})
}
