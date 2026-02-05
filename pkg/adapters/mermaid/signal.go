package mermaid

import (
	"fmt"
	"reflect"
	"strings"
)

// SignalStateMachine renders a Mermaid state diagram for the signal context.
// Accepts signal.State from pkg/signal package.
func SignalStateMachine(sig any, opts ...Option) string {
	options := &Options{Styles: DefaultStyles()}
	for _, opt := range opts {
		opt(options)
	}

	var sb strings.Builder

	// Extract fields via reflection
	v := reflect.ValueOf(sig)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	forceExitThreshold := getIntField(v, "ForceExitThreshold")
	hookTimeout := getField(v, "HookTimeout")
	stopping := getBoolField(v, "Stopping")
	received := getField(v, "Received")

	sb.WriteString("stateDiagram-v2\n")
	sb.WriteString("    [*] --> Running\n")

	// Transition to Graceful
	signals := "SIGTERM"
	if forceExitThreshold == 1 {
		signals = "SIGINT/SIGTERM"
	}
	sb.WriteString(fmt.Sprintf("    Running --> Graceful: %s\n", signals))

	sb.WriteString("    note right of Graceful\n")
	sb.WriteString("        Context Cancelled\n")
	sb.WriteString("        Hooks Running (LIFO)\n")
	sb.WriteString(fmt.Sprintf("        Timeout: %v\n", hookTimeout))
	sb.WriteString("    end note\n")

	// Force Exit path
	if forceExitThreshold > 0 {
		sb.WriteString(fmt.Sprintf("    Graceful --> ForceExit: Signal x%d\n", forceExitThreshold))
		sb.WriteString("    ForceExit --> [*]: os.Exit(1)\n")
	}

	// Natural completion
	sb.WriteString("    Graceful --> [*]: Hooks Complete\n")

	// State highlighting
	stoppedState := getBoolField(v, "Stopped")
	if stoppedState {
		sb.WriteString("    class [*] stopped\n")
	} else if stopping {
		sb.WriteString("    class Graceful running\n")
	} else if received == nil {
		sb.WriteString("    class Running running\n")
	} else {
		// Check if received is actually nil (for interface types)
		rv := reflect.ValueOf(received)
		if rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface {
			if rv.IsNil() {
				sb.WriteString("    class Running running\n")
			}
		}
	}

	if received != nil {
		rv := reflect.ValueOf(received)
		isNil := false
		if rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface ||
			rv.Kind() == reflect.Slice || rv.Kind() == reflect.Map ||
			rv.Kind() == reflect.Chan || rv.Kind() == reflect.Func {
			isNil = rv.IsNil()
		}
		if !isNil {
			sb.WriteString(fmt.Sprintf("    note left of Graceful: Received %v\n", received))
		}
	}

	sb.WriteString(options.Styles)

	return sb.String()
}

// renderSignalFragment renders a Mermaid fragment representing the signal context for use in composite diagrams.
func renderSignalFragment(sb *strings.Builder, sig any, id, indent string) {
	v := reflect.ValueOf(sig)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	enabled := getBoolField(v, "Enabled")
	stopping := getBoolField(v, "Stopping")
	stoppedState := getBoolField(v, "Stopped")
	received := getField(v, "Received")
	reason := getStringField(v, "Reason")

	statusMode := "Running"
	statusClass := "running"

	if stoppedState {
		statusMode = "Stopped"
		statusClass = "stopped"
	} else if !enabled {
		statusMode = "Stopped"
		statusClass = "stopped"
	} else if stopping {
		statusMode = "Stopping"
		statusClass = "stopping" // Use the red/pink style for shutdown feedback

		// Adjust label if reason is known
		if reason == "Signal:Terminate" {
			statusClass = "failed"
		}
	}

	receivedStr := "None"
	if received != nil {
		rv := reflect.ValueOf(received)
		// Only check IsNil for types that can be nil
		if rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface ||
			rv.Kind() == reflect.Slice || rv.Kind() == reflect.Map ||
			rv.Kind() == reflect.Chan || rv.Kind() == reflect.Func {
			if !rv.IsNil() {
				receivedStr = fmt.Sprintf("%v", received)
			}
		} else {
			receivedStr = fmt.Sprintf("%v", received)
		}
	}

	if reason == "" {
		reason = "None"
	}

	label := fmt.Sprintf("<b>⚡ Lifecycle Controller</b><br/>Mode: %s", statusMode)

	if receivedStr != "None" {
		label += fmt.Sprintf("<br/>Received: %s", receivedStr)
	}

	if reason != "None" {
		label += fmt.Sprintf("<br/>Reason: %s", reason)
	}

	sb.WriteString(fmt.Sprintf("%s%s[\"%s\"]:::signal\n", indent, id, label))
	sb.WriteString(fmt.Sprintf("%sclass %s %s\n", indent, id, statusClass))
}

// Helper functions for reflection
func getIntField(v reflect.Value, name string) int {
	field := v.FieldByName(name)
	if field.IsValid() && field.CanInt() {
		return int(field.Int())
	}
	return 0
}

func getBoolField(v reflect.Value, name string) bool {
	field := v.FieldByName(name)
	if field.IsValid() && field.Kind() == reflect.Bool {
		return field.Bool()
	}
	return false
}

func getStringField(v reflect.Value, name string) string {
	field := v.FieldByName(name)
	if field.IsValid() && field.Kind() == reflect.String {
		return field.String()
	}
	return ""
}

func getField(v reflect.Value, name string) any {
	field := v.FieldByName(name)
	if field.IsValid() && field.CanInterface() {
		return field.Interface()
	}
	return nil
}
