package tunnel

import (
	"context"
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

	services standardTunnelServices
}

// standardTunnelServices are the external dependencies that StandardTunnel needs to do its job
type standardTunnelServices struct {
	sql interface {
		GetStandardTunnelPrivateKeys(ctx context.Context, tunnelID uuid.UUID) ([]postgres.Key, error)
	}

	keystore keystore.Keystore
}

func isContextCancelled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

const sshDialTimeout = 15 * time.Second

func (t StandardTunnel) Start(ctx context.Context, options SSHOptions) error {
	st := stats.GetStats(ctx)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// generate our authentication strategy
	auth, err := t.generateAuthMethod(ctx)
	if err != nil {
		return errors.Wrap(err, "generate auth method")
	}

	// connect to remote SSH server
	addr := net.JoinHostPort(t.SSHHost, strconv.Itoa(t.SSHPort))
	st.WithEventTags(stats.Tags{"sshUser": t.SSHUser, "sshHost": t.SSHHost, "sshPort": t.SSHPort}).SimpleEvent("ssh.dial")
	sshConn, err := net.DialTimeout("tcp", addr, sshDialTimeout)
	if err != nil {
		return errors.Wrap(err, "dial remote")
	}
	defer func() {
		sshConn.Close()
	}()

	// init ssh connection & client
	c, chans, reqs, err := ssh.NewClientConn(
		sshConn, addr,
		&ssh.ClientConfig{
			User:            options.User,
			Auth:            auth,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		},
	)
	if err != nil {
		return errors.Wrap(err, "init ssh")
	}
	sshClient := ssh.NewClient(c, chans, reqs)

	// start keepalive handler
	keepaliveErr := make(chan error)
	go sshKeepalive(ctx, sshClient, sshConn, keepaliveErr)

	// open tunnel listener
	st.WithEventTags(stats.Tags{"tunnelPort": t.TunnelPort}).SimpleEvent("listener.start")
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", options.BindHost, t.TunnelPort))
	if err != nil {
		return errors.Wrap(err, "listen")
	}
	defer func() {
		listener.Close()
	}()

	// accept incoming conns and serve them up
	incomingConns := make(chan net.Conn)
	go func() {
		for {
			select {
			default:
				conn, err := listener.Accept()
				if err != nil && !isContextCancelled(ctx) {
					t.logger().WithError(err).Error("tunnel connection accept error")
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
		case tunnelConn := <-incomingConns:
			go func() {
				st := st.WithEventTags(stats.Tags{"remoteAddr": tunnelConn.RemoteAddr().String()}).WithPrefix("conn")
				ctx := stats.InjectContext(ctx, st)

				st.SimpleEvent("accept")
				st.Incr("accept", nil, 1)

				atomic.AddInt32(&activeConnections, 1)
				read, written, err := t.handleTunnelConnection(ctx, sshClient, tunnelConn)
				atomic.AddInt32(&activeConnections, -1)
				st.Gauge("bytesRead", float64(read), nil, 1)
				st.Gauge("bytesWritten", float64(written), nil, 1)
				st = st.WithEventTags(stats.Tags{"bytesRead": read, "bytesWritten": written})

				if err != nil {
					st.ErrorEvent("error", err)
					tunnelConn.Write([]byte(errors.Wrap(err, conncheckErrorPrefix).Error()))
					return
				}
				st.SimpleEvent("close")
			}()

		case <-statsTicker.C:
			// explicit tunnelId tag here, so it appears on the metric
			st.WithTags(stats.Tags{"tunnelId": t.ID.String()}).Gauge("activeConnections", float64(atomic.LoadInt32(&activeConnections)), nil, 1)

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

	// connect to upstream service
	st.WithEventTags(stats.Tags{"host": t.ServiceHost, "port": t.ServicePort}).SimpleEvent("upstream.dial")
	serviceConn, err := sshConn.Dial("tcp", fmt.Sprintf("%s:%d", t.ServiceHost, t.ServicePort))
	if err != nil {
		return 0, 0, errors.Wrap(err, "dial upstream service")
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
		go copy(g, tunnelConn, serviceConn, &written)
		go copy(g, serviceConn, tunnelConn, &read)
		g.Wait()
	}()

	// wait for an error or connection completion
	select {
	case <-ctx.Done():
		return read, written, nil
	}
}

// generateAuthMethod finds the SSH private keys that are configured for this tunnel and structure them for use by the SSH client library
func (t StandardTunnel) generateAuthMethod(ctx context.Context) ([]ssh.AuthMethod, error) {
	// get private keys from database
	keys, err := t.services.sql.GetStandardTunnelPrivateKeys(ctx, t.ID)
	if err != nil {
		return []ssh.AuthMethod{}, errors.Wrap(err, "could not look up private keys")
	}
	authMethods := make([]ssh.AuthMethod, len(keys))

	// parse private keys and prepare for SSH
	for i, key := range keys {
		contents, err := t.services.keystore.Get(ctx, key.ID)
		if err != nil {
			return []ssh.AuthMethod{}, errors.Wrapf(err, "could not get contents for key %s", key.ID)
		}
		signer, err := ssh.ParsePrivateKey([]byte(contents))
		if err != nil {
			return []ssh.AuthMethod{}, errors.Wrapf(err, "could not parse key %s", key.ID)
		}
		authMethods[i] = ssh.PublicKeys(signer)
	}

	return authMethods, nil
}

const keepaliveInterval = 1 * time.Minute
const keepaliveTimeout = 15 * time.Second

func sshKeepalive(ctx context.Context, client *ssh.Client, conn net.Conn, errChan chan<- error) {
	t := time.NewTicker(keepaliveInterval)
	defer t.Stop()

	err := func() error {
		for {
			deadline := time.Now().Add(keepaliveInterval).Add(keepaliveTimeout)
			err := conn.SetDeadline(deadline)
			if err != nil {
				return errors.Wrap(err, "set deadline")
			}
			select {
			case <-t.C:
				_, _, err = client.SendRequest("keepalive@passage.hightouch.io", true, nil)
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

// createStandardTunnelListFunc wraps our Postgres list function in something that converts the records into Standard structs so they can be passed to Manager which accepts the Tunnel interface
func createStandardTunnelListFunc(postgresList func(ctx context.Context) ([]postgres.StandardTunnel, error), services standardTunnelServices) ListFunc {
	return func(ctx context.Context) ([]Tunnel, error) {
		standardTunnels, err := postgresList(ctx)
		if err != nil {
			return []Tunnel{}, err
		}

		// convert all the SQL records to our primary struct
		tunnels := make([]Tunnel, len(standardTunnels))
		for i, record := range standardTunnels {
			tunnel := standardTunnelFromSQL(record)
			tunnel.services = services // inject dependencies
			tunnels[i] = tunnel
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
		SSHUser:     tunnel.SSHUser,
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
		SSHUser:     record.SSHUser,
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
