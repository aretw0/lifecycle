package sources

import (
	"context"
	"testing"
	"time"
)

func TestNewHealthCheckSource(t *testing.T) {
	checkFunc := func(ctx context.Context) error { return nil }

	tests := []struct {
		name         string
		opts         []HealthOption
		wantInterval time.Duration
		wantStrategy TriggerStrategy
	}{
		{
			name:         "Default",
			opts:         nil,
			wantInterval: 30 * time.Second,
			wantStrategy: TriggerEdge,
		},
		{
			name: "CustomInterval",
			opts: []HealthOption{
				WithInterval(5 * time.Second),
			},
			wantInterval: 5 * time.Second,
			wantStrategy: TriggerEdge,
		},
		{
			name: "CustomStrategy",
			opts: []HealthOption{
				WithStrategy(TriggerLevel),
			},
			wantInterval: 30 * time.Second,
			wantStrategy: TriggerLevel,
		},
		{
			name: "Mixed",
			opts: []HealthOption{
				WithInterval(10 * time.Minute),
				WithStrategy(TriggerLevel),
			},
			wantInterval: 10 * time.Minute,
			wantStrategy: TriggerLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewHealthCheckSource("test", checkFunc, tt.opts...)
			if s.Interval != tt.wantInterval {
				t.Errorf("NewHealthCheckSource() Interval = %v, want %v", s.Interval, tt.wantInterval)
			}
			if s.Strategy != tt.wantStrategy {
				t.Errorf("NewHealthCheckSource() Strategy = %v, want %v", s.Strategy, tt.wantStrategy)
			}
		})
	}
}
