package server

import (
	"crypto/rand"
	"crypto/rsa"
	"sync"
	"time"

	"github.com/apex/log"
	"github.com/gliderlabs/ssh"
	"github.com/hightouchio/passage/pkg/models"
	gossh "golang.org/x/crypto/ssh"
)

const retryDuration = time.Second

type Server struct {
	tunnels map[string]models.Tunnel
	lock    sync.RWMutex
	once    sync.Once
}

func NewServer() *Server {
	return &Server{
		tunnels: make(map[string]models.Tunnel),
		lock:    sync.RWMutex{},
		once:    sync.Once{},
	}
}

func (s *Server) SetTunnels(tunnels []models.Tunnel) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.tunnels = make(map[string]models.Tunnel)
	for _, tunnel := range tunnels {
		s.tunnels[tunnel.ID] = tunnel
	}

	s.once.Do(func() {
		go s.start()
	})
}

func (s *Server) start() {
	ticker := time.NewTicker(retryDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.startSSHServer(); err != nil {
				log.WithError(err).Error("start ssh server")
				continue
			}
		}
	}
}

func (s *Server) startSSHServer() error {
	signer, err := getSigner()
	if err != nil {
		return err
	}

	forwardHandler := &ssh.ForwardedTCPHandler{}
	sshServer := &ssh.Server{
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
		ReversePortForwardingCallback: func(ctx ssh.Context, bindHost string, bindPort uint32) bool {
			return bindHost == "localhost"
		},
	}

	if err = sshServer.SetOption(ssh.PublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
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
