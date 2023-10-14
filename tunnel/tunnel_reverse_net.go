package tunnel

import (
	"context"
	"github.com/gliderlabs/ssh"
	"github.com/hightouchio/passage/tunnel/discovery"
	"github.com/pkg/errors"
	gossh "golang.org/x/crypto/ssh"
	"io"
	"net"
	"strconv"
	"sync"
)

type ReverseForwardingHandler struct {
	httpProxyEnabled bool
	forwards         map[string]*TCPForwarder
	sync.Mutex

	GetTunnel func(bindPort int) (boundReverseTunnel, bool)
}

func (h *ReverseForwardingHandler) HandleSSHRequest(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (bool, []byte) {
	h.Lock()
	if h.forwards == nil {
		h.forwards = make(map[string]*TCPForwarder)
	}
	h.Unlock()

	switch req.Type {
	// Handle a request from the SSH connection to open port forwarding.
	case sshTCPForwardOpenEvent:
		var payload remoteForwardOpenRequest
		if err := gossh.Unmarshal(req.Payload, &payload); err != nil {
			return false, []byte{}
		}

		// Validate that the requested bind port is allowed for this tunnel.
		if srv.ReversePortForwardingCallback == nil || !srv.ReversePortForwardingCallback(ctx, payload.BindAddr, payload.BindPort) {
			return false, []byte("Port forwarding is disabled")
		}

		return h.openPortForwarding(ctx, payload)

	// Handle a request from the SSH connection to close port forwarding.
	case sshTCPForwardCloseEvent:
		var payload remoteForwardCancelRequest
		if err := gossh.Unmarshal(req.Payload, &payload); err != nil {
			return false, []byte{}
		}
		return h.closePortForwarding(ctx, payload)

	default:
		return false, nil
	}
}

// closePortForwarding handles a request from the SSH client to open port forwarding
func (h *ReverseForwardingHandler) openPortForwarding(ctx context.Context, payload remoteForwardOpenRequest) (bool, []byte) {
	tunnel, ok := h.GetTunnel(int(payload.BindPort))
	if !ok {
		// Couldn't find a valid registered tunnel for this request
		return false, []byte("Port forwarding is disabled")
	}

	// Get the connection out of the context
	conn := ctx.Value(ssh.ContextKeyConn).(*gossh.ServerConn)
	tunnelBindAddr := net.JoinHostPort(payload.BindAddr, strconv.Itoa(int(payload.BindPort)))

	// Initiate TCPForwarder to listen for tunnel connections.
	forwarder := &TCPForwarder{
		BindAddr:         tunnelBindAddr,
		HTTPProxyEnabled: h.httpProxyEnabled,

		// Implement GetUpstreamConn by opening a channel on the SSH connection.
		GetUpstreamConn: func(tConn net.Conn) (io.ReadWriteCloser, error) {
			// Address of tunnel client.
			originAddr, originPortStr, _ := net.SplitHostPort(tConn.RemoteAddr().String())
			originPort, _ := strconv.Atoi(originPortStr)

			// Construct port forwarding response
			payload := remoteForwardChannelData{
				// Tunnel listener address
				DestAddr: payload.BindAddr,
				DestPort: payload.BindPort,

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
		},

		Lifecycle: tunnel.lifecycle,
		Stats:     tunnel.stats,
	}

	// Start tunnel listener
	if err := forwarder.Listen(); err != nil {
		switch err.(type) {
		case bootError:
			tunnel.lifecycle.BootError(err)
		default:
			tunnel.lifecycle.Error(err)
		}

		return false, []byte("Port forwarding disabled")
	}

	h.Lock()
	h.forwards[tunnelBindAddr] = forwarder
	h.Unlock()

	// Start port forwarding
	go forwarder.Serve()

	// Mark tunnel healthy
	tunnel.discovery.UpdateHealth(tunnel.id, discovery.TunnelHealthy, "Tunnel is online")

	// Graceful shutdown if connection ends
	go func() {
		<-ctx.Done()
		h.closeTunnel(tunnelBindAddr)
	}()

	return true, gossh.Marshal(&remoteForwardSuccess{payload.BindPort})
}

// closePortForwarding handles a request from the SSH client to close port forwarding
func (h *ReverseForwardingHandler) closePortForwarding(ctx context.Context, payload remoteForwardCancelRequest) (bool, []byte) {
	addr := net.JoinHostPort(payload.BindAddr, strconv.Itoa(int(payload.BindPort)))
	h.closeTunnel(addr)

	// Get the tunnel to deregister it from the discovery service
	tunnel, ok := h.GetTunnel(int(payload.BindPort))
	if ok {
		tunnel.discovery.UpdateHealth(tunnel.id, discovery.TunnelUnhealthy, "Tunnel is offline")
	}

	return true, nil
}

func (h *ReverseForwardingHandler) closeTunnel(addr string) {
	h.Lock()
	ln, ok := h.forwards[addr]
	h.Unlock()
	if ok {
		ln.Close()
	}
}

const (
	sshTCPForwardOpenEvent  = "tcpip-forward"
	sshTCPForwardCloseEvent = "cancel-tcpip-forward"
	forwardedTCPChannelType = "forwarded-tcpip"
)

type remoteForwardOpenRequest struct {
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
