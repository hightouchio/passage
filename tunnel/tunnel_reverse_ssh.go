package tunnel

import (
	"context"
	"github.com/gliderlabs/ssh"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/stats"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	gossh "golang.org/x/crypto/ssh"
	"sync"
)

type SSHServer struct {
	BindAddr string
	HostKey  []byte

	server  *ssh.Server
	tunnels map[uuid.UUID]registeredTunnel
	close   chan bool

	sync.RWMutex
}

type registeredTunnel struct {
	id             uuid.UUID
	port           int
	authorizedKeys []ssh.PublicKey
	lifecycle      Lifecycle
	stats          stats.Stats
}

func NewSSHServer(addr string, hostKey []byte) *SSHServer {
	return &SSHServer{
		BindAddr: addr,
		HostKey:  hostKey,
		tunnels:  make(map[uuid.UUID]registeredTunnel),
		close:    make(chan bool),
	}
}

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
		entry := logrus.WithField("session_id", ctx.SessionID())

		success, authorizedTunnels := func() (bool, []registeredTunnel) {
			// Identify the set of tunnels that match the incoming public key
			authorizedTunnels := s.getAuthorizedTunnels(incomingKey)

			// Reject the SSH session if there are no authorized tunnels
			if len(authorizedTunnels) == 0 {
				entry.Debug("no authorized tunnels for public key")
				return false, []registeredTunnel{}
			}

			return true, authorizedTunnels
		}()

		// If there are any authorized tunnels, we can allow this session past the authentication step
		// TODO: Upsert authorized tunnels for the session
		ctx.SetValue("authorized_tunnels", authorizedTunnels)

		entry.WithFields(logrus.Fields{
			"remote_addr":        ctx.RemoteAddr().String(),
			"user":               ctx.User(),
			"key_type":           incomingKey.Type(),
			"fingerprint":        gossh.FingerprintSHA256(incomingKey),
			"success":            success,
			"authorized_tunnels": len(authorizedTunnels),
		}).Info("public key auth request")

		return success
	})); err != nil {
		return err
	}

	// Validate incoming port forward requests against the set of authorized tunnels for this session
	server.ReversePortForwardingCallback = func(ctx ssh.Context, bindHost string, bindPort uint32) bool {
		authorizedTunnels := ctx.Value("authorized_tunnels")

		entry := logrus.WithField("session_id", ctx.SessionID())
		success, tunnelId := func() (bool, string) {
			// If the authorized tunnels are nil or cannot be cast, reject the request
			if authorizedTunnels == nil {
				entry.Debug("no authorized tunnels for session")
				return false, ""
			}
			tunnels, ok := authorizedTunnels.([]registeredTunnel)
			if !ok {
				entry.Debug("no authorized tunnels for session")
				return false, ""
			}

			// If there are no valid tunnels, reject the forwarding request
			if len(tunnels) == 0 {
				entry.Debug("No authorized tunnels for session")
				return false, ""
			}

			// Check the requested bind port against the set of authorized tunnels
			var tunnelId uuid.UUID
			var success bool
			for _, tunnel := range tunnels {
				// If the requested port matches a valid tunnel port, we're good to go
				if int(bindPort) == tunnel.port {
					success = true
					tunnelId = tunnel.id
					break
				}
			}
			return success, tunnelId.String()
		}()

		logrus.WithFields(logrus.Fields{
			"remote_addr":  ctx.RemoteAddr().String(),
			"tunnel_id":    tunnelId,
			"bind_address": bindPort,
			"bind_port":    bindPort,
			"success":      success,
		}).Info("reverse port forwarding request")
		return success
	}

	// Handle reverse port forwarding requests
	handler := &ReverseForwardingHandler{
		GetTunnel: s.getTunnelFromBindPort,
	}
	server.RequestHandlers = map[string]ssh.RequestHandler{
		"tcpip-forward":        handler.HandleSSHRequest,
		"cancel-tcpip-forward": handler.HandleSSHRequest,
	}

	logrus.WithField("addr", s.BindAddr).Infof("Reverse tunnel sshd server listening on %s", s.BindAddr)
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

// GetTunnelFromBindPort resolves a given bind port to the registered tunnel that is bound to it.
func (s *SSHServer) getTunnelFromBindPort(bindPort int) (registeredTunnel, bool) {
	s.RLock()
	defer s.RUnlock()

	for _, tunnel := range s.tunnels {
		if tunnel.port == bindPort {
			return tunnel, true
		}
	}
	return registeredTunnel{}, false
}

// RegisterTunnel registers a Reverse Tunnel with this SSH Server.
func (s *SSHServer) RegisterTunnel(tunnelId uuid.UUID, bindPort int, authorizedKeys []ssh.PublicKey, lifecycle Lifecycle, st stats.Stats) {
	s.Lock()
	defer s.Unlock()

	logrus.WithFields(logrus.Fields{
		"tunnel_id": tunnelId,
		"bind_port": bindPort,
	}).Info("Registering tunnel")
	s.tunnels[tunnelId] = registeredTunnel{
		id:             tunnelId,
		port:           bindPort,
		authorizedKeys: authorizedKeys,
		lifecycle:      lifecycle,
		stats:          st,
	}
}

func (s *SSHServer) DeregisterTunnel(tunnelId uuid.UUID) {
	s.Lock()
	defer s.Unlock()

	logrus.WithFields(logrus.Fields{
		"tunnel_id": tunnelId,
	}).Info("Deregistering tunnel")

	delete(s.tunnels, tunnelId)
}

// getAuthorizedTunnels matches an incoming ssh.PublicKey against tunnels registered with this SSH server.
// This serves to determine the set of authorized bind ports that a given SSH connection can forward to.
func (s *SSHServer) getAuthorizedTunnels(incomingKey ssh.PublicKey) []registeredTunnel {
	s.RLock()
	defer s.RUnlock()

	var authorizedTunnels []registeredTunnel
	for _, tunnel := range s.tunnels {
		for _, authorizedKey := range tunnel.authorizedKeys {
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
