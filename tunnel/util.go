package tunnel

import (
	"context"
	"time"
)

// retry the given function until it succeeds
func retry(ctx context.Context, interval time.Duration, fn func() error) error {
	for {
		select {
		case <-ctx.Done():
			return nil

		default:
			if err := fn(); err != nil {
				time.Sleep(interval)
				continue
			}

			return nil
		}
	}
}

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
