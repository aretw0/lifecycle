package worker

import (
	"strings"
	"testing"
)

func TestMermaidTree_Styles(t *testing.T) {
	tests := []struct {
		name     string
		state    State
		contains []string
	}{
		{
			name: "Container",
			state: State{
				Name:   "redis",
				Status: StatusRunning,
				Metadata: map[string]string{
					"type": "container",
				},
			},
			contains: []string{"📦", "[[", "]]", "container"},
		},
		{
			name: "Process",
			state: State{
				Name:   "nginx",
				Status: StatusRunning,
				Metadata: map[string]string{
					"type": "process",
				},
			},
			contains: []string{"⚙️", "[", "]", "process"},
		},
		{
			name: "Function",
			state: State{
				Name:   "handler",
				Status: StatusRunning,
				Metadata: map[string]string{
					"type": "func",
				},
			},
			contains: []string{"λ", "(", ")", "func"},
		},
		{
			name: "Supervisor",
			state: State{
				Name:   "main",
				Status: StatusRunning,
				Metadata: map[string]string{
					"type": "supervisor",
				},
			},
			contains: []string{"🧠", "{{", "}}", "supervisor"},
		},
		{
			name: "Generic",
			state: State{
				Name:   "worker",
				Status: StatusRunning,
			},
			contains: []string{"🧬", "(", ")", "goroutine"},
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



