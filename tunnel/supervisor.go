package tunnel

import (
	"context"
	"github.com/sirupsen/logrus"
	"time"
)

type Supervisor struct {
	Tunnel     Tunnel
	SSHOptions SSHOptions

	RetryDuration time.Duration
}

func (s Supervisor) Start() {
	ticker := time.NewTicker(s.RetryDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx := context.Background()

			if err := s.Tunnel.Start(ctx, s.SSHOptions); err != nil {
				logrus.WithError(err).Error("could not start server")
			}
		}
	}
}
