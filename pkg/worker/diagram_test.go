package worker

import (
	"strings"
	"testing"
)

func TestMermaidState(t *testing.T) {
	tests := []struct {
		name     string
		state    State
		contains []string
	}{
		{
			name:     "Pending",
			state:    State{Name: "worker-1", Status: StatusPending},
			contains: []string{"class Pending active", "Name: worker-1"},
		},
		{
			name:     "Running",
			state:    State{Name: "worker-2", Status: StatusRunning, PID: 1234},
			contains: []string{"Pending --> Running: Start()", "class Running active", "PID: 1234"},
		},
		{
			name:     "Stopped",
			state:    State{Name: "worker-3", Status: StatusStopped, ExitCode: 0},
			contains: []string{"class Stopped active", "ExitCode: 0", "class Stopped stopped"},
		},
		{
			name:     "Failed",
			state:    State{Name: "worker-4", Status: StatusFailed, ExitCode: 1},
			contains: []string{"class Failed active", "ExitCode: 1", "class Failed failed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MermaidState(tt.state)
			for _, c := range tt.contains {
				if !strings.Contains(got, c) {
					t.Errorf("MermaidState() missing %q", c)
				}
			}
		})
	}
}

func TestMermaidTree(t *testing.T) {
	tests := []struct {
		name     string
		state    State
		contains []string
	}{
		{
			name:     "Leaf",
			state:    State{Name: "worker-1", Status: StatusPending},
			contains: []string{"graph TD", "worker-1", "Pending", "classDef pending"},
		},
		{
			name: "Tree",
			state: State{
				Name:   "root",
				Status: StatusRunning,
				Children: []State{
					{Name: "child-1", Status: StatusRunning},
					{Name: "child-2", Status: StatusStopped},
				},
			},
			contains: []string{"root --> root_0", "root --> root_1", "child-1", "child-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MermaidTree(tt.state)
			for _, c := range tt.contains {
				if !strings.Contains(got, c) {
					t.Errorf("MermaidTree() missing %q", c)
				}
			}
		})
	}
}
