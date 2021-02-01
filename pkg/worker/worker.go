package worker

import (
	"context"
	"time"

	"github.com/apex/log"
	"github.com/hightouchio/passage/pkg/ssh"
	"github.com/hightouchio/passage/pkg/tunnels"
)

type Worker struct {
	tunnels         *tunnels.Tunnels
	sshManager      *ssh.Manager
	pollingDuration time.Duration
}

func NewWorker(
	tunnels *tunnels.Tunnels,
	pollingDuration time.Duration,
) *Worker {
	return &Worker{
		tunnels:         tunnels,
		pollingDuration: pollingDuration,
		sshManager:      ssh.NewManager(),
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
	tunnels, err := w.tunnels.List(ctx)
	if err != nil {
		log.WithError(err).Error("list tunnels")
		return
	}
	w.sshManager.SetTunnels(tunnels)
}
