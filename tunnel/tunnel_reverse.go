package tunnel

import (
	"context"
	"fmt"
	"github.com/hightouchio/passage/stats"
	"github.com/hightouchio/passage/tunnel/discovery"
	"github.com/hightouchio/passage/tunnel/keystore"
	"time"

	"github.com/gliderlabs/ssh"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	gossh "golang.org/x/crypto/ssh"
)

type ReverseTunnel struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Enabled   bool      `json:"enabled"`

	SSHDPort   int `json:"sshdPort"`
	TunnelPort int `json:"tunnelPort"`

	services      ReverseTunnelServices
	serverOptions SSHServerOptions
}

func (t ReverseTunnel) Start(ctx context.Context, tunnelOptions TunnelOptions) error {
	st := stats.GetStats(ctx)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sshd, err := t.newSSHServer(ctx, t.serverOptions, tunnelOptions)
	if err != nil {
		return errors.Wrap(err, "init sshd")
	}
	defer func() {
		sshd.Close()
	}()

	errs := make(chan error)
	go func() {
		st.WithEventTags(stats.Tags{"port": t.SSHDPort}).SimpleEvent("sshd.start")
		errs <- sshd.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		return nil

	case err := <-errs:
		return err
	}
}

func (t ReverseTunnel) newSSHServer(ctx context.Context, serverOptions SSHServerOptions, tunnelOptions TunnelOptions) (*ssh.Server, error) {
	st := stats.GetStats(ctx)

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
		st := st.WithEventTags(stats.Tags{"remoteAddr": s.RemoteAddr().String()})
		st.SimpleEvent("session.start")
		st.Incr("session.start", nil, 1)
		select {
		case <-s.Context().Done():
			st.SimpleEvent("session.end")
		}
	}

	// get the server-side Host Key signers
	hostSigners, err := serverOptions.GetHostSigners()
	if err != nil {
		return nil, errors.Wrap(err, "could not get host signers")
	}
	server.HostSigners = hostSigners

	// validate port forwarding
	server.ReversePortForwardingCallback = func(ctx ssh.Context, bindHost string, bindPort uint32) bool {
		success := bindHost == tunnelOptions.BindHost && int(bindPort) == t.TunnelPort

		st.WithEventTags(stats.Tags{
			"sessionId":       ctx.SessionID(),
			"remoteAddr":      ctx.RemoteAddr().String(),
			"requestBindHost": bindHost,
			"requestBindPort": bindPort,
			"configBindHost":  tunnelOptions.BindHost,
			"configBindPort":  t.TunnelPort,
			"success":         success,
		}).SimpleEvent("session.portForwardRequest")

		return success
	}

	// integrate public key auth
	if err := server.SetOption(ssh.PublicKeyAuth(func(ctx ssh.Context, incomingKey ssh.PublicKey) bool {
		sessSt := st.WithEventTags(stats.Tags{
			"sessionId":  ctx.SessionID(),
			"remoteAddr": ctx.RemoteAddr().String(),
			"keyType":    incomingKey.Type(),
		})

		ok, err := t.isAuthorizedKey(ctx, incomingKey)
		if err != nil {
			sessSt.ErrorEvent("session.authRequest.error", err)
			return false
		}

		sessSt.WithTags(stats.Tags{"success": ok}).SimpleEvent("session.authRequest")
		return ok
	})); err != nil {
		return nil, err
	}

	return server, nil
}

// compare incoming connection key to the key authorized for this tunnel configuration
func (t ReverseTunnel) isAuthorizedKey(ctx context.Context, testKey ssh.PublicKey) (bool, error) {
	authorizedKeys, err := t.services.SQL.GetReverseTunnelAuthorizedKeys(ctx, t.ID)
	if err != nil {
		return false, errors.Wrap(err, "could not get keys from db")
	}

	// check all authorized keys configured for this tunnel
	for _, authorizedKey := range authorizedKeys {
		id := authorizedKey.ID
		// retrieve key contents
		key, err := t.services.Keystore.Get(ctx, id)
		if err != nil {
			return false, errors.Wrapf(err, "could not resolve contents for key %s", authorizedKey.ID.String())
		}

		// compare stored authorized key to test key
		authorizedKey, _, _, _, err := gossh.ParseAuthorizedKey([]byte(key))
		if err != nil {
			return false, errors.Wrapf(err, "could not parse key %d", id)
		}
		if ssh.KeysEqual(testKey, authorizedKey) {
			return true, nil
		}
	}

	return false, nil
}

func (t ReverseTunnel) GetConnectionDetails(discovery discovery.DiscoveryService) (ConnectionDetails, error) {
	tunnelHost, err := discovery.ResolveTunnelHost("standard", t.ID)
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
	SQL interface {
		GetReverseTunnelAuthorizedKeys(ctx context.Context, tunnelID uuid.UUID) ([]postgres.Key, error)
	}
	Keystore keystore.Keystore
}

func InjectReverseTunnelDependencies(f func(ctx context.Context) ([]ReverseTunnel, error), services ReverseTunnelServices, options SSHServerOptions) ListFunc {
	return func(ctx context.Context) ([]Tunnel, error) {
		// Get standard tunnels
		sts, err := f(ctx)
		if err != nil {
			return []Tunnel{}, err
		}
		// Inject ClientOptions into StandardTunnels
		tunnels := make([]Tunnel, len(sts))
		for i, st := range sts {
			st.services = services
			st.serverOptions = options
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

	return t.ID == t2.ID && t.TunnelPort == t2.TunnelPort && t.SSHDPort == t2.SSHDPort
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
