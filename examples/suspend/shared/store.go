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
	Mu    sync.Mutex
	State FactoryState
	Path  string
}

func NewStore(path string) *Store {
	return &Store{Path: path}
}

// Save persists the current state to disk.
func (s *Store) Save(ctx context.Context) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	bytes, err := json.MarshalIndent(s.State, "", "  ")
	if err != nil {
		return err
	}

	slog.Info("[STORE] Persisting state to disk...", "items", s.State, "path", s.Path)
	// Simulate I/O latency
	time.Sleep(100 * time.Millisecond)
	return os.WriteFile(s.Path, bytes, 0644)
}

// Load reads state from disk if it exists.
func (s *Store) Load() {
	if s.Path == "" {
		s.Path = StateFile
	}
	bytes, err := os.ReadFile(s.Path)
	if err != nil {
		return
	}
	json.Unmarshal(bytes, &s.State)

	// CRITICAL RECOVERY:
	// If items were Produced but not Processed, they were in the in-memory channel
	// and are now lost due to restart. We must "Rewind" production to ensure
	// they are re-generated.
	if s.State.ItemsProduced > s.State.ItemsProcessed {
		slog.Warn("[STORE] Detected in-flight items lost during restart. Rewinding.",
			"produced", s.State.ItemsProduced,
			"processed", s.State.ItemsProcessed,
			"rewind_to", s.State.ItemsProcessed)
		s.State.ItemsProduced = s.State.ItemsProcessed
	}
	slog.Info("[STORE] Loaded previous state", "state", s.State)
}

// Cleanup removes the state file and resets internal counters.
func (s *Store) Cleanup() {
	slog.Info("[STORE] Goal reached! Cleaning up state file.")
	s.Mu.Lock()
	s.State.ItemsProduced = 0
	s.State.ItemsProcessed = 0
	s.Mu.Unlock()
	os.Remove(s.Path)
}



