package tunnel

import (
	"github.com/gliderlabs/ssh"
	"github.com/hightouchio/passage/stats"
	"github.com/pkg/errors"
	gossh "golang.org/x/crypto/ssh"
	"io"
	"net"
	"strconv"
)

// ForwardedTCPHandler can be enabled by creating a ForwardedTCPHandler and
// adding the HandleSSHRequest callback to the server's RequestHandlers under
// tcpip-forward and cancel-tcpip-forward.
//
// From github.com/gliderlabs/ssh
type ForwardedTCPHandler struct {
	forwarder *TCPForwarder

	lifecycle Lifecycle
	stats     stats.Stats
}

const (
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

func (h *ForwardedTCPHandler) HandleSSHRequest(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (bool, []byte) {
	conn := ctx.Value(ssh.ContextKeyConn).(*gossh.ServerConn)

	switch req.Type {
	case "tcpip-forward":
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

		// Initiate TCPForwarder to listen for tunnel connections.
		h.forwarder = &TCPForwarder{
			BindAddr: net.JoinHostPort(reqPayload.BindAddr, strconv.Itoa(int(reqPayload.BindPort))),

			// Implement GetUpstreamConn by opening a channel on the SSH connection.
			GetUpstreamConn: func(tConn net.Conn) (io.ReadWriteCloser, error) {
				// Address of tunnel client.
				originAddr, originPortStr, _ := net.SplitHostPort(tConn.RemoteAddr().String())
				originPort, _ := strconv.Atoi(originPortStr)

				// Construct port forwarding response
				payload := remoteForwardChannelData{
					// Tunnel listener address
					DestAddr: reqPayload.BindAddr,
					DestPort: reqPayload.BindPort,

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

			Lifecycle: h.lifecycle,
			Stats:     h.stats,
		}

		// Start the TCP Forwarder
		go func() {
			if err := h.forwarder.Listen(); err != nil {
				switch err.(type) {
				case bootError:
					h.lifecycle.BootError(err)
				default:
					h.lifecycle.Error(err)
				}
			}
		}()

		// Graceful shutdown if connection ends
		go func() {
			<-ctx.Done()
			h.forwarder.Close()
		}()

		return true, gossh.Marshal(&remoteForwardSuccess{reqPayload.BindPort})

	case "cancel-tcpip-forward":
		var reqPayload remoteForwardCancelRequest
		if err := gossh.Unmarshal(req.Payload, &reqPayload); err != nil {
			return false, []byte{}
		}
		//addr := net.JoinHostPort(reqPayload.BindAddr, strconv.Itoa(int(reqPayload.BindPort)))

		if h.forwarder != nil {
			h.forwarder.Close()
		}

		return true, nil

	default:
		return false, nil
	}
}
