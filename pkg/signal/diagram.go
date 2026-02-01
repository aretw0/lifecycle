package signal

import (
	"fmt"
	"strings"

	"github.com/aretw0/lifecycle/internal/diagram"
)

// MermaidState returns a Mermaid state diagram string representing the lifecycle configuration.
func MermaidState(s State) string {
	var sb strings.Builder

	sb.WriteString("stateDiagram-v2\n")
	sb.WriteString("    [*] --> Running\n")

	// Transition to Graceful
	signals := "SIGTERM"
	if s.InterruptCancel {
		signals = "SIGINT/SIGTERM"
	}
	sb.WriteString(fmt.Sprintf("    Running --> Graceful: %s\n", signals))

	sb.WriteString("    note right of Graceful\n")
	sb.WriteString("        Context Cancelled\n")
	sb.WriteString("        Hooks Running (LIFO)\n")
	sb.WriteString(fmt.Sprintf("        Timeout: %s\n", s.HookTimeout))
	sb.WriteString("    end note\n")

	// Force Exit path
	if s.ForceExitThreshold > 0 {
		sb.WriteString(fmt.Sprintf("    Graceful --> ForceExit: Signal x%d\n", s.ForceExitThreshold))
		sb.WriteString("    ForceExit --> [*]: os.Exit(1)\n")
	}

	// Natural completion
	sb.WriteString("    Graceful --> [*]: Hooks Complete\n")

	// State highlighting
	if s.Stopping {
		sb.WriteString("    class Graceful active\n")
	} else if s.Received == nil {
		sb.WriteString("    class Running active\n")
	}

	if s.Received != nil {
		sb.WriteString(fmt.Sprintf("    note left of Graceful: Received %s\n", s.Received))
	}

	sb.WriteString(diagram.Styles())

	return sb.String()
}

// RenderFragment renders a Mermaid fragment representing the signal context for use in composite diagrams.
func RenderFragment(sb *strings.Builder, sig State, id, indent string) {
	statusMode := "Running"
	statusClass := "active" // Default: Blue/Standard (Listening)

	if !sig.Enabled {
		statusMode = "Stopped"
		statusClass = "stopped" // Green: Manually stopped (Listener inactive)
	} else if sig.Stopping {
		statusMode = "Graceful"
		// Default Graceful is Warning/Pending (Yellow)
		statusClass = "pending"

		switch sig.Reason {
		case ReasonTerminate:
			statusClass = "failed" // Red: System Termination
		case ReasonInterrupt:
			statusClass = "pending" // Yellow: User Interrupt
		case ReasonManualStop, ReasonManualCancel:
			// If cancelled manually (Cancel called) but monitor is still enabled
			statusClass = "pending"
		}
	}

	received := "None"
	if sig.Received != nil {
		received = sig.Received.String()
	}

	reason := sig.Reason
	if reason == "" {
		reason = ReasonNone
	}

	// S["..."]:::signal
	// class S statusClass
	label := fmt.Sprintf("<b>⚡ Signal Listener</b><br/>Mode: %s<br/>Received: %s<br/>Reason: %s", statusMode, received, reason)
	sb.WriteString(fmt.Sprintf("%s%s[\"%s\"]:::signal\n", indent, id, label))
	sb.WriteString(fmt.Sprintf("%sclass %s %s\n", indent, id, statusClass))
}
