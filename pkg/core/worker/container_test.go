package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aretw0/lifecycle/pkg/core/container"
)

func TestContainerWorker(t *testing.T) {
	mock := container.NewMockContainer("test-container")
	cw := NewContainerWorker("worker-1", mock)

	ctx := context.Background()

	// Start the worker
	if err := cw.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Verify mock state
	if mock.Status() != container.StatusRunning {
		t.Errorf("Mock container should be running, got %v", mock.Status())
	}

	// Check State inspection
	state := cw.State()
	if state.Name != "worker-1" {
		t.Errorf("Expected name worker-1, got %s", state.Name)
	}
	if state.Status != StatusRunning {
		t.Errorf("Expected status Running, got %v", state.Status)
	}
	if state.Metadata["type"] != "container" {
		t.Errorf("Expected type container, got %s", state.Metadata["type"])
	}

	// Verify String representation
	if str := cw.String(); str != "ContainerWorker(worker-1, test-container)" {
		t.Errorf("Unexpected String(): %s", str)
	}

	// Stop the worker
	if err := cw.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if mock.Status() != container.StatusStopped {
		t.Errorf("Mock container should be stopped, got %v", mock.Status())
	}

	// Wait for completion
	waitCh := cw.Wait()
	select {
	case err := <-waitCh:
		if err != nil {
			t.Errorf("Wait returned error: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("Wait timed out")
	}
}

func TestContainerWorker_StartFailure(t *testing.T) {
	// Create a mock that fails start
	mock := &failingMock{
		MockContainer: container.NewMockContainer("fail-container"),
		failStart:     true,
	}
	cw := NewContainerWorker("fail-worker", mock)

	if err := cw.Start(context.Background()); err == nil {
		t.Error("Expected Start error")
	}
}

type failingMock struct {
	*container.MockContainer
	failStart bool
}

func (m *failingMock) Start(ctx context.Context) error {
	if m.failStart {
		return errors.New("simulated start failure")
	}
	return m.MockContainer.Start(ctx)
}
