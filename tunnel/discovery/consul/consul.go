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
		// TODO: Refactor this to support multiple healthchecks
		//	Prioritize self-reported status over connectivity check status
		if c.CheckID == getTunnelHealthcheckId(id, "check_in") {
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

func (d Discovery) RegisterHealthcheck(tunnelId uuid.UUID, options discovery.HealthcheckOptions) error {
	if err := d.Consul.Agent().CheckRegister(&consul.AgentCheckRegistration{
		ID:        getTunnelHealthcheckId(tunnelId, options.ID),
		ServiceID: getTunnelServiceId(tunnelId),
		Name:      options.Name,

		AgentServiceCheck: consul.AgentServiceCheck{
			// Register a TTL check
			TTL: formatConsulDuration(options.TTL),

			// Default to the warning status
			Status: string(discovery.HealthcheckWarning),
		},
	}); err != nil {
		return errors.Wrapf(err, "could not register healthcheck %s for tunnel %s", options.ID, tunnelId.String())
	}
	return nil
}

func (d Discovery) DeregisterHealthcheck(tunnelId uuid.UUID, id string) error {
	if err := d.Consul.Agent().CheckDeregister(getTunnelHealthcheckId(tunnelId, id)); err != nil {
		return errors.Wrapf(err, "could not deregister healthcheck %s for tunnel %s", id, tunnelId.String())
	}
	return nil
}

func (d Discovery) UpdateHealthcheck(tunnelId uuid.UUID, id string, status discovery.HealthcheckStatus, message string) error {
	if err := d.Consul.Agent().UpdateTTL(getTunnelHealthcheckId(tunnelId, id), message, string(status)); err != nil {
		return errors.Wrap(err, "could not mark tunnel unhealthy")
	}
	return nil
}

func getTunnelServiceId(id uuid.UUID) string {
	return fmt.Sprintf("tunnel:%s", id.String())
}

func getTunnelHealthcheckId(id uuid.UUID, checkId string) string {
	return fmt.Sprintf("%s:%s", getTunnelServiceId(id), checkId)
}

func formatConsulDuration(d time.Duration) string {
	return fmt.Sprintf("%ds", int(d.Seconds()))
}
