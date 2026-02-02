package control

import "fmt"

// SuspendEvent is triggered when the application should pause processing.
// This is typically used for "Durable Execution" where a process needs to be
// moved or upgraded without losing state.
type SuspendEvent struct{}

func (e SuspendEvent) String() string {
	return "lifecycle/suspend"
}

// ResumeEvent is triggered when the application should resume processing.
type ResumeEvent struct{}

func (e ResumeEvent) String() string {
	return "lifecycle/resume"
}

// StatusEvent is an internal event for periodic status updates.
type StatusEvent struct {
	Component string
	State     string
	Metadata  map[string]string
}

func (e StatusEvent) String() string {
	return fmt.Sprintf("status/%s", e.Component)
}
