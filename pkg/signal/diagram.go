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
