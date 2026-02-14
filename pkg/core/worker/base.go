package worker

import (
	"context"
	"sync"

	"github.com/aretw0/lifecycle/pkg/core/introspection"
	"github.com/aretw0/procio/termio"
)

// BaseWorker provides default implementations for common Worker interface methods.
// It is designed to be embedded in custom worker types to reduce boilerplate.
//
// Example:
//
//	type MyWorker struct {
//	    lifecycle.BaseWorker
//	    // ... custom fields
//	}
//
//	func NewMyWorker() *MyWorker {
//	    return &MyWorker{
//	        BaseWorker: lifecycle.NewBaseWorker("MyWorker"),
//	    }
//	}
//
//	func (w *MyWorker) Start(ctx context.Context) error {
//	    return w.StartFunc(ctx, w.Run)
//	}
//
//	func (w *MyWorker) Run(ctx context.Context) error {
//	    // ... business logic
//	    return nil
//	}
//
// The embedding pattern provides default implementations for:
//   - Stop(ctx) — no-op (context cancellation handles cleanup)
//   - Wait() — returns done channel
//   - String() — returns worker name
//   - State() — returns minimal state with name
//   - Watch(ctx) — returns state change events (StateWatcher)
//
// These can be overridden if custom behavior is needed.
type BaseWorker struct {
	name     string
	done     chan error
	finished chan struct{}
	status   Status
	mu       sync.RWMutex

	// StateWatchers (Event-Driven Introspection)
	stateWatchers []chan introspection.StateChange[State]
	watchersMu    sync.RWMutex

	// Standardized State Fields (for centralized logic)
	StopRequested bool
	Killed        bool
	ExitCode      int
	Err           error
}

// NewBaseWorker creates a new BaseWorker with the given name.
// The name is immutable after creation (construct a new worker to change it).
func NewBaseWorker(name string) *BaseWorker {
	return &BaseWorker{
		name:     name,
		done:     make(chan error, 1),
		finished: make(chan struct{}),
		status:   StatusCreated,
	}
}

// DeriveFinalStatus determines the final status based on the strict Intent vs Outcome logic.
// This centralizes the state machine rules:
// Killed -> StatusKilled
// Err != nil -> StatusFailed
// StopRequested -> StatusStopped
// Default -> StatusFinished
func (b *BaseWorker) DeriveFinalStatus() Status {
	if b.Killed {
		return StatusKilled
	}
	if b.Err != nil && termio.IsInterrupted(b.Err) {
		return StatusStopped
	}
	if b.Err != nil {
		return StatusFailed
	}
	if b.StopRequested {
		return StatusStopped
	}
	return StatusFinished
}

// SetStatus updates the worker's status and emits a state change event.
// This should be called by worker implementations when status changes.
func (b *BaseWorker) SetStatus(new Status) {
	b.mu.Lock()
	old := b.status
	b.status = new
	b.mu.Unlock()

	if old != new {
		b.emitStateChange(State{Name: b.name, Status: old}, State{Name: b.name, Status: new})
	}
}

// Finish is the terminal checkpoint for a worker. It centralizes the final state
// transition logic, metrics, and signaling.
func (b *BaseWorker) Finish(err error) {
	b.mu.Lock()
	// Capture the raw error
	b.Err = err

	// Determine and set terminal status
	oldStatus := b.status
	b.status = b.DeriveFinalStatus()
	newStatus := b.status
	b.mu.Unlock()

	// Emit state change for the terminal transition
	b.emitStateChange(State{Name: b.name, Status: oldStatus}, State{Name: b.name, Status: newStatus})

	// Signal completion
	// 1. Send error to buffered channel (single-result observer)
	b.done <- err
	close(b.done)

	// 2. Close finished channel (broadcast observer - multiple waiters allowed)
	close(b.finished)
}

// Stop satisfies the Worker interface. In BaseWorker, it only handles
// the "Strict Wait" protocol using the provided context.
// Embedding types should override this to trigger their specific cleanup
// (e.g. canceling a context or signaling a process) but should call
// this base implementation if they want to wait for quiescence.
func (b *BaseWorker) Stop(ctx context.Context) error {
	b.mu.Lock()
	b.StopRequested = true

	// If not yet running, return immediately (nothing to quiesce)
	if b.status == StatusCreated || b.status == StatusPending {
		b.mu.Unlock()
		return nil
	}
	b.mu.Unlock()

	// Wait for quiescence or timeout
	select {
	case <-b.finished:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Wait returns the done channel.
// The channel is populated by StartFunc and closed when the worker exits.
func (b *BaseWorker) Wait() <-chan error {
	return b.done
}

// String returns the worker name.
// This is used for logging and debugging.
func (b *BaseWorker) String() string {
	return b.name
}

// State returns the current worker state (base fields only).
func (b *BaseWorker) State() State {
	return b.ExportState(nil)
}

// ExportState allows embedding workers to safely construct their state while holding the lock.
// It builds the base state and passes it to the optional extension function 'fn'.
func (b *BaseWorker) ExportState(fn func(*State)) State {
	b.mu.RLock()
	defer b.mu.RUnlock()

	s := State{
		Name:   b.name,
		Status: b.status,
	}

	if fn != nil {
		fn(&s)
	}

	return s
}

// StartFunc is a helper that runs fn in a goroutine and manages the done channel.
// It's a common pattern for Start() implementations:
//
//	func (w *MyWorker) Start(ctx context.Context) error {
//	    return w.StartFunc(ctx, w.Run)
//	}
//
// The function result is sent to the done channel, then the channel is closed.
func (b *BaseWorker) StartFunc(ctx context.Context, fn func(context.Context) error) error {
	go func() {
		b.Finish(fn(ctx))
	}()
	return nil
}

// ComponentType returns the component type for introspection.
func (b *BaseWorker) ComponentType() string {
	return "worker"
}
