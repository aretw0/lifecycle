package sources

import (
	"context"
	"fmt"

	"github.com/aretw0/lifecycle/pkg/control"
)

// FileWatchEvent represents a filesystem change.
type FileWatchEvent struct {
	Path string
	Op   string // CREATE, WRITE, REMOVE, etc.
}

func (e FileWatchEvent) String() string {
	return fmt.Sprintf("FileWatch(path=%s, op=%s)", e.Path, e.Op)
}

// FileWatchSource watches for file changes (e.g., config updates).
// TODO: Integrate with fsnotify or Loam.
type FileWatchSource struct {
	Path   string
	events chan control.Event
}

func NewFileWatchSource(path string) *FileWatchSource {
	return &FileWatchSource{
		Path:   path,
		events: make(chan control.Event),
	}
}

func (s *FileWatchSource) Events() <-chan control.Event {
	return s.events
}

func (s *FileWatchSource) Start(ctx context.Context) error {
	defer close(s.events)
	// TODO: Watch logic.
	<-ctx.Done()
	return ctx.Err()
}
