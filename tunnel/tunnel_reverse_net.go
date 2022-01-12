package tunnel

import (
	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
)

// ForwardedTCPHandler can be enabled by creating a ForwardedTCPHandler and
// adding the HandleSSHRequest callback to the server's RequestHandlers under
// tcpip-forward and cancel-tcpip-forward.
//
// From github.com/gliderlabs/ssh
type ForwardedTCPHandler struct {
	forwards map[string]net.Listener
	sync.Mutex
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
	h.Lock()
	if h.forwards == nil {
		h.forwards = make(map[string]net.Listener)
	}
	h.Unlock()

	switch req.Type {
	case "tcpip-forward":
		// Parse remote forward request
		var reqPayload remoteForwardRequest
		if err := gossh.Unmarshal(req.Payload, &reqPayload); err != nil {
			// TODO: log parse failure
			return false, []byte{}
		}
		if srv.ReversePortForwardingCallback == nil || !srv.ReversePortForwardingCallback(ctx, reqPayload.BindAddr, reqPayload.BindPort) {
			return false, []byte("port forwarding is disabled")
		}

		addr := net.JoinHostPort(reqPayload.BindAddr, strconv.Itoa(int(reqPayload.BindPort)))
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			// TODO: log listen failure
			return false, []byte{}
		}

		_, destPortStr, _ := net.SplitHostPort(ln.Addr().String())
		destPort, _ := strconv.Atoi(destPortStr)

		// Register forward
		h.Lock()
		h.forwards[addr] = ln
		h.Unlock()

		// Wait for channel closure to derregister
		go func() {
			<-ctx.Done()
			h.Lock()
			ln, ok := h.forwards[addr]
			h.Unlock()
			if ok {
				ln.Close()
			}
		}()

		go func() {
			for {
				c, err := ln.Accept()
				if err != nil {
					// TODO: log accept failure
					break
				}

				originAddr, orignPortStr, _ := net.SplitHostPort(c.RemoteAddr().String())
				originPort, _ := strconv.Atoi(orignPortStr)
				payload := gossh.Marshal(&remoteForwardChannelData{
					DestAddr:   reqPayload.BindAddr,
					DestPort:   uint32(destPort),
					OriginAddr: originAddr,
					OriginPort: uint32(originPort),
				})

				go func() {
					ch, reqs, err := conn.OpenChannel(forwardedTCPChannelType, payload)
					if err != nil {
						// TODO: log failure to open channel
						log.Println(err)
						c.Close()
						return
					}
					go gossh.DiscardRequests(reqs)

					go func() {
						defer ch.Close()
						defer c.Close()
						io.Copy(ch, c)
					}()

					go func() {
						defer ch.Close()
						defer c.Close()
						io.Copy(c, ch)
					}()
				}()
			}

			// Clean out forward
			h.Lock()
			delete(h.forwards, addr)
			h.Unlock()
		}()
		return true, gossh.Marshal(&remoteForwardSuccess{uint32(destPort)})

	case "cancel-tcpip-forward":
		var reqPayload remoteForwardCancelRequest
		if err := gossh.Unmarshal(req.Payload, &reqPayload); err != nil {
			// TODO: log parse failure
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
