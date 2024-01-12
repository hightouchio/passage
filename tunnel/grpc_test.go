package tunnel

import (
	"context"
	"github.com/hightouchio/passage/tunnel/proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"testing"
)

func Test_grpc(t *testing.T) {
	conn, err := grpc.Dial("localhost:8081", grpc.WithInsecure())
	if err != nil {
		t.Fatal(errors.Wrap(err, "grpc dial"))
	}

	client := proto.NewPassageClient(conn)
	response, err := client.GetTunnel(context.Background(), &proto.GetTunnelRequest{
		Id: "hello world",
	})
	if err != nil {
		t.Fatal(errors.Wrap(err, "grpc get tunnel"))
	}

	t.Logf("%+v", response)
}
