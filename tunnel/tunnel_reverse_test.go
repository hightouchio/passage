package tunnel

import (
	"context"
	"fmt"
	"github.com/DataDog/datadog-go/statsd"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/stats"
	keystoreInMemory "github.com/hightouchio/passage/tunnel/keystore/in_memory"
	"github.com/hightouchio/passage/tunnel/postgres"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"net"
	"sync"
	"testing"
	"time"
)

func TestReverseTunnel_Basic(t *testing.T) {
	runReverseTunnelTest(t,
		[]testInstruction{
			writeLine("hello world"),
			writeLine("whats up"),
			readAndAssertLine("world hello"),
		},
		[]testInstruction{
			readAndAssertLine("hello world"),
			writeLine("world hello"),
			readAndAssertLine("whats up"),
		},
	)
}

const (
	bindHost = "0.0.0.0"
)

func runReverseTunnelTest(t *testing.T, clientInstructions, serviceInstructions []testInstruction) {
	logrus.SetLevel(logrus.DebugLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sshdPort := getFreePort()
	tunnelPort := getFreePort()

	// Server
	go func() {
		logger := logrus.New()
		logger.SetLevel(logrus.DebugLevel)
		st := stats.New(&statsd.NoOpClient{}, logger)

		ctx = stats.InjectContext(ctx, st)
		ctx = injectCtxLifecycle(ctx, lifecycleLogger{st})

		keystore := keystoreInMemory.New()
		database := MockReverseDatabase{}

		tunnel := &ReverseTunnel{
			ID:         uuid.New(),
			SSHDPort:   sshdPort,
			TunnelPort: tunnelPort,
			services: ReverseTunnelServices{
				Keystore: keystore,
				SQL:      database,
				Logger:   logrus.New(),
			},
			serverOptions: SSHServerOptions{
				BindHost: bindHost,
			},
		}
		if err := tunnel.Start(ctx, TunnelOptions{BindHost: bindHost}); err != nil {
			t.Error(errors.Wrap(err, "server failed to start"))
			return
		}
	}()

	listenerOpened := make(chan bool)

	tunnelClientConn := make(chan net.Conn)
	tunnelServiceConn := make(chan net.Conn)

	// Port forwarding client.
	go func() {
		time.Sleep(1 * time.Second)

		// Dial SSH server.
		client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", bindHost, sshdPort), &ssh.ClientConfig{
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			User:            "test",
			Auth:            []ssh.AuthMethod{},
		})
		if err != nil {
			logrus.Error(errors.Wrap(err, "could not open client"))
			return
		}
		defer client.Close()

		// Open port forwarding.
		listener, err := client.Listen("tcp", fmt.Sprintf("%s:%d", bindHost, tunnelPort))
		if err != nil {
			logrus.Error(errors.Wrap(err, "could not open listener"))
			return
		}
		defer listener.Close()
		listenerOpened <- true

		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Accept conn.
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				tunnelServiceConn <- conn
			}
		}
	}()

	// Forwarding initiator (simulate tunnel client)
	go func() {
		<-listenerOpened
		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", bindHost, tunnelPort))
		if err != nil {
			logrus.Error(errors.Wrap(err, "could not open tunnel port"))
			return
		}
		tunnelClientConn <- conn
	}()

	wg := sync.WaitGroup{}
	wg.Add(2)

	go newTunneledConn(<-tunnelClientConn).runAssertions(t, &wg, clientInstructions)
	go newTunneledConn(<-tunnelServiceConn).runAssertions(t, &wg, serviceInstructions)

	wg.Wait()
}

type MockReverseDatabase struct {
}

func (d MockReverseDatabase) GetReverseTunnelAuthorizedKeys(ctx context.Context, tunnelID uuid.UUID) ([]postgres.Key, error) {
	return []postgres.Key{}, nil
}
