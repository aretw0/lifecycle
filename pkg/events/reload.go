package events

import (
	"context"
	"fmt"
	"time"

	"github.com/aretw0/lifecycle/pkg/core/metrics"
)

// ReloadHandler handles configuration reload
type ReloadHandler struct {
	OnReload func(ctx context.Context) error
}

func (r *ReloadHandler) HandleEvent(ctx context.Context, e Event) error {
	defer func(start time.Time) {
		metrics.GetProvider().ObserveHandlerDuration("reload", time.Since(start))
	}(time.Now())

	fmt.Printf("control: reloading config triggered by %s\n", e)
	if r.OnReload != nil {
		return r.OnReload(ctx)
	}
	return nil
}

func NewReload(onReload func(context.Context) error) Handler {
	return &ReloadHandler{OnReload: onReload}
}
