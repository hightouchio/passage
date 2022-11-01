package tunnel

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/hightouchio/passage/stats"
	"github.com/hightouchio/passage/tunnel/discovery"
	"github.com/hightouchio/passage/tunnel/keystore"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

type NormalTunnel struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Enabled   bool      `json:"enabled"`

	TunnelPort  int    `json:"tunnelPort"`
	SSHUser     string `json:"sshUser"`
	SSHHost     string `json:"sshHost"`
	SSHPort     int    `json:"sshPort"`
	ServiceHost string `json:"serviceHost"`
	ServicePort int    `json:"servicePort"`

	clientOptions SSHClientOptions
	services      NormalTunnelServices

	upstream *normalTunnelUpstream
}

func (t *NormalTunnel) Start(ctx context.Context, options TunnelOptions) error {
	lifecycle := getCtxLifecycle(ctx)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Get a list of key signers to use for authentication
	keySigners, err := t.getAuthSigners(ctx)
	if err != nil {
		return bootError{event: "generate_auth_signers", err: err}
	}

	// Determine SSH user to use, either from the database or from the config.
	var sshUser string
	if t.SSHUser != "" {
		sshUser = t.SSHUser
	} else {
		sshUser = t.clientOptions.User
	}

	// Dial external SSH server
	lifecycle.BootEvent("remote_dial", stats.Tags{"ssh_host": t.SSHHost, "ssh_port": t.SSHPort})
	addr := net.JoinHostPort(t.SSHHost, strconv.Itoa(t.SSHPort))
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return bootError{event: "remote_dial", err: errors.Wrapf(err, "resolve addr %s", addr)}
	}
	sshConn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return bootError{event: "remote_dial", err: err}
	}
	defer sshConn.Close()

	// Configure TCP keepalive for SSH connection
	sshConn.SetKeepAlive(true)
	sshConn.SetKeepAlivePeriod(t.clientOptions.KeepaliveInterval)

	// Init SSH connection protocol
	lifecycle.BootEvent("ssh_connect", stats.Tags{"ssh_user": sshUser, "ssh_auth_method_count": len(keySigners)})
	c, chans, reqs, err := ssh.NewClientConn(
		sshConn, addr,
		&ssh.ClientConfig{
			User:            sshUser,
			Auth:            []ssh.AuthMethod{ssh.PublicKeys(keySigners...)},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		},
	)
	defer c.Close()
	if err != nil {
		return bootError{event: "ssh_connect", err: err}
	}
	sshClient := ssh.NewClient(c, chans, reqs)
	t.upstream = &normalTunnelUpstream{sshClient: sshClient}

	// Start sending keepalive packets to the upstream SSH server
	go func() {
		if err := sshKeepaliver(ctx, sshConn, sshClient, t.clientOptions.KeepaliveInterval, t.clientOptions.DialTimeout); err != nil {
			defer cancel()
			lifecycle.Error(errors.Wrap(err, "ssh keepalive failed"))
		}
	}()

	return nil
}

func (t *NormalTunnel) Dial(downstream net.Conn, addr string) (io.ReadWriteCloser, error) {
	if t.upstream == nil {
		return nil, fmt.Errorf("upstream does not exist")
	}
	return t.upstream.Dial(downstream, addr)
}

// normalTunnelUpstream provides an interface for opening new connections on a SSH connection opened upstream
type normalTunnelUpstream struct {
	sshClient *ssh.Client
}

func (u normalTunnelUpstream) Dial(_ net.Conn, addr string) (io.ReadWriteCloser, error) {
	serviceConn, err := u.sshClient.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return serviceConn, nil
}

// getAuthSigners finds the SSH keys that are configured for this tunnel and structure them for use by the SSH client library
func (t *NormalTunnel) getAuthSigners(ctx context.Context) ([]ssh.Signer, error) {
	// get private keys from database
	keys, err := t.services.SQL.GetNormalTunnelPrivateKeys(ctx, t.ID)
	if err != nil {
		return []ssh.Signer{}, errors.Wrap(err, "could not look up private keys")
	}
	signers := make([]ssh.Signer, len(keys))

	// parse private keys and prepare for SSH
	for i, key := range keys {
		contents, err := t.services.Keystore.Get(ctx, key.ID)
		if err != nil {
			return []ssh.Signer{}, errors.Wrapf(err, "could not get contents for key %s", key.ID)
		}
		signer, err := ssh.ParsePrivateKey(contents)
		if err != nil {
			return []ssh.Signer{}, errors.Wrapf(err, "could not parse key %s", key.ID)
		}

		signers[i] = signer
	}

	return signers, nil
}

func (t *NormalTunnel) GetConnectionDetails(discovery discovery.DiscoveryService) (ConnectionDetails, error) {
	tunnelHost, err := discovery.ResolveTunnelHost(Normal, t.ID)
	if err != nil {
		return ConnectionDetails{}, errors.Wrap(err, "could not resolve tunnel host")
	}

	return ConnectionDetails{
		Host: tunnelHost,
		Port: t.TunnelPort,
	}, nil
}

// NormalTunnelServices are the external dependencies that NormalTunnel needs to do its job
type NormalTunnelServices struct {
	SQL interface {
		GetNormalTunnelPrivateKeys(ctx context.Context, tunnelID uuid.UUID) ([]postgres.Key, error)
	}
	Keystore keystore.Keystore
	Logger   *logrus.Logger
}

func InjectNormalTunnelDependencies(f func(ctx context.Context) ([]*NormalTunnel, error), services NormalTunnelServices, options SSHClientOptions) ListFunc {
	return func(ctx context.Context) ([]Tunnel, error) {
		sts, err := f(ctx)
		if err != nil {
			return []Tunnel{}, err
		}
		tunnels := make([]Tunnel, len(sts))
		for i, st := range sts {
			st.services = services
			st.clientOptions = options
			tunnels[i] = st
		}
		return tunnels, nil
	}
}

func (t *NormalTunnel) Equal(v interface{}) bool {
	t2, ok := v.(*NormalTunnel)
	if !ok {
		return false
	}

	return t.ID == t2.ID &&
		t.SSHUser == t2.SSHUser &&
		t.SSHHost == t2.SSHHost &&
		t.SSHPort == t2.SSHPort &&
		t.TunnelPort == t2.TunnelPort &&
		t.ServiceHost == t2.ServiceHost &&
		t.ServicePort == t2.ServicePort
}

// sqlFromNormalTunnel converts tunnel data into something that can be inserted into the DB
func sqlFromNormalTunnel(tunnel NormalTunnel) postgres.NormalTunnel {
	return postgres.NormalTunnel{
		SSHUser:     sql.NullString{String: tunnel.SSHUser, Valid: tunnel.SSHUser != ""},
		SSHHost:     tunnel.SSHHost,
		SSHPort:     tunnel.SSHPort,
		ServiceHost: tunnel.ServiceHost,
		ServicePort: tunnel.ServicePort,
	}
}

// convert a SQL DB representation of a postgres.NormalTunnel into the primary NormalTunnel struct
func normalTunnelFromSQL(record postgres.NormalTunnel) *NormalTunnel {
	return &NormalTunnel{
		ID:          record.ID,
		CreatedAt:   record.CreatedAt,
		Enabled:     record.Enabled,
		TunnelPort:  record.TunnelPort,
		SSHUser:     record.SSHUser.String,
		SSHHost:     record.SSHHost,
		SSHPort:     record.SSHPort,
		ServiceHost: record.ServiceHost,
		ServicePort: record.ServicePort,
	}
}

func (t *NormalTunnel) GetID() uuid.UUID {
	return t.ID
}

func (t *NormalTunnel) GetError() *string {
	// TODO: Fix this
	return nil
}
