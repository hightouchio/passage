package tunnel

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/hightouchio/passage/stats"
	"github.com/hightouchio/passage/tunnel/discovery"
	"github.com/hightouchio/passage/tunnel/keystore"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

type NormalTunnel struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Enabled   bool      `json:"enabled"`

	TunnelPort  int     `json:"tunnelPort"`
	SSHUser     string  `json:"sshUser"`
	SSHHost     string  `json:"sshHost"`
	SSHPort     int     `json:"sshPort"`
	ServiceHost string  `json:"serviceHost"`
	ServicePort int     `json:"servicePort"`
	HTTPProxy   bool    `json:"httpProxy"`
	Error       *string `json:"error"`

	clientOptions SSHClientOptions
	services      NormalTunnelServices
}

func (t NormalTunnel) Start(ctx context.Context, options TunnelOptions) error {
	err := t.start(ctx, options)
	if err != nil {
		if err := t.services.SQL.UpdateNormalTunnelError(ctx, t.ID, err.Error()); err != nil {
			return errors.Wrap(err, "failed to persist tunnel start error")
		}
		return err
	}
	return nil
}

func (t NormalTunnel) start(ctx context.Context, options TunnelOptions) error {
	lifecycle := getCtxLifecycle(ctx)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Register tunnel with service discovery.
	if err := t.services.Discovery.RegisterTunnel(t.ID, t.TunnelPort); err != nil {
		return bootError{event: "service_discovery_register", err: err}
	}
	// TODO: Deregister tunnel when it *completely* shuts down

	// Update service discovery that SSH connection established, but not quite online
	t.services.Discovery.UpdateHealth(t.ID, discovery.TunnelWarning, "Booting")

	// Get a list of key signers to use for authentication
	keySigners, err := t.getAuthSigners(ctx)
	if err != nil {
		t.services.Discovery.UpdateHealth(t.ID, discovery.TunnelUnhealthy, fmt.Sprintf("Failed to generate authentication payload: %s", err.Error()))
		return bootError{event: "generate_auth_signers", err: err}
	}

	// Determine SSH user to use, either from the database or from the config.
	var sshUser string
	if t.SSHUser != "" {
		sshUser = t.SSHUser
	} else {
		sshUser = t.clientOptions.User
	}

	// Dial external SSH server
	lifecycle.BootEvent("remote_dial", stats.Tags{"ssh_host": t.SSHHost, "ssh_port": t.SSHPort})
	addr := net.JoinHostPort(t.SSHHost, strconv.Itoa(t.SSHPort))
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		t.services.Discovery.UpdateHealth(t.ID, discovery.TunnelUnhealthy, fmt.Sprintf("Failed to resolve remote address: %s", err.Error()))
		return bootError{event: "remote_dial", err: errors.Wrapf(err, "resolve addr %s", addr)}
	}
	sshConn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		t.services.Discovery.UpdateHealth(t.ID, discovery.TunnelUnhealthy, fmt.Sprintf("Failed to connect to remote server: %s", err.Error()))
		return bootError{event: "remote_dial", err: err}
	}
	defer sshConn.Close()
	// Configure TCP keepalive for SSH connection
	sshConn.SetKeepAlive(true)
	sshConn.SetKeepAlivePeriod(t.clientOptions.KeepaliveInterval)

	// Init SSH connection protocol
	lifecycle.BootEvent("ssh_connect", stats.Tags{"ssh_user": sshUser, "ssh_auth_method_count": len(keySigners)})
	c, chans, reqs, err := ssh.NewClientConn(
		sshConn, addr,
		&ssh.ClientConfig{
			User:            sshUser,
			Auth:            []ssh.AuthMethod{ssh.PublicKeys(keySigners...)},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		},
	)
	if err != nil {
		t.services.Discovery.UpdateHealth(t.ID, discovery.TunnelUnhealthy, err.Error())
		return bootError{event: "ssh_connect", err: err}
	}
	sshClient := ssh.NewClient(c, chans, reqs)

	// Update service discovery that SSH connection established, but not quite online
	t.services.Discovery.UpdateHealth(t.ID, discovery.TunnelWarning, "SSH connection established")

	// Start sending keepalive packets to the upstream SSH server
	go func() {
		if err := sshKeepaliver(ctx, sshConn, sshClient, t.clientOptions.KeepaliveInterval, t.clientOptions.DialTimeout); err != nil {
			lifecycle.Error(errors.Wrap(err, "ssh keepalive failed"))
			// Update service discovery that SSH connection established, but not quite online
			t.services.Discovery.UpdateHealth(t.ID, discovery.TunnelUnhealthy, fmt.Sprintf("SSH keepalive failed: %s", err.Error()))
			cancel()
		}
	}()

	// Configure TCPForwarder
	forwarder := &TCPForwarder{
		BindAddr:          net.JoinHostPort(options.BindHost, strconv.Itoa(t.TunnelPort)),
		HTTPProxyEnabled:  t.HTTPProxy,
		KeepaliveInterval: 5 * time.Second,

		// Implement GetUpstreamConn by initiating upstream connections through the SSH client.
		GetUpstreamConn: func(conn net.Conn) (io.ReadWriteCloser, error) {
			serviceConn, err := sshClient.Dial("tcp", net.JoinHostPort(t.ServiceHost, strconv.Itoa(t.ServicePort)))
			if err != nil {
				return nil, err
			}
			return serviceConn, err
		},

		Lifecycle: lifecycle,
		Stats:     stats.GetStats(ctx),
	}
	defer forwarder.Close()

	// Start tunnel listener
	if err := forwarder.Listen(); err != nil {
		t.services.Discovery.UpdateHealth(t.ID, discovery.TunnelUnhealthy, fmt.Sprintf("Failed to boot forwarder: %s", err.Error()))
		switch err.(type) {
		case bootError:
			lifecycle.BootError(err)
		default:
			lifecycle.Error(err)
		}
	}

	// Start port forwarding
	go func() {
		defer cancel()
		forwarder.Serve()
	}()

	// Start connectivity checker
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				if err := checkConnectivity(ctx, "localhost", t.TunnelPort); err != nil {
					lifecycle.Error(errors.Wrap(err, "connectivity check failed"))

					// Update service discovery that tunnel is unhealthy
					t.services.Discovery.UpdateHealth(t.ID, discovery.TunnelUnhealthy, err.Error())
					return
				}

				// Update service discovery that tunnel is healthy
				t.services.Discovery.UpdateHealth(t.ID, discovery.TunnelHealthy, "Tunnel is online")
			}
		}
	}()

	<-ctx.Done()
	return nil
}

// getAuthSigners finds the SSH keys that are configured for this tunnel and structure them for use by the SSH client library
func (t NormalTunnel) getAuthSigners(ctx context.Context) ([]ssh.Signer, error) {
	// get private keys from database
	keys, err := t.services.SQL.GetNormalTunnelPrivateKeys(ctx, t.ID)
	if err != nil {
		return []ssh.Signer{}, errors.Wrap(err, "could not look up private keys")
	}

	var signers []ssh.Signer

	// parse private keys and prepare for SSH
	for _, key := range keys {
		privateKeyBytes, err := t.services.Keystore.Get(ctx, key.ID)
		if err != nil {
			return []ssh.Signer{}, errors.Wrapf(err, "could not get contents for key %s", key.ID)
		}

		// Generate ssh.Signers for the private key
		keySigners, err := getSignersForPrivateKey(privateKeyBytes)
		if err != nil {
			return []ssh.Signer{}, errors.Wrap(err, "could not generate public key signers")
		}

		signers = append(signers, keySigners...)
	}

	return signers, nil
}

// NormalTunnelServices are the external dependencies that NormalTunnel needs to do its job
type NormalTunnelServices struct {
	SQL interface {
		GetNormalTunnelPrivateKeys(ctx context.Context, tunnelID uuid.UUID) ([]postgres.Key, error)
		UpdateNormalTunnelError(ctx context.Context, tunnelID uuid.UUID, error string) error
	}
	Keystore keystore.Keystore
	Logger   *logrus.Logger

	Discovery discovery.DiscoveryService
}

func InjectNormalTunnelDependencies(f func(ctx context.Context) ([]NormalTunnel, error), services NormalTunnelServices, options SSHClientOptions) ListFunc {
	return func(ctx context.Context) ([]Tunnel, error) {
		sts, err := f(ctx)
		if err != nil {
			return []Tunnel{}, err
		}
		tunnels := make([]Tunnel, len(sts))
		for i, st := range sts {
			st.services = services
			st.clientOptions = options
			tunnels[i] = st
		}
		return tunnels, nil
	}
}

func (t NormalTunnel) Equal(v interface{}) bool {
	t2, ok := v.(NormalTunnel)
	if !ok {
		return false
	}

	return t.ID == t2.ID &&
		t.SSHUser == t2.SSHUser &&
		t.SSHHost == t2.SSHHost &&
		t.SSHPort == t2.SSHPort &&
		t.TunnelPort == t2.TunnelPort &&
		t.ServiceHost == t2.ServiceHost &&
		t.ServicePort == t2.ServicePort &&
		t.HTTPProxy == t2.HTTPProxy
}

// sqlFromNormalTunnel converts tunnel data into something that can be inserted into the DB
func sqlFromNormalTunnel(tunnel NormalTunnel) postgres.NormalTunnel {
	return postgres.NormalTunnel{
		SSHUser:     sql.NullString{String: tunnel.SSHUser, Valid: tunnel.SSHUser != ""},
		SSHHost:     tunnel.SSHHost,
		SSHPort:     tunnel.SSHPort,
		ServiceHost: tunnel.ServiceHost,
		ServicePort: tunnel.ServicePort,
		HTTPProxy:   tunnel.HTTPProxy,
	}
}

// convert a SQL DB representation of a postgres.NormalTunnel into the primary NormalTunnel struct
func normalTunnelFromSQL(record postgres.NormalTunnel) NormalTunnel {
	return NormalTunnel{
		ID:          record.ID,
		CreatedAt:   record.CreatedAt,
		Enabled:     record.Enabled,
		TunnelPort:  record.TunnelPort,
		SSHUser:     record.SSHUser.String,
		SSHHost:     record.SSHHost,
		SSHPort:     record.SSHPort,
		ServiceHost: record.ServiceHost,
		ServicePort: record.ServicePort,
		HTTPProxy:   record.HTTPProxy,
	}
}

func (t NormalTunnel) GetID() uuid.UUID {
	return t.ID
}
