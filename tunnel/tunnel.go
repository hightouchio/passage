package tunnel

import "context"

type Tunnel interface {
	GetID() int
	Start(context.Context, SSHOptions) error
}

type SSHOptions struct {
	BindHost string
	HostKey  []byte

	User string
}
