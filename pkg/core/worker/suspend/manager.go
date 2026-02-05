package suspend

import (
	"context"
	"sync"
	"time"
)

// Manager provides channel-based suspend/resume with context cancellation support.
// It offers an alternative to sync.Cond for scenarios where context-aware waiting is needed.
//
// Example usage:
//
//	type Worker struct {
//	    lifecycle.BaseWorker
//	    suspendMgr *lifecycle.SuspendManager
//	}
//
//	func (w *Worker) Run(ctx context.Context) error {
//	    for {
//	        // Respect suspend with context cancellation
//	        if err := w.suspendMgr.Wait(ctx); err != nil {
//	            return err
//	        }
//	        // Do work...
//	    }
//	}
//
//	func (w *Worker) Suspend(ctx context.Context) error {
//	    w.suspendMgr.Pause()
//	    return nil
//	}
//
//	func (w *Worker) Resume(ctx context.Context) error {
//	    w.suspendMgr.Resume()
//	    return nil
//	}
type Manager struct {
	mu       sync.Mutex
	paused   bool
	resumeCh chan struct{}
}

// NewManager creates a new suspend manager in the running state.
func NewManager() *Manager {
	return &Manager{
		resumeCh: make(chan struct{}),
	}
}

// Pause requests that work should stop.
// This method is idempotent and safe to call multiple times.
func (m *Manager) Pause() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.paused {
		m.paused = true
		if m.resumeCh != nil {
			close(m.resumeCh)
			m.resumeCh = nil
		}
	}
}

// Resume allows work to continue after a pause.
// This method is idempotent and safe to call multiple times.
func (m *Manager) Resume() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.paused {
		m.paused = false
		m.resumeCh = make(chan struct{})
	}
}

// IsPaused returns true if currently paused.
func (m *Manager) IsPaused() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.paused
}

// Wait blocks until work can continue or the context is cancelled.
// Returns nil when resumed, or ctx.Err() if the context is cancelled.
//
// Call this at strategic points in your worker loop:
//
//	for {
//	    if err := mgr.Wait(ctx); err != nil {
//	        return err // Context cancelled during pause
//	    }
//	    // Continue working...
//	}
//
// Performance: Uses polling with 50ms interval when paused.
// For high-frequency suspend/resume (>10k/sec), consider sync.Cond instead.
func (m *Manager) Wait(ctx context.Context) error {
	// Fast path: if not paused, return immediately
	m.mu.Lock()
	if !m.paused {
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()

	// Paused: wait for resume or context cancellation
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			m.mu.Lock()
			paused := m.paused
			m.mu.Unlock()

			if !paused {
				return nil // Resumed
			}
		}
	}
}



