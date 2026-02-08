package events

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/aretw0/lifecycle/pkg/core/log"
)

// SmartSignalHandler implements a "Double-Tap" strategy for signals like SIGINT.
// 1st Trigger: Attempts to Intercept/Suspend the application via the intercept handler.
// 2nd Trigger (or if interceptor is already Active): Delegates to a Quit handler (Force Exit).
type SmartSignalHandler struct {
	intercept Handler
	quit      Handler
	quitting  atomic.Bool
	actionMsg string
	actionEv  Event
}

// NewSmartSignalHandler creates a handler that arbitrates between Interruption and Quit
// based on the current state of the application.
//
// If 'intercept' implements StateChecker, its IsActive() method is used to decide
// whether to escalate to 'quit'.
func NewSmartSignalHandler(intercept Handler, quit Handler) *SmartSignalHandler {
	return &SmartSignalHandler{
		intercept: intercept,
		quit:      quit,
		actionMsg: "Suspending...",
		actionEv:  SuspendEvent{},
	}
}

// SmartSignalOption configures the SmartSignalHandler.
type SmartSignalOption func(*SmartSignalHandler)

// WithActionMessage sets the log message for the first trigger.
func WithActionMessage(msg string) SmartSignalOption {
	return func(h *SmartSignalHandler) {
		h.actionMsg = msg
	}
}

// WithActionEvent sets the event emitted on the first trigger.
func WithActionEvent(ev Event) SmartSignalOption {
	return func(h *SmartSignalHandler) {
		h.actionEv = ev
	}
}

// WithInteractiveSemantics configures the handler to use "Interrupted" language.
func WithInteractiveSemantics() SmartSignalOption {
	return func(h *SmartSignalHandler) {
		h.actionMsg = "Interrupted..."
		h.actionEv = InterceptEvent{}
	}
}

// NewSmartSignalHandlerWithOpts creates a configured SmartSignalHandler.
func NewSmartSignalHandlerWithOpts(intercept Handler, quit Handler, opts ...SmartSignalOption) *SmartSignalHandler {
	h := NewSmartSignalHandler(intercept, quit)
	for _, opt := range opts {
		opt(h)
	}
	return h
}

func (h *SmartSignalHandler) HandleEvent(ctx context.Context, e Event) error {
	// 0. Check if already quitting
	if h.quitting.Load() {
		log.Debug("SmartSignalHandler: Quit already in progress, ignoring signal.")
		return nil
	}

	// 1. Decide if we should Intercept or Quit
	shouldQuit := false

	// If interceptor implements StateChecker, use it for persistent state (e.g. Suspend)
	if sc, ok := h.intercept.(StateChecker); ok {
		if sc.IsActive() {
			shouldQuit = true
		}
	}

	if !shouldQuit {
		// Attempt to Intercept
		err := h.intercept.HandleEvent(ctx, h.actionEv)

		// Dynamic Escalation: If the handler explicitly says it didn't handle it,
		// we proceed to the Quit logic.
		if errors.Is(err, ErrNotHandled) {
			shouldQuit = true
		} else {
			// If handled (nil) or returned a real error, we stop here.
			if err == nil {
				msg := h.actionMsg
				// Only suggest "Send again to Quit" if we suspect stateful escalation is likely,
				// or if the action message specifically includes it.
				if _, ok := h.intercept.(StateChecker); ok {
					msg += " (Send signal again to Quit)"
				}
				log.Info("SmartSignalHandler: " + msg)
			}
			return err
		}
	}

	// 2. Atomic "Test-and-Set" for Quit
	if h.quitting.CompareAndSwap(false, true) {
		log.Info("SmartSignalHandler: Already interrupted/active or unhandled. Quitting...")
		return h.quit.HandleEvent(ctx, e)
	}

	return nil
}
