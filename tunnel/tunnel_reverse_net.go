package tunnel

import (
	"github.com/gliderlabs/ssh"
	"github.com/pkg/errors"
	gossh "golang.org/x/crypto/ssh"
	"net"
	"strconv"
)

type ReverseForwardingHandler struct {
	GetTunnel func(bindPort int) (SSHServerRegisteredTunnel, bool)
}

type ReverseForwardingConnection struct {
	ssh.Context

	Dial  func(originAddr string) (gossh.Channel, error)
	Close func() error
}

func (h *ReverseForwardingHandler) HandleSSHRequest(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (bool, []byte) {
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
func (h *ReverseForwardingHandler) openPortForwarding(ctx ssh.Context, payload remoteForwardOpenRequest) (bool, []byte) {
	tunnel, ok := h.GetTunnel(int(payload.BindPort))
	if !ok {
		// Couldn't find a valid registered tunnel for this request
		return false, []byte("Port forwarding is disabled")
	}

	// Get a reference to the connection
	conn := ctx.Value(ssh.ContextKeyConn).(*gossh.ServerConn)

	// We've validated the connection and the port forwarding request.
	//	Pass this off to the tunnel to re-establish the connection.
	tunnel.Connections <- ReverseForwardingConnection{
		Context: ctx,

		// Dial exposes an interface to make upstream connections through this SSH tunnel
		Dial: func(originAddr string) (gossh.Channel, error) {
			originHost, originPortStr, _ := net.SplitHostPort(originAddr)
			originPort, _ := strconv.Atoi(originPortStr)

			// Open an upstream connection through a new `forwarded-tcpip` channel
			ch, reqs, err := conn.OpenChannel(forwardedTCPChannelType, gossh.Marshal(remoteForwardChannelData{
				// Pass along the origin address of this connection
				// TODO: Is this necessary?
				OriginAddr: originHost,
				OriginPort: uint32(originPort),

				// We should initiate an upstream connection to the port that was bound in this forwarding request.
				DestAddr: payload.BindAddr,
				DestPort: payload.BindPort,
			}))
			if err != nil {
				return nil, errors.Wrap(err, "could not open forwarding channel")
			}
			go gossh.DiscardRequests(reqs)

			return ch, nil
		},

		Close: func() error {
			return conn.Close()
		},
	}
	return true, gossh.Marshal(&remoteForwardSuccess{payload.BindPort})
}

// closePortForwarding handles a request from the SSH client to close port forwarding
func (h *ReverseForwardingHandler) closePortForwarding(ctx ssh.Context, payload remoteForwardCancelRequest) (bool, []byte) {
	// Get a reference to the connection
	conn := ctx.Value(ssh.ContextKeyConn).(*gossh.ServerConn)

	// We could just cancel the specific forwarder that was opened (indicated by BindAddr and BindPort on `payload`),
	//	but today we only support one forwarder per connection, so we should just close the connection
	conn.Close()

	return true, nil
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
