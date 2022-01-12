package consul

import (
	"fmt"
	"github.com/google/uuid"
	consulAPI "github.com/hashicorp/consul/api"
	"github.com/hightouchio/passage/tunnel/discovery"
)

type Discovery struct {
	Consul      *consulAPI.Client
	ServiceName string
	ServiceTags []string
}

//func NewConsulDiscovery(addr string) (Discovery, error) {
//	client, err := consulAPI.NewClient(&consulAPI.Config{
//		Address: addr,
//	})
//	if err != nil {
//		return Discovery{}, errors.Wrap(err, "could not init client")
//	}
//	return Discovery{client}, nil
//}

func (c Discovery) RegisterTunnel(tunnelID uuid.UUID, connectionDetails discovery.LocalConnectionDetails) error {
	serviceId := formatTunnelServiceId(tunnelID)
	return c.Consul.Agent().ServiceRegister(&consulAPI.AgentServiceRegistration{
		Name: c.ServiceName,
		ID:   serviceId,
		Tags: []string{
			fmt.Sprintf("tunnel_id:%s", tunnelID.String()),
		},

		Meta: map[string]string{
			"tunnel_id": tunnelID.String(),
		},

		// Tell Consul how to connect to this service
		Port:       connectionDetails.Port,
		SocketPath: connectionDetails.SocketPath,

		// Configure proxy sidecar
		Connect: &consulAPI.AgentServiceConnect{
			SidecarService: &consulAPI.AgentServiceRegistration{},
		},
	})
}

func (c Discovery) DeregisterTunnel(tunnelID uuid.UUID) error {
	return c.Consul.Agent().ServiceDeregister(formatTunnelServiceId(tunnelID))
}

func (c Discovery) ResolveTunnelHost(tunnelType string, tunnelID uuid.UUID) (string, error) {
	panic("implement me")
}

func formatTunnelServiceId(tunnelID uuid.UUID) string {
	return fmt.Sprintf("tunnel-%s", tunnelID.String())
}
