package tunnel

import (
	"context"
	"github.com/hightouchio/passage/log"
	"go.uber.org/zap"
	"io"
	"net"
	"strconv"
	"time"
)

// newEphemeralTCPListener returns a TCP listener bound to a random, unused port
func newEphemeralTCPListener(ip string) (*net.TCPListener, error) {
	return net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.ParseIP(ip),
		Port: 0,
	})
}

func portFromNetAddr(addr net.Addr) int {
	_, portStr, _ := net.SplitHostPort(addr.String())
	port, _ := strconv.Atoi(portStr)
	return port
}

// TODO: Support a maximum number of retries.
func testUpstreamConnection(
	ctx context.Context,
	logger *log.Logger,
	statusUpdate chan<- StatusUpdate,
	getUpstream func() (io.ReadWriteCloser, error),
) error {
	logger.Debug("Test upstream connection")

	if err := retry(ctx, 5*time.Second, func() error {
		if _, err := getUpstream(); err != nil {
			statusUpdate <- StatusUpdate{StatusError, err.Error()}
			logger.Errorw("Upstream dial failed", zap.Error(err))
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	logger.Debug("Upstream connection successful")
	return nil
}
