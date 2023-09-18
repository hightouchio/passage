package dns

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"net"
)

type Discovery struct {
	// HostnameNormal is the DNS name that resolves to the normal tunnel host
	HostNormal string
	// HostnameReverse is the DNS name that resolves to the reverse tunnel host
	HostReverse string
}

func (d Discovery) ResolveTunnelHost(tunnelType string, tunnelID uuid.UUID) (string, error) {
	lookup := func(name string) (string, error) {
		addrs, err := net.LookupAddr(name)
		if err != nil {
			return "", errors.Wrap(err, "could not resolve addr")
		}
		// Always return the first address
		return addrs[0], nil
	}

	switch tunnelType {
	case "normal":
		return lookup(d.HostNormal)
	case "reverse":
		return lookup(d.HostReverse)
	default:
		return "", fmt.Errorf("invalid tunnel type: %s", tunnelType)
	}
}
