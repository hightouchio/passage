package tunnel

import (
	"context"
	"github.com/hightouchio/passage/stats"
	"github.com/hightouchio/passage/tunnel/discovery"
	"github.com/hightouchio/passage/tunnel/keystore"
	"net"
	"strconv"
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
	lifecycle := getCtxLifecycle(ctx)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Configure SSHD
	serverAddr := net.JoinHostPort(t.serverOptions.BindHost, strconv.Itoa(t.SSHDPort))
	server := &ssh.Server{
		Addr: serverAddr,
		ChannelHandlers: map[string]ssh.ChannelHandler{
			"session": ssh.DefaultSessionHandler,
		},
	}

	if err := t.configureAuth(ctx, server, t.serverOptions); err != nil {
		return bootError{event: "configure_auth", err: err}
	}

	if err := t.configurePortForwarding(ctx, server, t.serverOptions, tunnelOptions); err != nil {
		return bootError{event: "configure_port_forwarding", err: err}
	}
	defer func() {
		lifecycle.Close()
		server.Close()
	}()

	errs := make(chan error)
	go func() {
		lifecycle.BootEvent("sshd_start", stats.Tags{"sshd_addr": serverAddr})
		errs <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		return nil

	case err := <-errs:
		return err
	}
}

func (t ReverseTunnel) configureAuth(ctx context.Context, server *ssh.Server, serverOptions SSHServerOptions) error {
	lifecycle := getCtxLifecycle(ctx)

	// Init host key signing
	hostSigners, err := serverOptions.GetHostSigners()
	if err != nil {
		return errors.Wrap(err, "could not get host signers")
	}
	server.HostSigners = hostSigners

	// Match incoming auth requests against stored public keys.
	if err := server.SetOption(ssh.PublicKeyAuth(func(ctx ssh.Context, incomingKey ssh.PublicKey) bool {
		lifecycle.SessionEvent(ctx.SessionID(), "auth_request", stats.Tags{
			"session_id":  ctx.SessionID(),
			"remote_addr": ctx.RemoteAddr().String(),
			"user":        ctx.User(),
			"key_type":    incomingKey.Type(),
			"fingerprint": gossh.FingerprintSHA256(incomingKey),
		})

		// Check if there's a public key match.
		ok, err := t.isAuthorizedKey(ctx, incomingKey)
		if err != nil {
			lifecycle.SessionError(ctx.SessionID(), errors.Wrap(err, "validate authorized key"))
			return false
		}

		if ok {
			lifecycle.SessionEvent(ctx.SessionID(), "auth_success", stats.Tags{})
		} else {
			lifecycle.SessionEvent(ctx.SessionID(), "auth_reject", stats.Tags{})
		}

		return ok
	})); err != nil {
		return err
	}

	return nil
}

func (t ReverseTunnel) configurePortForwarding(ctx context.Context, server *ssh.Server, serverOptions SSHServerOptions, tunnelOptions TunnelOptions) error {
	lifecycle := getCtxLifecycle(ctx)

	// SSH session handler. Hold connections open until cancelled.
	server.Handler = func(s ssh.Session) {
		lifecycle.Open()
		defer func() {
			lifecycle.SessionEvent("", "session_end", stats.Tags{})
			lifecycle.Close()
		}()

		lifecycle.SessionEvent("", "session_start", stats.Tags{
			"remote_addr": s.RemoteAddr().String(),
		})

		select {
		case <-s.Context().Done(): // Block until session ends
		case <-ctx.Done(): // or until server closes
		}
	}

	// Add request handlers for reverse port forwarding
	forwardHandler := &ForwardedTCPHandler{}
	server.RequestHandlers = map[string]ssh.RequestHandler{
		"tcpip-forward":        forwardHandler.HandleSSHRequest,
		"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
	}

	// Validate incoming port forward requests. SSH clients should only be able to forward to their assigned tunnel port (bind port).
	server.ReversePortForwardingCallback = func(ctx ssh.Context, bindHost string, bindPort uint32) bool {
		success := bindHost == t.serverOptions.BindHost && int(bindPort) == t.TunnelPort

		lifecycle.SessionEvent(ctx.SessionID(), "port_forward_request", stats.Tags{
			"session_id":        ctx.SessionID(),
			"remote_addr":       ctx.RemoteAddr().String(),
			"request_bind_host": bindHost,
			"request_bind_port": bindPort,
			"config_bind_host":  t.serverOptions.BindHost,
			"config_bind_port":  t.TunnelPort,
			"success":           success,
		})

		return success
	}

	return nil
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
			return false, errors.Wrapf(err, "could not get key %s", authorizedKey.ID.String())
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
	SQL interface {
		GetReverseTunnelAuthorizedKeys(ctx context.Context, tunnelID uuid.UUID) ([]postgres.Key, error)
	}
	Keystore keystore.Keystore
	Logger   *logrus.Logger
}

func InjectReverseTunnelDependencies(f func(ctx context.Context) ([]ReverseTunnel, error), services ReverseTunnelServices, options SSHServerOptions) ListFunc {
	return func(ctx context.Context) ([]Tunnel, error) {
		sts, err := f(ctx)
		if err != nil {
			return []Tunnel{}, err
		}
		// Inject dependencies
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
