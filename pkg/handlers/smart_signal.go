package handlers

import (
	"context"

	"github.com/aretw0/lifecycle/pkg/control"
	"github.com/aretw0/lifecycle/pkg/log"
)

// SmartSignalHandler implements a "Double-Tap" strategy for signals like SIGINT.
// 1st Trigger: Attempts to Suspend the application.
// 2nd Trigger (or if already Suspended): Delegates to a Quit handler (Force Exit).
type SmartSignalHandler struct {
	suspend *SuspendHandler
	quit    control.Handler
}

// NewSmartSignalHandler creates a handler that arbitrates between Suspend and Quit
// based on the current state of the application.
func NewSmartSignalHandler(s *SuspendHandler, q control.Handler) *SmartSignalHandler {
	return &SmartSignalHandler{
		suspend: s,
		quit:    q,
	}
}

func (h *SmartSignalHandler) HandleEvent(ctx context.Context, e control.Event) error {
	// Introspect state to decide action
	state := h.suspend.State().(map[string]any)
	isSuspended, ok := state["suspended"].(bool)
	if !ok {
		// Fallback if introspection fails (should generally not happen)
		log.Warn("SmartSignalHandler: failed to read suspend state, defaulting to quit")
		return h.quit.HandleEvent(ctx, e)
	}

	if !isSuspended {
		log.Info("SmartSignalHandler: Suspending... (Send signal again to Quit)")
		// Synthesize a SuspendEvent
		return h.suspend.HandleEvent(ctx, control.SuspendEvent{})
	}

	log.Info("SmartSignalHandler: Already suspended. Quitting...")
	return h.quit.HandleEvent(ctx, e)
}
