package mermaid

import (
	"fmt"
	"reflect"
	"strings"
)

// WorkerTreeDiagram returns a Mermaid diagram string representing the worker hierarchy.
// Accepts worker.State from pkg/worker package.
func WorkerTreeDiagram(s any, opts ...Option) string {
	options := &Options{Styles: DefaultStyles()}
	for _, opt := range opts {
		opt(options)
	}

	var sb strings.Builder

	// We use "graph TD" for tree visualization
	sb.WriteString("graph TD\n")

	// Definitions for styles
	sb.WriteString(options.Styles)

	// Render Root
	renderWorkerNode(&sb, s, "root", "    ")

	return sb.String()
}

// renderWorkerFragment appends the Mermaid tree nodes and links to the provided builder.
// This is useful for building composite diagrams.
func renderWorkerFragment(sb *strings.Builder, s any, rootID string, indent string) {
	renderWorkerNode(sb, s, rootID, indent)
}

func renderWorkerNode(sb *strings.Builder, s any, id, indent string) {
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Extract fields
	name := getStringField(v, "Name")
	status := getStringField(v, "Status")
	pid := getIntField(v, "PID")
	metadata := getMapField(v, "Metadata")
	children := getSliceField(v, "Children")

	// 1. Determine Identity & Metadata Enrichment
	icon, shapeStart, shapeEnd, idClass := getWorkerNodeStyle(metadata)

	// 2. Build Label
	label := buildWorkerNodeLabel(name, status, pid, metadata, icon)

	// 3. Define Node
	sb.WriteString(fmt.Sprintf("%s%s%s\"%s\"%s:::%s\n", indent, id, shapeStart, label, shapeEnd, idClass))

	// 4. Apply Status Class (state-based styling)
	statusClass := strings.ToLower(status)
	if statusClass == "" {
		statusClass = "pending"
	}
	sb.WriteString(fmt.Sprintf("%sclass %s %s\n", indent, id, statusClass))

	// 5. Render Children (Recursive)
	for i, child := range children {
		childID := fmt.Sprintf("%s_%d", id, i)
		renderWorkerNode(sb, child, childID, indent)
		sb.WriteString(fmt.Sprintf("%s%s --> %s\n", indent, id, childID))
	}
}

func getWorkerNodeStyle(metadata map[string]string) (icon, shapeStart, shapeEnd, idClass string) {
	workerType := "process" // default
	if metadata != nil {
		if t, ok := metadata["type"]; ok {
			workerType = t
		}
	}

	// Icon selection
	switch workerType {
	case "supervisor":
		icon = "🧠"
		idClass = "supervisor"
		shapeStart = "{{"
		shapeEnd = "}}"
	case "process":
		icon = "⚙️"
		idClass = "process"
		shapeStart = "["
		shapeEnd = "]"
	case "container":
		icon = "🐳"
		idClass = "container"
		shapeStart = "[["
		shapeEnd = "]]"
	case "func", "goroutine":
		icon = "λ"
		idClass = "goroutine"
		shapeStart = "("
		shapeEnd = ")"
	default:
		icon = "⚙️"
		idClass = "process"
		shapeStart = "["
		shapeEnd = "]"
	}

	return
}

func buildWorkerNodeLabel(name string, status string, pid int, metadata map[string]string, icon string) string {
	var parts []string

	// Icon + Name
	parts = append(parts, fmt.Sprintf("<b>%s %s</b>", icon, name))

	// Status
	if status != "" {
		parts = append(parts, fmt.Sprintf("Status: %s", status))
	}

	// PID (if running)
	if pid > 0 {
		parts = append(parts, fmt.Sprintf("PID: %d", pid))
	}

	// Additional metadata
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

// Reflection helpers for map and slice
func getMapField(v reflect.Value, name string) map[string]string {
	field := v.FieldByName(name)
	if field.IsValid() && field.Kind() == reflect.Map {
		result := make(map[string]string)
		iter := field.MapRange()
		for iter.Next() {
			k := iter.Key()
			v := iter.Value()
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
