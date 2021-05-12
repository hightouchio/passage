package tunnel

import (
	"context"
	"fmt"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
	"io"
	"net"
	"time"
)

type NormalTunnel struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"createdAt"`

	TunnelPort      uint32 `json:"port"`
	ServerEndpoint  string `json:"serverEndpoint"`
	ServerPort      uint32 `json:"serverPort"`
	ServiceEndpoint string `json:"serviceEndpoint"`
	ServicePort     uint32 `json:"servicePort"`

	services normalTunnelServices
}

// normalTunnelServices are the external dependencies that NormalTunnel needs to do its job
type normalTunnelServices struct {
	sql interface {
	}
}

func (t NormalTunnel) Start(ctx context.Context, options SSHOptions) error {
	t.Logger().WithField("tunnel_port", t.TunnelPort).Info("starting tunnel")

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", options.BindHost, t.TunnelPort))
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		localConn, err := listener.Accept()
		if err != nil {
			return err
		}
		go func() {
			defer localConn.Close()
			if err := t.handleConn(localConn, options); err != nil {
				t.Logger().WithError(err).Error("error handling client connection")
				listener.Close()
			}
		}()
	}
}

func (t NormalTunnel) handleConn(localConn net.Conn, options SSHOptions) error {
	auth, err := t.generateAuthMethod()

	t.Logger().WithFields(logrus.Fields{
		"hostname": t.ServerEndpoint,
		"port":     t.ServerPort,
	}).Debug("dialing remote ssh server")

	serverConn, err := ssh.Dial(
		"tcp",
		fmt.Sprintf("%s:%d", t.ServerEndpoint, t.ServerPort),
		&ssh.ClientConfig{
			User:            options.User,
			Auth:            auth,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		},
	)
	if err != nil {
		return err
	}
	defer serverConn.Close()

	t.Logger().WithFields(logrus.Fields{
		"hostname": t.ServiceEndpoint,
		"port": t.ServicePort,
	}).Debug("dialing tunneled service")

	remoteConn, err := serverConn.Dial("tcp", fmt.Sprintf("%s:%d", t.ServiceEndpoint, t.ServicePort))
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

	t.Logger().Info("started normal tunnel")

	return g.Wait()
}

func (t NormalTunnel) generateAuthMethod() ([]ssh.AuthMethod, error) {
	return []ssh.AuthMethod{}, nil

	// TODO: Re-enable when we wire up the keys
	//signer, err := ssh.ParsePrivateKey([]byte(t.PrivateKey))
	//if err != nil {
	//	return []ssh.AuthMethod{}, err
	//}
	//
	//return []ssh.AuthMethod{
	//	ssh.PublicKeys(signer),
	//}, nil
}

func (t NormalTunnel) getTunnelConnection(server string, remote string, config ssh.ClientConfig) (net.Conn, error) {
	serverConn, err := ssh.Dial("tcp", server, &config)
	if err != nil {
		return nil, err
	}

	remoteConn, err := serverConn.Dial("tcp", remote)
	if err != nil {
		return nil, err
	}

	return remoteConn, nil
}

func (t NormalTunnel) Logger() *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		"tunnel_type": "normal",
		"tunnel_id":   t.ID,
	})
}

func (t NormalTunnel) GetID() int {
	return t.ID
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

// convert a SQL DB representation of a postgres.NormalTunnel into the primary NormalTunnel struct
func normalTunnelFromSQL(record postgres.NormalTunnel) NormalTunnel {
	return NormalTunnel{
		ID:              record.ID,
		CreatedAt:       record.CreatedAt,
		TunnelPort:      record.TunnelPort,
		ServerEndpoint:  record.ServerEndpoint,
		ServerPort:      record.ServerPort,
		ServiceEndpoint: record.ServiceEndpoint,
		ServicePort:     record.ServicePort,
	}
}
