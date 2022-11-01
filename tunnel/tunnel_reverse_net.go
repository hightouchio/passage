package tunnel

import (
	"crypto/tls"
	"fmt"
	"github.com/gliderlabs/ssh"
	"github.com/hightouchio/passage/stats"
	"github.com/pkg/errors"
	gossh "golang.org/x/crypto/ssh"
	"io"
	"net"
	"strconv"
	"sync"
)

type ReverseForwardingHandler struct {
	tlsConfig *tls.Config
	stats     stats.Stats
	lifecycle Lifecycle
	upstream  *reverseTunnelUpstream
	sync.Mutex
}

type reverseTunnelUpstream struct {
	conns map[string]*gossh.ServerConn
	mux   sync.RWMutex
}

// Register an SSH connection as a forwarding target for an upstream address
func (r *reverseTunnelUpstream) Register(addr string, conn *gossh.ServerConn) {
	r.mux.Lock()
	defer r.mux.Unlock()
	if r.conns == nil {
		r.conns = make(map[string]*gossh.ServerConn)
	}
	r.conns[addr] = conn
}

// Deregister the reverse tunnel upstream for an address
func (r *reverseTunnelUpstream) Deregister(addr string) {
	r.mux.Lock()
	defer r.mux.Unlock()
	conn, ok := r.conns[addr]
	if !ok {
		return
	}
	delete(r.conns, addr)
	conn.Close()
}

func (r *reverseTunnelUpstream) Close() {
	r.mux.Lock()
	defer r.mux.Unlock()
	for addr, conn := range r.conns {
		conn.Close()
		delete(r.conns, addr)
	}
}

func (r *reverseTunnelUpstream) Dial(downstream net.Conn, addr string) (io.ReadWriteCloser, error) {
	r.mux.RLock()
	defer r.mux.RUnlock()

	// Find the corresponding SSH connection to use to forward this connection
	conn, ok := r.conns[addr]
	if !ok {
		return nil, fmt.Errorf("forwarding not configured for addr %s", addr)
	}

	// Address of requested upstream connection.
	destAddr, destPortStr, _ := net.SplitHostPort(addr)
	destPort, _ := strconv.Atoi(destPortStr)

	// Address of tunnel client.
	originAddr, originPortStr, _ := net.SplitHostPort(downstream.RemoteAddr().String())
	originPort, _ := strconv.Atoi(originPortStr)

	// Construct port forwarding response
	payload := remoteForwardChannelData{
		// Tunnel listener address
		DestAddr: destAddr,
		DestPort: uint32(destPort),

		// Tunnel client address
		OriginAddr: originAddr,
		OriginPort: uint32(originPort),
	}

	// Open SSH channel.
	ch, reqs, err := conn.OpenChannel(forwardedTCPChannelType, gossh.Marshal(payload))
	if err != nil {
		return nil, errors.Wrap(err, "could not open channel")
	}
	// Discard all other incoming requests.
	go gossh.DiscardRequests(reqs)

	return ch, nil
}

func (h *ReverseForwardingHandler) HandleSSHRequest(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (bool, []byte) {
	conn := ctx.Value(ssh.ContextKeyConn).(*gossh.ServerConn)

	switch req.Type {
	// Handle a remote forwarding request
	case sshTCPForwardOpenEvent:
		// Parse remote forward request
		var reqPayload remoteForwardRequest
		if err := gossh.Unmarshal(req.Payload, &reqPayload); err != nil {
			// TODO: log parse failure
			return false, []byte{}
		}

		// Validate that the requested bind port is allowed for this tunnel.
		if srv.ReversePortForwardingCallback == nil || !srv.ReversePortForwardingCallback(ctx, reqPayload.BindAddr, reqPayload.BindPort) {
			return false, []byte("port forwarding is disabled")
		}

		bindAddr := net.JoinHostPort(reqPayload.BindAddr, strconv.Itoa(int(reqPayload.BindPort)))

		// Register this connection as available for forwarding to the given BindAddr
		h.upstream.Register(bindAddr, conn)

		return true, gossh.Marshal(&remoteForwardSuccess{reqPayload.BindPort})

	// Handle a remote forwarding close request
	case sshTCPForwardCloseEvent:
		var reqPayload remoteForwardCancelRequest
		if err := gossh.Unmarshal(req.Payload, &reqPayload); err != nil {
			return false, []byte{}
		}
		h.upstream.Deregister(net.JoinHostPort(reqPayload.BindAddr, strconv.Itoa(int(reqPayload.BindPort))))
		return true, nil

	default:
		return false, nil
	}
}

const (
	sshTCPForwardOpenEvent  = "tcpip-forward"
	sshTCPForwardCloseEvent = "cancel-tcpip-forward"
	forwardedTCPChannelType = "forwarded-tcpip"
)

type remoteForwardRequest struct {
	BindAddr string
	BindPort uint32
}

type remoteForwardSuccess struct {
	BindPort uint32
}

type remoteForwardCancelRequest struct {
	BindAddr string
	BindPort uint32
}

type remoteForwardChannelData struct {
	DestAddr   string
	DestPort   uint32
	OriginAddr string
	OriginPort uint32
}
