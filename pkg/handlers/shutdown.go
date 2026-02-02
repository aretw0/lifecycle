package handlers

import (
	"context"
	"time"

	"github.com/aretw0/lifecycle/pkg/control"
	"github.com/aretw0/lifecycle/pkg/log"
	"github.com/aretw0/lifecycle/pkg/metrics"
)

// ShutdownHandler creates a handler that cancels the given context (triggering shutdown).
// Since handlers receive a context, they can't cancel the *parent* unless they have access to the CancelFunc.
// We must provide the CancelFunc to the constructor.
type ShutdownHandler struct {
	Cancel context.CancelFunc
}

func (r *ShutdownHandler) HandleEvent(ctx context.Context, e control.Event) error {
	defer func(start time.Time) {
		metrics.GetProvider().ObserveHandlerDuration("shutdown", time.Since(start))
	}(time.Now())

	// TODO: Log reason based on event
	log.Info("shutdown triggered by event", "event", e.String())
	r.Cancel()
	return nil
}

// NewShutdown returns a handler that cancels context.
func NewShutdown(cancel context.CancelFunc) control.Handler {
	return &ShutdownHandler{Cancel: cancel}
}
