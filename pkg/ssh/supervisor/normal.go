package supervisor

import (
	"time"

	"github.com/apex/log"
	"github.com/hightouchio/passage/pkg/models"
)

const normalSupervisorRetryDuration = time.Second

type NormalSupervisor struct {
	tunnel models.Tunnel
}

func NewNormalSupervisor(tunnel models.Tunnel) *NormalSupervisor {
	return &NormalSupervisor{
		tunnel: tunnel,
	}
}

func (s *NormalSupervisor) Start() {
	go s.start()
}

func (s *NormalSupervisor) start() {
	ticker := time.NewTicker(normalSupervisorRetryDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.startSSHClient(); err != nil {
				log.Error("start ssh client")
			}
		}
	}
}

func (s *NormalSupervisor) startSSHClient() error {
	return nil
}
