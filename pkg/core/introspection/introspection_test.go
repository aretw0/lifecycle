package introspection

import (
	"context"
	"strings"
	"testing"
	"time"
)

// MockState is a simple state for testing.
type MockState struct {
	Value string
}

// MockComponent implements ComponentEvent for testing.
type MockComponent struct {
	id       string
	compType string
	ts       time.Time
	eType    string
}

func (m *MockComponent) ComponentID() string {
	return m.id
}

func (m *MockComponent) ComponentType() string {
	return m.compType
}

func (m *MockComponent) Timestamp() time.Time {
	return m.ts
}

func (m *MockComponent) EventType() string {
	return m.eType
}

// MockIntrospectable implements Introspectable for testing.
type MockIntrospectable struct {
	state any
}

func (m *MockIntrospectable) State() any {
	return m.state
}

// MockTypedWatcher implements TypedWatcher for testing.
type MockTypedWatcher[S any] struct {
	compType     string
	currentState S
	changeChan   chan StateChange[S]
}

func NewMockTypedWatcher[S any](initialState S) *MockTypedWatcher[S] {
	return &MockTypedWatcher[S]{
		compType:     "worker", // Default for valid aggregation
		currentState: initialState,
		changeChan:   make(chan StateChange[S], 10),
	}
}

func (m *MockTypedWatcher[S]) ComponentType() string {
	return m.compType
}

func (m *MockTypedWatcher[S]) State() S {
	return m.currentState
}

func (m *MockTypedWatcher[S]) Watch(ctx context.Context) <-chan StateChange[S] {
	ch := make(chan StateChange[S], 10)

	go func() {
		defer close(ch)

		for {
			select {
			case change := <-m.changeChan:
				select {
				case ch <- change:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch
}

func (m *MockTypedWatcher[S]) SendChange(change StateChange[S]) {
	// Ensure tests that don't set ComponentID get a deterministic id.
	if change.ComponentID == "" {
		change.ComponentID = "test-id"
	}

	m.changeChan <- change
}

// MockEventSource implements EventSource for testing.
type MockEventSource struct {
	eventChan chan ComponentEvent
}

func NewMockEventSource() *MockEventSource {
	return &MockEventSource{
		eventChan: make(chan ComponentEvent, 10),
	}
}

func (m *MockEventSource) Events(ctx context.Context) <-chan ComponentEvent {
	ch := make(chan ComponentEvent, 10)

	go func() {
		defer close(ch)

		for {
			select {
			case event := <-m.eventChan:
				select {
				case ch <- event:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch
}

func (m *MockEventSource) SendEvent(event ComponentEvent) {
	m.eventChan <- event
}

func TestStateChange_Fields(t *testing.T) {
	now := time.Now()
	oldState := MockState{Value: "old"}
	newState := MockState{Value: "new"}

	change := StateChange[MockState]{
		ComponentID:   "comp-1",
		ComponentType: "worker",
		OldState:      oldState,
		NewState:      newState,
		Timestamp:     now,
	}

	if change.ComponentID != "comp-1" {
		t.Errorf("ComponentID = %q, want %q", change.ComponentID, "comp-1")
	}

	if change.ComponentType != "worker" {
		t.Errorf("ComponentType = %q, want %q", change.ComponentType, "worker")
	}

	if change.OldState.Value != "old" {
		t.Errorf("OldState.Value = %q, want %q", change.OldState.Value, "old")
	}

	if change.NewState.Value != "new" {
		t.Errorf("NewState.Value = %q, want %q", change.NewState.Value, "new")
	}

	if !change.Timestamp.Equal(now) {
		t.Errorf("Timestamp = %v, want %v", change.Timestamp, now)
	}
}

func TestStateSnapshot_Fields(t *testing.T) {
	now := time.Now()
	payload := MockState{Value: "test"}

	snapshot := StateSnapshot{
		ComponentID:   "snap-1",
		ComponentType: "supervisor",
		Timestamp:     now,
		Payload:       payload,
	}

	if snapshot.ComponentID != "snap-1" {
		t.Errorf("ComponentID = %q, want %q", snapshot.ComponentID, "snap-1")
	}

	if snapshot.ComponentType != "supervisor" {
		t.Errorf("ComponentType = %q, want %q", snapshot.ComponentType, "supervisor")
	}

	if !snapshot.Timestamp.Equal(now) {
		t.Errorf("Timestamp = %v, want %v", snapshot.Timestamp, now)
	}

	state, ok := snapshot.Payload.(MockState)
	if !ok {
		t.Errorf("Payload is not MockState")
	}

	if state.Value != "test" {
		t.Errorf("Payload.Value = %q, want %q", state.Value, "test")
	}
}

func TestStateSnapshot_PayloadTypes(t *testing.T) {
	tests := []struct {
		name    string
		payload any
	}{
		{name: "string", payload: "test"},
		{name: "int", payload: 42},
		{name: "struct", payload: MockState{Value: "data"}},
		{name: "nil", payload: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := StateSnapshot{
				Payload: tt.payload,
			}

			if snapshot.Payload != tt.payload {
				t.Errorf("Payload mismatch")
			}
		})
	}
}

func TestMockComponent_Implements_ComponentEvent(t *testing.T) {
	var _ ComponentEvent = (*MockComponent)(nil)
}

func TestComponentEvent_MockComponent(t *testing.T) {
	now := time.Now()
	comp := &MockComponent{
		id:       "comp-1",
		compType: "worker",
		ts:       now,
		eType:    "started",
	}

	if comp.ComponentID() != "comp-1" {
		t.Errorf("ComponentID() = %q, want %q", comp.ComponentID(), "comp-1")
	}

	if comp.ComponentType() != "worker" {
		t.Errorf("ComponentType() = %q, want %q", comp.ComponentType(), "worker")
	}

	if !comp.Timestamp().Equal(now) {
		t.Errorf("Timestamp() = %v, want %v", comp.Timestamp(), now)
	}

	if comp.EventType() != "started" {
		t.Errorf("EventType() = %q, want %q", comp.EventType(), "started")
	}
}

func TestIntrospectable_MockIntrospectable(t *testing.T) {
	state := MockState{Value: "test"}
	introspectable := &MockIntrospectable{state: state}

	if introspectable.State() != state {
		t.Error("State() did not return expected value")
	}
}

func TestIntrospectable_Different_States(t *testing.T) {
	tests := []struct {
		name  string
		state any
	}{
		{name: "string", state: "test"},
		{name: "int", state: 42},
		{name: "struct", state: MockState{Value: "data"}},
		{name: "nil", state: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			introspectable := &MockIntrospectable{state: tt.state}

			if introspectable.State() != tt.state {
				t.Errorf("State() did not return %v", tt.state)
			}
		})
	}
}

func TestStateChange_Multiple_Types(t *testing.T) {
	// String states
	change1 := StateChange[string]{
		OldState: "old",
		NewState: "new",
	}

	if change1.OldState != "old" || change1.NewState != "new" {
		t.Error("String StateChange failed")
	}

	// Int states
	change2 := StateChange[int]{
		OldState: 1,
		NewState: 2,
	}

	if change2.OldState != 1 || change2.NewState != 2 {
		t.Error("Int StateChange failed")
	}
}

func TestStateSnapshot_Empty_Payload(t *testing.T) {
	snapshot := StateSnapshot{
		Payload: nil,
	}

	if snapshot.Payload != nil {
		t.Error("Expected nil Payload")
	}
}

func TestComponentTypes_Constants(t *testing.T) {
	types := []string{"worker", "signal", "supervisor"}

	for _, compType := range types {
		snapshot := StateSnapshot{
			ComponentType: compType,
		}

		if snapshot.ComponentType != compType {
			t.Errorf("ComponentType = %q, want %q", snapshot.ComponentType, compType)
		}
	}
}

func TestTypedWatcher_MockTypedWatcher(t *testing.T) {
	initialState := MockState{Value: "initial"}
	watcher := NewMockTypedWatcher(initialState)

	if watcher.State() != initialState {
		t.Error("State() did not return initial state")
	}
}

func TestTypedWatcher_Watch_Channel_Closed(t *testing.T) {
	watcher := NewMockTypedWatcher(MockState{Value: "test"})
	ctx, cancel := context.WithCancel(context.Background())

	ch := watcher.Watch(ctx)
	if ch == nil {
		t.Error("Watch() returned nil channel")
	}

	// Cancel the context and verify channel closes
	cancel()

	time.Sleep(50 * time.Millisecond) // Let goroutine finish

	// Confirm channel is closed by reading from it
	_, ok := <-ch
	if ok {
		t.Error("Channel should be closed after context cancellation")
	}
}

func TestWatcherAdapter_NewWatcherAdapter(t *testing.T) {
	watcher := NewMockTypedWatcher(MockState{Value: "test"})
	adapter := NewWatcherAdapter("worker", watcher)

	if adapter == nil {
		t.Error("NewWatcherAdapter returned nil")
	}
}

func TestWatcherAdapter_Snapshots(t *testing.T) {
	watcher := NewMockTypedWatcher(MockState{Value: "initial"})
	adapter := NewWatcherAdapter("worker", watcher)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	snapshots := adapter.Snapshots(ctx)
	if snapshots == nil {
		t.Error("Snapshots() returned nil channel")
	}

	// Send a state change
	change := StateChange[MockState]{
		OldState: MockState{Value: "initial"},
		NewState: MockState{Value: "updated"},
	}

	watcher.SendChange(change)

	// Verify we receive a StateSnapshot
	select {
	case snapshot, ok := <-snapshots:
		if !ok {
			t.Error("Snapshot channel was closed unexpectedly")
		}

		if snapshot.ComponentType != "worker" {
			t.Errorf("ComponentType = %q, want %q", snapshot.ComponentType, "worker")
		}

		if snapshot.ComponentID != "test-id" {
			t.Errorf("ComponentID = %q, want %q", snapshot.ComponentID, "test-id")
		}

		if snapshot.Payload != change.NewState {
			t.Error("Payload does not match NewState")
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for snapshot")
	}
}

func TestEventSource_MockEventSource(t *testing.T) {
	source := NewMockEventSource()
	if source == nil {
		t.Error("NewMockEventSource returned nil")
	}
}

func TestEventSource_Events_Channel(t *testing.T) {
	source := NewMockEventSource()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := source.Events(ctx)
	if events == nil {
		t.Error("Events() returned nil channel")
	}

	// Send an event
	event := &MockComponent{
		id:       "comp-1",
		compType: "worker",
		ts:       time.Now(),
		eType:    "started",
	}

	source.SendEvent(event)

	// Verify we receive the event
	select {
	case received, ok := <-events:
		if !ok {
			t.Error("Event channel was closed unexpectedly")
		}

		if received.ComponentID() != "comp-1" {
			t.Errorf("ComponentID() = %q, want %q", received.ComponentID(), "comp-1")
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for event")
	}
}

func TestTypedWatcher_Implements_Interface(t *testing.T) {
	var _ TypedWatcher[MockState] = (*MockTypedWatcher[MockState])(nil)
}

func TestEventSource_Implements_Interface(t *testing.T) {
	var _ EventSource = (*MockEventSource)(nil)
}

// Tests for AggregateWatchers
// Note: Full aggregation tested via WatcherAdapter integration
func TestAggregateWatchers_Nil_Input(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Should handle nil input gracefully
	snapshots := AggregateWatchers(ctx, nil)
	if snapshots == nil {
		t.Error("AggregateWatchers returned nil channel")
	}

	cancel()

	// Channel should close cleanly
	_, ok := <-snapshots
	if ok {
		t.Error("Channel should be closed")
	}
}

func TestAggregateWatchers_Context_Cancellation(t *testing.T) {
	watcher := NewMockTypedWatcher(MockState{Value: "test"})
	ctx, cancel := context.WithCancel(context.Background())

	snapshots := AggregateWatchers(ctx, watcher)

	// Cancel context
	cancel()

	time.Sleep(50 * time.Millisecond)

	// Channel should close
	_, ok := <-snapshots
	if ok {
		t.Error("Channel should be closed after context cancellation")
	}
}

// Tests for AggregateEvents
func TestAggregateEvents_Single_Source(t *testing.T) {
	source := NewMockEventSource()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := AggregateEvents(ctx, source)
	if events == nil {
		t.Error("AggregateEvents returned nil channel")
	}

	// Send an event
	event := &MockComponent{
		id:       "comp-1",
		compType: "worker",
		ts:       time.Now(),
		eType:    "started",
	}

	source.SendEvent(event)

	// Verify we receive the event
	select {
	case received, ok := <-events:
		if !ok {
			t.Error("Event channel was closed unexpectedly")
		}

		if received.ComponentID() != "comp-1" {
			t.Errorf("ComponentID() = %q, want %q", received.ComponentID(), "comp-1")
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for event")
	}
}

func TestAggregateEvents_Multiple_Sources(t *testing.T) {
	source1 := NewMockEventSource()
	source2 := NewMockEventSource()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := AggregateEvents(ctx, source1, source2)

	// Send events from both sources
	source1.SendEvent(&MockComponent{
		id:       "comp-1",
		compType: "worker",
		ts:       time.Now(),
		eType:    "started",
	})

	source2.SendEvent(&MockComponent{
		id:       "comp-2",
		compType: "supervisor",
		ts:       time.Now(),
		eType:    "restarted",
	})

	// Expect to receive events from both
	received := 0
	timeout := time.After(2 * time.Second)

	for received < 2 {
		select {
		case event, ok := <-events:
			if !ok {
				t.Error("Event channel was closed unexpectedly")
			}

			if event.ComponentID() == "comp-1" || event.ComponentID() == "comp-2" {
				received++
			}
		case <-timeout:
			t.Errorf("Timeout waiting for events. Received: %d, expected: 2", received)
			return
		}
	}
}

func TestAggregateEvents_Context_Cancellation(t *testing.T) {
	source := NewMockEventSource()
	ctx, cancel := context.WithCancel(context.Background())

	events := AggregateEvents(ctx, source)

	// Cancel context
	cancel()

	time.Sleep(50 * time.Millisecond)

	// Channel should close
	_, ok := <-events
	if ok {
		t.Error("Channel should be closed after context cancellation")
	}
}

// Tests for Mermaid functions
func TestDefaultStyles_Returns_NonEmpty_String(t *testing.T) {
	styles := DefaultStyles()
	if styles == "" {
		t.Error("DefaultStyles() returned empty string")
	}

	// Verify it contains expected Mermaid class definitions
	if !strings.Contains(styles, "classDef") {
		t.Error("DefaultStyles() should contain Mermaid class definitions")
	}

	// Verify it contains expected state classes
	expectedClasses := []string{"created", "running", "stopped", "failed", "supervisor"}
	for _, cls := range expectedClasses {
		if !strings.Contains(styles, cls) {
			t.Errorf("DefaultStyles() should contain class definition for %q", cls)
		}
	}
}

func TestWithStyles_Option(t *testing.T) {
	customStyles := "classDef custom fill:#fff;"
	opts := &MermaidOptions{}
	option := WithStyles(customStyles)
	option(opts)

	if opts.Styles != customStyles {
		t.Errorf("WithStyles() should set custom styles, got %q", opts.Styles)
	}
}

// Helper function tests (used internally by Mermaid functions)
// Note: These functions are tested indirectly through public APIs

// Tests for WatcherAdapter functionality
func TestWatcherAdapter_Integration_With_Aggregator(t *testing.T) {
	watcher := NewMockTypedWatcher(MockState{Value: "initial"})
	adapter := NewWatcherAdapter("worker", watcher)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get snapshots from adapter
	snapshots := adapter.Snapshots(ctx)

	// Send a change
	watcher.SendChange(StateChange[MockState]{
		ComponentID:   "worker-1",
		ComponentType: "worker",
		OldState:      MockState{Value: "old"},
		NewState:      MockState{Value: "new"},
	})

	// Verify we get a snapshot with correct ComponentType from adapter
	select {
	case snapshot, ok := <-snapshots:
		if !ok {
			t.Error("Snapshot channel closed unexpectedly")
		}

		if snapshot.ComponentType != "worker" {
			t.Errorf("ComponentType = %q, want %q", snapshot.ComponentType, "worker")
		}

		if snapshot.ComponentID != "worker-1" {
			t.Errorf("ComponentID = %q, want %q", snapshot.ComponentID, "worker-1")
		}

		if snapshot.Payload != (MockState{Value: "new"}) {
			t.Error("Payload should match NewState")
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for snapshot")
	}
}

func TestAggregateWatchers_Consume(t *testing.T) {
	watcher := NewMockTypedWatcher(MockState{Value: "initial"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	snapshots := AggregateWatchers(ctx, watcher)

	// Send a change
	newState := MockState{Value: "updated"}
	watcher.SendChange(StateChange[MockState]{
		ComponentID:   "test-id",
		ComponentType: "worker",
		OldState:      MockState{Value: "initial"},
		NewState:      newState,
		Timestamp:     time.Now(),
	})

	select {
	case snap := <-snapshots:
		if snap.ComponentID != "test-id" {
			t.Errorf("Expected ComponentID test-id, got %s", snap.ComponentID)
		}
		if snap.ComponentType != "worker" {
			t.Errorf("Expected ComponentType worker, got %s", snap.ComponentType)
		}
		state, ok := snap.Payload.(MockState)
		if !ok || state.Value != "updated" {
			t.Errorf("Expected payload updated, got %v", snap.Payload)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for snapshot")
	}
}

func TestAggregateWatchers_InvalidInputs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Struct without Watch method
	snapshots := AggregateWatchers(ctx, struct{}{})
	if snapshots == nil {
		t.Error("AggregateWatchers returned nil channel")
	}
	if _, ok := <-snapshots; ok {
		t.Error("Expected closed channel for empty struct")
	}

	// 2. Struct with wrong package path
	snapshots = AggregateWatchers(ctx, time.Time{})
	if _, ok := <-snapshots; ok {
		t.Error("Expected closed channel for time.Time")
	}
}
