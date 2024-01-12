package tunnel

import (
	"context"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/tunnel/proto"
)

type GrpcServer struct {
	// Embed
	proto.UnimplementedPassageServer
}

func (g GrpcServer) GetTunnel(ctx context.Context, req *proto.GetTunnelRequest) (*proto.Tunnel, error) {
	log.Get().Named("gRPC").Infow("GetTunnel", "id", req.Id)
	return &proto.Tunnel{
		Id:       req.Id,
		Type:     proto.Tunnel_REVERSE,
		Enabled:  true,
		BindPort: 94983,
		Tunnel: &proto.Tunnel_StandardTunnel_{
			StandardTunnel: &proto.Tunnel_StandardTunnel{
				SshHost:     "localhost",
				SshPort:     1234,
				ServiceHost: "localhost",
				ServicePort: "4567",
			},
		},
	}, nil
}
