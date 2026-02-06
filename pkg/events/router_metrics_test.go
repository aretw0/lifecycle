package events

import (
	"context"
	"testing"

	"github.com/aretw0/lifecycle/pkg/core/metrics"
	"github.com/aretw0/lifecycle/pkg/core/metrics/mock"
)

// TestRouterWithMockMetricsProvider validates that router dispatch correctly
// emits events routed and handler executed metrics using the mock provider.
func TestRouterWithMockMetricsProvider(t *testing.T) {
	// Setup: inject mock provider
	mockProvider := mock.New()
	oldProvider := metrics.GetProvider()
	defer metrics.SetProvider(oldProvider)
	metrics.SetProvider(mockProvider)

	// Create router and register handler
	router := NewRouter()
	ctx := context.Background()
	eventType := "test.dispatch"

	router.HandleFunc(eventType, func(_ context.Context, e Event) error {
		return nil
	})

	// Dispatch event
	event := mockEvent{eventType}
	router.Dispatch(ctx, event)

	// Verify metrics were incremented by checking the map
	if count, ok := mockProvider.EventsRouted[eventType]; !ok || count != 1 {
		t.Errorf("EventsRouted[%s]: expected 1, got %v", eventType, count)
	}

	if count, ok := mockProvider.HandlersExecuted[eventType]; !ok || count != 1 {
		t.Errorf("HandlersExecuted[%s]: expected 1, got %v", eventType, count)
	}
}

// TestRouterWithMockMetricsHandlerError validates that handler errors increment
// the handler error metric.
func TestRouterWithMockMetricsHandlerError(t *testing.T) {
	mockProvider := mock.New()
	oldProvider := metrics.GetProvider()
	defer metrics.SetProvider(oldProvider)
	metrics.SetProvider(mockProvider)

	router := NewRouter()
	ctx := context.Background()
	eventType := "test.error"

	router.HandleFunc(eventType, func(_ context.Context, e Event) error {
		return &testError{"test failure"}
	})

	event := mockEvent{eventType}
	router.Dispatch(ctx, event)

	if count, ok := mockProvider.HandlerErrors[eventType]; !ok || count != 1 {
		t.Errorf("HandlerErrors[%s]: expected 1, got %v", eventType, count)
	}
}

// TestRouterDispatchNoMatchWithMetrics verifies that unmatched events only
// increment EventsRouted, not HandlersExecuted.
func TestRouterDispatchNoMatchWithMetrics(t *testing.T) {
	mockProvider := mock.New()
	oldProvider := metrics.GetProvider()
	defer metrics.SetProvider(oldProvider)
	metrics.SetProvider(mockProvider)

	router := NewRouter()
	ctx := context.Background()

	// Register handler for pattern "registered.*"
	router.HandleFunc("registered.*", func(_ context.Context, e Event) error {
		return nil
	})

	// Dispatch event that doesn't match
	event := mockEvent{"unregistered.event"}
	router.Dispatch(ctx, event)

	// Event was routed but no handler matched
	if count, ok := mockProvider.EventsRouted["unregistered.event"]; !ok || count != 1 {
		t.Errorf("EventsRouted[unregistered.event]: expected 1, got %v", count)
	}

	// No handler should be logged as executed
	if count, ok := mockProvider.HandlersExecuted["unregistered.event"]; ok && count > 0 {
		t.Errorf("HandlersExecuted[unregistered.event] should be 0, got %d", count)
	}
}

// testError is a simple error type for testing.
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
