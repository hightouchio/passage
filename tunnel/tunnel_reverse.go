package tunnel

import (
	"context"
	"fmt"
	"github.com/hightouchio/passage/stats"
	"github.com/hightouchio/passage/tunnel/discovery"
	"github.com/hightouchio/passage/tunnel/keystore"
	"io"
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

	SSHDPort      int `json:"sshdPort"`
	TunnelPort    int `json:"tunnelPort"`
	services      ReverseTunnelServices
	serverOptions SSHServerOptions

	upstream  *reverseTunnelUpstream
	sshServer *ssh.Server
}

func (t *ReverseTunnel) Start(ctx context.Context, tunnelOptions TunnelOptions) error {
	lifecycle := getCtxLifecycle(ctx)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Configure SSHD
	serverAddr := net.JoinHostPort(t.serverOptions.BindHost, strconv.Itoa(t.SSHDPort))
	t.sshServer = &ssh.Server{
		Addr: serverAddr,
		ChannelHandlers: map[string]ssh.ChannelHandler{
			"session": ssh.DefaultSessionHandler,
		},
	}
	defer t.sshServer.Close()

	if err := t.configureAuth(ctx, t.serverOptions); err != nil {
		return bootError{event: "configure_auth", err: err}
	}

	forwardingHandler, err := t.configurePortForwarding(ctx, t.serverOptions, tunnelOptions)
	if err != nil {
		return bootError{event: "configure_port_forwarding", err: err}
	}
	t.upstream = forwardingHandler.upstream
	defer t.upstream.Close()

	lifecycle.BootEvent("sshd_start", stats.Tags{"listen_addr": serverAddr})

	errs := make(chan error)
	go func() {
		if err := t.sshServer.ListenAndServe(); err != nil {
			errs <- bootError{event: "sshd_start", err: errors.Wrap(err, "could not start SSH server")}
		}
	}()

	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
		return nil
	}
}

// Dial creates a new upstream connection to a given address
func (t *ReverseTunnel) Dial(downstream net.Conn, addr string) (io.ReadWriteCloser, error) {
	if t.upstream == nil {
		return nil, fmt.Errorf("reverse tunnel does not have an upstream yet")
	}

	return t.upstream.Dial(downstream, addr)
}

func (t *ReverseTunnel) configureAuth(ctx context.Context, serverOptions SSHServerOptions) error {
	lifecycle := getCtxLifecycle(ctx)

	// Init host key signing
	hostSigners, err := serverOptions.GetHostSigners()
	if err != nil {
		return errors.Wrap(err, "could not get host signers")
	}
	t.sshServer.HostSigners = hostSigners

	// Match incoming auth requests against stored public keys.
	if err := t.sshServer.SetOption(ssh.PublicKeyAuth(func(ctx ssh.Context, incomingKey ssh.PublicKey) bool {
		lifecycle.BootEvent("auth_request", stats.Tags{
			"session_id":  ctx.SessionID(),
			"remote_addr": ctx.RemoteAddr().String(),
			"user":        ctx.User(),
			"key_type":    incomingKey.Type(),
			"fingerprint": gossh.FingerprintSHA256(incomingKey),
		})

		// Check if there's a public key match.
		ok, err := t.isAuthorizedKey(ctx, incomingKey)
		if err != nil {
			lifecycle.BootError(errors.Wrap(err, "validate authorized key"))
			return false
		}

		if ok {
			lifecycle.BootEvent("auth_success", stats.Tags{"session_id": ctx.SessionID()})
		} else {
			lifecycle.BootEvent("auth_reject", stats.Tags{"session_id": ctx.SessionID()})
		}

		return ok
	})); err != nil {
		return err
	}

	return nil
}

func (t *ReverseTunnel) configurePortForwarding(ctx context.Context, serverOptions SSHServerOptions, tunnelOptions TunnelOptions) (*ReverseForwardingHandler, error) {
	lifecycle := getCtxLifecycle(ctx)
	st := stats.GetStats(ctx)

	// SSH session handler. Hold connections open until cancelled.
	t.sshServer.Handler = func(s ssh.Session) {
		select {
		case <-s.Context().Done(): // Block until session ends
		case <-ctx.Done(): // or until server closes
		}
	}

	// Add request handlers for reverse port forwarding
	handler := &ReverseForwardingHandler{
		tlsConfig: tunnelOptions.TLSConfig,
		stats:     st,
		lifecycle: lifecycle,
		upstream:  &reverseTunnelUpstream{},
	}
	t.sshServer.RequestHandlers = map[string]ssh.RequestHandler{
		"tcpip-forward":        handler.HandleSSHRequest,
		"cancel-tcpip-forward": handler.HandleSSHRequest,
	}

	// Validate incoming port forward requests. SSH clients should only be able to forward to their assigned tunnel port (bind port).
	t.sshServer.ReversePortForwardingCallback = func(ctx ssh.Context, bindHost string, bindPort uint32) bool {
		success := bindHost == t.serverOptions.BindHost && int(bindPort) == t.TunnelPort

		lifecycle.BootEvent("port_forward_request", stats.Tags{
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

	return handler, nil
}

// compare incoming connection key to the key authorized for this tunnel configuration
func (t *ReverseTunnel) isAuthorizedKey(ctx context.Context, testKey ssh.PublicKey) (bool, error) {
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

func (t *ReverseTunnel) GetConnectionDetails(discovery discovery.DiscoveryService) (ConnectionDetails, error) {
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

func InjectReverseTunnelDependencies(f func(ctx context.Context) ([]*ReverseTunnel, error), services ReverseTunnelServices, options SSHServerOptions) ListFunc {
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

func (t *ReverseTunnel) Equal(v interface{}) bool {
	t2, ok := v.(*ReverseTunnel)
	if !ok {
		return false
	}

	return t.ID == t2.ID && t.TunnelPort == t2.TunnelPort && t.SSHDPort == t2.SSHDPort
}

// convert a SQL DB representation of a postgres.ReverseTunnel into the primary ReverseTunnel struct
func reverseTunnelFromSQL(record postgres.ReverseTunnel) *ReverseTunnel {
	return &ReverseTunnel{
		ID:         record.ID,
		CreatedAt:  record.CreatedAt,
		Enabled:    record.Enabled,
		TunnelPort: record.TunnelPort,
		SSHDPort:   record.SSHDPort,
	}
}

func (t *ReverseTunnel) GetID() uuid.UUID {
	return t.ID
}

func (t *ReverseTunnel) GetError() *string {
	// TODO: Implement
	return nil
}
