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
	// Node Label
	label := fmt.Sprintf("<b>%s</b><br/>%s", s.Name, s.Status)
	if s.PID > 0 {
		label += fmt.Sprintf("<br/>PID: %d", s.PID)
	}
	if s.Error != nil {
		label += fmt.Sprintf("<br/>Err: %v", s.Error)
	}

	// Determine shape/style based on status
	// StatusPending -> rect
	// StatusRunning -> rounded rect
	// StatusStopped -> cylinder? or rect
	// Check status
	styleClass := "pending"
	switch s.Status {
	case StatusRunning:
		styleClass = "running"
	case StatusStopped:
		styleClass = "stopped"
	case StatusFailed:
		styleClass = "failed"
	}

	// Write Node
	// id["label"]:::styleClass
	sb.WriteString(fmt.Sprintf("    %s[\"%s\"]:::%s\n", id, label, styleClass))

	// Render Children
	for i, child := range s.Children {
		childID := fmt.Sprintf("%s_%d", id, i)
		renderNode(sb, child, childID)
		// Link
		sb.WriteString(fmt.Sprintf("    %s --> %s\n", id, childID))
	}
}
