package tunnel

import (
	"context"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/tunnel/discovery"
	"github.com/hightouchio/passage/tunnel/proto"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/emptypb"
	"slices"
)

type GrpcServer struct {
	API    API
	Logger *log.Logger

	// Embed
	proto.UnimplementedPassageServer
}

func (g GrpcServer) CreateStandardTunnel(ctx context.Context, req *proto.CreateStandardTunnelRequest) (*proto.Tunnel, error) {
	var sshUser string
	if req.SshUser != nil {
		sshUser = *req.SshUser
	}

	response, err := g.API.CreateNormalTunnel(ctx, CreateNormalTunnelRequest{
		NormalTunnel: NormalTunnel{
			SSHHost:     req.SshHost,
			SSHPort:     int(req.SshPort),
			SSHUser:     sshUser,
			ServiceHost: req.ServiceHost,
			ServicePort: int(req.ServicePort),
		},
		CreateKeyPair: false,
		Keys:          nil,
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not create tunnel")
	}

	return response.ToProtoTunnel(), nil
}

func (g GrpcServer) CreateReverseTunnel(ctx context.Context, req *proto.CreateReverseTunnelRequest) (*proto.Tunnel, error) {
	keyUuids := make([]uuid.UUID, len(req.PublicKeys))
	for i, key := range req.PublicKeys {
		id, err := uuid.Parse(key)
		if err != nil {
			return nil, errors.Wrapf(err, "could not parse key ID %s", key)
		}
		keyUuids[i] = id
	}

	response, err := g.API.CreateReverseTunnel(ctx, CreateReverseTunnelRequest{
		Keys:          keyUuids,
		CreateKeyPair: req.CreateKeyPair,
	})

	if err != nil {
		return nil, errors.Wrap(err, "could not create tunnel")
	}

	return response.ToProtoTunnel(), nil
}

func (g GrpcServer) GetTunnel(ctx context.Context, req *proto.GetTunnelRequest) (*proto.GetTunnelResponse, error) {
	trace.SpanFromContext(ctx).SetAttributes(attribute.String("tunnel.id", req.Id))

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse tunnel ID")
	}

	response, err := g.API.GetTunnel(ctx, GetTunnelRequest{ID: id})
	if err != nil {
		return nil, errors.Wrap(err, "could not get tunnel")
	}

	return &proto.GetTunnelResponse{
		Tunnel:    response.ToProtoTunnel(),
		Instances: formatTunnelInstances(response.Instances),
	}, nil
}

func formatTunnelInstances(instances []discovery.TunnelInstance) []*proto.TunnelInstance {
	convertStatus := func(status discovery.HealthcheckStatus) proto.TunnelHealthcheck_Status {
		switch status {
		case discovery.HealthcheckPassing:
			return proto.TunnelHealthcheck_PASSING
		case discovery.HealthcheckWarning:
			return proto.TunnelHealthcheck_WARNING
		case discovery.HealthcheckCritical:
			return proto.TunnelHealthcheck_CRITICAL
		}
		return proto.TunnelHealthcheck_CRITICAL
	}

	protoInstances := make([]*proto.TunnelInstance, len(instances))
	for i, instance := range instances {
		healthchecks := make([]*proto.TunnelHealthcheck, len(instance.Checks))
		for i, check := range instance.Checks {
			healthchecks[i] = &proto.TunnelHealthcheck{
				Id:      check.ID,
				Status:  convertStatus(check.Status),
				Message: check.Message,
			}
		}

		protoInstances[i] = &proto.TunnelInstance{
			Host:         instance.Host,
			Port:         instance.Port,
			Status:       convertStatus(instance.Status),
			Healthchecks: healthchecks,
		}
	}

	// Sort instances by status. Best candidate instances should come first.
	slices.SortFunc(protoInstances, func(a, b *proto.TunnelInstance) int {
		if a.Status == proto.TunnelHealthcheck_PASSING && b.Status != proto.TunnelHealthcheck_PASSING {
			return -1
		} else if a.Status != proto.TunnelHealthcheck_PASSING && b.Status == proto.TunnelHealthcheck_PASSING {
			return 1
		} else {
			return 0
		}
	})

	return protoInstances
}

func (g GrpcServer) DeleteTunnel(ctx context.Context, req *proto.DeleteTunnelRequest) (*emptypb.Empty, error) {
	trace.SpanFromContext(ctx).SetAttributes(attribute.String("tunnel.id", req.Id))

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse tunnel ID")
	}
	_, err = g.API.DeleteTunnel(ctx, DeleteTunnelRequest{
		ID: id,
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not delete tunnel")
	}
	return &emptypb.Empty{}, nil
}
