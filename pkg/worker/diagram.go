package worker

import (
	"fmt"
	"strings"

	"github.com/aretw0/lifecycle/internal/diagram"
)

// MermaidState returns a simple Mermaid state diagram (FSM) for a single worker.
// It visualizes the lifecycle transitions: Pending -> Running -> Stopped/Failed.
// This is useful for understanding the internal behavior of a worker type.
func MermaidState(s State) string {
	var sb strings.Builder

	sb.WriteString("stateDiagram-v2\n")
	sb.WriteString("    [*] --> Pending\n")

	// Pending -> Running
	sb.WriteString("    Pending --> Running: Start()\n")

	// Active State styling
	sb.WriteString(fmt.Sprintf("    class %s active\n", s.Status))

	// Terminal states
	sb.WriteString("    Running --> Stopped: Success (Exit 0)\n")
	sb.WriteString("    Running --> Failed: Error (Exit != 0)\n")

	// Styling
	sb.WriteString(diagram.Styles())
	sb.WriteString("    class Stopped stopped\n")
	sb.WriteString("    class Failed failed\n")

	// Details note
	sb.WriteString(fmt.Sprintf("    note right of %s\n", s.Status))
	sb.WriteString(fmt.Sprintf("        Name: %s\n", s.Name))
	if s.PID > 0 {
		sb.WriteString(fmt.Sprintf("        PID: %d\n", s.PID))
	}
	if s.Status == StatusStopped || s.Status == StatusFailed {
		sb.WriteString(fmt.Sprintf("        ExitCode: %d\n", s.ExitCode))
	}
	if s.Error != nil {
		sb.WriteString(fmt.Sprintf("        Error: %v\n", s.Error))
	}
	sb.WriteString("    end note\n")

	return sb.String()
}

// MermaidTree returns a Mermaid diagram string representing the worker structure (Tree).
// It renders a hierarchical tree (graph TD) showing parent-child relationships.
func MermaidTree(s State) string {
	var sb strings.Builder

	// We use "graph TD" for tree visualization
	sb.WriteString("graph TD\n")

	// Definitions for styles
	sb.WriteString(diagram.Styles())

	// Render Root
	renderNode(&sb, s, "root")

	return sb.String()
}

// RenderTreeFragment appends the Mermaid tree nodes and links to the provided builder.
// This is useful for building composite diagrams.
func RenderTreeFragment(sb *strings.Builder, s State, rootID string) {
	renderNode(sb, s, rootID)
}

func renderNode(sb *strings.Builder, s State, id string) {
	// 1. Determine Identity & Metadata Enrichment
	icon := "🧬 " // Default: Goroutine/Generic
	shapeStart, shapeEnd := "(", ")"
	idClass := "goroutine"

	if _, ok := s.Metadata["image"]; ok {
		icon = "📦 "
		shapeStart, shapeEnd = "[[", "]]"
		idClass = "container"
	} else if _, ok := s.Metadata["path"]; ok {
		icon = "⚙️ "
		shapeStart, shapeEnd = "[", "]"
		idClass = "process"
	}

	// 2. Build Label
	var labelParts []string
	labelParts = append(labelParts, fmt.Sprintf("<b>%s%s</b>", icon, s.Name))
	labelParts = append(labelParts, string(s.Status))

	if s.PID > 0 {
		labelParts = append(labelParts, fmt.Sprintf("PID: %d", s.PID))
	}

	// Add significant metadata
	if ip, ok := s.Metadata["ip"]; ok {
		labelParts = append(labelParts, fmt.Sprintf("🌐 %s", ip))
	}
	if ports, ok := s.Metadata["ports"]; ok {
		labelParts = append(labelParts, fmt.Sprintf("🔌 %s", ports))
	}
	if image, ok := s.Metadata["image"]; ok {
		labelParts = append(labelParts, fmt.Sprintf("<i>%s</i>", image))
	}

	label := strings.Join(labelParts, "<br/>")

	// 3. Determine Color Class
	statusClass := "pending"
	switch s.Status {
	case StatusRunning:
		statusClass = "running"
	case StatusStopped:
		statusClass = "stopped"
	case StatusFailed:
		statusClass = "failed"
	}

	// 4. Write Node and Class Assignment
	// We use the 'class ID className' syntax to avoid generating too many colons in a single line,
	// which ensures better compatibility across different Mermaid renderers.
	sb.WriteString(fmt.Sprintf("    %s%s\"%s\"%s\n", id, shapeStart, label, shapeEnd))
	sb.WriteString(fmt.Sprintf("    class %s %s,%s\n", id, statusClass, idClass))

	// Render Children
	for i, child := range s.Children {
		childID := fmt.Sprintf("%s_%d", id, i)
		renderNode(sb, child, childID)
		// Link
		sb.WriteString(fmt.Sprintf("    %s --> %s\n", id, childID))
	}
}
