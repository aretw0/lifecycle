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

// Watchdog is a system service that runs continuously, ignoring suspension.
type Watchdog struct {
	done chan error
}

func NewWatchdog() *Watchdog {
	return &Watchdog{done: make(chan error, 1)}
}

func (w *Watchdog) Start(ctx context.Context) error {
	go func() {
		w.done <- w.Run(ctx)
		close(w.done)
	}()
	return nil
}

func (w *Watchdog) Stop(ctx context.Context) error { return nil }
func (w *Watchdog) Wait() <-chan error             { return w.done }
func (w *Watchdog) String() string                 { return "Watchdog" }
func (w *Watchdog) State() lifecycle.WorkerState   { return lifecycle.WorkerState{Name: "Watchdog"} }

func (w *Watchdog) Run(ctx context.Context) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			slog.Info("[WATCHDOG] System Healthy. (I never sleep!)")
		}
	}
}

// FactoryState represents the persisted state of our factory.
type FactoryState struct {
	ItemsProduced  int `json:"items_produced"`
	ItemsProcessed int `json:"items_processed"`
}

const (
	StateFile  = "factory_state.json"
	TargetGoal = 100
)

// Store manages persistence.
type Store struct {
	mu    sync.Mutex
	state FactoryState
	path  string
}

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

func (s *Store) Cleanup() {
	slog.Info("[STORE] Goal reached! Cleaning up state file.")
	os.Remove(s.path)
}

// Generator produces raw materials.
type Generator struct {
	output chan int
	paused bool
	mu     sync.Mutex
	cond   *sync.Cond
	store  *Store
	done   chan error
}

// Satisfy worker.Suspendable
func (g *Generator) Start(ctx context.Context) error {
	go func() {
		// Run blocks until context is cancelled or goal met
		g.done <- g.Run(ctx)
		close(g.done)
	}()
	return nil
}
func (g *Generator) Stop(ctx context.Context) error { return nil }
func (g *Generator) Wait() <-chan error             { return g.done }
func (g *Generator) String() string                 { return "Generator" }
func (g *Generator) State() lifecycle.WorkerState   { return lifecycle.WorkerState{Name: "Generator"} }

func NewGenerator(output chan int, store *Store) *Generator {
	g := &Generator{
		output: output,
		store:  store,
		done:   make(chan error, 1),
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
			return ctx.Err()
		default:
		}

		// Produce item
		g.store.mu.Lock()
		if g.store.state.ItemsProduced >= TargetGoal {
			g.store.mu.Unlock()
			slog.Info("[GENERATOR] Target reached. Stopping.")
			// Supervisor will NOT restart us because we return nil (RestartOnFailure).
			return nil
		}
		g.store.state.ItemsProduced++
		item := g.store.state.ItemsProduced
		g.store.mu.Unlock()

		slog.Info("[GENERATOR] Produced item", "id", item)

		select {
		case g.output <- item:
		case <-ctx.Done():
			return ctx.Err()
		}

		time.Sleep(2000 * time.Millisecond)
	}
}

func (g *Generator) Suspend(ctx context.Context) error {
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
	input    <-chan int
	store    *Store
	paused   bool
	pauseReq bool
	mu       sync.Mutex
	cond     *sync.Cond
	done     chan error
}

// Satisfy worker.Suspendable
func (w *Worker) Start(ctx context.Context) error {
	go func() {
		w.done <- w.Run(ctx)
		close(w.done)
	}()
	return nil
}
func (w *Worker) Stop(ctx context.Context) error { return nil }
func (w *Worker) Wait() <-chan error             { return w.done }
func (w *Worker) String() string                 { return "Worker" }
func (w *Worker) State() lifecycle.WorkerState   { return lifecycle.WorkerState{Name: "Worker"} }

func NewWorker(input <-chan int, store *Store) *Worker {
	w := &Worker{
		input: input,
		store: store,
		done:  make(chan error, 1),
	}
	w.cond = sync.NewCond(&w.mu)
	return w
}

func (w *Worker) Run(ctx context.Context) error {
	slog.Info("[WORKER] Waiting for work...")
	for {
		w.mu.Lock()

		// 1. Quiescence Check
		// If a pause was requested, we stop HERE, before taking new work.
		// We set paused=true and broadcast to wake up Suspend().
		if w.pauseReq {
			w.paused = true
			w.cond.Broadcast()
			slog.Info("[WORKER] Quiescent. Paused.")
		}

		for w.paused {
			w.cond.Wait()
			slog.Info("[WORKER] Resuming...")
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
			case <-time.After(1500 * time.Millisecond):
			case <-ctx.Done():
				return ctx.Err()
			}

			w.store.mu.Lock()
			w.store.state.ItemsProcessed++
			count := w.store.state.ItemsProcessed
			w.store.mu.Unlock()

			slog.Info("[WORKER] Finished item", "id", item)

			if count >= TargetGoal {
				return nil
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Suspend blocks until the worker is actually paused.
func (w *Worker) Suspend(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	slog.Info("[WORKER] Suspend requested. Waiting for quiescent state...")
	w.pauseReq = true

	// Wait until the worker loop confirms it is paused
	for !w.paused {
		// Verify context to avoid hanging forever
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Use a temporary release of lock or Wait?
		// Worker does w.cond.Broadcast() when entering pause.
		// But cond.Wait() requires lock. We have it.
		w.cond.Wait()
	}

	slog.Info("[WORKER] Suspended successfully.")
	return nil
}

func (w *Worker) Resume(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	slog.Info("[WORKER] Resuming processing.")
	w.pauseReq = false
	w.paused = false
	w.cond.Broadcast()
	return nil
}

// Blocker is a worker that refuses to suspend quickly, testing the USER's patience.
// It allows demonstrating the "Force Exit" safety net during a stuck suspension.
type Blocker struct {
	done chan error
}

func (b *Blocker) Start(ctx context.Context) error {
	b.done = make(chan error, 1)
	go func() {
		<-ctx.Done()
		b.done <- nil
		close(b.done)
	}()
	return nil
}
func (b *Blocker) Stop(ctx context.Context) error { return nil }
func (b *Blocker) Wait() <-chan error             { return b.done }
func (b *Blocker) String() string                 { return "Blocker" }
func (b *Blocker) State() lifecycle.WorkerState   { return lifecycle.WorkerState{Name: "Blocker"} }

func (b *Blocker) Resume(ctx context.Context) error { return nil }
func (b *Blocker) Suspend(ctx context.Context) error {
	slog.Warn("[BLOCKER] Suspending... (I am slow! Press Ctrl+C again 2x to Force Quit, or wait)")
	select {
	case <-time.After(5 * time.Second):
		slog.Info("[BLOCKER] Finally suspended.")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func main() {
	// 0. Logging
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, opts)))

	// 1. Persistence
	statePath := StateFile
	if _, err := os.Stat("examples/suspend"); err == nil {
		statePath = "examples/suspend/" + StateFile
	}
	store := &Store{path: statePath}
	store.Load()

	// 2. Control Plane
	suspendHandler := lifecycle.NewSuspendHandler()

	// Shared communication
	sharedChan := make(chan int)

	// 3. Supervisor
	sup := lifecycle.NewSupervisor("factory", lifecycle.SupervisorStrategyOneForOne,
		lifecycle.SupervisorSpec{
			Name: "watchdog",
			Factory: func() (lifecycle.Worker, error) {
				return NewWatchdog(), nil
			},
			RestartPolicy: lifecycle.RestartAlways,
		},
		lifecycle.SupervisorSpec{
			Name: "generator",
			Factory: func() (lifecycle.Worker, error) {
				return NewGenerator(sharedChan, store), nil
			},
			RestartPolicy: lifecycle.RestartOnFailure,
		},
		lifecycle.SupervisorSpec{
			Name: "worker",
			Factory: func() (lifecycle.Worker, error) {
				return NewWorker(sharedChan, store), nil
			},
			RestartPolicy: lifecycle.RestartOnFailure,
		},
		lifecycle.SupervisorSpec{
			Name: "blocker", // The bad neighbor
			Factory: func() (lifecycle.Worker, error) {
				return &Blocker{}, nil
			},
			RestartPolicy: lifecycle.RestartAlways,
		},
	)

	// 4. Run Logic (Interactive with Input)
	err := lifecycle.Run(lifecycle.Job(func(ctx context.Context) error {
		// Channels to synchronize UI
		suspendedCh := make(chan struct{})
		resumedCh := make(chan struct{})
		quitCh := make(chan struct{})

		// Router Setup
		router := lifecycle.NewRouter()

		// Map internal events to Handlers
		router.Handle("lifecycle/suspend", suspendHandler)
		router.Handle("lifecycle/resume", suspendHandler)

		// Smart Signal Handling: Ctrl+C -> Suspend -> Quit
		smartHandler := lifecycle.NewSmartSignalHandler(suspendHandler, lifecycle.HandlerFunc(func(ctx context.Context, e lifecycle.Event) error {
			// Now safe to call directly thanks to internal idempotency in SmartSignalHandler
			close(quitCh)
			return nil
		}))
		router.Handle("Signal(interrupt)", smartHandler)

		// Handle Quit locally
		router.Handle("input/quit", lifecycle.HandlerFunc(func(ctx context.Context, e lifecycle.Event) error {
			// For mixed sources, we technically still risk a race if user scripts 'q' + Ctrl+C simultaneously,
			// but for interactive usage, the SmartSignalHandler protection covers the main "spam" case.
			select {
			case <-quitCh:
			default:
				close(quitCh)
			}
			return nil
		}))

		// Add Sources
		router.AddSource(lifecycle.NewOSSignalSource(os.Interrupt))
		inputSource := lifecycle.NewInputSource()
		router.AddSource(inputSource)

		// Allow Auto-Suspend to speak
		simSource := make(chan lifecycle.Event)
		router.AddSource(&ChannelSource{Ch: simSource})

		// Hooks to update UI State
		suspendHandler.OnSuspend(func(ctx context.Context) error {
			// Reset signal count is now handled automatically by time-based reset

			if suspendable, ok := sup.(lifecycle.Suspendable); ok {
				if err := suspendable.Suspend(ctx); err != nil {
					return err
				}
			}
			if err := store.Save(ctx); err != nil {
				return err
			}
			select {
			case suspendedCh <- struct{}{}:
			default:
			}
			return nil
		})

		suspendHandler.OnResume(func(ctx context.Context) error {
			// Reset here too, just in case a signal woke us up (not applicable here but good practice)
			if sc, ok := ctx.(*lifecycle.Context); ok {
				sc.ResetSignalCount()
			}

			if suspendable, ok := sup.(lifecycle.Suspendable); ok {
				if err := suspendable.Resume(ctx); err != nil {
					return err
				}
			}
			select {
			case resumedCh <- struct{}{}:
			default:
			}
			return nil
		})

		// Start Components
		lifecycle.Go(ctx, router.Start)
		if err := sup.Start(ctx); err != nil {
			return err
		}
		defer sup.Stop(context.WithoutCancel(ctx))

		lifecycle.Go(ctx, func(ctx context.Context) error {
			<-sup.Wait()
			return nil
		})

		// Wait for interrupts
		// We use the router to detect the FIRST signal (Suspend)
		// But we need to poll/check for the "Press again to force quit" warning?
		// Actually, the router only handles events.
		// We can add a specialized handler or just let the default "SmartSignalHandler" do its job.
		// But we want to show a warning.
		// Let's hook into the SmartSignal behavior? No, that's internal.
		// We can use a Ticker or just let the logs speak.
		// The user asked for "Avisa que o próximo vai cancelar".
		// We can do this by wrapping the Signal Source?

		// Simplest way: The SmartSignalHandler logs "Suspending...".
		// The Blockers log "Suspending... Press Ctrl+C again...".
		// The SignalContext logs "force exit threshold reached".

		// Reviewing the logs from Step 276:
		// "SmartSignalHandler: Suspending..."
		// "[BLOCKER] Suspending... (I am slow! Press Ctrl+C again 2x to Force Quit...)"

		// This already kind of exists in the Blocker.
		// But let's verify if we can check the state in the main loop to show a dynamic "FURY METER".

		// UI Loop
		slog.Info(">>> FACTORY RUNNING <<<")
		slog.Info("Commands: [s]uspend, [r]esume, [q]uit")
		slog.Info("Auto-Suspend in 10s (unless you interact)")

		autoSuspend := time.NewTimer(10 * time.Second)
		defer autoSuspend.Stop()
		autoSuspendActive := true

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()

			case <-quitCh:
				fmt.Println("👋 Quitting via command...")
				return nil

			case <-autoSuspend.C:
				if !autoSuspendActive {
					continue
				}
				// Check goal
				store.mu.Lock()
				done := store.state.ItemsProcessed >= TargetGoal
				store.mu.Unlock()
				if !done {
					slog.Warn(">>> AUTO-SUSPEND TRIGGERED <<<")
					simSource <- lifecycle.SuspendEvent{}
				}

			case <-suspendedCh:
				// Reset Loop State for UI
				autoSuspend.Stop()
				if autoSuspendActive {
					fmt.Println("\n🛑 SYSTEM SUSPENDED (Auto).")
				} else {
					fmt.Println("\n🛑 SYSTEM SUSPENDED (Manual).")
				}
				fmt.Println("👉 Commands: [r]esume | [q]uit")

			case <-resumedCh:
				// User interacted to resume, so WE DISABLE Auto-Suspend
				autoSuspendActive = false
				fmt.Println("\n🟢 SYSTEM RESUMED.")
				fmt.Println("👉 Manual Mode Active. Commands: [s]uspend | [q]uit")

			// Check Goal
			case <-time.After(1 * time.Second):
				store.mu.Lock()
				count := store.state.ItemsProcessed
				store.mu.Unlock()
				if count >= TargetGoal {
					slog.Info("🏆 GOAL REACHED! Shutting down factory.")
					store.Cleanup()
					return nil
				}
			}
		}
	}), lifecycle.WithInterrupt(false))

	if err != nil {
		if err != context.Canceled {
			fmt.Printf("Factory exited with error: %v\n", err)
		}
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
