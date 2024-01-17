package tunnel

import (
	"context"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/tunnel/proto"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"
)

type GrpcServer struct {
	API    API
	Logger *log.Logger

	// Embed
	proto.UnimplementedPassageServer
}

func (g GrpcServer) GetTunnel(ctx context.Context, req *proto.GetTunnelRequest) (*proto.Tunnel, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse tunnel ID")
	}

	response, err := g.API.GetTunnel(ctx, GetTunnelRequest{ID: id})
	if err != nil {
		return nil, errors.Wrap(err, "could not get tunnel")
	}

	// Render the response as a protobuf tunnel
	return response.ToProtoTunnel(), nil
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

func (g GrpcServer) DeleteTunnel(ctx context.Context, req *proto.DeleteTunnelRequest) (*emptypb.Empty, error) {
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
