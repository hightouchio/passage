package tunnel

import (
	"context"
	"github.com/google/uuid"
)

type Tunnel interface {
	GetID() uuid.UUID
	Start(context.Context, SSHOptions) error
}

type SSHOptions struct {
	BindHost string
	HostKey  []byte
}
