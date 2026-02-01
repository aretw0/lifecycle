package sources

import (
	"context"
	"fmt"

	"github.com/aretw0/lifecycle/pkg/control"
)

// WebhookEvent represents an HTTP triggering event.
type WebhookEvent struct {
	Payload string
}

func (e WebhookEvent) String() string {
	return fmt.Sprintf("Webhook(payload=%s)", e.Payload)
}

// WebhookSource listens for HTTP requests.
// TODO: Implement actual HTTP server listener or handler.
type WebhookSource struct {
	events chan control.Event
}

func NewWebhookSource() *WebhookSource {
	return &WebhookSource{
		events: make(chan control.Event),
	}
}

func (s *WebhookSource) Events() <-chan control.Event {
	return s.events
}

func (s *WebhookSource) Start(ctx context.Context) error {
	defer close(s.events)
	// TODO: Start HTTP server using lifecycle.Group?
	<-ctx.Done()
	return ctx.Err()
}
