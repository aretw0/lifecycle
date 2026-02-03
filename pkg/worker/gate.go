package worker

import (
	"context"
	"log/slog"
	"sync"
)

// QuiescenceGate orchestrates safe suspension (pause) for a worker.
// It ensures the worker finishes its current unit of work before pausing.
type QuiescenceGate struct {
	mu           sync.Mutex
	cond         *sync.Cond
	pauseRequest bool
	paused       bool
}

// NewQuiescenceGate creates a new gate ready for use.
func NewQuiescenceGate() *QuiescenceGate {
	g := &QuiescenceGate{}
	g.cond = sync.NewCond(&g.mu)
	return g
}

// Check should be called by the Worker before starting a new unit of work.
// If a pause was requested, this method blocks until resumed.
// Returns an error if context is cancelled while waiting.
func (g *QuiescenceGate) Check(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// If pause was requested, enter paused state
	if g.pauseRequest {
		slog.Debug("lifecycle: worker quiescence reached, pausing")
		g.paused = true
		g.cond.Broadcast() // Wake up any WaitPaused callers
	}

	for g.paused {
		// Check context before/during wait to avoid hanging
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// TODO: cond.Wait() doesn't respect context cancellation automatically.
		// To be truly robust we'd need a separate channel or periodic check.
		// For now, we rely on Resume/Broadcast to wake us up, or external cancellation
		// if the worker loop checks context periodically.
		// A robust implementation might use a WaitWithContext pattern if needed.
		g.cond.Wait()
	}

	return nil
}

// RequestPause signals the worker to pause at the next safe opportunity.
func (g *QuiescenceGate) RequestPause() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.pauseRequest = true
	// We don't broadcast here because we are asking the worker to notice flag...
	// but if the worker is currently blocked on Check() (already paused), strictly it wouldn't be.
}

// WaitPaused blocks until the worker has actually entered the paused state.
func (g *QuiescenceGate) WaitPaused(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	for !g.paused {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		g.cond.Wait()
	}
	return nil
}

// Resume wakes up the worker.
func (g *QuiescenceGate) Resume() {
	g.mu.Lock()
	defer g.mu.Unlock()

	slog.Debug("lifecycle: resuming worker")
	g.pauseRequest = false
	g.paused = false
	g.cond.Broadcast() // Wake up the worker blocked in Check()
}
