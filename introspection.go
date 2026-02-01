package lifecycle

import (
	"strings"

	"github.com/aretw0/lifecycle/internal/diagram"
	"github.com/aretw0/lifecycle/pkg/signal"
	"github.com/aretw0/lifecycle/pkg/worker"
)

// SystemDiagram returns a unified Mermaid diagram representing the entire application lifecycle.
// It combines the SignalContext (Control Plane) and the Worker hierarchy (Data Plane).
func SystemDiagram(sig SignalState, work WorkerState) string {
	var sb strings.Builder

	sb.WriteString("graph TD\n")

	// 1. Signal Context Subgraph
	sb.WriteString("    subgraph ControlPlane [Signal Context]\n")
	signal.RenderFragment(&sb, sig, "S", "        ")
	sb.WriteString("    end\n\n")

	// 2. Worker Subgraph (The worker tree)
	sb.WriteString("    subgraph DataPlane [Supervision Tree]\n")
	worker.RenderTreeFragment(&sb, work, "root", "        ")
	sb.WriteString("    end\n\n")

	// 3. Connection
	sb.WriteString("    S -- cancels --> root\n")

	// 4. Styles
	sb.WriteString(diagram.Styles())

	return sb.String()
}
