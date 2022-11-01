package tunnel

import (
	"io"
	"net"
)

type Upstream interface {
	Dial(downstream net.Conn, addr string) (io.ReadWriteCloser, error)
}
