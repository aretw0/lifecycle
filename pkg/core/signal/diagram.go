package signal

import (
	"fmt"
	"strings"

	"github.com/aretw0/introspection"
)

// MermaidState returns a Mermaid state diagram string representing the lifecycle configuration.
func MermaidState(s State) string {
	initialToGraceful := "SIGTERM"
	if s.ForceExitThreshold == 1 {
		initialToGraceful = "SIGINT/SIGTERM"
	}

	return introspection.StateMachineDiagram(s, &introspection.StateMachineConfig{
		InitialState:      "Running",
		GracefulState:     "Graceful",
		ForcedState:       "ForceExit",
		InitialToGraceful: initialToGraceful,
		GracefulToForced:  "Signal",
		GracefulToFinal:   "Hooks Complete",
		NoteGenerator: func(state any) string {
			s := state.(State)
			var sb strings.Builder
			sb.WriteString("        Context Cancelled\n")
			sb.WriteString("        Hooks Running (LIFO)\n")
			sb.WriteString(fmt.Sprintf("        Timeout: %v\n", s.HookTimeout))
			if s.Received != nil {
				sb.WriteString(fmt.Sprintf("        Received: %v\n", s.Received))
			}
			return sb.String()
		},
	})
}

// PrimaryStyler determines the CSS class for the signal component based on its state.
func PrimaryStyler(state any) string {
	s, ok := state.(State)
	if !ok {
		return "running"
	}

	if !s.Enabled {
		return "stopped" // Green: Manually stopped (Listener inactive)
	}

	if s.Stopping {
		switch s.Reason {
		case ReasonTerminate:
			return "failed" // Red: System Termination
		case ReasonInterrupt, ReasonManualStop, ReasonManualCancel:
			return "pending" // Yellow: User Interrupt or manual cancel
		default:
			return "pending"
		}
	}

	return "active" // Blue: Listening
}

// PrimaryLabeler builds the HTML label for the signal component.
func PrimaryLabeler(state any) string {
	s, ok := state.(State)
	if !ok {
		return "<b>⚡ Signal Listener</b>"
	}

	statusMode := "Running"
	if !s.Enabled {
		statusMode = "Stopped"
	} else if s.Stopping {
		statusMode = "Graceful"
	}

	received := "None"
	if s.Received != nil {
		received = s.Received.String()
	}

	reason := s.Reason
	if reason == "" {
		reason = ReasonNone
	}

	return fmt.Sprintf("<b>⚡ Signal Listener</b><br/>Mode: %s<br/>Received: %s<br/>Reason: %s", statusMode, received, reason)
}
