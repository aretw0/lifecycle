// Package lifecycle provides a centralized library for managing application lifecycles
// and interactive I/O within the aretw0 ecosystem (Trellis, Tobot, Fiscus).
//
// It offers a robust "Dual Signal" context (SignalContext) that differentiates between
// soft interruption (SIGINT/Ctrl+C) and hard termination (SIGTERM), allowing applications
// to implement "Are you sure?" prompts or graceful shutdown sequences.
//
// Additionally, it provides platform-agnostic terminal I/O utilities (in subpackage pkg/termio)
// that support interruptible reads, preventing goroutine leaks on Windows and Posix systems
// when an application is cancelling its context.
package lifecycle
