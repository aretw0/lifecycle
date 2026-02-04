package shared

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	"log/slog"
)

const (
	StateFile  = "factory_state.json"
	TargetGoal = 100
)

// FactoryState represents the persisted state of our factory.
type FactoryState struct {
	ItemsProduced  int `json:"items_produced"`
	ItemsProcessed int `json:"items_processed"`
}

// Store manages persistence of factory state.
type Store struct {
	mu    sync.Mutex
	state FactoryState
	path  string
}

// Save persists the current state to disk.
func (s *Store) Save(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	bytes, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}

	slog.Info("[STORE] Persisting state to disk...", "items", s.state, "path", s.path)
	// Simulate I/O latency
	time.Sleep(100 * time.Millisecond)
	return os.WriteFile(s.path, bytes, 0644)
}

// Load reads state from disk if it exists.
func (s *Store) Load() {
	if s.path == "" {
		s.path = StateFile
	}
	bytes, err := os.ReadFile(s.path)
	if err == nil {
		json.Unmarshal(bytes, &s.state)
		// CRITICAL RECOVERY:
		// If items were Produced but not Processed, they were in the in-memory channel
		// and are now lost due to restart. We must "Rewind" production to ensure
		// they are re-generated.
		if s.state.ItemsProduced > s.state.ItemsProcessed {
			slog.Warn("[STORE] Detected in-flight items lost during restart. Rewinding.",
				"produced", s.state.ItemsProduced,
				"processed", s.state.ItemsProcessed,
				"rewind_to", s.state.ItemsProcessed)
			s.state.ItemsProduced = s.state.ItemsProcessed
		}
		slog.Info("[STORE] Loaded previous state", "state", s.state)
	}
}

// Cleanup removes the state file.
func (s *Store) Cleanup() {
	slog.Info("[STORE] Goal reached! Cleaning up state file.")
	os.Remove(s.path)
}
