package tunnel

import (
	"context"
	"github.com/gliderlabs/ssh"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	gossh "golang.org/x/crypto/ssh"
	"net"
	"sync"
	"time"
)

type SSHServer struct {
	BindAddr string
	HostKey  []byte

	server               *ssh.Server
	registeredForwarders map[uuid.UUID]registeredForwarder

	sync.RWMutex
}

type registeredForwarder struct {
	id             uuid.UUID
	port           int
	authorizedKeys []ssh.PublicKey
}

func NewSSHServer(addr string, hostKey []byte) *SSHServer {
	return &SSHServer{
		BindAddr:             addr,
		HostKey:              hostKey,
		registeredForwarders: make(map[uuid.UUID]registeredForwarder),
	}
}

func (s *SSHServer) Start(ctx context.Context) error {
	lifecycle := getCtxLifecycle(ctx)

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
	server.Handler = func(s ssh.Session) {
		select {
		case <-s.Context().Done(): // Block until session ends
		case <-ctx.Done(): // or until server closes
		}
	}

	// Validate incoming public keys, match them against registered forwarders, and store the list of authorized
	// 	forwarders in the session context for future reference when evaluating reverse port forwarding requests.
	if err := server.SetOption(ssh.PublicKeyAuth(func(ctx ssh.Context, incomingKey ssh.PublicKey) bool {
		entry := logrus.WithField("session_id", ctx.SessionID())

		success, authorizedForwarders := func() (bool, []registeredForwarder) {
			// Identify the set of forwarders that match the incoming public key
			authorizedForwarders := s.getAuthorizedForwarders(incomingKey)

			// Reject the SSH session if there are no authorized forwarders
			if len(authorizedForwarders) == 0 {
				entry.Debug("no authorized forwarders for public key")
				return false, []registeredForwarder{}
			}

			return true, authorizedForwarders
		}()

		// If there are any authorized forwarders, we can allow this session past the authentication step
		// TODO: Upsert authorized forwarders for the session
		ctx.SetValue("authorized_forwarders", authorizedForwarders)

		entry.WithFields(logrus.Fields{
			"remote_addr":           ctx.RemoteAddr().String(),
			"user":                  ctx.User(),
			"key_type":              incomingKey.Type(),
			"fingerprint":           gossh.FingerprintSHA256(incomingKey),
			"success":               success,
			"authorized_forwarders": len(authorizedForwarders),
		}).Info("public key auth request")

		return success
	})); err != nil {
		return err
	}

	// Validate incoming port forward requests against the set of authorized forwarders for this session
	server.ReversePortForwardingCallback = func(ctx ssh.Context, bindHost string, bindPort uint32) bool {
		authorizedForwarders := ctx.Value("authorized_forwarders")

		entry := logrus.WithField("session_id", ctx.SessionID())
		success, forwarderId := func() (bool, string) {
			// If the authorized forwarders are nil or cannot be cast, reject the request
			if authorizedForwarders == nil {
				entry.Debug("no authorized forwarders for session")
				return false, ""
			}
			forwarders, ok := authorizedForwarders.([]registeredForwarder)
			if !ok {
				entry.Debug("no authorized forwarders for session")
				return false, ""
			}

			// If there are no valid forwarders, reject the forwarding request
			if len(forwarders) == 0 {
				entry.Debug("No authorized forwarders for session")
				return false, ""
			}

			// Check the requested bind port against the set of authorized forwarders
			var forwarderId uuid.UUID
			var success bool
			for _, forwarder := range forwarders {
				// If the requested port matches a valid forwarder port, we're good to go
				if int(bindPort) == forwarder.port {
					success = true
					forwarderId = forwarder.id
					break
				}
			}
			return success, forwarderId.String()
		}()

		logrus.WithFields(logrus.Fields{
			"remote_addr":  ctx.RemoteAddr().String(),
			"forwarder_id": forwarderId,
			"bind_address": bindPort,
			"bind_port":    bindPort,
			"success":      success,
		}).Info("reverse port forwarding request")
		return success
	}

	// Handle reverse port forwarding requests
	handler := &ReverseForwardingHandler{
		lifecycle: lifecycle,
	}
	server.RequestHandlers = map[string]ssh.RequestHandler{
		"tcpip-forward":        handler.HandleSSHRequest,
		"cancel-tcpip-forward": handler.HandleSSHRequest,
	}

	logrus.WithField("addr", s.BindAddr).Infof("Reverse tunnel sshd server listening on %s", s.BindAddr)
	s.server = server
	if err := s.server.ListenAndServe(); err != nil {
		return errors.Wrap(err, "listen and serve")
	}

	return nil
}

func (s *SSHServer) Close() error {
	if s.server == nil {
		return nil
	}
	return s.server.Close()
}

func (s *SSHServer) RegisterForwarder(forwarderId uuid.UUID, forwardPort int, authorizedKeys []ssh.PublicKey) {
	s.Lock()
	defer s.Unlock()

	logrus.WithFields(logrus.Fields{
		"forwarder_id":   forwarderId,
		"forwarder_port": forwardPort,
	}).Info("Registering forwarder")
	s.registeredForwarders[forwarderId] = registeredForwarder{
		id:             forwarderId,
		port:           forwardPort,
		authorizedKeys: authorizedKeys,
	}
}

func (s *SSHServer) DeregisterForwarder(forwarderId uuid.UUID) {
	s.Lock()
	defer s.Unlock()

	logrus.WithFields(logrus.Fields{
		"forwarder_id": forwarderId,
	}).Info("Deregistering forwarder")

	delete(s.registeredForwarders, forwarderId)
}

func (s *SSHServer) getAuthorizedForwarders(incomingKey ssh.PublicKey) []registeredForwarder {
	s.RLock()
	defer s.RUnlock()

	var authorizedForwarders []registeredForwarder
	for _, forwarder := range s.registeredForwarders {
		for _, authorizedKey := range forwarder.authorizedKeys {
			if ssh.KeysEqual(authorizedKey, incomingKey) {
				authorizedForwarders = append(authorizedForwarders, forwarder)
			}
		}
	}

	return authorizedForwarders
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

type SSHClientOptions struct {
	User              string
	DialTimeout       time.Duration
	KeepaliveInterval time.Duration
}

// sshKeepaliver regularly sends a keepalive request and returns an error if there is a failure
func sshKeepaliver(ctx context.Context, conn net.Conn, client *gossh.Client, interval, timeout time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			// Only break out of the keepaliver if we get an error
			if err := sshKeepalivePing(ctx, conn, client, timeout); err != nil {
				return err
			}

			// Reset deadline to the predicted next tick, plus a one-second grace period.
			if err := conn.SetDeadline(time.Now().Add(interval + (1 * time.Second))); err != nil {
				return errors.Wrap(err, "reset deadline")
			}
		}
	}
}

// sshKeepalivePing sends a keepalive message and waits for a response, using the gossh client libraries
func sshKeepalivePing(ctx context.Context, conn net.Conn, client *gossh.Client, timeout time.Duration) error {
	// Set deadline for request.
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return errors.Wrap(err, "set conn deadline")
	}

	result := make(chan error)
	go func() {
		// Keepalive over the SSH connection
		_, _, err := client.SendRequest("keepalive@passage", true, nil)
		result <- err
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-result:
		return err
	}
}
