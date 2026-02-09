package worker

import (
	"context"
	"log/slog"
	"sync"
)

// SuspendGate orchestrates safe suspension (pause) for a worker.
// It ensures the worker finishes its current unit of work before pausing,
// satisfying the strict quiescence requirement of the Suspendable interface.
type SuspendGate struct {
	mu           sync.Mutex
	pauseRequest bool
	paused       bool
	resumeCh     chan struct{} // Channel used to signal resume
	quiescedCh   chan struct{} // Channel used to signal that worker reached quiescence
}

// NewSuspendGate creates a new gate ready for use.
func NewSuspendGate() *SuspendGate {
	return &SuspendGate{
		resumeCh:   make(chan struct{}),
		quiescedCh: make(chan struct{}),
	}
}

// Check should be called by the Worker loop before starting a new unit of work.
// If a pause was requested, this method blocks until resumed or context is cancelled.
func (g *SuspendGate) Check(ctx context.Context) error {
	g.mu.Lock()

	// If no pause requested, just proceed
	if !g.pauseRequest {
		g.mu.Unlock()
		return nil
	}

	// Wait for resume or cancellation
	slog.Debug("lifecycle: worker quiescence reached, pausing")
	g.paused = true
	resCh := g.resumeCh

	// Signal WaitPaused callers that we are now paused
	close(g.quiescedCh)
	g.quiescedCh = make(chan struct{})
	g.mu.Unlock()

	select {
	case <-resCh:
		g.mu.Lock()
		g.paused = false
		g.mu.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Suspend signals the worker to pause at the next safe opportunity (the next Check call).
// It blocks until the worker loop confirms it has reached the paused state.
func (g *SuspendGate) Suspend(ctx context.Context) error {
	g.RequestPause()
	g.mu.Lock()
	if g.paused {
		g.mu.Unlock()
		return nil
	}
	qCh := g.quiescedCh
	g.mu.Unlock()

	select {
	case <-qCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// RequestPause signals the worker to pause at the next safe opportunity (the next Check call).
// Unlike Suspend, it does not block for confirmation.
func (g *SuspendGate) RequestPause() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.pauseRequest = true
}

// Resume wakes up the worker and allows it to proceed past the Check call.
func (g *SuspendGate) Resume() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.pauseRequest {
		return
	}

	slog.Debug("lifecycle: resuming worker")
	g.pauseRequest = false
	close(g.resumeCh)
	g.resumeCh = make(chan struct{})
}

// IsPaused returns true if the worker is currently suspended.
func (g *SuspendGate) IsPaused() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.paused
}
