package tunnel

import (
	"context"
	"github.com/gliderlabs/ssh"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/stats"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	gossh "golang.org/x/crypto/ssh"
	"net"
	"sync"
)

type SSHServer struct {
	BindAddr string
	HostKey  []byte

	server  *ssh.Server
	tunnels map[uuid.UUID]SSHServerRegisteredTunnel
	close   chan bool
	logger  *log.Logger
	stats   stats.Stats

	sync.RWMutex
}

type SSHServerRegisteredTunnel struct {
	ID             uuid.UUID
	RegisteredPort int
	AuthorizedKeys []ssh.PublicKey
	Listener       *net.TCPListener

	StatusUpdate chan<- StatusUpdate
	Logger       *log.Logger
	Stats        stats.Stats
}

func NewSSHServer(addr string, hostKey []byte, logger *log.Logger, st stats.Stats) *SSHServer {
	return &SSHServer{
		BindAddr: addr,
		HostKey:  hostKey,

		logger:  logger,
		stats:   st,
		tunnels: make(map[uuid.UUID]SSHServerRegisteredTunnel),
		close:   make(chan bool),
	}
}

var ErrSshServerClosed = ssh.ErrServerClosed

func (s *SSHServer) Start(ctx context.Context) error {
	server := &ssh.Server{
		Addr: s.BindAddr,
		ChannelHandlers: map[string]ssh.ChannelHandler{
			"session": ssh.DefaultSessionHandler,
		},
	}

	// Set sshd host key
	hostSigners, err := s.getHostSigners()
	if err != nil {
		return errors.Wrap(err, "get host signers")
	}
	server.HostSigners = hostSigners

	// SSH session handler. Hold connections open until cancelled.
	server.Handler = func(session ssh.Session) {
		select {
		case <-session.Context().Done(): // Block until session ends
		case <-s.close: // or until server closes
		case <-ctx.Done(): // or until start context is cancelled
		}
	}

	// Validate incoming public keys, match them against registered tunnels, and store the list of authorized
	// 	tunnels in the session context for future reference when evaluating reverse port forwarding requests.
	if err := server.SetOption(ssh.PublicKeyAuth(func(ctx ssh.Context, incomingKey ssh.PublicKey) bool {
		logger := s.logger.With(zap.String("session_id", ctx.SessionID()))

		success, authorizedTunnels := func() (bool, []SSHServerRegisteredTunnel) {
			// Identify the set of tunnels that match the incoming public key
			authorizedTunnels := s.getAuthorizedTunnels(incomingKey)

			// Reject the SSH session if there are no authorized tunnels
			if len(authorizedTunnels) == 0 {
				logger.Debug("no authorized tunnels for public key")
				return false, []SSHServerRegisteredTunnel{}
			}

			return true, authorizedTunnels
		}()

		logger.With(
			zap.String("remote_addr", ctx.RemoteAddr().String()),
			zap.String("user", ctx.User()),
			zap.String("key_type", incomingKey.Type()),
			zap.String("fingerprint", gossh.FingerprintSHA256(incomingKey)),
			zap.Bool("success", success),
			zap.Int("authorized_tunnels", len(authorizedTunnels)),
		).Info("Handle authentication attempt")
		s.stats.Incr(StatSshdConnectionsRequests, stats.Tags{"success": success}, 1)

		// Register the authorized tunnels onto the ssh.Context
		registerAuthorizedTunnels(ctx, authorizedTunnels)

		return success
	})); err != nil {
		return err
	}

	// Validate incoming port forward requests against the set of authorized tunnels for this session
	server.ReversePortForwardingCallback = func(ctx ssh.Context, bindHost string, bindPort uint32) bool {
		logger := s.logger.With(zap.String("session_id", ctx.SessionID()))

		success, tunnelId := func() (bool, string) {
			tunnels := getAuthorizedTunnels(ctx)

			// If there are no valid tunnels, reject the forwarding request
			if len(tunnels) == 0 {
				logger.Debug("No authorized tunnels for session")
				return false, ""
			}

			// Check the requested bind port against the set of authorized tunnels
			var tunnelId uuid.UUID
			var success bool
			for _, tunnel := range tunnels {
				// If the requested port matches a valid tunnel port, we're good to go
				if int(bindPort) == tunnel.RegisteredPort {
					success = true
					tunnelId = tunnel.ID
					break
				}
			}
			return success, tunnelId.String()
		}()

		logger.With(
			zap.String("remote_addr", ctx.RemoteAddr().String()),
			zap.String("tunnel_id", tunnelId),
			zap.String("bind_address", bindHost),
			zap.Uint32("bind_port", bindPort),
			zap.Bool("success", success),
		).Info("Reverse port forwarding request")
		return success
	}

	// Handle reverse port forwarding requests
	handler := &ReverseForwardingHandler{
		GetTunnel: s.getTunnelFromRegisteredPort,
	}
	server.RequestHandlers = map[string]ssh.RequestHandler{
		"tcpip-forward":        handler.HandleSSHRequest,
		"cancel-tcpip-forward": handler.HandleSSHRequest,
	}

	s.logger.With(zap.String("bind_addr", s.BindAddr)).Infof("Listening on %s", s.BindAddr)
	s.server = server
	if err := s.server.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

func (s *SSHServer) Close() error {
	close(s.close)
	if s.server == nil {
		return nil
	}
	return s.server.Close()
}

// getTunnelFromRegisteredPort resolves a given port to the registered tunnel associated with it
func (s *SSHServer) getTunnelFromRegisteredPort(port int) (SSHServerRegisteredTunnel, bool) {
	s.RLock()
	defer s.RUnlock()

	for _, tunnel := range s.tunnels {
		if tunnel.RegisteredPort == port {
			return tunnel, true
		}
	}
	return SSHServerRegisteredTunnel{}, false
}

// RegisterTunnel registers a Reverse Tunnel with this SSH Server.
func (s *SSHServer) RegisterTunnel(tunnel SSHServerRegisteredTunnel) {
	s.Lock()
	defer s.Unlock()

	s.logger.With(
		zap.String("tunnel_id", tunnel.ID.String()),
		zap.Int("registered_port", tunnel.RegisteredPort),
	).Debug("Registering tunnel")

	s.tunnels[tunnel.ID] = tunnel
}

// DeregisterTunnel removes the reverse tunnel from the SSH server
func (s *SSHServer) DeregisterTunnel(tunnelId uuid.UUID) {
	s.Lock()
	defer s.Unlock()

	s.logger.With(
		zap.String("tunnel_id", tunnelId.String()),
	).Debug("Deregistering tunnel")

	delete(s.tunnels, tunnelId)
}

// getAuthorizedTunnels matches an incoming ssh.PublicKey against tunnels registered with this SSH server.
// This serves to determine the set of authorized bind ports that a given SSH connection can forward to.
func (s *SSHServer) getAuthorizedTunnels(incomingKey ssh.PublicKey) []SSHServerRegisteredTunnel {
	s.RLock()
	defer s.RUnlock()

	var authorizedTunnels []SSHServerRegisteredTunnel
	for _, tunnel := range s.tunnels {
		for _, authorizedKey := range tunnel.AuthorizedKeys {
			if ssh.KeysEqual(authorizedKey, incomingKey) {
				authorizedTunnels = append(authorizedTunnels, tunnel)
			}
		}
	}

	return authorizedTunnels
}

func (s *SSHServer) getHostSigners() ([]ssh.Signer, error) {
	var hostSigners []ssh.Signer
	if len(s.HostKey) != 0 {
		signers, err := getSignersForPrivateKey(s.HostKey)
		if err != nil {
			return hostSigners, err
		}
		for _, signer := range signers {
			// Convert from `x/crypto/ssh` Signer to `gliderlabs/ssh` Signer
			hostSigners = append(hostSigners, signer)
		}
	}

	return hostSigners, nil
}

// registerAuthorizedTunnels adds new authorized tunnels to the ssh.Context
func registerAuthorizedTunnels(ctx ssh.Context, newAuthorizedTunnels []SSHServerRegisteredTunnel) {
	authorizedTunnels := getAuthorizedTunnels(ctx)
	authorizedTunnels = append(authorizedTunnels, newAuthorizedTunnels...)
	ctx.SetValue("authorized_tunnels", authorizedTunnels)
}

// getAuthorizedTunnels extracts the authorized tunnels from the ssh.Context
func getAuthorizedTunnels(ctx ssh.Context) []SSHServerRegisteredTunnel {
	ctxTunnels := ctx.Value("authorized_tunnels")
	if ctxTunnels == nil {
		return []SSHServerRegisteredTunnel{}
	}
	tunnels, ok := ctxTunnels.([]SSHServerRegisteredTunnel)
	if !ok {
		return []SSHServerRegisteredTunnel{}
	}
	return tunnels
}
