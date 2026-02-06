package events

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileWatchSource(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filewatch-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	_ = os.WriteFile(testFile, []byte("initial"), 0644)

	source := NewFileWatchSource(testFile) // Correct API

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := source.Events()

	go func() {
		_ = source.Start(ctx)
	}()

	// Wait for watcher to initialize
	time.Sleep(100 * time.Millisecond)

	// Trigger change: WRITE
	_ = os.WriteFile(testFile, []byte("changed"), 0644)

	// Verify event
	select {
	case ev := <-ch:
		// Topic should be "file/WRITE" (based on mapFsnotifyOp and String())
		if ev.String() != "file/WRITE" {
			t.Errorf("Expected event file/WRITE, got %s", ev.String())
		}

		fev, ok := ev.(fileChangeEvent) // Unexported but accessible in same package
		if !ok {
			t.Fatalf("Expected fileChangeEvent, got %T", ev)
		}
		if fev.path != testFile {
			t.Errorf("Expected path %s, got %s", testFile, fev.path)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timed out waiting for file change event")
	}
}
