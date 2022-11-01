package tunnel

import (
	"crypto/tls"
	consul "github.com/hashicorp/consul/api"
	"github.com/phayes/freeport"
	"github.com/pkg/errors"
	"net"
	"strconv"
	"time"
)

//// Register tunnel with Consul
//serviceId := fmt.Sprintf("tunnel_%s", s.Tunnel.GetID())
//err := s.Consul.Agent().ServiceRegister(&consul.AgentServiceRegistration{
//Name: serviceId,
//Tags: []string{s.Tunnel.GetID().String()},
//Port: 57500,
//Connect: &consul.AgentServiceConnect{
//Native: true,
//},
//})
//if err != nil {
//lifecycle.BootError(errors.Wrap(err, "could not register consul service"))
//continue
//}
//
//// Register service with Consul connect
//service, err := connect.NewService(serviceId, s.Consul)
//if err != nil {
//lifecycle.BootError(errors.Wrap(err, "could not register connect service"))
//continue
//}

//// Wait for Consul service to become ready
//<-service.ReadyWait()

type TLSListener struct {
	BindHost          string
	Consul            *consul.Client
	TLSConfig         *tls.Config
	KeepaliveInterval time.Duration

	addr  net.Addr
	conns chan net.Conn
}

func (l *TLSListener) Addr() net.Addr {
	return l.addr
}

func (l *TLSListener) Start() error {
	if l.conns == nil {
		l.conns = make(chan net.Conn, 0)
	}

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
	listener, err := tls.Listen("tcp", l.addr.String(), nil)
	if err != nil {
		return bootError{event: "listener_start", err: err}
	}

	// Accept connections and forward to chan
	for {
		conn, err := listener.Accept()
		if err != nil {
			return errors.Wrap(err, "could not accept")
		}

		// Configure keepalive for tunnel clients
		tcpConn := conn.(*net.TCPConn)
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(l.KeepaliveInterval)

		// Forward the original TLS conn on to consumers
		l.conns <- conn
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
func (l *TLSListener) Accept() (net.Conn, error) {
	return <-l.conns, nil
}

func (l *TLSListener) Close() error {
	return nil
}
