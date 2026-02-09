package events

import (
	"context"
	"github.com/aretw0/lifecycle/pkg/core/log"
	"github.com/aretw0/lifecycle/pkg/core/metrics"
	"time"
)

// ShutdownHandler creates a handler that cancels the given context (triggering shutdown).
// Since handlers receive a context, they can't cancel the *parent* unless they have access to the CancelFunc.
// We must provide the CancelFunc to the constructor.
type ShutdownHandler struct {
	Cancel context.CancelFunc
}

func (r *ShutdownHandler) HandleEvent(ctx context.Context, e Event) error {
	defer func(start time.Time) {
		metrics.GetProvider().ObserveHandlerDuration("shutdown", time.Since(start))
	}(time.Now())

	// TODO: Log reason based on event
	log.Info("shutdown triggered by event", "event", e.String())
	r.Cancel()
	return nil
}

// NewShutdown returns a handler that cancels context.
// It is automatically wrapped in Once to ensure idempotency.
func NewShutdown(cancel context.CancelFunc) Handler {
	return Once(&ShutdownHandler{Cancel: cancel})
}

// NewShutdownFunc returns a handler that executes the given function once.
// Useful for wrapping generic close/cleanup operations as shutdown triggers.
func NewShutdownFunc(fn func()) Handler {
	return Once(HandlerFunc(func(ctx context.Context, e Event) error {
		if fn != nil {
			fn()
		}
		return nil
	}))
}
