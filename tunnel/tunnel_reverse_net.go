package tunnel

import (
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
	httpProxyEnabled bool
	stats            stats.Stats
	lifecycle        Lifecycle
	forwards         map[string]*TCPForwarder
	sync.Mutex
}

func (h *ReverseForwardingHandler) HandleSSHRequest(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (bool, []byte) {
	h.Lock()
	if h.forwards == nil {
		h.forwards = make(map[string]*TCPForwarder)
	}
	h.Unlock()
	conn := ctx.Value(ssh.ContextKeyConn).(*gossh.ServerConn)

	switch req.Type {
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

		// Initiate TCPForwarder to listen for tunnel connections.
		forwarder := &TCPForwarder{
			BindAddr:         bindAddr,
			HTTPProxyEnabled: h.httpProxyEnabled,

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

		// Start tunnel listener
		if err := forwarder.Listen(); err != nil {
			switch err.(type) {
			case bootError:
				h.lifecycle.BootError(err)
			default:
				h.lifecycle.Error(err)
			}

			return false, []byte("bind address already in use")
		}

		h.Lock()
		h.forwards[bindAddr] = forwarder
		h.Unlock()

		// Start port forwarding
		go forwarder.Serve()

		// Graceful shutdown if connection ends
		go func() {
			<-ctx.Done()
			h.Lock()
			f, ok := h.forwards[bindAddr]
			h.Unlock()
			if ok {
				f.Close()
			}
		}()

		return true, gossh.Marshal(&remoteForwardSuccess{reqPayload.BindPort})

	case sshTCPForwardCloseEvent:
		var reqPayload remoteForwardCancelRequest
		if err := gossh.Unmarshal(req.Payload, &reqPayload); err != nil {
			return false, []byte{}
		}

		addr := net.JoinHostPort(reqPayload.BindAddr, strconv.Itoa(int(reqPayload.BindPort)))
		h.Lock()
		ln, ok := h.forwards[addr]
		h.Unlock()
		if ok {
			ln.Close()
		}
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
