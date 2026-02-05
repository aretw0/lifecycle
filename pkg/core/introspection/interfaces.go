package introspection

// Introspectable is an interface for components that can report their internal state.
// This is used for generating visualization and status reports.
//
// Note: For type-safe state watching, use TypedWatcher[S] instead.
type Introspectable interface {
	// State returns a serializable DTO (Data Transfer Object) representing the component's state.
	State() any
}



