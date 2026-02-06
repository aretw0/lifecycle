package events

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestWebhookSource(t *testing.T) {
	source := NewWebhookSource("127.0.0.1:0") // Explicit local binding

	// We need to trigger the server start
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Capture events
	ch := source.Events()

	go func() {
		_ = source.Start(ctx)
	}()

	// Wait for server to be ready (it's async)
	time.Sleep(100 * time.Millisecond)

	addr := source.Addr()

	// Send a request
	payload := map[string]string{"foo": "bar"}
	body, _ := json.Marshal(payload)

	resp, err := http.Post("http://"+addr+"/webhook", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to send webhook: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("Expected 202 Accepted, got %d", resp.StatusCode)
	}

	// Verify event
	select {
	case ev := <-ch:
		wev, ok := ev.(WebhookEvent)
		if !ok {
			t.Fatalf("Expected WebhookEvent, got %T", ev)
		}
		if wev.Path != "/webhook" {
			t.Errorf("Expected path /webhook, got %s", wev.Path)
		}
		if !bytes.Equal(wev.Payload, body) {
			t.Errorf("Expected payload %s, got %s", string(body), string(wev.Payload))
		}
	case <-time.After(1 * time.Second):
		t.Error("Timed out waiting for webhook event")
	}
}
