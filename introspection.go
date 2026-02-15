package lifecycle

import (
	"github.com/aretw0/introspection"
)

// SystemDiagram returns a unified Mermaid diagram representing the entire application lifecycle.
// It combines the SignalContext (Control Plane) and the Worker hierarchy (Data Plane)
// using the generic introspection.ComponentDiagram API with lifecycle-specific configuration.
func SystemDiagram(sig SignalState, work WorkerState) string {
	return introspection.ComponentDiagram(sig, work, LifecycleDiagramConfig())
}
