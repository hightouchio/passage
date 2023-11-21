package tunnel

import "fmt"

type StatusUpdateFn func(status Status, message string)

type Status string

const (
	StatusBooting Status = "booting"
	StatusOnline  Status = "healthy"
	StatusError   Status = "error"
)

func newTunnelStatusUpdater(tunnel Tunnel) StatusUpdateFn {
	return func(status Status, message string) {
		fmt.Printf("Tunnel(%s): %s - %s\n", tunnel.GetID(), status, message)
	}
}
