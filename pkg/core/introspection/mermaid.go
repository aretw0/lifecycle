package introspection

import (
	"fmt"
	"reflect"
	"strings"
)

// MermaidOption configures the rendering behavior.
type MermaidOption func(*MermaidOptions)

type MermaidOptions struct {
	Styles string // Custom Mermaid class definitions
}

// DefaultStyles returns the standard Mermaid class definitions for lifecycle diagrams.
func DefaultStyles() string {
	return `    classDef created fill:#f8f9fa,stroke:#dee2e6,color:#6c757d;
    classDef pending fill:#eef2ff,stroke:#c7d2fe,color:#4338ca;
    classDef starting fill:#cfe2ff,stroke:#b8d4ff,color:#004085;
    classDef running fill:#d1ecf1,stroke:#bee5eb,color:#0c5460;
    classDef suspended fill:#fff3cd,stroke:#ffe69c,color:#856404;
    classDef stopping fill:#f8d7da,stroke:#f5c6cb,color:#721c24;
    classDef stopped fill:#e9ecef,stroke:#adb5bd,color:#495057;
    classDef finished fill:#d4edda,stroke:#c3e6cb,color:#155724;
    classDef killed fill:#343a40,stroke:#212529,color:#ffffff;
    classDef failed fill:#f8d7da,stroke:#f5c6cb,color:#721c24;
    classDef container stroke-width:3px,stroke-dasharray: 0;
    classDef process stroke-width:1px;
    classDef goroutine stroke-dasharray: 5 5;
    classDef supervisor stroke-width:2px,stroke-dasharray: 0;
    classDef signal stroke-width:2px,stroke-dasharray: 0;
    classDef active fill:#eef2ff,stroke:#4338ca,stroke-width:2px;
`
}

// WithStyles allows custom Mermaid class definitions.
func WithStyles(styles string) MermaidOption {
	return func(o *MermaidOptions) {
		o.Styles = styles
	}
}

// SystemDiagram renders a full system topology diagram combining signal context and worker tree.
// Accepts signal.State and worker.State (or pointers to them) as any.
func SystemDiagram(sig, work any, opts ...MermaidOption) string {
	options := &MermaidOptions{Styles: DefaultStyles()}
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

// SignalStateMachine renders a Mermaid state diagram for the signal context.
func SignalStateMachine(sig any, opts ...MermaidOption) string {
	options := &MermaidOptions{Styles: DefaultStyles()}
	for _, opt := range opts {
		opt(options)
	}

	var sb strings.Builder

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

	if forceExitThreshold > 0 {
		sb.WriteString(fmt.Sprintf("    Graceful --> ForceExit: Signal x%d\n", forceExitThreshold))
		sb.WriteString("    ForceExit --> [*]: os.Exit(1)\n")
	}

	sb.WriteString("    Graceful --> [*]: Hooks Complete\n")

	stoppedState := getBoolField(v, "Stopped")
	if stoppedState {
		sb.WriteString("    class [*] stopped\n")
	} else if stopping {
		sb.WriteString("    class Graceful stopping\n")
	} else if received == nil {
		sb.WriteString("    class Running running\n")
	} else {
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

// WorkerTreeDiagram returns a Mermaid diagram string representing the worker hierarchy.
func WorkerTreeDiagram(s any, opts ...MermaidOption) string {
	options := &MermaidOptions{Styles: DefaultStyles()}
	for _, opt := range opts {
		opt(options)
	}

	var sb strings.Builder
	sb.WriteString("graph TD\n")
	sb.WriteString(options.Styles)
	renderWorkerNode(&sb, s, "root", "    ")
	return sb.String()
}

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

	if !enabled {
		statusMode = "Disabled"
		statusClass = "stopped"
	} else if stoppedState {
		statusMode = "Stopped"
		statusClass = "stopped"
	} else if stopping {
		statusMode = "Stopping"
		statusClass = "pending"

		if reason == "Signal:Terminate" {
			statusClass = "failed"
		}
	}

	receivedStr := "None"
	if received != nil {
		rv := reflect.ValueOf(received)
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

func renderWorkerFragment(sb *strings.Builder, s any, rootID string, indent string) {
	renderWorkerNode(sb, s, rootID, indent)
}

func renderWorkerNode(sb *strings.Builder, s any, id, indent string) {
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	name := getStringField(v, "Name")
	status := getStringField(v, "Status")
	pid := getIntField(v, "PID")
	metadata := getMapField(v, "Metadata")
	children := getSliceField(v, "Children")

	icon, shapeStart, shapeEnd, idClass := getWorkerNodeStyle(metadata)
	label := buildWorkerNodeLabel(name, status, pid, metadata, icon)

	sb.WriteString(fmt.Sprintf("%s%s%s\"%s\"%s:::%s\n", indent, id, shapeStart, label, shapeEnd, idClass))

	statusClass := strings.ToLower(status)
	if statusClass == "" {
		statusClass = "pending"
	}
	sb.WriteString(fmt.Sprintf("%sclass %s %s\n", indent, id, statusClass))

	for i, child := range children {
		childID := fmt.Sprintf("%s_%d", id, i)
		renderWorkerNode(sb, child, childID, indent)
		sb.WriteString(fmt.Sprintf("%s%s --> %s\n", indent, id, childID))
	}
}

func getWorkerNodeStyle(metadata map[string]string) (icon, shapeStart, shapeEnd, idClass string) {
	workerType := "process"
	if metadata != nil {
		if t, ok := metadata["type"]; ok {
			workerType = t
		}
	}

	switch workerType {
	case "supervisor":
		icon, shapeStart, shapeEnd, idClass = "🧠", "{{", "}}", "supervisor"
	case "process":
		icon, shapeStart, shapeEnd, idClass = "⚙️", "[", "]", "process"
	case "container":
		icon, shapeStart, shapeEnd, idClass = "📦", "[[", "]]", "container"
	case "func", "goroutine":
		icon, shapeStart, shapeEnd, idClass = "λ", "(", ")", "goroutine"
	default:
		icon, shapeStart, shapeEnd, idClass = "⚙️", "[", "]", "process"
	}
	return
}

func buildWorkerNodeLabel(name string, status string, pid int, metadata map[string]string, icon string) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("<b>%s %s</b>", icon, name))
	if status != "" {
		parts = append(parts, fmt.Sprintf("Status: %s", status))
	}
	if pid > 0 {
		parts = append(parts, fmt.Sprintf("PID: %d", pid))
	}
	if metadata != nil {
		if image, ok := metadata["image"]; ok && image != "" {
			parts = append(parts, fmt.Sprintf("Image: %s", image))
		}
		if restarts, ok := metadata["restarts"]; ok && restarts != "0" {
			parts = append(parts, fmt.Sprintf("🔄 Restarts: %s", restarts))
		}
	}
	return strings.Join(parts, "<br/>")
}

// Reflection helpers
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

func getMapField(v reflect.Value, name string) map[string]string {
	field := v.FieldByName(name)
	if field.IsValid() && field.Kind() == reflect.Map {
		result := make(map[string]string)
		iter := field.MapRange()
		for iter.Next() {
			k, v := iter.Key(), iter.Value()
			if k.Kind() == reflect.String && v.Kind() == reflect.String {
				result[k.String()] = v.String()
			}
		}
		return result
	}
	return nil
}

func getSliceField(v reflect.Value, name string) []any {
	field := v.FieldByName(name)
	if field.IsValid() && field.Kind() == reflect.Slice {
		result := make([]any, field.Len())
		for i := 0; i < field.Len(); i++ {
			result[i] = field.Index(i).Interface()
		}
		return result
	}
	return nil
}
