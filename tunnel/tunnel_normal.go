package tunnel

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
	"io"
	"net"
	"time"
)

type NormalTunnel struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"createdAt"`

	TunnelPort  int    `json:"port"`
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

func (t NormalTunnel) Start(ctx context.Context, options SSHOptions) error {
	t.logger().WithField("tunnel_port", t.TunnelPort).Info("starting tunnel")

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", options.BindHost, t.TunnelPort))
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		localConn, err := listener.Accept()
		if err != nil {
			// TODO: Question for Josh
			return errors.Wrap(err, "could not accept")
		}

		connCtx := context.Background() // new context for each connection
		go func() {
			defer localConn.Close()

			if err := t.handleConn(connCtx, localConn); err != nil {
				t.logger().WithError(err).Error("error handling client connection")
				localConn.Write([]byte(errors.Wrap(err, conncheckErrorPrefix).Error()))
			}
		}()
	}
}

func (t NormalTunnel) handleConn(ctx context.Context, localConn net.Conn) error {
	// generate the authent
	auth, err := t.generateAuthMethod(ctx)
	if err != nil {
		return errors.Wrap(err, "could not generate auth methods")
	}

	t.logger().WithFields(logrus.Fields{
		"hostname": t.SSHHost,
		"port":     t.SSHPort,
	}).Debug("dialing remote ssh server")

	serverConn, err := ssh.Dial(
		"tcp",
		fmt.Sprintf("%s:%d", t.SSHHost, t.SSHPort),
		&ssh.ClientConfig{
			User:            t.SSHUser,
			Auth:            auth,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		},
	)
	if err != nil {
		return err
	}
	defer serverConn.Close()

	t.logger().WithFields(logrus.Fields{
		"hostname": t.ServiceHost,
		"port":     t.ServicePort,
	}).Debug("dialing tunneled service")

	remoteConn, err := serverConn.Dial("tcp", fmt.Sprintf("%s:%d", t.ServiceHost, t.ServicePort))
	if err != nil {
		return err
	}
	defer remoteConn.Close()

	g := new(errgroup.Group)
	g.Go(func() error {
		_, err := io.Copy(remoteConn, localConn)
		return err
	})

	g.Go(func() error {
		_, err := io.Copy(localConn, remoteConn)
		return err
	})

	t.logger().Info("started normal tunnel")

	return g.Wait()
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
		Host: "localhost", // TODO: need to get current server public IP
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
