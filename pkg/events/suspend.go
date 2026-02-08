package events

import (
	"context"
	"sync"

	"github.com/aretw0/lifecycle/pkg/core/log"
	"github.com/aretw0/lifecycle/pkg/core/worker"
)

// SuspendHook is a function called when a suspend/resume event occurs.
type SuspendHook func(ctx context.Context) error

// SuspendHandler manages Suspend and Resume
// It allows registering hooks that are executed when these events occur.
type SuspendHandler struct {
	mu            sync.RWMutex
	onSuspend     []SuspendHook
	onResume      []SuspendHook
	suspended     bool
	transitioning bool
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

	// If already transitioning, ignore to prevent duplicate hook execution
	if h.transitioning {
		h.mu.Unlock()
		log.Debug("lifecycle: already transitioning state")
		return nil
	}

	var hooks []SuspendHook
	var nextState bool
	var logMsg string

	switch e.(type) {
	case SuspendEvent, InterceptEvent:
		if h.suspended {
			h.mu.Unlock()
			log.Debug("lifecycle: already suspended/intercepted")
			return nil
		}
		logMsg = "lifecycle: suspending application"
		if _, ok := e.(InterceptEvent); ok {
			logMsg = "lifecycle: intercepting application"
		}
		hooks = append([]SuspendHook(nil), h.onSuspend...)
		nextState = true

	case ResumeEvent:
		if !h.suspended {
			h.mu.Unlock()
			log.Debug("lifecycle: not suspended")
			return nil
		}
		logMsg = "lifecycle: resuming application"
		hooks = append([]SuspendHook(nil), h.onResume...)
		nextState = false

	default:
		h.mu.Unlock()
		return nil
	}

	// Mark as transitioning before unlocking to run hooks
	h.transitioning = true
	h.mu.Unlock()

	log.Info(logMsg)
	err := h.executeHooks(ctx, hooks)

	h.mu.Lock()
	h.transitioning = false
	if err == nil {
		h.suspended = nextState
	}
	h.mu.Unlock()

	return err
}

func (h *SuspendHandler) executeHooks(ctx context.Context, hooks []SuspendHook) error {
	for _, hook := range hooks {
		if err := hook(ctx); err != nil {
			return err
		}
	}
	return nil
}

// IsActive returns true if the system is currently suspended.
// Implements StateChecker interface.
func (h *SuspendHandler) IsActive() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.suspended
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
