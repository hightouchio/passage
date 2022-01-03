package tunnel

import (
	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
	"time"
)

type SSHServerOptions struct {
	BindHost string
	HostKey  []byte
}

func (o SSHServerOptions) GetHostSigners() ([]ssh.Signer, error) {
	var hostSigners []ssh.Signer
	if len(o.HostKey) != 0 {
		hostSigner, err := gossh.ParsePrivateKey(o.HostKey)
		if err != nil {
			return []ssh.Signer{}, err
		}
		hostSigners = []ssh.Signer{hostSigner}
	}
	return hostSigners, nil
}

type SSHClientOptions struct {
	User              string
	DialTimeout       time.Duration
	KeepaliveInterval time.Duration
}
