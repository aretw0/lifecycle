package events

import (
	"context"
	"fmt"
	"sync"

	"github.com/aretw0/lifecycle/pkg/core/log"
	"github.com/aretw0/lifecycle/pkg/core/worker"
)

// SuspendHook is a function called when a suspend/resume event occurs.
type SuspendHook func(ctx context.Context) error

// SuspendHandler manages Suspend and Resume 
// It allows registering hooks that are executed when these events occur.
type SuspendHandler struct {
	mu        sync.RWMutex
	onSuspend []SuspendHook
	onResume  []SuspendHook
	suspended bool
}

// NewSuspendHandler creates a new handler for suspend/resume 
func NewSuspendHandler() *SuspendHandler {
	return &SuspendHandler{}
}

// OnSuspend adds a hook to be executed on suspend.
func (h *SuspendHandler) OnSuspend(fn SuspendHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onSuspend = append(h.onSuspend, fn)
}

// OnResume adds a hook to be executed on resume.
func (h *SuspendHandler) OnResume(fn SuspendHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onResume = append(h.onResume, fn)
}

// Manage registers a worker.Suspendable component to be managed by this handler.
// It automatically wires up the Suspend and Resume methods to the respective 
func (h *SuspendHandler) Manage(s worker.Suspendable) {
	h.OnSuspend(s.Suspend)
	h.OnResume(s.Resume)
}

// HandleEvent processes SuspendEvent and ResumeEvent.
func (h *SuspendHandler) HandleEvent(ctx context.Context, e Event) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	switch e.(type) {
	case SuspendEvent:
		if h.suspended {
			log.Debug("lifecycle: already suspended")
			return nil
		}
		log.Info("lifecycle: suspending application")
		if err := h.executeHooks(ctx, h.onSuspend); err != nil {
			return fmt.Errorf("suspend failed: %w", err)
		}
		h.suspended = true
		return nil

	case ResumeEvent:
		if !h.suspended {
			log.Debug("lifecycle: not suspended")
			return nil
		}
		log.Info("lifecycle: resuming application")
		if err := h.executeHooks(ctx, h.onResume); err != nil {
			return fmt.Errorf("resume failed: %w", err)
		}
		h.suspended = false
		return nil

	default:
		return nil
	}
}

func (h *SuspendHandler) executeHooks(ctx context.Context, hooks []SuspendHook) error {
	for _, hook := range hooks {
		if err := hook(ctx); err != nil {
			return err
		}
	}
	return nil
}

// State returns the current state of the handler.
func (h *SuspendHandler) State() any {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return map[string]any{
		"suspended":     h.suspended,
		"suspend_hooks": len(h.onSuspend),
		"resume_hooks":  len(h.onResume),
	}
}


