package tunnel

import (
	"context"
	"github.com/hightouchio/passage/log"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	gossh "golang.org/x/crypto/ssh"
	"net"
	"strconv"
	"time"
)

type SSHClientOptions struct {
	Host          string
	Port          int
	User          string
	GetKeySigners func(context.Context) ([]gossh.Signer, error)

	DialTimeout       time.Duration
	KeepaliveInterval time.Duration
}

func NewSSHClient(ctx context.Context, options SSHClientOptions) (*gossh.Client, <-chan error, error) {
	logger := log.FromContext(ctx).Named("SSH")
	logger.With(
		zap.String("user", options.User),
		zap.String("host", options.Host),
		zap.Int("port", options.Port),
	).Infof("Connect to ssh://%s@%s:%d", options.User, options.Host, options.Port)

	// Validate the address
	addr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(options.Host, strconv.Itoa(options.Port)))
	if err != nil {
		return nil, nil, errors.Wrap(err, "resolve address")
	}

	// Dial remote SSH server
	logger.With(zap.String("addr", addr.String())).Debugf("Dial %s", addr.String())
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to connect to remote server")
	}

	// Configure TCP keepalive for SSH connection
	logger.Debugw("Set TCP keepalive", zap.Duration("interval", options.KeepaliveInterval))
	if err := conn.SetKeepAlive(true); err != nil {
		return nil, nil, errors.Wrap(err, "failed to enable keepalive")
	}
	if err := conn.SetKeepAlivePeriod(options.KeepaliveInterval); err != nil {
		return nil, nil, errors.Wrap(err, "failed to set keepalive period")
	}

	// Get a list of key signers to use for authentication
	logger.Debugw("Get key signers")
	keySigners, err := options.GetKeySigners(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "generate auth signers")
	}

	// Open client connection
	logger.With(
		zap.String("ssh_user", options.User),
		zap.Int("ssh_auth_method_count", len(keySigners)),
	).Debugf("Authenticating as user %s", options.User)
	c, chans, reqs, err := gossh.NewClientConn(
		conn, addr.String(),
		&gossh.ClientConfig{
			User:            options.User,
			Auth:            []gossh.AuthMethod{gossh.PublicKeys(keySigners...)},
			HostKeyCallback: gossh.InsecureIgnoreHostKey(), // TODO: Fix
		},
	)
	if err != nil {
		return nil, nil, errors.Wrap(err, "establish SSH connection")
	}
	logger.Info("Client connection established")
	sshClient := gossh.NewClient(c, chans, reqs)

	// Start sending keepalive packets to the upstream SSH server
	keepaliveErrors := make(chan error)
	go func() {
		if err := sshKeepalive(ctx, conn, sshClient, options.KeepaliveInterval, options.DialTimeout); err != nil {
			if !errors.Is(err, net.ErrClosed) {
				logger.Errorw("Keepalive failed", zap.Error(err))
				keepaliveErrors <- err
			}
		}
	}()
	return sshClient, keepaliveErrors, nil
}

// sshKeepalive regularly sends a keepalive request and returns an error if there is a failure
func sshKeepalive(ctx context.Context, conn net.Conn, client *gossh.Client, interval, timeout time.Duration) error {
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
