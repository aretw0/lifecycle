package sources

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aretw0/lifecycle/pkg/control"
	"github.com/aretw0/lifecycle/pkg/metrics"
)

// FileOp represents a file operation.
type FileOp string

const (
	FileOpCreate FileOp = "CREATE"
	FileOpWrite  FileOp = "WRITE"
	FileOpRemove FileOp = "REMOVE"
	FileOpRename FileOp = "RENAME"
	FileOpChmod  FileOp = "CHMOD"
)

// FileWatchEvent represents a filesystem change.
type FileWatchEvent struct {
	Path string
	Op   FileOp
}

func (e FileWatchEvent) String() string {
	return fmt.Sprintf("file/%s", e.Op)
}

// FileWatchSource watches for file changes via polling.
type FileWatchSource struct {
	Path     string
	Interval time.Duration
	events   chan control.Event
}

// NewFileWatchSource creates a new source that polls the given file path.
func NewFileWatchSource(path string, interval time.Duration) *FileWatchSource {
	return &FileWatchSource{
		Path:     path,
		Interval: interval,
		events:   make(chan control.Event),
	}
}

func (s *FileWatchSource) Events() <-chan control.Event {
	return s.events
}

func (s *FileWatchSource) Start(ctx context.Context) error {
	defer close(s.events)

	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()

	var lastModTime time.Time
	info, err := os.Stat(s.Path)
	if err == nil {
		lastModTime = info.ModTime()
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			info, err := os.Stat(s.Path)
			if os.IsNotExist(err) {
				if !lastModTime.IsZero() {
					// File was deleted
					s.emit(ctx, FileWatchEvent{Path: s.Path, Op: FileOpRemove})
					lastModTime = time.Time{}
				}
				continue
			}

			if err != nil {
				continue // Ignore other errors
			}

			// File exists. Was it just created?
			if lastModTime.IsZero() {
				lastModTime = info.ModTime()
				s.emit(ctx, FileWatchEvent{Path: s.Path, Op: FileOpCreate})
				continue
			}

			if info.ModTime().After(lastModTime) {
				lastModTime = info.ModTime()
				s.emit(ctx, FileWatchEvent{Path: s.Path, Op: FileOpWrite})
			}
		}
	}
}

func (s *FileWatchSource) emit(ctx context.Context, e FileWatchEvent) {
	select {
	case s.events <- e:
		metrics.GetProvider().IncEventEmitted("filewatch")
	case <-ctx.Done():
	}
}
