package tunnel

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type fakeConnection struct {
	bytesWritten, bytesRead uint64
}

func (f *fakeConnection) GetBytesWritten() uint64 {
	return f.bytesWritten
}

func (f *fakeConnection) GetBytesRead() uint64 {
	return f.bytesRead
}

type fakeConnectionPair struct {
	client, server *fakeConnection
	tick           func()
}

func fakePair(ctx context.Context, deltas chan<- forwarderStatsPayload) fakeConnectionPair {
	client := &fakeConnection{}
	server := &fakeConnection{}

	tick := make(chan time.Time)
	go connectionStatProducer(ctx, client, server, deltas, tick)

	return fakeConnectionPair{
		client: client,
		server: server,
		tick: func() {
			tick <- time.Now()
		},
	}
}

func (p fakeConnectionPair) clientWrite(bytes uint64) {
	p.client.bytesWritten += bytes
	p.server.bytesRead += bytes
}

func (p fakeConnectionPair) serverWrite(bytes uint64) {
	p.client.bytesRead += bytes
	p.server.bytesWritten += bytes
}

func Test_StatReporter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	deltas := make(chan forwarderStatsPayload)

	// Create three connection pairs to simulate metric aggregation
	pair1 := fakePair(ctx, deltas)
	pair2 := fakePair(ctx, deltas)
	pair3 := fakePair(ctx, deltas)

	// Create a ticker to control the aggregator
	tickAggC := make(chan time.Time)
	tickAgg := func() {
		tickAggC <- time.Now()
	}

	// Consume the aggregated reports
	report := make(chan forwarderStatsPayload)
	go connectionStatAggregator(ctx, deltas, func(stats forwarderStatsPayload) {
		report <- stats
	}, tickAggC)

	// Total of 135 bytes written by the client, 175 bytes written by the server
	pair1.clientWrite(10)
	pair1.tick()

	pair2.serverWrite(50)
	pair2.tick()

	pair3.serverWrite(125)
	pair3.clientWrite(125)
	pair3.tick()

	// Sleep so the deltas are consumed
	time.Sleep(300 * time.Millisecond)

	// Tick the aggregator
	tickAgg()

	// Assert current state of the result
	assert.Equal(t, forwarderStatsPayload{
		ClientBytesSent:       135,
		ClientBytesReceived:   175,
		UpstreamBytesSent:     175,
		UpstreamBytesReceived: 135,
	}, <-report)

	pair2.clientWrite(350)
	pair2.serverWrite(250)
	pair2.tick()

	pair3.serverWrite(50)
	pair3.serverWrite(500)
	pair3.clientWrite(27)
	pair3.tick()

	// Sleep so the deltas are consumed
	time.Sleep(1000 * time.Millisecond)

	tickAgg()

	assert.Equal(t, forwarderStatsPayload{
		ClientBytesSent:       377,
		ClientBytesReceived:   800,
		UpstreamBytesSent:     800,
		UpstreamBytesReceived: 377,
	}, <-report)
}
