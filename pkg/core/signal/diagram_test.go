package signal

import (
	"strings"
	"testing"
	"time"
)

func TestMermaid(t *testing.T) {
	tests := []struct {
		name     string
		state    State
		contains []string
	}{
		{
			name: "Default",
			state: State{
				Config: Config{
					ForceExitThreshold: 1,
					HookTimeout:        5 * time.Second,
				},
			},
			contains: []string{
				"Running --> Graceful: SIGINT/SIGTERM",
				"Timeout: 5s",
				"Graceful --> ForceExit: Signal x1",
			},
		},
		{
			name: "NoInterrupt",
			state: State{
				Config: Config{
					ForceExitThreshold: 2,
					HookTimeout:        5 * time.Second,
				},
			},
			contains: []string{
				"Running --> Graceful: SIGTERM", // Should not see SIGINT
			},
		},
		{
			name: "NoForceExit",
			state: State{
				Config: Config{
					ForceExitThreshold: 0,
					HookTimeout:        100 * time.Millisecond,
				},
			},
			contains: []string{
				"Timeout: 100ms",
			},
		},
		{
			name: "StoppingHighlight",
			state: State{
				Config: Config{
					ForceExitThreshold: 1,
					HookTimeout:        5 * time.Second,
				},
				Status: Status{
					Stopping: true,
				},
			},
			contains: []string{
				"class Graceful active",
				"classDef active",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MermaidState(tt.state)
			for _, substr := range tt.contains {
				if !strings.Contains(got, substr) {
					t.Errorf("Mermaid() output missing %q, got:\n%s", substr, got)
				}
			}
			// Special check for negative assertions (simple way)
			if tt.name == "NoForceExit" && strings.Contains(got, "ForceExit") {
				t.Error("Mermaid() should not contain ForceExit when threshold is 0")
			}
		})
	}
}
