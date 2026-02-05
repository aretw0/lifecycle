package worker

import (
	"context"
	"sync"

	"github.com/aretw0/lifecycle/pkg/introspection"
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
	name   string
	done   chan error
	status Status
	mu     sync.RWMutex

	// StateWatchers (Event-Driven Introspection)
	stateWatchers []chan introspection.StateChange[State]
	watchersMu    sync.RWMutex
}

// NewBaseWorker creates a new BaseWorker with the given name.
// The name is immutable after creation (construct a new worker to change it).
func NewBaseWorker(name string) BaseWorker {
	return BaseWorker{
		name:   name,
		done:   make(chan error, 1),
		status: StatusCreated,
	}
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

// Stop is a no-op implementation.
// Most workers rely on context cancellation for cleanup.
// Override this method if your worker needs explicit stop logic.
func (b *BaseWorker) Stop(ctx context.Context) error {
	return nil
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

// State returns the current worker state with thread-safe access.
// DO NOT OVERRIDE THIS METHOD. Override buildState() instead.
func (b *BaseWorker) State() State {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.buildState()
}

// buildState constructs the worker state.
// Override this method in subclasses to add custom fields.
// The caller (State()) already holds the read lock, so this is safe.
func (b *BaseWorker) buildState() State {
	return State{
		Name:   b.name,
		Status: b.status,
	}
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
		b.done <- fn(ctx)
		close(b.done)
	}()
	return nil
}
