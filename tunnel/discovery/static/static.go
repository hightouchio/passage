package static

import (
	"github.com/google/uuid"
)

// StaticDiscovery is a tunnel discovery provider that returns a static host
type Discovery struct {
	Host string
}

func (d Discovery) ResolveTunnelHost(_ string, _ uuid.UUID) (string, error) {
	return d.Host, nil
}
