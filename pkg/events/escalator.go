package events

import (
	"context"
	"errors"
	"sync/atomic"
)

// Escalator is a handler that escalates from a Primary handler to a Fallback handler
// based on whether the Primary handler successfully handled the event.
//
// Pattern: "Double-Tap" or "Try-Then-Force".
//
// Logic:
// 1. If already escalating, call Fallback.
// 2. Call Primary.
// 3. If Primary returns ErrNotHandled, switch to escalating and call Fallback.
// 4. If Primary returns nil (handled), remain in Primary mode (reset).
// 5. If Primary returns other error, return it (stop).
type Escalator struct {
	primary    Handler
	fallback   Handler
	escalating atomic.Bool
}

// NewEscalator creates a new Escalator handler.
// primary: The initial handler to try (e.g., "Interrupt", "Suspend", "Clear Line").
// fallback: The handler to use if primary fails to handle or upon escalation (e.g., "Quit", "Force Exit").
func NewEscalator(primary Handler, fallback Handler) *Escalator {
	return &Escalator{
		primary:  primary,
		fallback: fallback,
	}
}

// HandleEvent implements the Handler interface.
func (h *Escalator) HandleEvent(ctx context.Context, e Event) error {
	// 1. Check if we are already in the escalated state
	if h.escalating.Load() {
		return h.fallback.HandleEvent(ctx, e)
	}

	// 2. Attempt Primary
	err := h.primary.HandleEvent(ctx, e)

	// 3. Check for escalation trigger
	if errors.Is(err, ErrNotHandled) {
		// Atomic switch to ensure thread safety if multiple signals arrive
		if h.escalating.CompareAndSwap(false, true) {
			return h.fallback.HandleEvent(ctx, e)
		}
		// If race lost, another thread escalated, so we also fall through to fallback
		return h.fallback.HandleEvent(ctx, e)
	}

	// 4. Return result (nil or actual error)
	return err
}

// Reset resets the escalator to the primary state.
func (h *Escalator) Reset() {
	h.escalating.Store(false)
}
