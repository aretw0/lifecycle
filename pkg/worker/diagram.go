package worker

import (
	"fmt"
	"strings"
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

	// Styling for terminal states
	sb.WriteString("    classDef success fill:#d4edda,stroke:#28a745,color:black;\n")
	sb.WriteString("    classDef failure fill:#f8d7da,stroke:#dc3545,color:black;\n")
	sb.WriteString("    class Stopped success\n")
	sb.WriteString("    class Failed failure\n")

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
	sb.WriteString("    classDef pending fill:#fff3cd,stroke:#ffecb5,color:#856404;\n")
	sb.WriteString("    classDef running fill:#d1ecf1,stroke:#bee5eb,color:#0c5460;\n")
	sb.WriteString("    classDef stopped fill:#d4edda,stroke:#c3e6cb,color:#155724;\n")
	sb.WriteString("    classDef failed fill:#f8d7da,stroke:#f5c6cb,color:#721c24;\n")

	// Render Root
	renderNode(&sb, s, "root")

	return sb.String()
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
