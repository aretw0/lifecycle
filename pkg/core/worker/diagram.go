package worker

import (
	"fmt"
	"strings"
	"time"

	"github.com/aretw0/introspection"
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
	sb.WriteString("    Running --> Stopped: Requested (Exit 0)\n")
	sb.WriteString("    Running --> Finished: Natural (Exit 0)\n")
	sb.WriteString("    Running --> Killed: Force Kill\n")
	sb.WriteString("    Running --> Failed: Error (Exit != 0)\n")

	// Styling
	sb.WriteString(introspection.DefaultStyles())
	sb.WriteString("    class Stopped stopped\n")
	sb.WriteString("    class Finished finished\n")
	sb.WriteString("    class Killed killed\n")
	sb.WriteString("    class Failed failed\n")

	// Details note
	sb.WriteString(fmt.Sprintf("    note right of %s\n", s.Status))
	sb.WriteString(fmt.Sprintf("        Name: %s\n", s.Name))
	if s.PID > 0 {
		sb.WriteString(fmt.Sprintf("        PID: %d\n", s.PID))
	}
	if s.Status == StatusStopped || s.Status == StatusFinished || s.Status == StatusKilled || s.Status == StatusFailed {
		sb.WriteString(fmt.Sprintf("        ExitCode: %d\n", s.ExitCode))
	}
	if s.Error != nil {
		sb.WriteString(fmt.Sprintf("        Error: %v\n", s.Error))
	}
	if s.Restarts > 0 {
		sb.WriteString(fmt.Sprintf("        Restarts: %d\n", s.Restarts))
	}
	if !s.StartedAt.IsZero() {
		sb.WriteString(fmt.Sprintf("        Uptime: %v\n", time.Since(s.StartedAt).Round(time.Second)))
	}
	if s.Health != nil {
		healthIcon := "❤️"
		if !s.Health.Healthy {
			healthIcon = "💔"
		}
		sb.WriteString(fmt.Sprintf("        Health: %s %s\n", healthIcon, s.Health.Message))
	}
	sb.WriteString("    end note\n")

	return sb.String()
}

// MermaidTree returns a Mermaid diagram string representing the worker structure (Tree).
// It renders a hierarchical tree (graph TD) showing parent-child relationships.
func MermaidTree(s State) string {
	return introspection.TreeDiagram(s, &introspection.DiagramConfig{
		SecondaryID: "root",
		NodeStyler:  NodeStyler,
		NodeLabeler: NodeLabeler,
	})
}

// NodeStyler provides worker-specific node styling based on metadata type field.
func NodeStyler(metadata map[string]string) (icon, shapeStart, shapeEnd, cssClass string) {
	icon = "🧬 " // Default: Goroutine/Generic
	shapeStart, shapeEnd = "(", ")"
	cssClass = string(TypeGoroutine)

	if tStr, ok := metadata["type"]; ok {
		// Normalize metadata to lowercase to match our canonical types
		switch Type(strings.ToLower(tStr)) {
		case TypeContainer:
			icon = "📦 "
			shapeStart, shapeEnd = "[[", "]]"
			cssClass = string(TypeContainer)
		case TypeProcess:
			icon = "⚙️ "
			shapeStart, shapeEnd = "[", "]"
			cssClass = string(TypeProcess)
		case TypeFunc:
			icon = "λ "
			shapeStart, shapeEnd = "(", ")"
			cssClass = string(TypeFunc)
		case TypeSupervisor:
			icon = "🧠 "
			shapeStart, shapeEnd = "{{", "}}" // Hexagon shape for supervisor/orchestrator
			cssClass = string(TypeSupervisor)
		}
	}
	return
}

// NodeLabeler provides worker-specific node label formatting.
func NodeLabeler(name, status string, pid int, metadata map[string]string, icon string) string {
	var labelParts []string
	labelParts = append(labelParts, fmt.Sprintf("<b>%s%s</b>", icon, name))
	if status != "" {
		labelParts = append(labelParts, status)
	}

	if pid > 0 {
		labelParts = append(labelParts, fmt.Sprintf("PID: %d", pid))
	}

	if metadata != nil {
		// Add significant metadata
		if ip, ok := metadata["ip"]; ok {
			labelParts = append(labelParts, fmt.Sprintf("🌐 %s", ip))
		}
		if ports, ok := metadata["ports"]; ok {
			labelParts = append(labelParts, fmt.Sprintf("🔌 %s", ports))
		}
		if image, ok := metadata["image"]; ok {
			labelParts = append(labelParts, fmt.Sprintf("<i>%s</i>", image))
		}
		if restarts, ok := metadata["restarts"]; ok && restarts != "0" {
			labelParts = append(labelParts, fmt.Sprintf("🔄 Restarts: %s", restarts))
		}
		if cb, ok := metadata["circuit_breaker"]; ok && cb == "triggered" {
			labelParts = append(labelParts, "<b>🚫 CIRCUIT BREAKER</b>")
		}

		// Health Bridge (for compatibility with current reflection-based introspection)
		if health, ok := metadata[MetadataHealth]; ok {
			healthIcon := "❤️"
			if health == "unhealthy" {
				healthIcon = "💔"
			}
			msg := ""
			if hmsg, ok := metadata[MetadataHealthMsg]; ok && hmsg != "" {
				msg = ": " + hmsg
			}
			labelParts = append(labelParts, fmt.Sprintf("%s Health%s", healthIcon, msg))
		}
	}

	return strings.Join(labelParts, "<br/>")
}
