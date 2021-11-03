package srv

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"net"
)

type Discovery struct {
	SrvRegistry string
	Prefix      string
}

func (d Discovery) ResolveTunnelHost(tunnelType string, _ uuid.UUID) (string, error) {
	_, targets, err := net.LookupSRV(fmt.Sprintf("%s_%s", d.Prefix, tunnelType), "tcp", d.SrvRegistry)
	if err != nil {
		return "", errors.Wrap(err, "could not resolve SRV")
	}
	if len(targets) == 0 {
		return "", errors.New("no targets found")
	}
	return targets[0].Target, nil
}