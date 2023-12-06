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

func (t NormalTunnel) Start(ctx context.Context, listener *net.TCPListener, statusUpdate chan<- StatusUpdate) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger := log.FromContext(ctx)

	// Establish a connection to the remote SSH server
	statusUpdate <- StatusUpdate{StatusBooting, "Initialized"}
	sshClient, keepalive, err := NewSSHClient(ctx, SSHClientOptions{
		Host: t.SSHHost,
		Port: t.SSHPort,

		// Select the SSH user to use for the client connection
		//	If the tunnel has explicitly set a user, use that.
		//	If not, fall back to the default.
		User: firstNotEmptyString(t.SSHUser, t.clientOptions.User),

		// Select the SSH auth methods to use for the client connection
		GetKeySigners: t.getAuthSigners,

		// Pass these options in from the global config
		DialTimeout:       t.clientOptions.DialTimeout,
		KeepaliveInterval: t.clientOptions.KeepaliveInterval,
	})
	if err != nil {
		return errors.Wrap(err, "SSH connect")
	}
	statusUpdate <- StatusUpdate{StatusBooting, "SSH connection established"}

	// Listen for keepalive failures
	go func() {
		defer cancel()

		select {
		case err, ok := <-keepalive:
			// If the channel closed, just ignore it
			if !ok {
				return
			}
			statusUpdate <- StatusUpdate{StatusError, fmt.Sprintf("SSH keepalive failed: %s", err.Error())}

		case <-ctx.Done():
			return
		}
	}()

	// Function which gets a connection to the upstream server
	upstreamAddr := net.JoinHostPort(t.ServiceHost, strconv.Itoa(t.ServicePort))
	getUpstreamConn := func() (net.Conn, error) {
		return sshClient.Dial("tcp", upstreamAddr)
	}

	// Initially test the upstream connection to make sure it works
	// 	Retry it until it succeeds.
	// TODO: Support a maximum number of retries.
	if err := retry(ctx, 5*time.Second, func() error {
		logger.With(zap.String("addr", upstreamAddr)).Debugf("Testing upstream connection %s", upstreamAddr)
		if _, err := getUpstreamConn(); err != nil {
			statusUpdate <- StatusUpdate{StatusError, err.Error()}
			logger.Errorw("Upstream dial failed", zap.Error(err))
			return err
		}

		logger.Debug("Upstream connection successful")
		return nil
	}); err != nil {
		return errors.Wrap(err, "upstream dial")
	}

	// Create a TCPForwarder, which will bidirectionally proxy connections and traffic between a local
	//	tunnel listener and a remote SSH connection.
	forwarder := &TCPForwarder{
		Listener: listener,

		// Implement GetUpstreamConn by initiating upstream connections through the SSH client.
		GetUpstreamConn: func(conn net.Conn) (io.ReadWriteCloser, error) {
			serviceConn, err := getUpstreamConn()
			if err != nil {
				// Any upstream dial errors should be reported as part of the tunnel status
				statusUpdate <- StatusUpdate{StatusError, err.Error()}

				return nil, err
			}
			return serviceConn, err
		},

		KeepaliveInterval: 5 * time.Second,
		Stats:             stats.GetStats(ctx),
		logger:            logger.Named("Forwarder"),
	}
	defer forwarder.Close()

	// Start port forwarding
	logger.Info("Starting forwarder")
	go func() {
		defer cancel()
		if err := forwarder.Serve(); err != nil {
			// If it's simply a closed error, we can return without logging an error.
			if !errors.Is(err, net.ErrClosed) {
				logger.Errorw("Forwarder serve", zap.Error(err))
			}
		}
	}()
	statusUpdate <- StatusUpdate{StatusReady, "Tunnel is online"}

	// Update the tunnel self-reported status every 30 seconds
	statusTicker := time.NewTicker(30 * time.Second)
	defer statusTicker.Stop()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			case <-statusTicker.C:
				statusUpdate <- StatusUpdate{StatusReady, "Tunnel is online"}
			}
		}
	}()

	<-ctx.Done()
	return nil
}

// firstNotEmptyString returns the first string that is not empty
func firstNotEmptyString(options ...string) string {
	if len(options) == 0 {
		return ""
	}

	for _, str := range options {
		if str != "" {
			return str
		}
	}

	return ""
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
