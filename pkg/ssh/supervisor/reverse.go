package supervisor

import (
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/gliderlabs/ssh"
	"github.com/hightouchio/passage/pkg/models"
	gossh "golang.org/x/crypto/ssh"
)

const reverseSupervisorRetryDuration = time.Second

type ReverseSupervisor struct {
	bindHost      string
	hostKey       *string
	reverseTunnel models.ReverseTunnel
}

func NewReverseSupervisor(
	bindHost string,
	hostKey *string,
	reverseTunnel models.ReverseTunnel,
) *ReverseSupervisor {
	return &ReverseSupervisor{
		bindHost:      bindHost,
		hostKey:       hostKey,
		reverseTunnel: reverseTunnel,
	}
}

func (s *ReverseSupervisor) Start() {
	go s.start()
}

func (s *ReverseSupervisor) start() {
	ticker := time.NewTicker(reverseSupervisorRetryDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.startSSHServer(); err != nil {
				log.WithError(err).Error("start ssh server")
			}
		}
	}
}

func (s *ReverseSupervisor) startSSHServer() error {
	var hostSigners []ssh.Signer
	if s.hostKey != nil {
		hostSigner, err := gossh.ParsePrivateKey([]byte(*s.hostKey))
		if err != nil {
			return err
		}
		hostSigners = []ssh.Signer{hostSigner}
	}

	forwardHandler := &ssh.ForwardedTCPHandler{}
	sshServer := &ssh.Server{
		Addr: fmt.Sprintf(":%d", s.reverseTunnel.SSHPort),
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
		HostSigners: hostSigners,
		ReversePortForwardingCallback: func(ctx ssh.Context, bindHost string, bindPort uint32) bool {
			return bindHost == s.bindHost && bindPort == s.reverseTunnel.Port
		},
	}

	if err := sshServer.SetOption(ssh.PublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
		authorizedKey, _, _, _, err := gossh.ParseAuthorizedKey([]byte(s.reverseTunnel.PublicKey))
		if err != nil {
			return false
		}
		return ssh.KeysEqual(key, authorizedKey)
	})); err != nil {
		return err
	}

	return sshServer.ListenAndServe()
}
