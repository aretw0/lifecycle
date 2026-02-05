package events

import (
	"testing"
)

func TestEventTypeStrings(t *testing.T) {
	tests := []struct {
		name     string
		event    interface{}
		expected string
	}{
		{
			name:     "SuspendEvent",
			event:    SuspendEvent{},
			expected: "lifecycle/suspend",
		},
		{
			name:     "ResumeEvent",
			event:    ResumeEvent{},
			expected: "lifecycle/resume",
		},
		{
			name:     "ShutdownEvent",
			event:    ShutdownEvent{},
			expected: "lifecycle/shutdown",
		},
		{
			name:     "ClearLineEvent",
			event:    ClearLineEvent{},
			expected: "lifecycle/clear-line",
		},
		{
			name:     "TerminateEvent",
			event:    TerminateEvent{},
			expected: "lifecycle/terminate",
		},
		{
			name:     "StatusEvent",
			event:    StatusEvent{Component: "test"},
			expected: "status/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.event.(interface{ String() string }).String()
			if str != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, str)
			}
		})
	}
}
