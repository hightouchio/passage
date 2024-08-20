package tunnel

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/hightouchio/passage/log"
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

	HealthcheckEnabled bool `json:"healthcheck_enabled"`

	// Deprecated
	TunnelPort int `json:"tunnelPort"`

	clientOptions SSHClientOptions
	services      NormalTunnelServices
}

func (t NormalTunnel) Start(ctx context.Context, listener *net.TCPListener, statusUpdate chan<- StatusUpdate) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	logger := log.FromContext(ctx)

	// Establish a connection to the remote SSH server
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

	// Shut down the tunnel if the SSH connection ends
	go func() {
		if err := sshClient.Wait(); err != nil {
			cancel(errors.Wrap(err, "SSH connection closed"))
		}
	}()

	// Listen for keepalive failures
	go func() {
		defer cancel(nil)

		select {
		case <-ctx.Done():
			return
		case err, ok := <-keepalive:
			// If the channel closed, just ignore it
			if !ok {
				return
			}
			statusUpdate <- StatusUpdate{StatusError, fmt.Sprintf("SSH keepalive failed: %s", err.Error())}
		}
	}()

	// Function which gets a connection to the upstream server
	getUpstreamConn := func() (io.ReadWriteCloser, error) {
		return sshClient.Dial("tcp", net.JoinHostPort(t.ServiceHost, strconv.Itoa(t.ServicePort)))
	}

	if t.HealthcheckEnabled {
		logger.Debug("Starting upstream healthcheck")
		// Start upstream reachability test
		go upstreamHealthcheck(ctx, t, logger, t.services.Discovery, getUpstreamConn)
	}

	// If the context has been cancelled at this point in time, stop the tunnel.
	if ctx.Err() != nil {
		return nil
	}

	// Create a TCPForwarder, which will bidirectionally proxy connections and traffic between a local
	//	tunnel listener and a remote SSH connection.
	forwarder := &TCPForwarder{
		Listener:          listener,
		GetUpstreamConn:   getUpstreamConn,
		KeepaliveInterval: 5 * time.Second,
		Stats:             stats.GetStats(ctx),
		logger:            logger.Named("Forwarder"),
	}
	defer forwarder.Close()

	// Start port forwarding
	logger.Debug("Starting forwarder")
	go func() {
		defer logger.Debug("Forwarder stopped")
		if err := forwarder.Serve(); err != nil {
			// If it's simply a closed error, we can return without logging an error.
			if !errors.Is(err, net.ErrClosed) {
				cancel(errors.Wrap(err, "forwarder serve"))
			}
		}
	}()

	// Continually report tunnel status until the tunnel shuts down
	go intervalStatusReporter(ctx, statusUpdate, func() StatusUpdate {
		// If we're at this point in the tunnel, we're online
		return StatusUpdate{Status: StatusReady, Message: "Tunnel is online"}
	})

	<-ctx.Done()

	// If the context was simply cancelled (with no error), return nil
	//	If the context was cancelled with a real error, return that
	if cause := context.Cause(ctx); !errors.Is(cause, context.Canceled) {
		return cause
	} else {
		return nil
	}
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
	}
	Keystore keystore.Keystore

	Discovery discovery.Service
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
		t.ServicePort == t2.ServicePort &&
		t.HealthcheckEnabled == t2.HealthcheckEnabled
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
		ID:                 record.ID,
		CreatedAt:          record.CreatedAt,
		Enabled:            record.Enabled,
		SSHUser:            record.SSHUser.String,
		SSHHost:            record.SSHHost,
		SSHPort:            record.SSHPort,
		ServiceHost:        record.ServiceHost,
		ServicePort:        record.ServicePort,
		HealthcheckEnabled: record.HealthcheckEnabled,
		TunnelPort:         record.TunnelPort,
	}
}

func (t NormalTunnel) GetID() uuid.UUID {
	return t.ID
}
