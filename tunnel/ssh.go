package tunnel

import (
	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

type SSHOptions struct {
	BindHost string
	HostKey  []byte
}

func (o SSHOptions) GetHostSigners() ([]ssh.Signer, error) {
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
