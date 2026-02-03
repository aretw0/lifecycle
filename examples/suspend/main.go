package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/aretw0/lifecycle"
)

// FactoryState represents the persisted state of our factory.
type FactoryState struct {
	ItemsProduced  int `json:"items_produced"`
	ItemsProcessed int `json:"items_processed"`
}

const (
	StateFile  = "factory_state.json"
	TargetGoal = 20
)

// Store manages persistence.
type Store struct {
	mu    sync.Mutex
	state FactoryState
}

func (s *Store) Save(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	bytes, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}

	slog.Info("[STORE] Persisting state to disk...", "items", s.state)
	// Simulate I/O latency
	time.Sleep(100 * time.Millisecond)
	return os.WriteFile(StateFile, bytes, 0644)
}

func (s *Store) Load() {
	bytes, err := os.ReadFile(StateFile)
	if err == nil {
		json.Unmarshal(bytes, &s.state)
		slog.Info("[STORE] Loaded previous state", "state", s.state)
	}
}

func (s *Store) Cleanup() {
	slog.Info("[STORE] Goal reached! Cleaning up state file.")
	os.Remove(StateFile)
}

// Generator produces raw materials.
type Generator struct {
	output chan int
	paused bool
	mu     sync.Mutex
	cond   *sync.Cond
	store  *Store
}

func NewGenerator(store *Store) *Generator {
	g := &Generator{
		output: make(chan int),
		store:  store,
	}
	g.cond = sync.NewCond(&g.mu)
	return g
}

func (g *Generator) Run(ctx context.Context) error {
	slog.Info("[GENERATOR] Started.")
	for {
		g.mu.Lock()
		for g.paused {
			slog.Info("[GENERATOR] Paused. Waiting for signal...")
			g.cond.Wait()
			slog.Info("[GENERATOR] Resuming production...")
		}
		g.mu.Unlock()

		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// Produce item
		g.store.mu.Lock()
		if g.store.state.ItemsProduced >= TargetGoal {
			g.store.mu.Unlock()
			return nil // Goal reached
		}
		g.store.state.ItemsProduced++
		item := g.store.state.ItemsProduced
		g.store.mu.Unlock()

		slog.Info("[GENERATOR] Produced item", "id", item)

		select {
		case g.output <- item:
		case <-ctx.Done():
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func (g *Generator) Pause(ctx context.Context) error {
	g.mu.Lock()
	g.paused = true
	g.mu.Unlock()
	return nil
}

func (g *Generator) Resume(ctx context.Context) error {
	g.mu.Lock()
	g.paused = false
	g.cond.Broadcast()
	g.mu.Unlock()
	return nil
}

// Worker processes materials.
type Worker struct {
	input  <-chan int
	store  *Store
	paused bool
	mu     sync.Mutex
}

func NewWorker(input <-chan int, store *Store) *Worker {
	return &Worker{
		input: input,
		store: store,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	slog.Info("[WORKER] Waiting for work...")
	for {
		// Check pause BEFORE taking new work
		w.mu.Lock()
		if w.paused {
			w.mu.Unlock()
			// If paused, we just spin/sleep briefly in this simple example
			// In a real app we'd use a Cond or channel like Generator
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(100 * time.Millisecond):
				continue
			}
		}
		w.mu.Unlock()

		select {
		case item, ok := <-w.input:
			if !ok {
				return nil
			}
			slog.Info("[WORKER] Processing item...", "id", item)

			// Simulate "Heavy" work that must finish even if Suspend hits
			select {
			case <-time.After(800 * time.Millisecond):
			case <-ctx.Done():
				// Even if context cancels, we might want to finish?
				// For this demo, we respect context.
				return nil
			}

			w.store.mu.Lock()
			w.store.state.ItemsProcessed++
			count := w.store.state.ItemsProcessed
			w.store.mu.Unlock()

			slog.Info("[WORKER] Finished item", "id", item)

			if count >= TargetGoal {
				return nil // We are done
			}

		case <-ctx.Done():
			return nil
		}
	}
}

func (w *Worker) Pause(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	slog.Info("[WORKER] Pause requested. Will finish current job (if any) then stop.")
	w.paused = true
	return nil
}

func (w *Worker) Resume(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	slog.Info("[WORKER] Resuming processing.")
	w.paused = false
	return nil
}

// Watchdog never sleeps.
func Watchdog(ctx context.Context) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			slog.Info("[WATCHDOG] System Healthy. (I never sleep!)")
		}
	}
}

func main() {
	// 0. Logging
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, opts)))

	// 1. Persistence
	store := &Store{}
	store.Load()

	// 2. Control Plane
	suspendHandler := lifecycle.NewSuspendHandler()

	// Generators & Workers
	gen := NewGenerator(store)
	worker := NewWorker(gen.output, store)

	// 3. Hooks (The Glue)
	suspendHandler.OnSuspend(func(ctx context.Context) error {
		// Ordering matters! Pause input first.
		if err := gen.Pause(ctx); err != nil {
			return err
		}
		if err := worker.Pause(ctx); err != nil {
			return err
		}
		return store.Save(ctx)
	})

	suspendHandler.OnResume(func(ctx context.Context) error {
		// Resume worker first, then generator
		if err := worker.Resume(ctx); err != nil {
			return err
		}
		return gen.Resume(ctx)
	})

	// 4. Router (Trigger Mapping)
	router := lifecycle.NewRouter()
	router.Handle("lifecycle/suspend", suspendHandler) // Triggered via signals or manually
	router.Handle("lifecycle/resume", suspendHandler)

	// We use a standard "Simulator" goroutine to trigger events
	// to make the example self-driving and cross-platform (Windows/Linux).

	simSource := make(chan lifecycle.Event)
	router.AddSource(&ChannelSource{Ch: simSource})

	// 5. Run Logic
	err := lifecycle.Run(lifecycle.Job(func(ctx context.Context) error {
		// Start Router
		lifecycle.Go(ctx, router.Start)

		// Start Watchdog
		lifecycle.Go(ctx, Watchdog)

		// Start Worker
		lifecycle.Go(ctx, worker.Run)

		// Start Generator
		lifecycle.Go(ctx, gen.Run)

		// Start Simulator (The "User" hitting buttons)
		lifecycle.Go(ctx, func(ctx context.Context) error {
			// Wait a bit, then suspend
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(3 * time.Second):
			}

			slog.Warn(">>> SIMULATING USER: SUSPEND COMMAND <<<")
			simSource <- lifecycle.SuspendEvent{} // Internal event

			// Wait while suspended
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(3 * time.Second):
			}

			slog.Warn(">>> SIMULATING USER: RESUME COMMAND <<<")
			simSource <- lifecycle.ResumeEvent{}

			return nil
		})

		// Wait for completion (Polling the store state)
		ticker := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				store.mu.Lock()
				count := store.state.ItemsProcessed
				store.mu.Unlock()
				if count >= TargetGoal {
					slog.Info("GOAL REACHED! Shutting down factory.")
					store.Cleanup()
					return nil // Causes lifecycle.Run to exit cleanly
				}
			}
		}
	}))

	if err != nil {
		fmt.Printf("Factory exited with error: %v\n", err)
	}
}

// ChannelSource adapts a channel to the Source interface
type ChannelSource struct {
	Ch <-chan lifecycle.Event
}

func (s *ChannelSource) Events() <-chan lifecycle.Event { return s.Ch }
func (s *ChannelSource) Start(ctx context.Context) error {
	<-ctx.Done()
	return nil
}
