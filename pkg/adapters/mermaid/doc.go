// Package mermaid provides Mermaid diagram rendering for lifecycle introspection.
//
// This adapter transforms immutable state snapshots (from State() methods) into
// Mermaid diagrams for development, debugging, and visualization.
//
// # Usage
//
// Basic rendering:
//
//	import "github.com/aretw0/lifecycle/pkg/adapters/mermaid"
//
//	ctx := lifecycle.NewSignalContext(context.Background())
//	sup := lifecycle.NewSupervisor("root", lifecycle.SupervisorStrategyOneForOne)
//
//	// Full system diagram
//	fmt.Println(mermaid.SystemDiagram(ctx.State(), sup.State()))
//
// Custom styles:
//
//	custom := mermaid.WithStyles(`
//	    classDef stopped fill:#00ff00;
//	`)
//	diagram := mermaid.SystemDiagram(ctx.State(), sup.State(), custom)
//
// # Scope
//
// This adapter renders **static snapshots** (poll-based) and can be used with
// event-driven introspection (StateWatcher) for real-time updates.
//
// # Extending
//
// To create adapters for other formats (JSON, GraphViz, OpenTelemetry), follow
// the same pattern: accept `any` state DTOs, render to target format.
package mermaid
