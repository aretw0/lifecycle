package termio

import (
	"bytes"
	"os"
	"testing"
)

func TestUpgrade_NonFile(t *testing.T) {
	buf := bytes.NewBufferString("hello")
	upgraded, err := Upgrade(buf)
	if err != nil {
		t.Fatalf("Upgrade failed: %v", err)
	}
	// Upgrade returns io.ReadCloser. Since bytes.Buffer isn't a ReadCloser,
	// it might be wrapped in a NopCloser.
	// Logic: if rc, ok := r.(io.ReadCloser); ok { return rc } else { return io.NopCloser(r) }

	// We just want to ensure it reads correctly
	out := make([]byte, 5)
	n, err := upgraded.Read(out)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if n != 5 || string(out) != "hello" {
		t.Errorf("Unexpected read content: %s", out)
	}
}

func TestUpgrade_FileNonTerminal(t *testing.T) {
	// Create a temporary file
	f, err := os.CreateTemp("", "lifecycle_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	upgraded, err := Upgrade(f)
	if err != nil {
		t.Fatalf("Upgrade failed: %v", err)
	}

	// Should return the file itself as it is a ReadCloser
	if upgraded != f {
		t.Errorf("Expected original file object for non-terminal file")
	}
}
