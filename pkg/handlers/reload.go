package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/aretw0/lifecycle/pkg/control"
	"github.com/aretw0/lifecycle/pkg/metrics"
)

// ReloadHandler handles configuration reload events.
type ReloadHandler struct {
	// TODO: Add callback for actual reload logic
	OnReload func(ctx context.Context) error
}

func (r *ReloadHandler) HandleEvent(ctx context.Context, e control.Event) error {
	defer func(start time.Time) {
		metrics.GetProvider().ObserveHandlerDuration("reload", time.Since(start))
	}(time.Now())

	fmt.Printf("control: reloading config triggered by %s\n", e)
	if r.OnReload != nil {
		return r.OnReload(ctx)
	}
	return nil
}

func NewReload(onReload func(context.Context) error) control.Handler {
	return &ReloadHandler{OnReload: onReload}
}
