package tunnel

import "context"

type Server struct {
}

type NewReverseTunnelRequest struct {}
type NewReverseTunnelResponse struct {}

func (s Server) NewReverseTunnel(ctx context.Context, request NewReverseTunnelRequest) (*NewReverseTunnelResponse, error) {
	return nil, nil
}
