package events

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
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

// WebhookSource listens for HTTP requests and converts them into lifecycle 
type WebhookSource struct {
	BaseSource
	addr   string
	server *http.Server
}

// WebhookOption configures a WebhookSource.
type WebhookOption func(*WebhookSource)

// NewWebhookSource creates a new source listening on the given address (e.g., ":8080").
func NewWebhookSource(addr string, opts ...WebhookOption) *WebhookSource {
	s := &WebhookSource{
		BaseSource: NewBaseSource("webhook", 10),
		addr:       addr,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *WebhookSource) Start(ctx context.Context) error {
	defer s.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		event := WebhookEvent{
			Method:  r.Method,
			Path:    r.URL.Path,
			Payload: body,
		}

		select {
		case <-ctx.Done():
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			_ = s.Emit(ctx, event) // Best effort in HTTP handler, or should we use TryEmit?
			// Using Emit(ctx) is safer for backpressure but might block the HTTP goroutine.
			// Given it's a webhook, backpressure is likely desired.
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprintf(w, "Event accepted: %s\n", event)
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


