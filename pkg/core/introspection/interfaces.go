package introspection

// Introspectable is an interface for components that can report their internal state.
// This is used for generating visualization and status reports.
//
// Note: For type-safe state watching, use TypedWatcher[S] instead.
type Introspectable interface {
	// State returns a serializable DTO (Data Transfer Object) representing the component's state.
	State() any
}

// Component identifies the type of a system component.
// Implementing this interface allows the introspection system to correctly classify
// the component (e.g. "worker", "supervisor", "signal") without relying on package paths.
type Component interface {
	// ComponentType returns the type of the component.
	ComponentType() string
}
