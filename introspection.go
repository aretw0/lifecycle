package lifecycle

import (
	"fmt"
	"strings"

	"github.com/aretw0/lifecycle/internal/diagram"
	"github.com/aretw0/lifecycle/pkg/worker"
)

// SystemDiagram returns a unified Mermaid diagram representing the entire application lifecycle.
// It combines the SignalContext (Control Plane) and the Worker hierarchy (Data Plane).
func SystemDiagram(sig SignalState, work WorkerState) string {
	var sb strings.Builder

	sb.WriteString("graph TD\n")

	// 1. Signal Context Subgraph
	sb.WriteString("    subgraph ControlPlane [Signal Context]\n")

	// Convert SignalState to a single node representation for the subgraph
	status := "Running"
	if sig.Stopping {
		status = "Graceful"
	}
	received := "None"
	if sig.Received != nil {
		received = sig.Received.String()
	}

	sb.WriteString(fmt.Sprintf("        S[\"<b>Signal Handler</b><br/>Mode: %s<br/>Received: %s\"]", status, received))
	if sig.Stopping {
		sb.WriteString(":::running")
	} else {
		sb.WriteString(":::pending")
	}
	sb.WriteString("\n")
	sb.WriteString("    end\n\n")

	// 2. Worker Subgraph (The worker tree)
	sb.WriteString("    subgraph DataPlane [Supervision Tree]\n")
	worker.RenderTreeFragment(&sb, work, "root")
	sb.WriteString("    end\n\n")

	// 3. Connection
	sb.WriteString("    S -- cancels --> root\n")

	// 4. Styles
	sb.WriteString(diagram.Styles())

	return sb.String()
}
