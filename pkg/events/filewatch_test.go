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
	select {
	case <-source.ready: // deterministic synchronization
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for watcher to start")
	}

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

func TestFileWatchSource_Recursive(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filewatch-recursive-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(subDir, "test.txt")
	_ = os.WriteFile(testFile, []byte("initial"), 0644)

	source := NewFileWatchSource(tmpDir, WithRecursive(true))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := source.Events()

	go func() {
		_ = source.Start(ctx)
	}()

	// Wait for watcher to initialize
	select {
	case <-source.ready:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for watcher to start")
	}

	// Trigger change in subdirectory watcher setup
	_ = os.WriteFile(testFile, []byte("changed"), 0644)

	// Verify event
	select {
	case ev := <-ch:
		if ev.String() != "file/WRITE" {
			t.Errorf("Expected event file/WRITE, got %s", ev.String())
		}
	case <-time.After(1 * time.Second):
		t.Error("Timed out waiting for recursive file change event")
	}
}

func TestFileWatchSource_Filter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filewatch-filter-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	goodFile := filepath.Join(tmpDir, "good.txt")
	badFile := filepath.Join(tmpDir, "bad.lock")

	_ = os.WriteFile(goodFile, []byte("initial"), 0644)
	_ = os.WriteFile(badFile, []byte("initial"), 0644)

	// Filter ignores any file ending in .lock
	filter := func(path string) bool {
		return filepath.Ext(path) != ".lock"
	}

	source := NewFileWatchSource(tmpDir, WithRecursive(true), WithFilter(filter))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := source.Events()

	go func() {
		_ = source.Start(ctx)
	}()

	// Wait for watcher to initialize
	select {
	case <-source.ready:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for watcher to start")
	}

	// Trigger change on ignored file
	_ = os.WriteFile(badFile, []byte("changed"), 0644)

	// Verify NO event happens for badFile
	select {
	case ev := <-ch:
		t.Errorf("Received unexpected event for filtered file: %s", ev.String())
	case <-time.After(200 * time.Millisecond):
		// Expected timeout
	}

	// Trigger change on allowed file
	_ = os.WriteFile(goodFile, []byte("changed"), 0644)

	// Verify event happens for goodFile
	select {
	case ev := <-ch: // Wait longer in case of slow disk
		if ev.String() != "file/WRITE" {
			t.Errorf("Expected event file/WRITE, got %s", ev.String())
		}
	case <-time.After(1 * time.Second):
		t.Error("Timed out waiting for allowed file change event")
	}
}
