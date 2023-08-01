package tunnel

import (
	"context"
	"github.com/gliderlabs/ssh"
	"github.com/pkg/errors"
	gossh "golang.org/x/crypto/ssh"
	"net"
	"time"
)

type SSHServerOptions struct {
	BindHost     string
	HostKey      []byte
	AuthDisabled bool
}

func (o SSHServerOptions) GetHostSigners() ([]ssh.Signer, error) {
	var hostSigners []ssh.Signer
	if len(o.HostKey) != 0 {
		signers, err := getSignersForPrivateKey(o.HostKey)
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
