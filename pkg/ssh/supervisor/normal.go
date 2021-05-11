package supervisor

import (
	"fmt"
	"github.com/hightouchio/passage/tunnel"
	"io"
	"net"
	"time"

	"github.com/apex/log"
	"golang.org/x/crypto/ssh"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
)

const normalSupervisorRetryDuration = time.Second

type NormalSupervisor struct {
	bindHost string
	user     string
	tunnel   tunnel.NormalTunnel
}

func NewNormalSupervisor(
	bindHost string,
	user string,
	tunnel tunnel.NormalTunnel,
) *NormalSupervisor {
	return &NormalSupervisor{
		bindHost: bindHost,
		user:     user,
		tunnel:   tunnel,
	}
}

func (s *NormalSupervisor) Start() {
	go s.start()
}

func (s *NormalSupervisor) start() {
	ticker := time.NewTicker(normalSupervisorRetryDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.listen(); err != nil {
				log.WithError(err).Error("ssh client listener")
			}
		}
	}
}

func (s *NormalSupervisor) listen() error {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.bindHost, s.tunnel.Port))
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
			if err := s.handleConn(localConn); err != nil {
				log.WithError(err).Error("handle ssh client connection")
				listener.Close()
			}
		}()
	}
}

func (s *NormalSupervisor) handleConn(localConn net.Conn) error {
	signer, err := gossh.ParsePrivateKey([]byte(s.tunnel.PrivateKey))
	if err != nil {
		return err
	}

	serverConn, err := ssh.Dial(
		"tcp",
		fmt.Sprintf("%s:%d", s.tunnel.ServerEndpoint, s.tunnel.ServerPort),
		&ssh.ClientConfig{
			User: s.user,
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		},
	)
	if err != nil {
		return err
	}
	defer serverConn.Close()

	remoteConn, err := serverConn.Dial(
		"tcp",
		fmt.Sprintf("%s:%d", s.tunnel.ServiceEndpoint, s.tunnel.ServicePort),
	)
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

func (s *NormalSupervisor) getTunnelConnection(server string, remote string, config ssh.ClientConfig) (net.Conn, error) {
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
