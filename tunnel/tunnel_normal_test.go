package tunnel

//import (
//	"context"
//	"github.com/DataDog/datadog-go/statsd"
//	"github.com/gliderlabs/ssh"
//	"github.com/google/uuid"
//	"github.com/hightouchio/passage/stats"
//	keystoreInMemory "github.com/hightouchio/passage/tunnel/keystore/in_memory"
//	"github.com/hightouchio/passage/tunnel/postgres"
//	"github.com/pkg/errors"
//	"github.com/sirupsen/logrus"
//	"net"
//	"strconv"
//	"sync"
//	"testing"
//	"time"
//)
//
//const (
//	bindHost = "0.0.0.0"
//)
//
//func TestNormalTunnel_Basic(t *testing.T) {
//	runNormalTunnelTest(t,
//		[]testInstruction{
//			writeLine("hello world"),
//			writeLine("whats up"),
//			readAndAssertLine("world hello"),
//		},
//		[]testInstruction{
//			readAndAssertLine("hello world"),
//			writeLine("world hello"),
//			readAndAssertLine("whats up"),
//		},
//	)
//}
//
//func runNormalTunnelTest(t *testing.T, clientInstructions, serviceInstructions []testInstruction) {
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	tunnelPort := getFreePort()
//	sshdPort := getFreePort()
//	servicePort := getFreePort()
//
//	tunnelClientConn := make(chan net.Conn)
//	tunnelServiceConn := make(chan net.Conn)
//
//	// Start upstream service
//	go func() {
//		listener, err := net.Listen("tcp", net.JoinHostPort(bindHost, strconv.Itoa(servicePort)))
//		if err != nil {
//			logrus.Error(errors.Wrap(err, "could not listen to service port"))
//			return
//		}
//		defer listener.Close()
//		conn, err := listener.Accept()
//		if err != nil {
//			logrus.Error(errors.Wrap(err, "could not accept service conn"))
//			return
//		}
//		logrus.Debug("service: tunneled connection accepted")
//		tunnelServiceConn <- conn
//	}()
//
//	// Start client sshd server
//	go func() {
//		sshServerAddr := net.JoinHostPort(bindHost, strconv.Itoa(sshdPort))
//		sshServer := &ssh.Server{
//			Addr: sshServerAddr,
//			Handler: ssh.Handler(func(s ssh.Session) {
//				<-ctx.Done()
//			}),
//			ChannelHandlers: map[string]ssh.ChannelHandler{
//				"direct-tcpip": ssh.DirectTCPIPHandler,
//			},
//			LocalPortForwardingCallback: ssh.LocalPortForwardingCallback(func(ctx ssh.Context, dhost string, dport uint32) bool {
//				return true
//			}),
//		}
//		logrus.Debug("sshd: start listen")
//		if err := sshServer.ListenAndServe(); err != nil {
//			logrus.Error(errors.Wrap(err, "could not open test sshd server"))
//		}
//	}()
//
//	// Start tunnel server
//	go func() {
//		logger := logrus.New()
//		logger.SetLevel(logrus.DebugLevel)
//		st := stats.New(&statsd.NoOpClient{})
//
//		ctx = stats.InjectContext(ctx, st)
//
//		keystore := keystoreInMemory.New()
//		database := MockNormalDatabase{}
//
//		tunnel := &NormalTunnel{
//			ID:          uuid.New(),
//			SSHHost:     "localhost",
//			SSHPort:     sshdPort,
//			SSHUser:     "test",
//			ServiceHost: "localhost",
//			ServicePort: servicePort,
//			services: NormalTunnelServices{
//				Keystore: keystore,
//				SQL:      database,
//			},
//			clientOptions: SSHClientOptions{
//				User:              "test",
//				KeepaliveInterval: 60 * time.Second,
//			},
//		}
//
//		listener, err := newEphemeralTCPListener("0.0.0.0")
//		if err != nil {
//			t.Error(errors.Wrap(err, "open listener"))
//			return
//		}
//
//		logrus.Debug("passage: start tunnel")
//		if err := tunnel.Start(ctx, listener, make(chan<- StatusUpdate)); err != nil {
//			t.Error(errors.Wrap(err, "server failed to start"))
//			return
//		}
//	}()
//
//	go func() {
//		time.Sleep(2 * time.Second)
//		conn, err := net.Dial("tcp", net.JoinHostPort("localhost", strconv.Itoa(tunnelPort)))
//		if err != nil {
//			logrus.Error(errors.Wrap(err, "could not connect to tunnel port"))
//		}
//		logrus.Debug("client: open conn")
//		tunnelClientConn <- conn
//	}()
//
//	wg := sync.WaitGroup{}
//	wg.Add(2)
//	go newTunneledConn(<-tunnelClientConn).runAssertions(t, &wg, clientInstructions)
//	go newTunneledConn(<-tunnelServiceConn).runAssertions(t, &wg, serviceInstructions)
//	wg.Wait()
//}
//
//type MockNormalDatabase struct {
//}
//
//func (d MockNormalDatabase) GetNormalTunnelPrivateKeys(ctx context.Context, tunnelID uuid.UUID) ([]postgres.Key, error) {
//	return []postgres.Key{}, nil
//}
