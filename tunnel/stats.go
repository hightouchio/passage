package tunnel

import (
	"context"
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
