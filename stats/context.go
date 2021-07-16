package stats

import (
	"context"
)

const contextStatsKey = "_stats"

func InjectContext(ctx context.Context, stats Stats) context.Context {
	return context.WithValue(ctx, contextStatsKey, stats)
}

func GetStats(ctx context.Context) Stats {
	entry, ok := ctx.Value(contextStatsKey).(Stats)
	if !ok {
		panic("no stats in context")
	}
	return entry
}
