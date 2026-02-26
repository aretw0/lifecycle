package events

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync" // Added import
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

// WebhookSource listens for HTTP requests and converts them into lifecycle events.
type WebhookSource struct {
	BaseSource
	addr            string
	maxPayloadBytes int64
	mu              sync.Mutex
	ln              net.Listener
	server          *http.Server
}

// WebhookOption configures a WebhookSource.
type WebhookOption func(*WebhookSource)

// WithMaxPayloadBytes configures the maximum request body size in bytes.
// Default is 1MB to prevent OOM attacks.
func WithMaxPayloadBytes(n int64) WebhookOption {
	return func(s *WebhookSource) {
		s.maxPayloadBytes = n
	}
}

// NewWebhookSource creates a new source listening on the given address (e.g., ":8080").
func NewWebhookSource(addr string, opts ...WebhookOption) *WebhookSource {
	s := &WebhookSource{
		BaseSource:      NewBaseSource("webhook", 10),
		addr:            addr,
		maxPayloadBytes: 1024 * 1024, // 1MB default limit
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Addr returns the address the source is listening on.
// This is useful when using dynamic ports (":0").
func (s *WebhookSource) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ln != nil {
		return s.ln.Addr().String()
	}
	return s.addr
}

func (s *WebhookSource) Start(ctx context.Context) error {
	defer s.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, s.maxPayloadBytes)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
			return
		}
		event := WebhookEvent{
			Method:  r.Method,
			Path:    r.URL.Path,
			Payload: body,
		}

		select {
		case <-ctx.Done():
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			// Emit blocks to enforce backpressure on the HTTP client.
			if err := s.Emit(ctx, event); err != nil {
				http.Error(w, "Server shutting down", http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprintf(w, "Event accepted: %s\n", event)
		}
	})

	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.ln = ln
	s.mu.Unlock()

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := s.server.Serve(s.ln); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return s.server.Shutdown(shutdownCtx)
	case err := <-errChan:
		return err
	}
}
