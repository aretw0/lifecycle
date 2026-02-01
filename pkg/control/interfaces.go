package control

import "context"

// Event is a stimulus that triggers a reaction.
// It can be a system signal, a webhook, a time tick, or a custom application event.
type Event interface {
	String() string
}

// Reaction is an action taken in response to an event.
// It receives the context and performs a task (e.g., Shutdown, Reload, Log).
type Reaction func(ctx context.Context) error

// Source is a producer of events.
// It listens for external or internal triggers and emits them to the Events channel.
// The Start method should block until the context is done or a fatal error occurs.
type Source interface {
	// Events returns a read-only channel where the source emits events.
	Events() <-chan Event

	// Start begins the listening process. It should be non-blocking or managed
	// by the caller (Control Router). The implementation should respect the context.
	Start(ctx context.Context) error
}
