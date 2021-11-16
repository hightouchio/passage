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
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

type StandardTunnel struct {
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
	services      StandardTunnelServices
}

func (t StandardTunnel) Start(ctx context.Context, options TunnelOptions) error {
	st := stats.GetStats(ctx)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Get a list of key signers to use for authentication
	keySigners, err := t.getAuthSigners(ctx)
	if err != nil {
		return errors.Wrap(err, "generate key signers")
	}

	// Determine SSH user to use, either from the database or from the config.
	var sshUser string
	if t.SSHUser != "" {
		sshUser = t.SSHUser
	} else {
		sshUser = t.clientOptions.User
	}

	// Dial remote SSH server
	addr := net.JoinHostPort(t.SSHHost, strconv.Itoa(t.SSHPort))
	st.WithEventTags(stats.Tags{"ssh_host": t.SSHHost, "ssh_port": t.SSHPort}).SimpleEvent("dial")
	sshConn, err := net.DialTimeout("tcp", addr, t.clientOptions.DialTimeout)
	if err != nil {
		return errors.Wrap(err, "dial remote")
	}
	defer func() {
		sshConn.Close()
	}()

	// Init SSH connection protocol
	st.WithEventTags(stats.Tags{"ssh_user": sshUser, "auth_method_count": len(keySigners)}).SimpleEvent("ssh")
	c, chans, reqs, err := ssh.NewClientConn(
		sshConn, addr,
		&ssh.ClientConfig{
			User:            sshUser,
			Auth:            []ssh.AuthMethod{ssh.PublicKeys(keySigners...)},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		},
	)
	if err != nil {
		return errors.Wrap(err, "ssh")
	}
	sshClient := ssh.NewClient(c, chans, reqs)

	// Start connection Keepalive.
	keepaliveErr := make(chan error)
	go sshKeepalive(ctx, sshClient, sshConn, t.clientOptions, keepaliveErr)

	// Listen for incoming tunnel connections.
	st.WithEventTags(stats.Tags{"port": t.TunnelPort}).SimpleEvent("listener.start")
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", options.BindHost, t.TunnelPort))
	if err != nil {
		return errors.Wrap(err, "listen")
	}
	defer func() {
		listener.Close()
	}()
	incomingConns := make(chan net.Conn)
	go func() {
		for {
			select {
			default:
				// Accept incoming tunnel connections
				conn, err := listener.Accept()
				if err != nil && !isContextCancelled(ctx) {
					t.logger().WithError(err).Error("tunnel conn accept error")
					break
				}
				incomingConns <- conn

			case <-ctx.Done():
				return
			}
		}
	}()

	statsTicker := time.NewTicker(1 * time.Second)
	defer statsTicker.Stop()
	var activeConnections int32
	for {
		select {
		// Handle incoming tunnel connections.
		case tunnelConn := <-incomingConns:
			go func() {
				st := st.WithEventTags(stats.Tags{"remote_addr": tunnelConn.RemoteAddr().String()}).WithPrefix("conn")
				ctx := stats.InjectContext(ctx, st)

				st.SimpleEvent("accept")
				st.Incr("accept", nil, 1)

				atomic.AddInt32(&activeConnections, 1)
				read, written, err := t.handleTunnelConnection(ctx, sshClient, tunnelConn)
				atomic.AddInt32(&activeConnections, -1)
				st.Gauge("read", float64(read), nil, 1)
				st.Gauge("write", float64(written), nil, 1)
				st = st.WithEventTags(stats.Tags{"read": read, "write": written})

				if err != nil {
					st.ErrorEvent("error", err)
					tunnelConn.Write([]byte(errors.Wrap(err, conncheckErrorPrefix).Error()))
					return
				}
				st.SimpleEvent("close")
			}()

		case <-statsTicker.C:
			// explicit tunnelId tag here, so it appears on the metric
			st.WithTags(stats.Tags{"tunnel_id": t.ID.String()}).Gauge("active_connections", float64(atomic.LoadInt32(&activeConnections)), nil, 1)

		case <-keepaliveErr:
			return errors.Wrap(err, "keepalive")

		case <-ctx.Done():
			return nil
		}
	}
}

func (t StandardTunnel) handleTunnelConnection(ctx context.Context, sshConn *ssh.Client, tunnelConn net.Conn) (int64, int64, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	st := stats.GetStats(ctx)

	// Dial upstream service.
	st.WithEventTags(stats.Tags{"upstream_host": t.ServiceHost, "upstream_port": t.ServicePort}).SimpleEvent("upstream.dial")
	serviceConn, err := sshConn.Dial("tcp", fmt.Sprintf("%s:%d", t.ServiceHost, t.ServicePort))
	if err != nil {
		return 0, 0, errors.Wrap(err, "dial upstream")
	}
	defer serviceConn.Close()

	copy := func(g *sync.WaitGroup, src io.Reader, dst io.Writer, written *int64) error {
		defer g.Done()
		byteCount, err := io.Copy(dst, src)
		*written = byteCount
		return err
	}

	var read, written int64
	go func() {
		defer cancel()
		g := new(sync.WaitGroup)
		g.Add(2)
		// Copy data bidirectionally.
		go copy(g, tunnelConn, serviceConn, &written)
		go copy(g, serviceConn, tunnelConn, &read)
		g.Wait()
	}()

	// Wait for an error or connection completion
	select {
	case <-ctx.Done():
		return read, written, nil
	}
}

// getAuthSigners finds the SSH keys that are configured for this tunnel and structure them for use by the SSH client library
func (t StandardTunnel) getAuthSigners(ctx context.Context) ([]ssh.Signer, error) {
	// get private keys from database
	keys, err := t.services.SQL.GetStandardTunnelPrivateKeys(ctx, t.ID)
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

		t.logger().WithField("fingerprint", ssh.FingerprintSHA256(signer.PublicKey())).Debug("using ssh key")
		signers[i] = signer
	}

	return signers, nil
}

func sshKeepalive(ctx context.Context, client *ssh.Client, conn net.Conn, options SSHClientOptions, errChan chan<- error) {
	t := time.NewTicker(options.KeepaliveInterval)
	defer t.Stop()

	err := func() error {
		for {
			deadline := time.Now().Add(options.KeepaliveInterval).Add(options.KeepaliveTimeout)
			err := conn.SetDeadline(deadline)
			if err != nil {
				return errors.Wrap(err, "set deadline")
			}
			select {
			case <-t.C:
				_, _, err = client.SendRequest("keepalive@passage", true, nil)
				if err != nil {
					return errors.Wrap(err, "send keep alive")
				}
			case <-ctx.Done():
				return nil
			}
		}
	}()
	if err != nil {
		errChan <- err
	}
	close(errChan)
}

func (t StandardTunnel) GetConnectionDetails(discovery discovery.DiscoveryService) (ConnectionDetails, error) {
	tunnelHost, err := discovery.ResolveTunnelHost("standard", t.ID)
	if err != nil {
		return ConnectionDetails{}, errors.Wrap(err, "could not resolve tunnel host")
	}

	return ConnectionDetails{
		Host: tunnelHost,
		Port: t.TunnelPort,
	}, nil
}

// StandardTunnelServices are the external dependencies that StandardTunnel needs to do its job
type StandardTunnelServices struct {
	SQL interface {
		GetStandardTunnelPrivateKeys(ctx context.Context, tunnelID uuid.UUID) ([]postgres.Key, error)
	}
	Keystore keystore.Keystore
}

func InjectStandardTunnelDependencies(f func(ctx context.Context) ([]StandardTunnel, error), services StandardTunnelServices, options SSHClientOptions) ListFunc {
	return func(ctx context.Context) ([]Tunnel, error) {
		// Get standard tunnels
		sts, err := f(ctx)
		if err != nil {
			return []Tunnel{}, err
		}
		// Inject ClientOptions into StandardTunnels
		tunnels := make([]Tunnel, len(sts))
		for i, st := range sts {
			st.services = services
			st.clientOptions = options
			tunnels[i] = st
		}
		return tunnels, nil
	}
}

func (t StandardTunnel) Equal(v interface{}) bool {
	t2, ok := v.(StandardTunnel)
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

// sqlFromStandardTunnel converts tunnel data into something that can be inserted into the DB
func sqlFromStandardTunnel(tunnel StandardTunnel) postgres.StandardTunnel {
	return postgres.StandardTunnel{
		SSHUser:     sql.NullString{String: tunnel.SSHUser, Valid: tunnel.SSHUser != ""},
		SSHHost:     tunnel.SSHHost,
		SSHPort:     tunnel.SSHPort,
		ServiceHost: tunnel.ServiceHost,
		ServicePort: tunnel.ServicePort,
	}
}

// convert a SQL DB representation of a postgres.StandardTunnel into the primary StandardTunnel struct
func standardTunnelFromSQL(record postgres.StandardTunnel) StandardTunnel {
	return StandardTunnel{
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

func (t StandardTunnel) GetID() uuid.UUID {
	return t.ID
}

func (t StandardTunnel) logger() *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		"tunnel_type": "standard",
		"tunnel_id":   t.ID.String(),
	})
}
