package postgres

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type TracedClient struct {
	Client
	tracer trace.Tracer
}

func WithTracing(client Client, tracer trace.Tracer) TracedClient {
	return TracedClient{
		Client: client,
		tracer: tracer,
	}
}

func (c TracedClient) CreateReverseTunnel(ctx context.Context, data ReverseTunnel, authorizedKeys []uuid.UUID) (ReverseTunnel, error) {
	ctx, span := c.startSpan(ctx, "CreateReverseTunnel")
	defer span.End()

	return c.Client.CreateReverseTunnel(ctx, data, authorizedKeys)
}

func (c TracedClient) GetReverseTunnel(ctx context.Context, id uuid.UUID) (ReverseTunnel, error) {
	ctx, span := c.startSpan(ctx, "GetReverseTunnel", id.String())
	defer span.End()
	return c.Client.GetReverseTunnel(ctx, id)
}

func (c TracedClient) UpdateReverseTunnel(ctx context.Context, id uuid.UUID, data map[string]interface{}) (ReverseTunnel, error) {
	ctx, span := c.startSpan(ctx, "UpdateReverseTunnel", id.String())
	defer span.End()
	return c.Client.UpdateReverseTunnel(ctx, id, data)
}

func (c TracedClient) ListReverseActiveTunnels(ctx context.Context) ([]ReverseTunnel, error) {
	ctx, span := c.startSpan(ctx, "ListActiveReverseTunnels")
	defer span.End()
	return c.Client.ListReverseActiveTunnels(ctx)
}

func (c TracedClient) CreateNormalTunnel(ctx context.Context, data NormalTunnel) (NormalTunnel, error) {
	ctx, span := c.startSpan(ctx, "CreateNormalTunnel")
	defer span.End()
	return c.Client.CreateNormalTunnel(ctx, data)
}

func (c TracedClient) GetNormalTunnel(ctx context.Context, id uuid.UUID) (NormalTunnel, error) {
	ctx, span := c.startSpan(ctx, "GetNormalTunnel", id.String())
	defer span.End()
	return c.Client.GetNormalTunnel(ctx, id)
}

func (c TracedClient) UpdateNormalTunnel(ctx context.Context, id uuid.UUID, data map[string]interface{}) (NormalTunnel, error) {
	ctx, span := c.startSpan(ctx, "UpdateNormalTunnel", id.String())
	defer span.End()
	return c.Client.UpdateNormalTunnel(ctx, id, data)
}

func (c TracedClient) ListNormalActiveTunnels(ctx context.Context) ([]NormalTunnel, error) {
	ctx, span := c.startSpan(ctx, "ListNormalActiveTunnels")
	defer span.End()
	return c.Client.ListNormalActiveTunnels(ctx)
}

func (c TracedClient) DeleteTunnel(ctx context.Context, tunnelID uuid.UUID) error {
	ctx, span := c.startSpan(ctx, "DeleteTunnel", tunnelID.String())
	defer span.End()
	return c.Client.DeleteTunnel(ctx, tunnelID)
}

func (c TracedClient) AuthorizeKeyForTunnel(ctx context.Context, tunnelType string, tunnelID uuid.UUID, keyID uuid.UUID) error {
	ctx, span := c.startSpan(ctx, "AuthorizeKeyForTunnel", tunnelID.String())
	defer span.End()
	return c.Client.AuthorizeKeyForTunnel(ctx, tunnelType, tunnelID, keyID)
}

func (c TracedClient) startSpan(ctx context.Context, name string, opts ...string) (context.Context, trace.Span) {
	var attributes []attribute.KeyValue
	switch len(opts) {
	case 1:
		attributes = append(attributes, attribute.String("tunnel.id", opts[0]))
	case 2:
		attributes = append(attributes,
			attribute.String("tunnel.id", opts[0]),
		)
	}

	return c.tracer.Start(ctx, fmt.Sprintf("Postgres/%s", name),
		trace.WithAttributes(attributes...),
	)
}
