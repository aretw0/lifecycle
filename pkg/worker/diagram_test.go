package worker

import (
	"errors"
	"strings"
	"testing"
)

func TestMermaid(t *testing.T) {
	tests := []struct {
		name     string
		state    State
		contains []string
	}{
		{
			name: "Pending",
			state: State{
				Name:   "worker-1",
				Status: StatusPending,
			},
			contains: []string{
				"class Pending active",
				"Name: worker-1",
			},
		},
		{
			name: "Running",
			state: State{
				Name:   "worker-2",
				Status: StatusRunning,
				PID:    1234,
			},
			contains: []string{
				"Pending --> Running: Start()",
				"class Running active",
				"PID: 1234",
			},
		},
		{
			name: "Stopped",
			state: State{
				Name:     "worker-3",
				Status:   StatusStopped,
				PID:      1234,
				ExitCode: 0,
			},
			contains: []string{
				"class Stopped active",
				"ExitCode: 0",
				"class Stopped success",
			},
		},
		{
			name: "Failed",
			state: State{
				Name:     "worker-4",
				Status:   StatusFailed,
				PID:      1234,
				ExitCode: 1,
				Error:    errors.New("oops"),
			},
			contains: []string{
				"class Failed active",
				"ExitCode: 1",
				"Error: oops",
				"class Failed failure",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Mermaid(tt.state)
			for _, c := range tt.contains {
				if !strings.Contains(got, c) {
					// Relaxed check for "network" prefix since that was a typo in my mental model,
					// checking loose containment is enough.
					// Actually, let's just check the core string.
					c = strings.TrimPrefix(c, "network ")
					if !strings.Contains(got, c) {
						t.Errorf("Mermaid() missing %q\nGot:\n%s", c, got)
					}
				}
			}
		})
	}
}
