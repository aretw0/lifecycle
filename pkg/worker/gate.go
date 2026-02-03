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
	pauseRequest bool
	paused       bool
	resumeCh     chan struct{} // Channel used to signal resume
	quiescedCh   chan struct{} // Channel used to signal that worker reached quiescence
}

// NewQuiescenceGate creates a new gate ready for use.
func NewQuiescenceGate() *QuiescenceGate {
	return &QuiescenceGate{
		resumeCh:   make(chan struct{}),
		quiescedCh: make(chan struct{}),
	}
}

// Check should be called by the Worker before starting a new unit of work.
// If a pause was requested, this method blocks until resumed or context is cancelled.
func (g *QuiescenceGate) Check(ctx context.Context) error {
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

// RequestPause signals the worker to pause at the next safe opportunity.
func (g *QuiescenceGate) RequestPause() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.pauseRequest = true
}

// WaitPaused blocks until the worker has actually entered the paused state.
func (g *QuiescenceGate) WaitPaused(ctx context.Context) error {
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

// Resume wakes up the worker.
func (g *QuiescenceGate) Resume() {
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
