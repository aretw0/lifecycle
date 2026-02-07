package events

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestWebhookSource(t *testing.T) {
	// Start on random port
	source := NewWebhookSource(":0")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Capture events
	eventsChan := source.Events()
	errChan := make(chan error, 1) // Buffer to avoid blocking if Start returns quickly

	go func() {
		// Start blocks until context cancelled
		if err := source.Start(ctx); err != nil {
			errChan <- err
		}
		close(errChan)
	}()

	// Wait for listener to be active
	deadline := time.Now().Add(1 * time.Second)
	var url string
	for time.Now().Before(deadline) {
		// Use public thread-safe method
		addr := source.Addr()
		// Wait for the listener to bind to a concrete port (resolving ":0").
		if addr != ":0" && !strings.HasSuffix(addr, ":0") {
			url = fmt.Sprintf("http://%s/test/path", addr)
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if url == "" {
		t.Fatal("WebhookSource failed to start listener")
	}

	// Send POST request
	payload := "test-payload"
	resp, err := http.Post(url, "text/plain", strings.NewReader(payload))
	if err != nil {
		t.Fatalf("HTTP POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted { // 202
		t.Errorf("Expected status 202 Accepted, got %d", resp.StatusCode)
	}

	// Verify Event
	select {
	case e := <-eventsChan:
		we, ok := e.(WebhookEvent)
		if !ok {
			t.Fatalf("Expected WebhookEvent, got %T", e)
		}
		if we.Path != "/test/path" {
			t.Errorf("Expected path /test/path, got %s", we.Path)
		}
		if string(we.Payload) != payload {
			t.Errorf("Expected payload %s, got %s", payload, string(we.Payload))
		}
		if we.String() != "webhook/test/path" {
			t.Errorf("Unexpected String(): %s", we.String())
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for event")
	case err := <-errChan:
		t.Fatalf("Start returned error: %v", err)
	}

	// Test Shutdown
	cancel()
	select {
	case err := <-errChan:
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("Unexpected error during shutdown: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Min source Start did not return after context cancel")
	}
}
