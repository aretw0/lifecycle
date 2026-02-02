package control

import "context"

// Event is a stimulus that triggers a reaction.
// It can be a system signal, a webhook, a time tick, or a custom application event.
type Event interface {
	String() string
}

// Handler responds to an event.
type Handler interface {
	HandleEvent(ctx context.Context, e Event) error
}

// HandlerFunc matches the signature of a Handler.
type HandlerFunc func(ctx context.Context, e Event) error

// HandleEvent calls f(ctx, e).
func (f HandlerFunc) HandleEvent(ctx context.Context, e Event) error {
	return f(ctx, e)
}

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

// RouteInfo describes a registered event route.
type RouteInfo struct {
	Pattern string
	Handler string // Name of the handler
}

// Introspectable is an interface for components that can report their internal state.
// This is used for generating visualization and status reports.
type Introspectable interface {
	// State returns a serializable DTO (Data Transfer Object) representing the component's state.
	State() any
}
