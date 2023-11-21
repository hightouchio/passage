package tunnel

import (
	"context"
	"fmt"
	"github.com/hightouchio/passage/log"
	"go.uber.org/zap"
)

// Lifecycle provides callbacks for the tunnel to self-report status
type Lifecycle interface {
	// Start is called when a Tunnel begins booting up.
	Start()

	// BootEvent is called for logging purposes during relevant events in the Tunnel boot process.
	BootEvent(event string, args ...any)

	// BootError is called when an anomaly occurs while booting a Tunnel.
	BootError(err error)

	// Open is called when the Tunnel is ready to accept incoming connections.
	Open()

	// SessionEvent is called for logging purposes during important events in a Tunnel session (connection)
	SessionEvent(sessionId string, event string, args ...any)
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
	logger *log.Logger
}

func (l lifecycleLogger) Start() {
	l.logger.Info("Starting tunnel")
}

func (l lifecycleLogger) BootEvent(event string, args ...any) {
	l.logger.With(args...).Infof("Boot event: %s", event)
}

func (l lifecycleLogger) BootError(err error) {
	l.logger.Errorw("Boot error", zap.Error(err))
}

func (l lifecycleLogger) SessionEvent(sessionId string, event string, args ...any) {
	l.logger.With(
		zap.String("session_id", sessionId),
	).With(args...).Infof("Session event: %s", event)
}

func (l lifecycleLogger) SessionError(sessionId string, err error) {
	l.logger.With(
		zap.String("session_id", sessionId),
	).Errorw("Session error", zap.Error(err))
}

func (l lifecycleLogger) Open() {
	l.logger.Info("Open")
}

func (l lifecycleLogger) Error(err error) {
	l.logger.Error(zap.Error(err))
}

func (l lifecycleLogger) Stop() {
	l.logger.Info("Stop")
}

func (l lifecycleLogger) Close() {
	l.logger.Info("Close")
}

type NoopLifecycle struct {
}

func (n NoopLifecycle) Start() {
	// no-op
}

func (n NoopLifecycle) BootEvent(event string, args ...any) {
	// no-op
}

func (n NoopLifecycle) BootError(err error) {
	// no-op
}

func (n NoopLifecycle) Open() {
	// no-op
}

func (n NoopLifecycle) SessionEvent(sessionId string, event string, args ...any) {
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
