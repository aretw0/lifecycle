// Package introspection provides a cross-cutting observation layer for the lifecycle ecosystem.
//
// It defines the mechanisms for capturing, aggregating, and visualizing the
// state of various components (Data Plane and Control Plane) without creating
// circular dependencies.
//
// # Visualization (Mermaid)
//
// The package includes the canonical Mermaid diagram generation logic. It uses
// reflection and standardized state DTOs to render system topology and component
// states (Running, Graceful, Stopped).
//
// # State Watcher
//
// It provides the [StateWatcher] interface and [Aggregator] to collect real-time
// snapshots from multiple [Introspectable] components, enabling live monitoring
// and debugging.
package introspection
