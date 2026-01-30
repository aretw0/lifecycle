package worker

import (
	"fmt"
	"strings"
)

// Mermaid returns a Mermaid state diagram string representing the worker state.
func Mermaid(s State) string {
	var sb strings.Builder

	sb.WriteString("stateDiagram-v2\n")
	sb.WriteString("    [*] --> Pending\n")

	// Pending -> Running
	if s.Status == StatusRunning || s.Status == StatusStopped || s.Status == StatusFailed {
		sb.WriteString("    Pending --> Running: Start()\n")
	}

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
