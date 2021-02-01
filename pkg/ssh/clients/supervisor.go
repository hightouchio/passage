package clients

import (
	"time"

	"github.com/apex/log"
	"github.com/hightouchio/passage/pkg/models"
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
			if err := s.startSSHClient(); err != nil {
				log.Error("start ssh client")
			}
		}
	}
}

// TODO
func (s *supervisor) startSSHClient() error {
	return nil
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
