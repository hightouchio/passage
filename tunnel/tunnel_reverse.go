package tunnel

import (
	"context"
	"github.com/gliderlabs/ssh"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/stats"
	"github.com/hightouchio/passage/tunnel/discovery"
	"github.com/hightouchio/passage/tunnel/keystore"
	"go.uber.org/zap"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/pkg/errors"
)

type ReverseTunnel struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Enabled   bool      `json:"enabled"`

	SSHDPort           int  `json:"sshdPort"`
	TunnelPort         int  `json:"tunnelPort"`
	HealthcheckEnabled bool `json:"healthcheck_enabled"`

	authorizedKeysHash string
	services           ReverseTunnelServices
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
	t.services.SSHServer.RegisterTunnel(SSHServerRegisteredTunnel{
		ID:             t.ID,
		AuthorizedKeys: authorizedKeys,

		// This is not actually the port that the tunnel is listening on,
		//	but the port that the tunnel is *registered* on, which is how we uniquely identify incoming requests
		//	for tunnels.
		RegisteredPort: t.TunnelPort,

		// Pass the receiver channel, so we can receive SSH connections from the SSHD server
		Connections: connectionChan,
	})
	defer t.services.SSHServer.DeregisterTunnel(t.ID)

	// Keep track of the number of active connections on this tunnel
	var connectionCount atomic.Int32

	// Regularly report status based on the number of active connections
	go intervalStatusReporter(ctx, statusUpdate, func() StatusUpdate {
		if connectionCount.Load() > 0 {
			return StatusUpdate{StatusReady, "Tunnel is online"}
		} else {
			return StatusUpdate{StatusBooting, "Tunnel is waiting for connections"}
		}
	})

	// Regularly report server-wide stats
	go intervalMetricReporter(ctx, func() {
		// Report the current client connection count
		stats.GetStats(ctx).Gauge(StatTunnelReverseForwardClientConnectionCount, float64(connectionCount.Load()), stats.Tags{}, 1)
	})

	// Handle incoming SSH port forwarding connections
	logger.Info("Tunnel registered with SSH server. Waiting for connections")
	connWg := sync.WaitGroup{}
	func() {
		for {
			select {
			case <-ctx.Done():
				return

				// Wait for incoming connections
			case conn, ok := <-connectionChan:
				if !ok {
					return
				}

				connectionCount.Add(1)
				connWg.Add(1)
				go func() {
					defer connWg.Done()
					defer connectionCount.Add(-1)

					t.handleConnection(ctx, conn, logger, listener)
				}()
			}
		}
	}()

	// Wait for all connections to close
	connWg.Wait()
	return nil
}

// handleConnection handles an SSH protocol connection and sets up port forwarding
func (t ReverseTunnel) handleConnection(
	ctx context.Context,
	conn ReverseForwardingConnection,
	log *log.Logger,
	listener *net.TCPListener,
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

	if t.HealthcheckEnabled {
		logger.Debug("Starting upstream healthcheck")
		// Start upstream reachability test
		// TODO: If we have multiple forwarders, the healthchecks can conflict.
		//		We should probably have a single healthcheck for the tunnel
		go upstreamHealthcheck(ctx, t, logger, t.services.Discovery, conn.Dial)
	}

	// Create a TCPForwarder, which will bidirectionally proxy connections and traffic between a local
	//	tunnel listener and a remote SSH connection.
	forwarder := &TCPForwarder{
		Listener:          listener,
		GetUpstreamConn:   conn.Dial,
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
	SSHServer *SSHServer
	Keystore  keystore.Keystore
	Discovery discovery.Service
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

	return t.ID == t2.ID &&
		t.SSHDPort == t2.SSHDPort &&
		t.TunnelPort == t2.TunnelPort &&
		t.authorizedKeysHash == t2.authorizedKeysHash &&
		t.HealthcheckEnabled == t2.HealthcheckEnabled
}

// convert a SQL DB representation of a postgres.ReverseTunnel into the primary ReverseTunnel struct
func reverseTunnelFromSQL(record postgres.ReverseTunnel) ReverseTunnel {
	return ReverseTunnel{
		ID:                 record.ID,
		CreatedAt:          record.CreatedAt,
		Enabled:            record.Enabled,
		TunnelPort:         record.TunnelPort,
		SSHDPort:           record.SSHDPort,
		HealthcheckEnabled: record.HealthcheckEnabled,
		authorizedKeysHash: record.AuthorizedKeysHash,
	}
}

func (t ReverseTunnel) GetID() uuid.UUID {
	return t.ID
}
