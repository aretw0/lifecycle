package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/aretw0/lifecycle/pkg/control"
)

// ErrSuspendNotImplemented is returned when a suspend event is received but not supported.
var ErrSuspendNotImplemented = errors.New("lifecycle: suspend not implemented")

// SuspendHandler acknowledges a suspend event but currently returns an error
// indicating that durable execution suspension is not yet supported.
type SuspendHandler struct{}

// NewSuspendHandler creates a new handler for suspend events.
func NewSuspendHandler() *SuspendHandler {
	return &SuspendHandler{}
}

// HandleEvent logs a warning and returns ErrSuspendNotImplemented.
func (h *SuspendHandler) HandleEvent(ctx context.Context, e control.Event) error {
	// TODO: Integrate with Durable Execution engine (when available)
	// For now, we just acknowledge the intent.
	fmt.Printf("lifecycle: received suspend request from %s (not supported yet)\n", e)
	return ErrSuspendNotImplemented
}
