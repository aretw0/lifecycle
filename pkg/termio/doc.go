// Package termio provides interruptible I/O primitives and terminal handling.
//
// It solves common issues with blocking I/O in Go CLI tools, particularly on Windows,
// where a blocked read from stdin can prevent signal delivery or cause hangs.
//
// Key features:
//   - InterruptibleReader: A reader that respects context cancellation.
//   - Open: Platform-safe terminal opening (uses CONIN$ on Windows).
//   - Upgrade: Automatic detection and upgrade of readers to terminal-aware handles.
package termio
