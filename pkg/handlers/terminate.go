package handlers

import (
	"context"
	"errors"

	"github.com/aretw0/lifecycle/pkg/control"
)

// TerminateOption configures the TerminateHandler.
type TerminateOption func(*TerminateHandler)

// WithContinueOnFailure configures whether to proceed with shutdown even if suspension fails.
// Default is true.
func WithContinueOnFailure(continueOnFailure bool) TerminateOption {
	return func(h *TerminateHandler) {
		h.ContinueOnFailure = continueOnFailure
	}
}

// TerminateHandler chains a SuspendEvent (to save state) with a Shutdown.
// This implements the "Power Command" pattern: Composing primitives to create rich behaviors.
//
// Mindset: High-level operations should be composed from smaller, specialized handlers
// rather than hard-coding complexity.
type TerminateHandler struct {
	Suspend           control.Handler
	Shutdown          control.Handler
	ContinueOnFailure bool
}

// HandleEvent processes the terminate request by chaining suspend and shutdown phases.
// It collects errors from both phases and returns them joined.
func (h *TerminateHandler) HandleEvent(ctx context.Context, e control.Event) error {
	var errs []error

	// 1. Try to suspend (save state)
	if h.Suspend != nil {
		if err := h.Suspend.HandleEvent(ctx, control.SuspendEvent{}); err != nil {
			errs = append(errs, err)
			if !h.ContinueOnFailure {
				return err
			}
		}
	}

	// 2. Trigger Shutdown
	if h.Shutdown != nil {
		if err := h.Shutdown.HandleEvent(ctx, control.ShutdownEvent{}); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// NewTerminate creates a new handler that chains suspension and shutdown.
func NewTerminate(suspend control.Handler, shutdown control.Handler, opts ...TerminateOption) control.Handler {
	h := &TerminateHandler{
		Suspend:           suspend,
		Shutdown:          shutdown,
		ContinueOnFailure: true, // Default to pragmatic behavior
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}
