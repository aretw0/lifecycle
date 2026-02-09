package events

import "context"

// WithStateCheck wraps a handler and only executes it if the StateChecker reports valid state for handling.
//
// Semantics:
// If checker.IsActive() returns TRUE, it means the state "Exists" (e.g. already Suspended).
// In the context of a "Start Operation" (like Suspend), this means we CANNOT start it again.
// So we return ErrNotHandled to allow escalation (e.g. to Quit).
//
// If checker.IsActive() returns FALSE, we proceed to call the handler.
func WithStateCheck(h Handler, checker StateChecker) Handler {
	return HandlerFunc(func(ctx context.Context, e Event) error {
		if checker.IsActive() {
			return ErrNotHandled
		}
		return h.HandleEvent(ctx, e)
	})
}

// WithFixedEvent wraps a handler and passes the specified event to it, ignoring the original event.
// Useful for adapting generic signals (SignalEvent) to specific domain events (SuspendEvent).
func WithFixedEvent(h Handler, ev Event) Handler {
	return HandlerFunc(func(ctx context.Context, _ Event) error {
		return h.HandleEvent(ctx, ev)
	})
}
