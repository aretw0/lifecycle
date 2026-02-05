package lifecycle

import (
	"github.com/aretw0/lifecycle/pkg/core/introspection"
)

// SystemDiagram returns a unified Mermaid diagram representing the entire application lifecycle.
// It combines the SignalContext (Control Plane) and the Worker hierarchy (Data Plane).
func SystemDiagram(sig SignalState, work WorkerState) string {
	return introspection.SystemDiagram(sig, work)
}
