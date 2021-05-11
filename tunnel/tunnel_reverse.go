package tunnel

import (
	"context"
	"fmt"
	"github.com/apex/log"
	"github.com/gliderlabs/ssh"
	"github.com/hightouchio/passage/tunnel/postgres"
	gossh "golang.org/x/crypto/ssh"
	"time"
)

type ReverseTunnel struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	PublicKey string    `json:"publicKey"`
	Port      uint32    `json:"port"`
	SSHPort   uint32    `json:"sshPort"`
}

func (t ReverseTunnel) Start(ctx context.Context, options SSHOptions) error {
	var hostSigners []ssh.Signer

	if len(options.HostKey) != 0 {
		hostSigner, err := gossh.ParsePrivateKey(options.HostKey)
		if err != nil {
			return err
		}
		hostSigners = []ssh.Signer{hostSigner}
	}

	forwardHandler := &ssh.ForwardedTCPHandler{}
	sshServer := &ssh.Server{
		Addr: fmt.Sprintf(":%d", t.SSHPort),
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
			return bindHost == options.BindHost && bindPort == t.Port
		},
	}

	// compare incoming connection key to the key authorized for this tunnel configuration
	// TODO: check the key from the database in realtime
	if err := sshServer.SetOption(ssh.PublicKeyAuth(func(ctx ssh.Context, incomingKey ssh.PublicKey) bool {
		authorizedKey, _, _, _, err := gossh.ParseAuthorizedKey([]byte(t.PublicKey))
		if err != nil {
			return false
		}

		return ssh.KeysEqual(incomingKey, authorizedKey)
	})); err != nil {
		return err
	}

	log.WithField("ssh_port", t.SSHPort).Info("started reverse tunnel")

	return sshServer.ListenAndServe()
}

func (t ReverseTunnel) GetID() int {
	return t.ID
}

// createReverseTunnelListFunc wraps our Postgres list function in something that converts the records into ReverseTunnel structs so they can be passed to Manager which accepts the Tunnel interface
func createReverseTunnelListFunc(postgresList func(ctx context.Context) ([]postgres.ReverseTunnel, error)) ListFunc {
	return func(ctx context.Context) ([]Tunnel, error) {
		reverseTunnels, err := postgresList(ctx)
		if err != nil {
			return []Tunnel{}, err
		}

		// convert all the SQL records to our primary struct
		tunnels := make([]Tunnel, len(reverseTunnels))
		for i, tunnel := range reverseTunnels {
			tunnels[i] = reverseTunnelFromSQL(tunnel)
		}

		return tunnels, nil
	}
}

// convert a SQL DB representation of a postgres.ReverseTunnel into the primary ReverseTunnel struct
func reverseTunnelFromSQL(record postgres.ReverseTunnel) ReverseTunnel {
	return ReverseTunnel{
		ID:        record.ID,
		CreatedAt: record.CreatedAt,
		PublicKey: record.PublicKey,
		Port:      record.Port,
		SSHPort:   record.SSHPort,
	}
}
