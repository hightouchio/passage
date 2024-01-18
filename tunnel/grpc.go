package tunnel

import (
	"context"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/tunnel/proto"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/emptypb"
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
		Tunnel: response.ToProtoTunnel(),
		Instances: []*proto.TunnelInstance{
			{
				Host:   "localhost",
				Port:   5432,
				Status: proto.TunnelHealthcheck_WARNING,
				Healthchecks: []*proto.TunnelHealthcheck{
					{
						Id:      "test",
						Status:  proto.TunnelHealthcheck_CRITICAL,
						Message: "Hello world",
					},
				},
			},
		},
	}, nil
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
