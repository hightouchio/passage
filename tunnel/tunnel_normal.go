package tunnel

import (
	"context"
	"fmt"
	"github.com/apex/log"
	"github.com/hightouchio/passage/tunnel/postgres"
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
}

func (t NormalTunnel) Start(ctx context.Context, options SSHOptions) error {
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
				log.WithError(err).Error("handle ssh client connection")
				listener.Close()
			}
		}()
	}
}

func (t NormalTunnel) handleConn(localConn net.Conn, options SSHOptions) error {
	auth, err := t.generateAuthMethod()

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

func (t NormalTunnel) GetID() int {
	return t.ID
}

// createNormalTunnelListFunc wraps our Postgres list function in something that converts the records into Normal structs so they can be passed to Manager which accepts the Tunnel interface
func createNormalTunnelListFunc(postgresList func(ctx context.Context) ([]postgres.NormalTunnel, error)) ListFunc {
	return func(ctx context.Context) ([]Tunnel, error) {
		normalTunnels, err := postgresList(ctx)
		if err != nil {
			return []Tunnel{}, err
		}

		// convert all the SQL records to our primary struct
		tunnels := make([]Tunnel, len(normalTunnels))
		for i, tunnel := range normalTunnels {
			tunnels[i] = normalTunnelFromSQL(tunnel)
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
