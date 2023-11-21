package tunnel

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/stats"
	"github.com/hightouchio/passage/tunnel/discovery"
	"github.com/hightouchio/passage/tunnel/keystore"
	"go.uber.org/zap"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

type NormalTunnel struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Enabled   bool      `json:"enabled"`

	SSHUser     string `json:"sshUser"`
	SSHHost     string `json:"sshHost"`
	SSHPort     int    `json:"sshPort"`
	ServiceHost string `json:"serviceHost"`
	ServicePort int    `json:"servicePort"`

	clientOptions SSHClientOptions
	services      NormalTunnelServices
}

func (t NormalTunnel) Start(ctx context.Context, options TunnelOptions, statusUpdate StatusUpdateFn) error {
	logger := log.FromContext(ctx)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start listening on a local port.
	tunnelListener, err := newEphemeralTCPListener()
	if err != nil {
		return bootError{event: "open_listener", err: err}
	}
	defer tunnelListener.Close()
	logger.Infow("Open tunnel listener", "listen_addr", tunnelListener.Addr().String())
	listenerPort := portFromNetAddr(tunnelListener.Addr())

	// Register tunnel with service discovery.
	if err := t.services.Discovery.RegisterTunnel(t.ID, listenerPort); err != nil {
		return bootError{event: "service_discovery_register", err: err}
	}
	defer func() {
		if err := t.services.Discovery.DeregisterTunnel(t.ID); err != nil {
			logger.Errorw("Failed to deregister tunnel from service discovery", zap.Error(err))
		}
	}()

	// Update service discovery that SSH connection established, but not quite online
	statusUpdate(StatusBooting, "Booting")

	// Get a list of key signers to use for authentication
	keySigners, err := t.getAuthSigners(ctx)
	if err != nil {
		statusUpdate(discovery.TunnelUnhealthy, fmt.Sprintf("Failed to generate authentication payload: %s", err.Error()))
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
	addr := net.JoinHostPort(t.SSHHost, strconv.Itoa(t.SSHPort))
	logger.Infow("SSH remote dial", zap.String("addr", addr))
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)

	if err != nil {
		statusUpdate(discovery.TunnelUnhealthy, fmt.Sprintf("Failed to resolve remote address: %s", err.Error()))
		return bootError{event: "remote_dial", err: errors.Wrapf(err, "resolve addr %s", addr)}
	}
	sshConn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		statusUpdate(discovery.TunnelUnhealthy, fmt.Sprintf("Failed to connect to remote server: %s", err.Error()))
		return bootError{event: "remote_dial", err: err}
	}
	defer sshConn.Close()

	// Configure TCP keepalive for SSH connection
	sshConn.SetKeepAlive(true)
	sshConn.SetKeepAlivePeriod(t.clientOptions.KeepaliveInterval)

	// Init SSH connection protocol
	logger.With(
		zap.String("ssh_user", sshUser),
		zap.Int("ssh_auth_method_count", len(keySigners)),
	).Infow("SSH open client connection")
	c, chans, reqs, err := ssh.NewClientConn(
		sshConn, addr,
		&ssh.ClientConfig{
			User:            sshUser,
			Auth:            []ssh.AuthMethod{ssh.PublicKeys(keySigners...)},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		},
	)
	if err != nil {
		statusUpdate(discovery.TunnelUnhealthy, err.Error())
		return bootError{event: "ssh_connect", err: err}
	}
	sshClient := ssh.NewClient(c, chans, reqs)
	logger.Info("SSH connection established")
	statusUpdate(discovery.TunnelWarning, "SSH connection established")

	// Start sending keepalive packets to the upstream SSH server
	go func() {
		if err := sshKeepaliver(ctx, sshConn, sshClient, t.clientOptions.KeepaliveInterval, t.clientOptions.DialTimeout); err != nil {
			logger.Errorw("SSH keepalive failed", zap.Error(err))
			statusUpdate(discovery.TunnelUnhealthy, fmt.Sprintf("SSH keepalive failed: %s", err.Error()))
			cancel()
		}
	}()

	// Create a TCPForwarder, which will bidirectionally proxy connections and traffic between a local
	//	tunnel listener and a remote SSH connection.
	forwarder := &TCPForwarder{
		Listener: tunnelListener,

		// Implement GetUpstreamConn by initiating upstream connections through the SSH client.
		GetUpstreamConn: func(conn net.Conn) (io.ReadWriteCloser, error) {
			serviceConn, err := sshClient.Dial("tcp", net.JoinHostPort(t.ServiceHost, strconv.Itoa(t.ServicePort)))
			if err != nil {
				return nil, err
			}
			return serviceConn, err
		},

		KeepaliveInterval: 5 * time.Second,
		Stats:             stats.GetStats(ctx),
		logger:            logger.Named("Forwarder"),
	}
	defer forwarder.Close()

	logger.Info("Tunnel is ready")

	// Start port forwarding
	go func() {
		defer cancel()
		if err := forwarder.Serve(); err != nil {
			logger.Errorw("Forwarder serve", zap.Error(err))
			return
		}
	}()

	// Start the tunnel connectivity check
	go runTunnelConnectivityCheck(ctx, t.ID, logger, t.services.Discovery)

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
		t.ServiceHost == t2.ServiceHost &&
		t.ServicePort == t2.ServicePort
}

// sqlFromNormalTunnel converts tunnel data into something that can be inserted into the DB
func sqlFromNormalTunnel(tunnel NormalTunnel) postgres.NormalTunnel {
	return postgres.NormalTunnel{
		SSHUser:     sql.NullString{String: tunnel.SSHUser, Valid: tunnel.SSHUser != ""},
		SSHHost:     tunnel.SSHHost,
		SSHPort:     tunnel.SSHPort,
		ServiceHost: tunnel.ServiceHost,
		ServicePort: tunnel.ServicePort,
	}
}

// convert a SQL DB representation of a postgres.NormalTunnel into the primary NormalTunnel struct
func normalTunnelFromSQL(record postgres.NormalTunnel) NormalTunnel {
	return NormalTunnel{
		ID:          record.ID,
		CreatedAt:   record.CreatedAt,
		Enabled:     record.Enabled,
		SSHUser:     record.SSHUser.String,
		SSHHost:     record.SSHHost,
		SSHPort:     record.SSHPort,
		ServiceHost: record.ServiceHost,
		ServicePort: record.ServicePort,
	}
}

func (t NormalTunnel) GetID() uuid.UUID {
	return t.ID
}
