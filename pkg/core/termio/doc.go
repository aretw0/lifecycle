// Package termio provides interruptible I/O primitives and terminal handling.
//
// It solves common issues with blocking I/O in Go CLI tools, particularly on Windows,
// where a blocked read from stdin can prevent signal delivery or cause hangs.
//
// Key features:
//   - InterruptibleReader: A reader that respects context cancellation.
//   - Open: Platform-safe terminal opening (uses CONIN$ on Windows).
//   - Upgrade: Automatic detection and upgrade of readers to terminal-aware handles.
//
// # Safety
//
// The [InterruptibleReader] uses a "Shielded Return" strategy. If data arrives
// exactly as the context is cancelled, the reader prioritizes returning the data
// (Data First, Error Second). The *next* read will then check for cancellation.
// This ensures no data loss occurs when reading from pipes or streams.
//
// For interactive prompts (e.g., "Confirm Delete [y/N]"), use [InterruptibleReader.ReadInteractive].
// This method enforces "Error First" logic, discarding potential race-condition input
// to prioritize safety.
package termio
