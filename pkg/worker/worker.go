package worker

import (
	"context"
	"github.com/hightouchio/passage/tunnel"
	"time"

	"github.com/apex/log"
	"github.com/hightouchio/passage/pkg/ssh"
	"github.com/hightouchio/passage/pkg/tunnels"
)

type Worker struct {
	disableNormal   bool
	disableReverse  bool
	tunnels         *tunnels.Tunnels
	reverseTunnels  *tunnels.ReverseTunnels
	pollingDuration time.Duration
	sshManager      *ssh.Manager
}

func NewWorker(
	disableNormal bool,
	disableReverse bool,
	tunnels *tunnels.Tunnels,
	reverseTunnels *tunnels.ReverseTunnels,
	bindHost string,
	hostKey []byte,
	user string,
	pollingDuration time.Duration,
) *Worker {
	return &Worker{
		disableNormal:   disableNormal,
		disableReverse:  disableReverse,
		tunnels:         tunnels,
		reverseTunnels:  reverseTunnels,
		pollingDuration: pollingDuration,
		sshManager:      ssh.NewManager(bindHost, hostKey, user),
	}
}

func (w *Worker) Start() {
	go w.start()
}

func (w *Worker) start() {
	ticker := time.NewTicker(w.pollingDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			w.refresh(ctx)
			cancel()
		}
	}
}

func (w *Worker) refresh(ctx context.Context) {
	var err error

	var tunnels []tunnel.NormalTunnel
	if !w.disableNormal {
		tunnels, err = w.tunnels.List(ctx)
		if err != nil {
			log.WithError(err).Error("list tunnel")
			return
		}
	}

	var reverseTunnels []tunnel.ReverseTunnel
	if !w.disableReverse {
		reverseTunnels, err = w.reverseTunnels.List(ctx)
		if err != nil {
			log.WithError(err).Error("list reverse tunnel")
			return
		}
	}

	w.sshManager.SetTunnels(tunnels, reverseTunnels)
}
