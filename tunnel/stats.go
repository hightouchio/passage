package tunnel

import (
	"context"
	"github.com/hightouchio/passage/stats"
	"time"
)

const (
	StatTunnelCount = "passage.tunnel.count"

	StatTunnelClientActiveConnectionCount = "passage.tunnel.client.connection_count"
	StatTunnelClientBytesSent             = "passage.tunnel.client.bytes_sent"
	StatTunnelClientBytesReceived         = "passage.tunnel.client.bytes_rcvd"

	StatTunnelUpstreamBytesSent     = "passage.tunnel.upstream.bytes_sent"
	StatTunnelUpstreamBytesReceived = "passage.tunnel.upstream.bytes_rcvd"

	StatTunnelReverseForwardClientConnectionCount = "passage.tunnel.reverse.forward_client_connection_count"

	StatSshdConnectionsRequests          = "passage.sshd.connection_requests"
	StatSshReversePortForwardingRequests = "passage.sshd.forwarding_connection_requests"
)

type ConnectionStatsPayload struct {
	ClientBytesSent       uint64
	ClientBytesReceived   uint64
	UpstreamBytesSent     uint64
	UpstreamBytesReceived uint64
}

// reportTunnelConnectionStats reports the number of bytes read and written to the statsd client
func reportTunnelConnectionStats(st stats.Stats, payload ConnectionStatsPayload) {
	st.Count(StatTunnelClientBytesSent, int64(payload.ClientBytesSent), stats.Tags{}, 1)
	st.Count(StatTunnelClientBytesReceived, int64(payload.ClientBytesReceived), stats.Tags{}, 1)
	st.Count(StatTunnelUpstreamBytesSent, int64(payload.UpstreamBytesSent), stats.Tags{}, 1)
	st.Count(StatTunnelUpstreamBytesReceived, int64(payload.UpstreamBytesReceived), stats.Tags{}, 1)
}

// Standardized metric reporting interval
const metricReportInterval = 1 * time.Second

func intervalMetricReporter(ctx context.Context, fn func()) {
	ticker := time.NewTicker(metricReportInterval)
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
