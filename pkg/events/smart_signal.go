package events

import (
	"context"

	"sync/atomic"

	"github.com/aretw0/lifecycle/pkg/core/log"
)

// SmartSignalHandler implements a "Double-Tap" strategy for signals like SIGINT.
// 1st Trigger: Attempts to Suspend the application.
// 2nd Trigger (or if already Suspended): Delegates to a Quit handler (Force Exit).
type SmartSignalHandler struct {
	suspend  *SuspendHandler
	quit     Handler
	quitting atomic.Bool
}

// NewSmartSignalHandler creates a handler that arbitrates between Suspend and Quit
// based on the current state of the application.
func NewSmartSignalHandler(s *SuspendHandler, q Handler) *SmartSignalHandler {
	return &SmartSignalHandler{
		suspend: s,
		quit:    q,
	}
}

func (h *SmartSignalHandler) HandleEvent(ctx context.Context, e Event) error {
	// 0. Check if already quitting
	if h.quitting.Load() {
		log.Debug("SmartSignalHandler: Quit already in progress, ignoring signal.")
		return nil
	}

	// Introspect state to decide action
	state := h.suspend.State().(map[string]any)
	isSuspended, ok := state["suspended"].(bool)
	if !ok {
		// Fallback if introspection fails (should generally not happen)
		log.Warn("SmartSignalHandler: failed to read suspend state, defaulting to quit")
		if h.quitting.CompareAndSwap(false, true) {
			return h.quit.HandleEvent(ctx, e)
		}
		return nil
	}

	if !isSuspended {
		log.Info("SmartSignalHandler: Suspending... (Send signal again to Quit)")
		// Synthesize a SuspendEvent
		return h.suspend.HandleEvent(ctx, SuspendEvent{})
	}

	// Atomic "Test-and-Set" for Quit
	if h.quitting.CompareAndSwap(false, true) {
		log.Info("SmartSignalHandler: Already suspended. Quitting...")
		return h.quit.HandleEvent(ctx, e)
	}

	return nil
}


