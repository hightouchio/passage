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
	"os"
	"time"
)

type ReverseTunnel struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Enabled   bool      `json:"enabled"`

	SSHDPort   int `json:"sshdPort"`
	TunnelPort int `json:"tunnelPort"`

	services reverseTunnelServices
}

// reverseTunnelServices are the external dependencies that ReverseTunnel needs to do its job
type reverseTunnelServices struct {
	sql interface {
		GetReverseTunnelAuthorizedKeys(ctx context.Context, tunnelID uuid.UUID) ([]postgres.Key, error)
	}
}

func (t ReverseTunnel) Start(ctx context.Context, options SSHOptions) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sshd, err := t.newSSHServer(options)
	if err != nil {
		return errors.Wrap(err, "init sshd")
	}
	defer func() {
		t.logger().Debug("start sshd")
		sshd.Close()
	}()

	errs := make(chan error)
	go func() {
		t.logger().WithField("port", t.SSHDPort).Debug("start sshd")
		errs <- sshd.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		return nil

	case err := <-errs:
		return err
	}

}

func (t ReverseTunnel) newSSHServer(options SSHOptions) (*ssh.Server, error) {
	server := &ssh.Server{
		Addr: fmt.Sprintf(":%d", t.SSHDPort),
		ChannelHandlers: map[string]ssh.ChannelHandler{
			"session":      ssh.DefaultSessionHandler,
			"direct-tcpip": ssh.DirectTCPIPHandler,
		},
	}

	// add request handlers
	forwardHandler := &ssh.ForwardedTCPHandler{}
	server.RequestHandlers = map[string]ssh.RequestHandler{
		"tcpip-forward":        forwardHandler.HandleSSHRequest,
		"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
	}

	// request session handler
	server.Handler = func(s ssh.Session) {
		t.logger().WithField("remote_addr", s.RemoteAddr().String()).Debug("new session")
		select {}
	}

	// get the server-side Host Key signers
	hostSigners, err := options.GetHostSigners()
	if err != nil {
		return nil, errors.Wrap(err, "could not get host signers")
	}
	server.HostSigners = hostSigners

	// validate port forwarding
	server.ReversePortForwardingCallback = func(ctx ssh.Context, bindHost string, bindPort uint32) bool {
		return bindHost == options.BindHost && int(bindPort) == t.TunnelPort
	}

	// integrate public key auth
	if err := server.SetOption(ssh.PublicKeyAuth(func(ctx ssh.Context, incomingKey ssh.PublicKey) bool {
		log := t.logger().WithField("key_type", incomingKey.Type())

		ok, err := t.isAuthorizedKey(ctx, incomingKey)
		if err != nil {
			log.WithError(err).Error("could not authorize key")
			return false
		}

		log.WithField("success", ok).Debug("public key auth attempt")
		return ok
	})); err != nil {
		return nil, err
	}

	return server, nil
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

func (t ReverseTunnel) Equal(v interface{}) bool {
	t2, ok := v.(ReverseTunnel)
	if !ok {
		return false
	}

	return t.ID == t2.ID && t.TunnelPort == t2.TunnelPort && t.SSHDPort == t2.SSHDPort
}

func (t ReverseTunnel) GetConnectionDetails() ConnectionDetails {
	return ConnectionDetails{
		Host: os.Getenv("TUNNEL_HOST_REVERSE"), // TODO: need to get current server public IP
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
		Enabled:    record.Enabled,
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
