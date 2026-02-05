package container

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
)

// MockContainer is a reference/mock implementation of the Container interface.
type MockContainer struct {
	id     string
	status Status
	mu     sync.Mutex
	logs   *strings.Reader
}

func NewMockContainer(id string) *MockContainer {
	return &MockContainer{
		id:     id,
		status: StatusCreated,
		logs:   strings.NewReader(fmt.Sprintf("Logs for container %s\n", id)),
	}
}

func (m *MockContainer) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.status != StatusCreated && m.status != StatusPending {
		return fmt.Errorf("container already started (status: %s)", m.status)
	}
	m.status = StatusRunning
	return nil
}

func (m *MockContainer) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = StatusStopped
	return nil
}

func (m *MockContainer) Inspect(ctx context.Context) (InspectData, error) {
	return InspectData{
		Image: "redis:7-alpine",
		IP:    "172.17.0.2",
		Ports: []string{"6379/tcp"},
		Labels: map[string]string{
			"app": "lifecycle-demo",
		},
	}, nil
}

func (m *MockContainer) Logs(ctx context.Context) (io.ReadCloser, error) {
	return io.NopCloser(m.logs), nil
}

func (m *MockContainer) ID() string {
	return m.id
}

func (m *MockContainer) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status
}
