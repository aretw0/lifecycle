package events

import (
	"context"
	"log/slog"
	"path/filepath"

	"github.com/aretw0/lifecycle/pkg/core/metrics"
	"github.com/fsnotify/fsnotify"
)

// FileWatchSource watches a file for changes and emits events when it is modified.
// This is useful for configuration hot-reloading without process restart.
//
// This implementation uses fsnotify for efficient, event-driven file watching
// (supported on Linux, Windows, macOS, BSD).
//
// Example:
//
//	router := lifecycle.NewRouter()
//	AddSource(lifecycle.NewFileWatchSource("config.yaml"))
//	Handle("file/*", lifecycle.NewReloadHandler(loadConfig))
type FileWatchSource struct {
	BaseSource
	path string
}

// NewFileWatchSource creates a new file watcher for the specified path.
// The path will be cleaned using filepath.Clean.
//
// Unlike the legacy polling-based approach, this uses fsnotify for
// immediate event notification when files change.
func NewFileWatchSource(path string) *FileWatchSource {
	return &FileWatchSource{
		BaseSource: NewBaseSource("filewatch", 1),
		path:       filepath.Clean(path),
	}
}

// Start begins watching the file for changes.
// It blocks until the context is cancelled or an unrecoverable error occurs.
func (f *FileWatchSource) Start(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	if err := watcher.Add(f.path); err != nil {
		return err
	}

	slog.Info("FileWatchSource: watching", "path", f.path)

	for {
		select {
		case <-ctx.Done():
			slog.Info("FileWatchSource: stopped watching", "path", f.path)
			return ctx.Err()

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			// Emit event on Write, Create, or Rename (for atomic saves)
			// Modern editors (VS Code, vim, etc) use atomic saves:
			// 1. Write to temp file
			// 2. Rename temp file to target (this is what we detect)
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {
				slog.Info("FileWatchSource: detected change", "path", f.path, "op", event.Op)

				// If file was renamed away, re-watch the new file
				if event.Has(fsnotify.Rename) || event.Has(fsnotify.Remove) {
					// Re-add watch in case editor replaced file
					watcher.Remove(f.path)
					if err := watcher.Add(f.path); err != nil {
						slog.Warn("FileWatchSource: failed to re-watch after rename", "path", f.path, "error", err)
					}
				}

				// Emit event using BaseSource helper
				if err := f.Emit(ctx, fileChangeEvent{path: f.path, op: mapFsnotifyOp(event.Op)}); err != nil {
					slog.Error("FileWatchSource: failed to emit event", "path", f.path, "error", err)
					return err
				}
				metrics.GetProvider().IncEventEmitted("filewatch")
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			slog.Error("FileWatchSource: error watching", "path", f.path, "error", err)
			// Continue watching despite errors
		}
	}
}

// mapFsnotifyOp converts fsnotify.Op to FileOp for backward compatibility.
func mapFsnotifyOp(op fsnotify.Op) FileOp {
	switch {
	case op&fsnotify.Create != 0:
		return FileOpCreate
	case op&fsnotify.Write != 0:
		return FileOpWrite
	case op&fsnotify.Remove != 0:
		return FileOpRemove
	case op&fsnotify.Rename != 0:
		return FileOpRename
	case op&fsnotify.Chmod != 0:
		return FileOpChmod
	default:
		return FileOpWrite // Default fallback
	}
}

// fileChangeEvent represents a file modification event.
type fileChangeEvent struct {
	path string
	op   FileOp
}

// String returns the event topic in the format "file/<op>".
func (e fileChangeEvent) String() string {
	return "file/" + string(e.op)
}

// FileOp represents a file operation (preserved for backward compatibility).
type FileOp string

const (
	FileOpCreate FileOp = "CREATE"
	FileOpWrite  FileOp = "WRITE"
	FileOpRemove FileOp = "REMOVE"
	FileOpRename FileOp = "RENAME"
	FileOpChmod  FileOp = "CHMOD"
)
