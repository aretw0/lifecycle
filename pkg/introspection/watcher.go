package introspection

import "context"

// TypedWatcher provides type-safe state watching for a specific state type S.
// Implementations can return their domain-specific state without any type assertions.
type TypedWatcher[S any] interface {
	// State returns the current state snapshot
	State() S

	// Watch returns a channel of type-safe state changes.
	// The channel is closed when the provided context is cancelled.
	Watch(ctx context.Context) <-chan StateChange[S]
}

// EventSource provides an event stream for observability.
type EventSource interface {
	// Events returns a channel of component events.
	// The channel is closed when the provided context is cancelled.
	Events(ctx context.Context) <-chan ComponentEvent
}
