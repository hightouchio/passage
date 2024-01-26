package tunnel

import (
	"testing"
	"time"
)

func Test_reportStats(t *testing.T) {
	ticker := make(chan time.Time)

	var payloads *[]ConnectionStatsPayload

	go reportStats(func() []ConnectionStatsPayload {
		return *payloads
	}, ticker)

	payloads = &[]ConnectionStatsPayload{
		{
			ClientBytesSent:     100,
			ClientBytesReceived: 100,
		},
	}

	ticker <- time.Now()

	payloads = &[]ConnectionStatsPayload{
		{
			ClientBytesSent:     200,
			ClientBytesReceived: 100,
		},

		{
			ClientBytesSent:     500,
			ClientBytesReceived: 500,
		},
	}

	ticker <- time.Now()
}
