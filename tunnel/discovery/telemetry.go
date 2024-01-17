package discovery

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type TracedService struct {
	Service
	tracer trace.Tracer
}

func WithTracing(service Service, tracer trace.Tracer) Service {
	return TracedService{
		Service: service,
		tracer:  tracer,
	}
}

func (s TracedService) RegisterTunnel(ctx context.Context, id uuid.UUID, port int) error {
	ctx, span := s.startSpan(ctx, "RegisterTunnel", id.String())
	defer span.End()
	return s.Service.RegisterTunnel(ctx, id, port)
}

func (s TracedService) DeregisterTunnel(ctx context.Context, id uuid.UUID) error {
	ctx, span := s.startSpan(ctx, "DeregisterTunnel", id.String())
	defer span.End()
	return s.Service.DeregisterTunnel(ctx, id)
}

func (s TracedService) GetTunnel(ctx context.Context, id uuid.UUID) (TunnelDetails, error) {
	ctx, span := s.startSpan(ctx, "GetTunnel", id.String())
	defer span.End()
	return s.Service.GetTunnel(ctx, id)
}

func (s TracedService) RegisterHealthcheck(ctx context.Context, tunnelId uuid.UUID, options HealthcheckOptions) error {
	ctx, span := s.startSpan(ctx, "RegisterHealthcheck", tunnelId.String(), options.ID)
	defer span.End()
	return s.Service.RegisterHealthcheck(ctx, tunnelId, options)
}

func (s TracedService) DeregisterHealthcheck(ctx context.Context, tunnelId uuid.UUID, id string) error {
	ctx, span := s.startSpan(ctx, "DeregisterHealthcheck", tunnelId.String(), id)
	defer span.End()
	return s.Service.DeregisterHealthcheck(ctx, tunnelId, id)
}

func (s TracedService) UpdateHealthcheck(ctx context.Context, tunnelId uuid.UUID, id string, status HealthcheckStatus, message string) error {
	ctx, span := s.startSpan(ctx, "UpdateHealthcheck", tunnelId.String(), id)
	defer span.End()
	return s.Service.UpdateHealthcheck(ctx, tunnelId, id, status, message)
}

func (s TracedService) startSpan(ctx context.Context, name string, opts ...string) (context.Context, trace.Span) {
	var attributes []attribute.KeyValue
	switch len(opts) {
	case 1:
		attributes = append(attributes, attribute.String("tunnel.id", opts[0]))
	case 2:
		attributes = append(attributes,
			attribute.String("tunnel.id", opts[0]),
			attribute.String("tunnel.healthcheck.id", opts[1]),
		)
	}

	return s.tracer.Start(ctx, fmt.Sprintf("Discovery/%s", name),
		trace.WithAttributes(attributes...),
	)
}
