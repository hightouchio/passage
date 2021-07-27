package tunnel

import (
	"context"
	"fmt"
	"github.com/hightouchio/passage/stats"
	"io"
	"net"
	"os"
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

	services normalTunnelServices
}

// normalTunnelServices are the external dependencies that NormalTunnel needs to do its job
type normalTunnelServices struct {
	sql interface {
		GetNormalTunnelPrivateKeys(ctx context.Context, tunnelID uuid.UUID) ([]postgres.Key, error)
	}
}

func isContextCancelled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

func (t NormalTunnel) Start(ctx context.Context, options SSHOptions) error {
	st := stats.GetStats(ctx)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// generate our authentication strategy
	auth, err := t.generateAuthMethod(ctx)
	if err != nil {
		return errors.Wrap(err, "generate auth method")
	}

	// connect to remote SSH server
	st.WithEventTags(stats.Tags{"sshUser": t.SSHUser, "sshHost": t.SSHHost, "sshPort": t.SSHPort}).SimpleEvent("ssh.dial")
	sshConn, err := ssh.Dial(
		"tcp", fmt.Sprintf("%s:%d", t.SSHHost, t.SSHPort),
		&ssh.ClientConfig{
			User:            t.SSHUser,
			Auth:            auth,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		},
	)
	if err != nil {
		return errors.Wrap(err, "dial ssh")
	}
	defer func() {
		sshConn.Close()
	}()

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
				read, written, err := t.handleTunnelConnection(ctx, sshConn, tunnelConn)
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

		case <-ctx.Done():
			return nil
		}
	}
}

func (t NormalTunnel) handleTunnelConnection(ctx context.Context, sshConn *ssh.Client, tunnelConn net.Conn) (int64, int64, error) {
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
func (t NormalTunnel) generateAuthMethod(ctx context.Context) ([]ssh.AuthMethod, error) {
	// get private keys from database
	keys, err := t.services.sql.GetNormalTunnelPrivateKeys(ctx, t.ID)
	if err != nil {
		return []ssh.AuthMethod{}, errors.Wrap(err, "could not get keys from db")
	}

	// parse private keys and prepare for SSH
	authMethods := make([]ssh.AuthMethod, len(keys))
	for i, key := range keys {
		signer, err := ssh.ParsePrivateKey([]byte(key.Contents))
		if err != nil {
			return []ssh.AuthMethod{}, errors.Wrapf(err, "could not parse key %d", key.ID)
		}
		authMethods[i] = ssh.PublicKeys(signer)
	}

	return authMethods, nil
}

func (t NormalTunnel) GetConnectionDetails() (ConnectionDetails, error) {
	_, targets, err := net.LookupSRV("passage_normal", "tcp", os.Getenv("PASSAGE_SRV_REGISTRY"))
	if err != nil {
		return ConnectionDetails{}, errors.Wrap(err, "could not resolve SRV")
	}
	if len(targets) == 0 {
		return ConnectionDetails{}, errors.New("no targets found")
	}

	return ConnectionDetails{
		Host: targets[0].Target,
		Port: t.TunnelPort,
	}, nil
}

// createNormalTunnelListFunc wraps our Postgres list function in something that converts the records into Normal structs so they can be passed to Manager which accepts the Tunnel interface
func createNormalTunnelListFunc(postgresList func(ctx context.Context) ([]postgres.NormalTunnel, error), services normalTunnelServices) ListFunc {
	return func(ctx context.Context) ([]Tunnel, error) {
		normalTunnels, err := postgresList(ctx)
		if err != nil {
			return []Tunnel{}, err
		}

		// convert all the SQL records to our primary struct
		tunnels := make([]Tunnel, len(normalTunnels))
		for i, record := range normalTunnels {
			tunnel := normalTunnelFromSQL(record)
			tunnel.services = services // inject dependencies
			tunnels[i] = tunnel
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
		SSHUser:     tunnel.SSHUser,
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
		SSHUser:     record.SSHUser,
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
		"tunnel_type": "normal",
		"tunnel_id":   t.ID.String(),
	})
}
