package container

import (
	"context"
	"io"
	"testing"
)

func TestMockContainer_NewMockContainer(t *testing.T) {
	mock := NewMockContainer("test-container-123")

	if mock.ID() != "test-container-123" {
		t.Errorf("ID() = %q, want %q", mock.ID(), "test-container-123")
	}

	if mock.Status() != StatusCreated {
		t.Errorf("Status() = %v, want %v", mock.Status(), StatusCreated)
	}
}

func TestMockContainer_Start(t *testing.T) {
	ctx := context.Background()
	mock := NewMockContainer("test-container")

	err := mock.Start(ctx)
	if err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	if mock.Status() != StatusRunning {
		t.Errorf("Status() after Start = %v, want %v", mock.Status(), StatusRunning)
	}
}

func TestMockContainer_Start_AlreadyStarted(t *testing.T) {
	ctx := context.Background()
	mock := NewMockContainer("test-container")

	// First start succeeds
	if err := mock.Start(ctx); err != nil {
		t.Fatalf("First Start() failed: %v", err)
	}

	// Second start should fail
	err := mock.Start(ctx)
	if err == nil {
		t.Error("Second Start() should return error")
	}

	if mock.Status() != StatusRunning {
		t.Errorf("Status after failed Start = %v, want %v", mock.Status(), StatusRunning)
	}
}

func TestMockContainer_Stop(t *testing.T) {
	ctx := context.Background()
	mock := NewMockContainer("test-container")

	// Start the container first
	if err := mock.Start(ctx); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Stop it
	err := mock.Stop(ctx)
	if err != nil {
		t.Fatalf("Stop() returned error: %v", err)
	}

	if mock.Status() != StatusStopped {
		t.Errorf("Status() after Stop = %v, want %v", mock.Status(), StatusStopped)
	}
}

func TestMockContainer_Inspect(t *testing.T) {
	ctx := context.Background()
	mock := NewMockContainer("test-container")

	data, err := mock.Inspect(ctx)
	if err != nil {
		t.Fatalf("Inspect() returned error: %v", err)
	}

	if data.Image != "redis:7-alpine" {
		t.Errorf("Inspect().Image = %q, want %q", data.Image, "redis:7-alpine")
	}

	if data.IP != "172.17.0.2" {
		t.Errorf("Inspect().IP = %q, want %q", data.IP, "172.17.0.2")
	}

	if len(data.Ports) == 0 || data.Ports[0] != "6379/tcp" {
		t.Errorf("Inspect().Ports = %v, want [6379/tcp]", data.Ports)
	}

	if data.Labels["app"] != "lifecycle-demo" {
		t.Errorf("Inspect().Labels[app] = %v, want lifecycle-demo", data.Labels["app"])
	}
}

func TestMockContainer_Logs(t *testing.T) {
	ctx := context.Background()
	mock := NewMockContainer("test-container")

	logs, err := mock.Logs(ctx)
	if err != nil {
		t.Fatalf("Logs() returned error: %v", err)
	}

	if logs == nil {
		t.Error("Logs() returned nil")
	}

	defer logs.Close()

	// Read content
	content, err := io.ReadAll(logs)
	if err != nil {
		t.Fatalf("ReadAll() failed: %v", err)
	}

	if len(content) == 0 {
		t.Error("Logs content is empty")
	}
}

// Trivial ID check removed

func TestMockContainer_Status_Transitions(t *testing.T) {
	ctx := context.Background()
	mock := NewMockContainer("test-container")

	// Created -> Running
	if mock.Status() != StatusCreated {
		t.Errorf("Initial Status = %v, want %v", mock.Status(), StatusCreated)
	}

	mock.Start(ctx)
	if mock.Status() != StatusRunning {
		t.Errorf("After Start, Status = %v, want %v", mock.Status(), StatusRunning)
	}

	// Running -> Stopped
	mock.Stop(ctx)
	if mock.Status() != StatusStopped {
		t.Errorf("After Stop, Status = %v, want %v", mock.Status(), StatusStopped)
	}
}

func TestMockContainer_Concurrent_Status(t *testing.T) {
	mock := NewMockContainer("test-container")
	ctx := context.Background()

	// Start in a goroutine
	go func() {
		mock.Start(ctx)
	}()

	// Read status concurrently
	status := mock.Status()
	if status != StatusCreated && status != StatusRunning {
		t.Errorf("Concurrent Status() = %v, got unexpected value", status)
	}
}
