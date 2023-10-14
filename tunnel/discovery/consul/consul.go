package consul

import (
	"fmt"
	"github.com/google/uuid"
	consul "github.com/hashicorp/consul/api"
	"github.com/hightouchio/passage/tunnel/discovery"
	"github.com/pkg/errors"
	"time"
)

type Discovery struct {
	HostAddress string
	Consul      *consul.Client

	HealthcheckTTL time.Duration
}

func (d Discovery) RegisterTunnel(id uuid.UUID, port int) error {
	serviceId := getTunnelServiceId(id)
	err := d.Consul.Agent().ServiceRegister(&consul.AgentServiceRegistration{
		ID:   serviceId,
		Name: serviceId,

		Kind:    consul.ServiceKindTypical,
		Address: d.HostAddress,
		Port:    port,
		Tags:    []string{fmt.Sprintf("tunnel_id:%s", id.String())},

		Check: &consul.AgentServiceCheck{
			CheckID: getTunnelHealthcheckId(id),
			Name:    "Tunnel Healthcheck",
			TTL:     fmt.Sprintf("%ds", int(d.HealthcheckTTL.Seconds())),

			// Default to the Critical status before the first healthcheck is processed.
			Status: discovery.TunnelUnhealthy,
		},
	})
	if err != nil {
		return errors.Wrapf(err, "could not register tunnel %s", id.String())
	}

	return nil
}

func (d Discovery) DeregisterTunnel(id uuid.UUID) error {
	if err := d.Consul.Agent().ServiceDeregister(getTunnelServiceId(id)); err != nil {
		return errors.Wrapf(err, "could not deregister tunnel %s", id.String())
	}
	return nil
}

func (d Discovery) GetTunnel(id uuid.UUID) (discovery.TunnelDetails, error) {
	services, _, err := d.Consul.Health().Service(getTunnelServiceId(id), "", false, nil)
	if err != nil {
		return discovery.TunnelDetails{}, errors.Wrap(err, "could not get tunnel details")
	}

	if len(services) != 1 {
		return discovery.TunnelDetails{}, fmt.Errorf("expected 1 service, got %d", len(services))
	}
	service := services[0]

	var check *consul.HealthCheck
	for _, c := range service.Checks {
		if c.CheckID == getTunnelHealthcheckId(id) {
			check = c
			break
		}
	}
	if check == nil {
		return discovery.TunnelDetails{}, errors.Wrapf(err, "could not find healthcheck for tunnel %s", id.String())
	}

	return discovery.TunnelDetails{
		Host:         service.Service.Address,
		Port:         service.Service.Port,
		Status:       check.Status,
		StatusReason: check.Output,
	}, nil
}

func (d Discovery) UpdateHealth(id uuid.UUID, status, message string) error {
	if err := d.Consul.Agent().UpdateTTL(getTunnelHealthcheckId(id), message, status); err != nil {
		return errors.Wrap(err, "could not mark tunnel unhealthy")
	}
	return nil
}

func getTunnelServiceId(id uuid.UUID) string {
	return fmt.Sprintf("tunnel:%s", id.String())
}

func getTunnelHealthcheckId(id uuid.UUID) string {
	return fmt.Sprintf("%s:check_in", getTunnelServiceId(id))
}
