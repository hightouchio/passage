package tunnel

import (
	"context"
	"github.com/google/uuid"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/tunnel/proto"
	"github.com/pkg/errors"
)

type GrpcServer struct {
	API    *API
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
