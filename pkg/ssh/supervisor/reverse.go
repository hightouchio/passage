package supervisor

import (
	"fmt"
	"github.com/hightouchio/passage/pkg/models"
	"time"

	"github.com/apex/log"
	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

const reverseSupervisorRetryDuration = time.Second

type ReverseSupervisor struct {
	bindHost      string
	hostKey       []byte
	reverseTunnel models.ReverseTunnel
}

func NewReverseSupervisor(
	bindHost string,
	hostKey []byte,
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
	if len(s.hostKey) != 0 {
		hostSigner, err := gossh.ParsePrivateKey(s.hostKey)
		if err != nil {
			return err
		}
		hostSigners = []ssh.Signer{hostSigner}
	}

	forwardHandler := &ssh.ForwardedTCPHandler{}
	sshServer := &ssh.Server{
		Addr: fmt.Sprintf(":%d", s.reverseTunnel.SSHPort),
		Handler: func(s ssh.Session) {
			log.Debug("reverse tunnel handler triggered")
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

	// compare incoming connection key to the key authorized for this tunnel configuration
	// TODO: check the key from the database in realtime
	if err := sshServer.SetOption(ssh.PublicKeyAuth(func(ctx ssh.Context, incomingKey ssh.PublicKey) bool {
		authorizedKey, _, _, _, err := gossh.ParseAuthorizedKey([]byte(s.reverseTunnel.PublicKey))
		if err != nil {
			return false
		}

		return ssh.KeysEqual(incomingKey, authorizedKey)
	})); err != nil {
		return err
	}

	log.WithField("ssh_port", s.reverseTunnel.SSHPort).Info("started reverse tunnel")

	return sshServer.ListenAndServe()
}
