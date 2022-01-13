package tunnel

import (
	"context"
	"fmt"
	"github.com/DataDog/datadog-go/statsd"
	"github.com/hightouchio/passage/stats"
)

// Lifecycle provides callbacks for the tunnel to self-report status
type Lifecycle interface {
	// Start is called when a Tunnel begins booting up.
	Start()

	// BootEvent is called for logging purposes during relevant events in the Tunnel boot process.
	BootEvent(event string, tags stats.Tags)

	// BootError is called when an anomaly occurs while booting a Tunnel.
	BootError(err error)

	// Open is called when the Tunnel is ready to accept incoming connections.
	Open()

	// SessionEvent is called for logging purposes during important events in a Tunnel session (connection)
	SessionEvent(sessionId string, event string, tags stats.Tags)
	SessionError(sessionId string, err error)

	// Error is called when an error causes the tunnel to crash
	Error(err error)

	// Close is called upon the closure of a tunnel listener
	Close()

	// Stop is called upon graceful shutdown of the tunnel
	Stop()
}

type bootError struct {
	event string
	err   error
}

func (e bootError) Error() string {
	return fmt.Sprintf("boot error [%s]: %s", e.event, e.err.Error())
}

// lifecycleLogger logs events in a tunnel's lifecycle
type lifecycleLogger struct {
	st stats.Stats
}

func (l lifecycleLogger) Start() {
	l.st.SimpleEvent("start")
}

func (l lifecycleLogger) BootEvent(event string, tags stats.Tags) {
	tags["event"] = event
	l.st.Event(stats.Event{
		Event: statsd.Event{
			Title:     "boot_event",
			Text:      event,
			AlertType: statsd.Info,
		},
		Tags: tags,
	})
}

func (l lifecycleLogger) BootError(err error) {
	l.st.ErrorEvent("boot_error", err)
}

func (l lifecycleLogger) SessionEvent(sessionId string, event string, tags stats.Tags) {
	tags["event"] = event
	tags["session_id"] = sessionId

	l.st.Event(stats.Event{
		Event: statsd.Event{
			Title:     "session_event",
			Text:      event,
			AlertType: statsd.Info,
		},
		Tags: tags,
	})
}

func (l lifecycleLogger) SessionError(sessionId string, err error) {
	l.st.Event(stats.Event{
		Event: statsd.Event{
			Title:     "session_error",
			Text:      err.Error(),
			AlertType: statsd.Error,
		},
		Tags: stats.Tags{
			"session_id": sessionId,
		},
	})
}

func (l lifecycleLogger) Open() {
	l.st.SimpleEvent("open")
}

func (l lifecycleLogger) Error(err error) {
	l.st.Event(stats.Event{
		Event: statsd.Event{
			Title:     "error",
			Text:      err.Error(),
			AlertType: statsd.Error,
		},
	})
}

func (l lifecycleLogger) Close() {
	l.st.SimpleEvent("close")
}

func (l lifecycleLogger) Stop() {
	l.st.SimpleEvent("stop")
}

type NoopLifecycle struct {
}

func (n NoopLifecycle) Start() {
	// no-op
}

func (n NoopLifecycle) BootEvent(event string, tags stats.Tags) {
	// no-op
}

func (n NoopLifecycle) BootError(err error) {
	// no-op
}

func (n NoopLifecycle) Open() {
	// no-op
}

func (n NoopLifecycle) SessionEvent(sessionId string, event string, tags stats.Tags) {
	// no-op
}

func (n NoopLifecycle) SessionError(sessionId string, err error) {
	// no-op
}

func (n NoopLifecycle) Error(err error) {
	// no-op
}

func (n NoopLifecycle) Close() {
	// no-op
}

func (n NoopLifecycle) Stop() {
	// no-op
}

const ctxLifecycleKey = "_tunnel_lifecycle"

func getCtxLifecycle(ctx context.Context) Lifecycle {
	lc, ok := ctx.Value(ctxLifecycleKey).(Lifecycle)
	if !ok {
		return NoopLifecycle{}
	}
	return lc
}

func injectCtxLifecycle(ctx context.Context, lifecycle Lifecycle) context.Context {
	return context.WithValue(ctx, ctxLifecycleKey, lifecycle)
}
