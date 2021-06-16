package tunnel

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
)

type NormalTunnel struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Enabled   bool      `json:"enabled"`

	TunnelPort  int    `json:"tunnelPort"`
	SSHUser     string `json:"sshUser"`
	SSHHost     string `json:"sshHost"`
	SSHPort     int    `json:"sshPort"`
	ServiceHost string `json:"serviceHost"`
	ServicePort int    `json:"servicePort"`

	services normalTunnelServices
}

// normalTunnelServices are the external dependencies that NormalTunnel needs to do its job
type normalTunnelServices struct {
	sql interface {
		GetNormalTunnelPrivateKeys(ctx context.Context, tunnelID uuid.UUID) ([]postgres.Key, error)
	}
}

func isContextCancelled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

func (t NormalTunnel) Start(ctx context.Context, options SSHOptions) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// generate our authentication strategy
	auth, err := t.generateAuthMethod(ctx)
	if err != nil {
		return errors.Wrap(err, "generate auth method")
	}

	// connect to remote SSH server
	t.logger().WithFields(logrus.Fields{"user": t.SSHUser, "host": t.SSHHost, "port": t.SSHPort}).Debug("dial ssh")
	sshConn, err := ssh.Dial(
		"tcp", fmt.Sprintf("%s:%d", t.SSHHost, t.SSHPort),
		&ssh.ClientConfig{
			User:            t.SSHUser,
			Auth:            auth,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		},
	)
	if err != nil {
		return errors.Wrap(err, "dial ssh")
	}
	defer func() {
		t.logger().Debug("stop ssh connection")
		sshConn.Close()
	}()

	// open tunnel listener
	t.logger().WithField("tunnel_port", t.TunnelPort).Debug("start tunnel listener")
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", options.BindHost, t.TunnelPort))
	if err != nil {
		return errors.Wrap(err, "listen")
	}
	defer func() {
		t.logger().Debug("stop tunnel listener")
		listener.Close()
	}()

	// accept incoming conns and serve them up
	incomingConns := make(chan net.Conn)
	go func() {
		for {
			select {
			default:
				conn, err := listener.Accept()
				if err != nil && !isContextCancelled(ctx) {
					t.logger().WithError(err).Error("tunnel connection accept error")
					break
				}
				incomingConns <- conn

			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case tunnelConn := <-incomingConns:
			go func() {
				if err := t.handleTunnelConnection(ctx, sshConn, tunnelConn); err != nil {
					t.logger().WithError(err).Error("tunnel error")
					tunnelConn.Write([]byte(errors.Wrap(err, conncheckErrorPrefix).Error()))
				}
			}()

		case <-ctx.Done():
			return nil
		}
	}
}

func (t NormalTunnel) handleTunnelConnection(ctx context.Context, sshConn *ssh.Client, tunnelConn net.Conn) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// connect to upstream service
	t.logger().WithFields(logrus.Fields{"host": t.ServiceHost, "port": t.ServicePort}).Debug("dial upstream service")
	serviceConn, err := sshConn.Dial("tcp", fmt.Sprintf("%s:%d", t.ServiceHost, t.ServicePort))
	if err != nil {
		return errors.Wrap(err, "dial upstream service")
	}
	defer serviceConn.Close()

	errs := make(chan error)
	go func() {
		g := new(errgroup.Group)
		g.Go(func() error { _, err := io.Copy(serviceConn, tunnelConn); return err })
		g.Go(func() error { _, err := io.Copy(tunnelConn, serviceConn); return err })
		errs <- g.Wait()
	}()

	// wait for an error or connection completion
	select {
	case <-ctx.Done():
		t.logger().Debug("ordered shutdown")
		return nil
	case err := <-errs:
		return err
	}
}

// generateAuthMethod finds the SSH private keys that are configured for this tunnel and structure them for use by the SSH client library
func (t NormalTunnel) generateAuthMethod(ctx context.Context) ([]ssh.AuthMethod, error) {
	// get private keys from database
	keys, err := t.services.sql.GetNormalTunnelPrivateKeys(ctx, t.ID)
	if err != nil {
		return []ssh.AuthMethod{}, errors.Wrap(err, "could not get keys from db")
	}

	// parse private keys and prepare for SSH
	authMethods := make([]ssh.AuthMethod, len(keys))
	for i, key := range keys {
		signer, err := ssh.ParsePrivateKey([]byte(key.Contents))
		if err != nil {
			return []ssh.AuthMethod{}, errors.Wrapf(err, "could not parse key %d", key.ID)
		}
		authMethods[i] = ssh.PublicKeys(signer)
	}

	return authMethods, nil
}

func (t NormalTunnel) GetConnectionDetails() ConnectionDetails {
	return ConnectionDetails{
		Host: os.Getenv("TUNNEL_HOST_NORMAL"), // TODO: need to get current server public IP
		Port: t.TunnelPort,
	}
}

// createNormalTunnelListFunc wraps our Postgres list function in something that converts the records into Normal structs so they can be passed to Manager which accepts the Tunnel interface
func createNormalTunnelListFunc(postgresList func(ctx context.Context) ([]postgres.NormalTunnel, error), services normalTunnelServices) ListFunc {
	return func(ctx context.Context) ([]Tunnel, error) {
		normalTunnels, err := postgresList(ctx)
		if err != nil {
			return []Tunnel{}, err
		}

		// convert all the SQL records to our primary struct
		tunnels := make([]Tunnel, len(normalTunnels))
		for i, record := range normalTunnels {
			tunnel := normalTunnelFromSQL(record)
			tunnel.services = services // inject dependencies
			tunnels[i] = tunnel
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
		t.TunnelPort == t2.TunnelPort &&
		t.ServiceHost == t2.ServiceHost &&
		t.ServicePort == t2.ServicePort
}

// sqlFromNormalTunnel converts tunnel data into something that can be inserted into the DB
func sqlFromNormalTunnel(tunnel NormalTunnel) postgres.NormalTunnel {
	return postgres.NormalTunnel{
		SSHUser:     tunnel.SSHUser,
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
		TunnelPort:  record.TunnelPort,
		SSHUser:     record.SSHUser,
		SSHHost:     record.SSHHost,
		SSHPort:     record.SSHPort,
		ServiceHost: record.ServiceHost,
		ServicePort: record.ServicePort,
	}
}

func (t NormalTunnel) GetID() uuid.UUID {
	return t.ID
}

func (t NormalTunnel) logger() *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		"tunnel_type": "normal",
		"tunnel_id":   t.ID.String(),
	})
}
