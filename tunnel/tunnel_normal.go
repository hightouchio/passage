package tunnel

import (
	"context"
	"database/sql"
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
}

func (t NormalTunnel) Start(ctx context.Context, options TunnelOptions) error {
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

	// Resolve dial address
	addr := net.JoinHostPort(t.SSHHost, strconv.Itoa(t.SSHPort))
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return errors.Wrapf(err, "could not resolve addr %s", addr)
	}
	// Dial external SSH server
	st.WithEventTags(stats.Tags{"ssh_host": t.SSHHost, "ssh_port": t.SSHPort}).SimpleEvent("dial")
	sshConn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return errors.Wrap(err, "dial remote")
	}
	defer sshConn.Close()
	// Configure TCP keepalive for SSH connection
	sshConn.SetKeepAlive(true)
	sshConn.SetKeepAlivePeriod(t.clientOptions.KeepaliveInterval)

	// Init SSH connection protocol
	st.WithEventTags(stats.Tags{"ssh_user": sshUser, "auth_method_count": len(keySigners)}).SimpleEvent("ssh_connect")
	c, chans, reqs, err := ssh.NewClientConn(
		sshConn, addr,
		&ssh.ClientConfig{
			User:            sshUser,
			Auth:            []ssh.AuthMethod{ssh.PublicKeys(keySigners...)},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		},
	)
	if err != nil {
		return errors.Wrap(err, "ssh connect")
	}
	sshClient := ssh.NewClient(c, chans, reqs)

	// Resolve tunnel listener addr.
	listenAddr := net.JoinHostPort(options.BindHost, strconv.Itoa(t.TunnelPort))
	listenTcpAddr, err := net.ResolveTCPAddr("tcp", listenAddr)
	if err != nil {
		return errors.Wrap(err, "resolve tunnel listen addr")
	}

	// Listen for incoming tunnel connections.
	st.WithEventTags(stats.Tags{"listen_addr": listenAddr}).SimpleEvent("listener.start")
	listener, err := net.ListenTCP("tcp", listenTcpAddr)
	if err != nil {
		return errors.Wrap(err, "tunnel listen")
	}
	defer listener.Close()

	// Start sending keepalive packets to the SSH server
	keepaliveErr := make(chan error)
	go func() {
		if err := sshKeepaliver(ctx, sshConn, sshClient, t.clientOptions.KeepaliveInterval, t.clientOptions.DialTimeout); err != nil {
			keepaliveErr <- err
		}
	}()

	// Accept incoming connections and push them to this channel
	incomingConns := make(chan *net.TCPConn)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			default:
				// Accept incoming tunnel connections
				conn, err := listener.AcceptTCP()
				if err != nil {
					break
				}

				incomingConns <- conn
			}
		}
	}()

	// Report active connections every tick
	var activeConnections int32
	go func() {
		statsTicker := time.NewTicker(1 * time.Second)
		defer statsTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case <-statsTicker.C:
				// explicit tunnelId tag here, so it appears on the metric
				st.WithTags(stats.Tags{"tunnel_id": t.ID.String()}).Gauge("active_connections", float64(atomic.LoadInt32(&activeConnections)), nil, 1)
			}
		}
	}()

	// Handle incoming tunnel connections
	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			case tunnelConn := <-incomingConns:
				go func() {
					st := st.WithEventTags(stats.Tags{"remote_addr": tunnelConn.RemoteAddr().String()}).WithPrefix("conn")
					st.SimpleEvent("accept")
					st.Incr("accept", nil, 1)
					atomic.AddInt32(&activeConnections, 1)

					// Configure keepalive
					tunnelConn.SetKeepAlive(true)
					tunnelConn.SetKeepAlivePeriod(t.clientOptions.KeepaliveInterval)
					// Tunnel connection
					read, written, err := t.handleTunnelConnection(stats.InjectContext(ctx, st), sshClient, tunnelConn)

					atomic.AddInt32(&activeConnections, -1)
					// TODO: This is probably wrong because its overriding a shared value.
					st = st.WithEventTags(stats.Tags{"bytes_read": read, "bytes_written": written})
					st.Gauge("read", float64(read), nil, 1)
					st.Gauge("write", float64(written), nil, 1)
					if err != nil {
						st.ErrorEvent("error", err)
						tunnelConn.Write([]byte(errors.Wrap(err, conncheckErrorPrefix).Error()))
						return
					}
					st.SimpleEvent("close")
				}()
			}
		}
	}()

	select {
	case err := <-keepaliveErr:
		st.ErrorEvent("keepalive_failed", err)

	case <-ctx.Done():
	}

	return nil
}

// handleTunnelConnection handles incoming TCP connections on the tunnel listen port, dials the tunneled upstream, and copies bytes bidirectionally
func (t NormalTunnel) handleTunnelConnection(ctx context.Context, sshClient *ssh.Client, tunnelConn net.Conn) (int64, int64, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	st := stats.GetStats(ctx)

	// Dial upstream service.
	upstreamAddr := net.JoinHostPort(t.ServiceHost, strconv.Itoa(t.ServicePort))
	st.WithEventTags(stats.Tags{"upstream_addr": upstreamAddr}).SimpleEvent("upstream.dial")
	serviceConn, err := sshClient.Dial("tcp", upstreamAddr)
	if err != nil {
		return 0, 0, errors.Wrap(err, "dial upstream")
	}
	defer serviceConn.Close()

	// copyConn copies all bytes from io.Reader to io.Writer, records
	copyConn := func(g *sync.WaitGroup, src io.Reader, dst io.Writer, written *int64, errors chan<- error) {
		defer g.Done()

		byteCount, err := io.Copy(dst, src)
		if err != nil {
			errors <- err
		}
		// Record number of bytes written in this direction
		*written = byteCount
	}

	var read, written int64
	readErr := make(chan error)
	writeErr := make(chan error)

	// Copy all bytes from tunnel to service and service to tunnel
	go func() {
		defer cancel()
		g := new(sync.WaitGroup)
		g.Add(2)

		// Copy data bidirectionally.
		go copyConn(g, serviceConn, tunnelConn, &read, readErr)
		go copyConn(g, tunnelConn, serviceConn, &written, writeErr)
		g.Wait()

		// Close serviceConn before the end of the function, so we get a count of bytes written
		serviceConn.Close()
	}()

	select {
	case err := <-readErr:
		return read, written, errors.Wrap(err, "read error")
	case err := <-writeErr:
		return read, written, errors.Wrap(err, "write error")
	case <-ctx.Done():
		return read, written, nil
	}
}

// getAuthSigners finds the SSH keys that are configured for this tunnel and structure them for use by the SSH client library
func (t NormalTunnel) getAuthSigners(ctx context.Context) ([]ssh.Signer, error) {
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

		t.logger().WithField("fingerprint", ssh.FingerprintSHA256(signer.PublicKey())).Debug("using ssh key")
		signers[i] = signer
	}

	return signers, nil
}

func (t NormalTunnel) GetConnectionDetails(discovery discovery.DiscoveryService) (ConnectionDetails, error) {
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
}

func InjectNormalTunnelDependencies(f func(ctx context.Context) ([]NormalTunnel, error), services NormalTunnelServices, options SSHClientOptions) ListFunc {
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

func (t NormalTunnel) Equal(v interface{}) bool {
	t2, ok := v.(NormalTunnel)
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
func normalTunnelFromSQL(record postgres.NormalTunnel) NormalTunnel {
	return NormalTunnel{
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

func (t NormalTunnel) GetID() uuid.UUID {
	return t.ID
}

func (t NormalTunnel) logger() *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		"tunnel_type": Normal,
		"tunnel_id":   t.ID.String(),
	})
}
