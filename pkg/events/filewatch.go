package events

import (
	"context"
	"log/slog"
	"os"
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
	path      string
	recursive bool
	filter    func(path string) bool
	ready     chan struct{} // For deterministic test synchronization
}

// FileWatchOption configures the FileWatchSource
type FileWatchOption func(*FileWatchSource)

// WithRecursive enables recursive watching of all subdirectories.
func WithRecursive(enabled bool) FileWatchOption {
	return func(opts *FileWatchSource) {
		opts.recursive = enabled
	}
}

// WithFilter sets a function to ignore certain paths. If the filter returns false,
// the path is ignored. This is useful for omitting .git folders or locks.
func WithFilter(filter func(path string) bool) FileWatchOption {
	return func(opts *FileWatchSource) {
		opts.filter = filter
	}
}

// NewFileWatchSource creates a new file watcher for the specified path.
// The path will be cleaned using filepath.Clean.
//
// Unlike the legacy polling-based approach, this uses fsnotify for
// immediate event notification when files change.
func NewFileWatchSource(path string, opts ...FileWatchOption) *FileWatchSource {
	f := &FileWatchSource{
		BaseSource: NewBaseSource("filewatch", 1),
		path:       filepath.Clean(path),
		ready:      make(chan struct{}),
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// Start begins watching the file for changes.
// It blocks until the context is cancelled or an unrecoverable error occurs.
func (f *FileWatchSource) Start(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	if f.recursive {
		err = filepath.WalkDir(f.path, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if f.filter != nil && !f.filter(path) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				if err := watcher.Add(path); err != nil {
					slog.Warn("FileWatchSource: failed to add dir", "path", path, "error", err)
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	} else {
		// Only check filter for base path if specific filter provided
		if f.filter == nil || f.filter(f.path) {
			if err := watcher.Add(f.path); err != nil {
				return err
			}
		} else {
			slog.Warn("FileWatchSource: base path ignored by filter", "path", f.path)
		}
	}

	// Signal readiness
	select {
	case <-f.ready:
	default:
		close(f.ready)
	}

	slog.Info("FileWatchSource: watching", "path", f.path, "recursive", f.recursive)

	for {
		select {
		case <-ctx.Done():
			slog.Info("FileWatchSource: stopped watching", "path", f.path)
			return ctx.Err()

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {
				// 1. Check if the event path passes the filter
				if f.filter != nil && !f.filter(event.Name) {
					continue
				}

				slog.Info("FileWatchSource: detected change", "path", event.Name, "op", event.Op)

				// 2. Dynamic recursion support: If a new directory is created, watch it automatically
				if f.recursive && (event.Has(fsnotify.Create) || event.Has(fsnotify.Rename)) {
					// fsnotify events don't tell us if it's a directory easily without stat
					// but Add on a file vs dir behaves okay, or we can quickly check:
					// Just try to add it, if it's a directory it will add it to the watcher.
					// If it's a file, we don't strictly *need* to add it manually since
					// we watched its parent dir, but fsnotify typically wants parent dirs for new files anyway.
					// We'll trust watcher.Add (which is generally safe to call on files or dirs).
					// NOTE: fsnotify watching a dir auto-watches children files under it
					// for Linux/Windows, but discovering *new subdirectories* requires manual Add.

					// Let's only Add if it matches path filter (already checked)
					// Avoid doing costly stats here unless necessary. Adding blindly is usually cheap.
					_ = watcher.Add(event.Name)
				}

				// If file was renamed away, re-watch the new file (legacy specific behavior for single file watches)
				if !f.recursive && (event.Has(fsnotify.Rename) || event.Has(fsnotify.Remove)) {
					// Re-add watch in case editor replaced file
					watcher.Remove(f.path)
					if err := watcher.Add(f.path); err != nil {
						slog.Warn("FileWatchSource: failed to re-watch after rename", "path", f.path, "error", err)
					}
				}

				// Emit event using BaseSource helper
				if err := f.Emit(ctx, fileChangeEvent{path: event.Name, op: mapFsnotifyOp(event.Op)}); err != nil {
					slog.Error("FileWatchSource: failed to emit event", "path", event.Name, "error", err)
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
