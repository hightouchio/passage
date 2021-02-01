package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/gliderlabs/ssh"
	"github.com/hightouchio/passage/pkg/models"
	gossh "golang.org/x/crypto/ssh"
)

const retryDuration = time.Second

type supervisor struct {
	tunnel models.Tunnel
}

func newSupervisor(tunnel models.Tunnel) *supervisor {
	return &supervisor{
		tunnel: tunnel,
	}
}

func (s *supervisor) Start() {
	go s.start()
}

func (s *supervisor) start() {
	ticker := time.NewTicker(retryDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			switch s.tunnel.Type {
			case models.TunnelTypeNormal:
				if err := s.startSSHClient(); err != nil {
					log.Error("start ssh client")
				}
			case models.TunnelTypeReverse:
				if err := s.startSSHServer(); err != nil {
					log.Error("start ssh server")
				}
			}
		}
	}
}

// TODO
func (s *supervisor) startSSHClient() error {
	return nil
}

func (s *supervisor) startSSHServer() error {
	signer, err := getSigner()
	if err != nil {
		return err
	}

	forwardHandler := &ssh.ForwardedTCPHandler{}
	sshServer := &ssh.Server{
		Addr: fmt.Sprintf(":%d", s.tunnel.Port),
		Handler: func(s ssh.Session) {
			select {}
		},
		RequestHandlers: map[string]ssh.RequestHandler{
			"tcpip-forward":        forwardHandler.HandleSSHRequest,
			"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
		},
		ChannelHandlers: map[string]ssh.ChannelHandler{
			"session":      ssh.DefaultSessionHandler,
			"direct-tcpip": ssh.DirectTCPIPHandler,
		},
		HostSigners: []ssh.Signer{signer},
		LocalPortForwardingCallback: func(ctx ssh.Context, destinationHost string, destinationPort uint32) bool {
			return false
		},
		ReversePortForwardingCallback: func(ctx ssh.Context, bindHost string, bindPort uint32) bool {
			return bindHost == "0.0.0.0" && bindPort == s.tunnel.Port
		},
	}

	if err = sshServer.SetOption(ssh.PublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
		/*for _, authorizedKey := range s.variables.GetAuthorizedSSHKeys() {
			if ssh.KeysEqual(key, authorizedKey) {
				return true
			}
		}
		return false*/
		return true
	})); err != nil {
		return err
	}

	return sshServer.ListenAndServe()
}

func getSigner() (ssh.Signer, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	signer, err := gossh.NewSignerFromKey(key)
	if err != nil {
		return nil, err
	}

	return signer, nil
}

/*package main

// Forward from local port 9000 to remote port 9999

import (
    "io"
    "log"
    "net"
    "golang.org/x/crypto/ssh"
)

var (
    username         = "root"
    password         = "password"
    serverAddrString = "192.168.1.100:22"
    localAddrString  = "localhost:9000"
    remoteAddrString = "localhost:9999"
)

func forward(localConn net.Conn, config *ssh.ClientConfig) {
    // Setup sshClientConn (type *ssh.ClientConn)
    sshClientConn, err := ssh.Dial("tcp", serverAddrString, config)
    if err != nil {
        log.Fatalf("ssh.Dial failed: %s", err)
    }

    // Setup sshConn (type net.Conn)
    sshConn, err := sshClientConn.Dial("tcp", remoteAddrString)

    // Copy localConn.Reader to sshConn.Writer
    go func() {
        _, err = io.Copy(sshConn, localConn)
        if err != nil {
            log.Fatalf("io.Copy failed: %v", err)
        }
    }()

    // Copy sshConn.Reader to localConn.Writer
    go func() {
        _, err = io.Copy(localConn, sshConn)
        if err != nil {
            log.Fatalf("io.Copy failed: %v", err)
        }
    }()
}

func main() {
    // Setup SSH config (type *ssh.ClientConfig)
    config := &ssh.ClientConfig{
        User: username,
        Auth: []ssh.AuthMethod{
            ssh.Password(password),
        },
    }

    // Setup localListener (type net.Listener)
    localListener, err := net.Listen("tcp", localAddrString)
    if err != nil {
        log.Fatalf("net.Listen failed: %v", err)
    }

    for {
        // Setup localConn (type net.Conn)
        localConn, err := localListener.Accept()
        if err != nil {
            log.Fatalf("listen.Accept failed: %v", err)
        }
        go forward(localConn, config)
    }
}*/
