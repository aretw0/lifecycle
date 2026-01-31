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
	renderSignalFragment(&sb, sig, "S", "        ")
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

func renderSignalFragment(sb *strings.Builder, sig SignalState, id, indent string) {
	statusMode := "Running"
	statusClass := "pending"

	if sig.Stopping {
		statusMode = "Graceful"
		statusClass = "running" // Or "stopped" depending on semantics, but let's match loop active state
	} else {
		// Waiting for signal (Pending state in terms of lifecycle action)
		statusClass = "pending"
	}

	received := "None"
	if sig.Received != nil {
		received = sig.Received.String()
	}

	// S["..."]:::signal
	// class S statusClass
	label := fmt.Sprintf("<b>Signal Handler</b><br/>Mode: %s<br/>Received: %s", statusMode, received)
	sb.WriteString(fmt.Sprintf("%s%s[\"%s\"]:::signal\n", indent, id, label))
	sb.WriteString(fmt.Sprintf("%sclass %s %s\n", indent, id, statusClass))
}
