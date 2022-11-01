package tunnel

import (
	"context"
	"github.com/hightouchio/passage/stats"
	"github.com/phayes/freeport"
	"github.com/pkg/errors"
	"net"
	"strconv"
	"time"
)

type TCPListener struct {
	BindHost          string
	KeepaliveInterval time.Duration
	Lifecycle

	addr  net.Addr
	conns chan net.Conn
}

func (l *TCPListener) Addr() net.Addr {
	return l.addr
}

func (l *TCPListener) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Construct listen address
	port, err := freeport.GetFreePort()
	if err != nil {
		return errors.Wrap(err, "could not get free port")
	}
	addr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(l.BindHost, strconv.Itoa(port)))
	if err != nil {
		return errors.Wrap(err, "could not resolve bind address")
	}
	l.addr = addr

	// Open TLS listener
	l.Lifecycle.BootEvent("listener_start", stats.Tags{"bind_addr": addr.String()})
	listener, err := net.ListenTCP("tcp", addr)
	defer listener.Close()
	if err != nil {
		return bootError{event: "listener_start", err: err}
	}

	// Accept connections and forward to chan
	for {
		select {
		case <-ctx.Done():
			return nil

		default:
			conn, err := listener.AcceptTCP()
			if err != nil {
				return errors.Wrap(err, "could not accept")
			}

			// Configure keepalive for tunnel clients
			conn.SetKeepAlive(true)
			conn.SetKeepAlivePeriod(l.KeepaliveInterval)

			// Forward the original TCP conn on to consumers
			l.conns <- conn
		}
	}

	//f.listener = listener
	//f.Lifecycle.Open()

	// Wait for close signal
	//go func() {
	//	<-f.close
	//	f.listener.Close()
	//	f.Lifecycle.Close()
	//}()
}

// Accept gets the next available net.Conn
func (l *TCPListener) Accept() (net.Conn, error) {
	return <-l.conns, nil
}

func (l *TCPListener) Close() error {
	return nil
}
