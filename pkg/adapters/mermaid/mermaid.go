package mermaid

import (
	"strings"
)

// SystemDiagram renders a full system topology diagram combining signal context and worker tree.
// Accepts signal.State and worker.State from their respective packages.
func SystemDiagram(sig, work any, opts ...Option) string {
	options := &Options{Styles: DefaultStyles()}
	for _, opt := range opts {
		opt(options)
	}

	var sb strings.Builder

	sb.WriteString("graph TD\n")

	// 1. Signal Context Subgraph
	sb.WriteString("    subgraph ControlPlane [Signal Context]\n")
	renderSignalFragment(&sb, sig, "S", "        ")
	sb.WriteString("    end\n\n")

	// 2. Worker Subgraph (The worker tree)
	sb.WriteString("    subgraph DataPlane [Supervision Tree]\n")
	renderWorkerFragment(&sb, work, "root", "        ")
	sb.WriteString("    end\n\n")

	// 3. Connection
	sb.WriteString("    S -- governs --> root\n")

	// 4. Styles
	sb.WriteString(options.Styles)

	return sb.String()
}
