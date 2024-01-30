package tunnel

import (
	"context"
	"time"
)

func runOnceAndTick(ctx context.Context, interval time.Duration, fn func()) {
	fn()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fn()
		}
	}
}
