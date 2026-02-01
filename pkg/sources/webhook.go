package sources

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aretw0/lifecycle/pkg/control"
)

// WebhookEvent represents an HTTP triggering event.
type WebhookEvent struct {
	Method  string
	Path    string
	Payload []byte
}

func (e WebhookEvent) String() string {
	return fmt.Sprintf("webhook%s", e.Path) // e.g., "webhook/reload"
}

// WebhookSource listens for HTTP requests and converts them into lifecycle events.
type WebhookSource struct {
	addr   string
	server *http.Server
	events chan control.Event
}

// NewWebhookSource creates a new source listening on the given address (e.g., ":8080").
func NewWebhookSource(addr string) *WebhookSource {
	return &WebhookSource{
		addr:   addr,
		events: make(chan control.Event, 10), // TODO: Make this configurable?
	}
}

func (s *WebhookSource) Events() <-chan control.Event {
	return s.events
}

func (s *WebhookSource) Start(ctx context.Context) error {
	defer close(s.events)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		event := WebhookEvent{
			Method:  r.Method,
			Path:    r.URL.Path,
			Payload: body,
		}

		select {
		case s.events <- event:
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprintf(w, "Event accepted: %s\n", event)
		case <-ctx.Done():
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			w.WriteHeader(http.StatusTooManyRequests)
		}
	})

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(shutdownCtx)
	case err := <-errChan:
		return err
	}
}
