package tunnel

import (
	"context"
	"github.com/gliderlabs/ssh"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/stats"
	"github.com/hightouchio/passage/tunnel/discovery"
	"github.com/hightouchio/passage/tunnel/keystore"
	"go.uber.org/zap"
	"io"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/pkg/errors"
)

type ReverseTunnel struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Enabled   bool      `json:"enabled"`

	SSHDPort           int `json:"sshdPort"`
	authorizedKeysHash string

	services ReverseTunnelServices
}

func (t ReverseTunnel) Start(ctx context.Context, listener *net.TCPListener, statusUpdate chan<- StatusUpdate) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger := log.FromContext(ctx)

	logger.Debug("Get authorized keys")
	authorizedKeys, err := t.getAuthorizedKeys(ctx)
	if err != nil {
		return errors.Wrap(err, "get authorized keys")
	}

	// Create a channel to receive incoming SSH connections
	//	for this tunnel
	connectionChan := make(chan ReverseForwardingConnection)
	defer close(connectionChan)

	logger.Debug("Register tunnel with SSHD")
	t.services.GlobalSSHServer.RegisterTunnel(SSHServerRegisteredTunnel{
		ID:             t.ID,
		AuthorizedKeys: authorizedKeys,

		// This is not actually the port that the tunnel is listening on,
		//	but the port that the tunnel is *registered* on, which is how we uniquely identify incoming requests
		//	for tunnels.
		RegisteredPort: t.SSHDPort,

		// Pass the receiver channel, so we can receive SSH connections from the SSHD server
		Connections: connectionChan,
	})
	defer t.services.GlobalSSHServer.DeregisterTunnel(t.ID)

	// Register this tunnel with the global reverse SSH server
	logger.Info("Tunnel registered with ssh server. Waiting for connections")
	statusUpdate <- StatusUpdate{StatusBooting, "ssh server is online. Waiting for connections"}

	// Handle incoming SSH port forwarding connections
	for {
		select {
		case <-ctx.Done():
			return nil

			// Wait for incoming connections
		case conn, ok := <-connectionChan:
			if !ok {
				return nil
			}
			go handleConnection(ctx, conn, logger, listener, statusUpdate)
		}
	}
}

// handleConnection handles an SSH protocol connection and sets up port forwarding
func handleConnection(
	ctx context.Context,
	conn ReverseForwardingConnection,
	log *log.Logger,
	listener *net.TCPListener,
	statusUpdate chan<- StatusUpdate,
) {
	// Create a new context to handle this connection's lifecycle
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger := sshSessionLogger(log, conn)
	logger.Info("Start tunnel port forwarding")
	defer logger.Info("Stop tunnel port forwarding")

	// Wait for either server or client connection to close, and stop forwarding
	go func() {
		defer cancel()
		select {
		case <-ctx.Done(): // Wait for server connection to close
		case <-conn.Done(): // Wait for client connection to close
		}
	}()

	// Create a TCPForwarder, which will bidirectionally proxy connections and traffic between a local
	//	tunnel listener and a remote SSH connection.
	forwarder := &TCPForwarder{
		Listener: listener,

		// Implement GetUpstreamConn by opening a channel on the SSH connection.
		GetUpstreamConn: func(tConn net.Conn) (io.ReadWriteCloser, error) {
			conn, err := conn.Dial(tConn.RemoteAddr().String())
			if err != nil {
				// Any upstream dial errors should be reported as part of the tunnel status
				statusUpdate <- StatusUpdate{StatusError, err.Error()}
				return nil, err
			}

			return conn, nil
		},

		KeepaliveInterval: 5 * time.Second,
		Stats:             stats.GetStats(ctx),
		logger:            logger.Named("Forwarder"),
	}
	defer forwarder.Close()

	// Start port forwarding
	go func() {
		defer cancel()
		if err := forwarder.Serve(); err != nil {
			// If it's simply a closed error, we can return without logging an error.
			if !errors.Is(err, net.ErrClosed) {
				logger.Errorw("Forwarder serve", zap.Error(err))
			}
		}
	}()

	// Continually report tunnel status until the tunnel shuts down
	go tunnelStatusReporter(ctx, statusUpdate)

	// Wait for connection to end
	<-conn.Done()
}

func (t ReverseTunnel) getAuthorizedKeys(ctx context.Context) ([]ssh.PublicKey, error) {
	registeredKeys, err := t.services.SQL.GetReverseTunnelAuthorizedKeys(ctx, t.ID)
	if err != nil {
		return []ssh.PublicKey{}, errors.Wrap(err, "could not read keys from database")
	}

	authorizedKeys := make([]ssh.PublicKey, len(registeredKeys))
	for i, registeredKey := range registeredKeys {
		keyBytes, err := t.services.Keystore.Get(ctx, registeredKey.ID)
		if err != nil {
			return authorizedKeys, errors.Wrapf(err, "could not read key %s from keystore", registeredKey.ID.String())
		}

		key, _, _, _, err := ssh.ParseAuthorizedKey(keyBytes)
		if err != nil {
			return authorizedKeys, errors.Wrapf(err, "could not parse key %s", registeredKey.ID.String())
		}

		authorizedKeys[i] = key
	}

	return authorizedKeys, nil
}

// ReverseTunnelServices are the external dependencies that ReverseTunnel needs to do its job
type ReverseTunnelServices struct {
	SQL interface {
		GetReverseTunnelAuthorizedKeys(ctx context.Context, tunnelID uuid.UUID) ([]postgres.Key, error)
	}
	GlobalSSHServer *SSHServer
	Keystore        keystore.Keystore
	Discovery       discovery.DiscoveryService

	EnableIndividualSSHD bool
	GetIndividualSSHD    func(sshdPort int) *SSHServer
}

func InjectReverseTunnelDependencies(f func(ctx context.Context) ([]ReverseTunnel, error), services ReverseTunnelServices) ListFunc {
	return func(ctx context.Context) ([]Tunnel, error) {
		sts, err := f(ctx)
		if err != nil {
			return []Tunnel{}, err
		}
		// Inject dependencies
		tunnels := make([]Tunnel, len(sts))
		for i, st := range sts {
			st.services = services
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

	return t.ID == t2.ID && t.SSHDPort == t2.SSHDPort && t.authorizedKeysHash == t2.authorizedKeysHash
}

// convert a SQL DB representation of a postgres.ReverseTunnel into the primary ReverseTunnel struct
func reverseTunnelFromSQL(record postgres.ReverseTunnel) ReverseTunnel {
	return ReverseTunnel{
		ID:                 record.ID,
		CreatedAt:          record.CreatedAt,
		Enabled:            record.Enabled,
		SSHDPort:           record.SSHDPort,
		authorizedKeysHash: record.AuthorizedKeysHash,
	}
}

func (t ReverseTunnel) GetID() uuid.UUID {
	return t.ID
}
