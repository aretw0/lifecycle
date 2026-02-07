package events

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

// ShutdownEvent is triggered when the application should shut down gracefully.
// This is typically mapped to "exit" or "quit" commands.
type ShutdownEvent struct {
	Reason string
}

func (e ShutdownEvent) String() string {
	return "lifecycle/shutdown"
}

// ClearLineEvent is triggered when an interactive input is interrupted (e.g. Ctrl+C)
// but the process should NOT exit. Applications can use this to clear the current
// line and show a fresh prompt.
type ClearLineEvent struct{}

func (e ClearLineEvent) String() string {
	return "lifecycle/clear-line"
}

// TerminateEvent is a high-level event that chains Suspend and Shutdown.
// It represents a graceful exit that preserves system state.
type TerminateEvent struct{}

func (e TerminateEvent) String() string {
	return "lifecycle/terminate"
}

// ReloadEvent is triggered when the application should reload its configuration.
// It is intended for "Hot Reload" scenarios where a restart is not required.
type ReloadEvent struct{}

func (e ReloadEvent) String() string {
	return "lifecycle/reload"
}
