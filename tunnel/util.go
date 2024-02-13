package tunnel

import (
	"context"
	"fmt"
	"github.com/google/uuid"
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

// FilterTunnelsByIds wraps a ListFunc to restrict the returned tunnels by the set of enabled IDs
func FilterTunnelsByIds(fn ListFunc, mode string, filterList []uuid.UUID) (ListFunc, error) {
	if mode != "whitelist" && mode != "blacklist" {
		return nil, fmt.Errorf("invalid filter mode: %s", mode)
	}

	// Populate a map for constant time lookup
	lookup := make(map[uuid.UUID]bool)
	for _, id := range filterList {
		lookup[id] = true
	}

	// Generate a new ListFunc that filters the results of the original
	//	based on if the tunnel ID is in the set of enabled IDs
	return func(ctx context.Context) ([]Tunnel, error) {
		// Get the original set of enabled tunnels
		tunnels, err := fn(ctx)
		if err != nil {
			return []Tunnel{}, err
		}

		// Only return the tunnels that match the set of enabled IDs

		filtered := make([]Tunnel, 0)
		for _, tunnel := range tunnels {
			switch mode {
			case "whitelist":
				// In blacklist mode, if the tunnel is in the set of enabled IDs, add it to the list
				if lookup[tunnel.GetID()] {
					filtered = append(filtered, tunnel)
				}

			case "blacklist":
				// In blacklist mode, if the tunnel is not in the set of enabled IDs, add it to the list
				if !lookup[tunnel.GetID()] {
					filtered = append(filtered, tunnel)
				}
			}
		}

		return filtered, nil
	}, nil
}
