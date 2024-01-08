package consul

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	consul "github.com/hashicorp/consul/api"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/tunnel/discovery"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"strings"
	"time"
)

type Discovery struct {
	HostAddress    string
	Consul         *consul.Client
	HealthcheckTTL time.Duration
	Log            *log.Logger
}

func NewConsulDiscovery(consul *consul.Client, hostAddress string, healthcheckTTL time.Duration) *Discovery {
	return &Discovery{
		HostAddress:    hostAddress,
		Consul:         consul,
		HealthcheckTTL: healthcheckTTL,
		Log:            log.Get().Named("Consul"),
	}
}

// Wait for discovery service to be ready
func (d Discovery) Wait(ctx context.Context) error {
	d.Log.Debug("Waiting for readiness")

	for {
		// If the context is cancelled, give up and return an error
		if ctx.Err() != nil {
			return ctx.Err()
		}

		_, err := d.Consul.Status().Leader()
		if err == nil {
			break
		}

		d.Log.Warnw("Error connecting", zap.Error(err))

		// Try again
		time.Sleep(5 * time.Second)
	}

	d.Log.Info("Ready")

	// No error means the service is ready
	return nil
}

func (d Discovery) RegisterTunnel(id uuid.UUID, port int) error {
	d.Log.With(zap.String("tunnel_id", id.String())).Debugf("Register tunnel %s", id.String())

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
	d.Log.With(zap.String("tunnel_id", id.String())).Debugf("Deregister tunnel %s", id.String())

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

	if len(services) == 0 {
		return discovery.TunnelDetails{}, fmt.Errorf("tunnel %s not found", id.String())
	}

	// Search for a healthy tunnel service instance and return it
	for _, service := range services {
		if service.Checks.AggregatedStatus() != consul.HealthPassing {
			continue
		}
		return formatTunnelDetails(id, service), nil
	}

	// If there are no healthy tunnel service instances, return the first
	return formatTunnelDetails(id, services[0]), nil
}

func formatTunnelDetails(tunnelId uuid.UUID, service *consul.ServiceEntry) discovery.TunnelDetails {
	response := discovery.TunnelDetails{
		Host:   service.Service.Address,
		Port:   service.Service.Port,
		Checks: make([]discovery.HealthcheckDetails, 0),
	}

	// All relevant custom checks must start with this prefix
	checkPrefix := getTunnelServiceId(tunnelId)
	for _, c := range service.Checks {
		// Only consider checks that have this prefix (this means that Passage manages them)
		if strings.HasPrefix(c.CheckID, checkPrefix) {
			response.Checks = append(response.Checks, discovery.HealthcheckDetails{
				ID:      strings.Replace(c.CheckID, checkPrefix, "", 1)[1:],
				Status:  c.Status,
				Message: c.Output,
			})
		}
	}

	return response
}

func (d Discovery) RegisterHealthcheck(tunnelId uuid.UUID, options discovery.HealthcheckOptions) error {
	consulCheckId := getTunnelHealthcheckId(tunnelId, options.ID)
	d.Log.With(
		zap.String("tunnel_id", tunnelId.String()),
		zap.Dict("check",
			zap.String("id", options.ID),
		),
	).Debugf("Register healthcheck: %s", consulCheckId)

	if err := d.Consul.Agent().CheckRegister(&consul.AgentCheckRegistration{
		ServiceID: getTunnelServiceId(tunnelId),
		ID:        consulCheckId,
		Name:      options.Name,

		AgentServiceCheck: consul.AgentServiceCheck{
			// Register a TTL check
			TTL: formatConsulDuration(options.TTL),

			// Default to the warning status
			Status: string(discovery.HealthcheckCritical),
		},
	}); err != nil {
		return errors.Wrapf(err, "could not register healthcheck %s for tunnel %s", options.ID, tunnelId.String())
	}
	return nil
}

func (d Discovery) DeregisterHealthcheck(tunnelId uuid.UUID, id string) error {
	consulCheckId := getTunnelHealthcheckId(tunnelId, id)
	d.Log.With(
		zap.String("tunnel_id", tunnelId.String()),
		zap.Dict("check",
			zap.String("id", id),
		),
	).Debugf("Deregister healthcheck: %s", consulCheckId)

	if err := d.Consul.Agent().CheckDeregister(consulCheckId); err != nil {
		return errors.Wrapf(err, "could not deregister healthcheck %s for tunnel %s", id, tunnelId.String())
	}
	return nil
}

func (d Discovery) UpdateHealthcheck(tunnelId uuid.UUID, id string, status discovery.HealthcheckStatus, message string) error {
	consulCheckId := getTunnelHealthcheckId(tunnelId, id)

	d.Log.With(
		zap.String("tunnel_id", tunnelId.String()),
		zap.Dict("check",
			zap.String("id", id),
			zap.String("status", string(status)),
			zap.String("message", message),
		),
	).Debugf("Healthcheck update: %s: %s (%s)", consulCheckId, status, message)

	if err := d.Consul.Agent().UpdateTTL(consulCheckId, message, string(status)); err != nil {
		return errors.Wrap(err, "could not mark tunnel unhealthy")
	}
	return nil
}

func getTunnelServiceId(id uuid.UUID) string {
	return fmt.Sprintf("tunnel-%s", id.String())
}

func getTunnelHealthcheckId(id uuid.UUID, checkId string) string {
	return fmt.Sprintf("%s-%s", getTunnelServiceId(id), checkId)
}

func formatConsulDuration(d time.Duration) string {
	return fmt.Sprintf("%ds", int(d.Seconds()))
}
