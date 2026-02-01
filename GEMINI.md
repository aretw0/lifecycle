# GEMINI.md

## Project Overview

**lifecycle** is a Go library for robust infrastructure signaling and interactive I/O. It solves two specific problems:

1. **Dual-Signal Handling**: Distinguishing `SIGINT` (User Interrupt) from `SIGTERM` (System Termination).
2. **Interruptible I/O**: Wrapping blocking `io.Reader` calls to allow context cancellation, essential for preventing goroutine leaks during shutdown.
3. **Durable Primitives** (v1.4): Shielding critical sections (`lifecycle.Do`) to allow "Durable Execution" engines to run safely.

It is designed to be the foundational entry point for Go CLIs.

## Project Structure & Documentation

* **[TECHNICAL.md](docs/TECHNICAL.md)**: Architecture, State Machine, and Constraints (Data Loss vs Safety).
* **[PLANNING.md](docs/PLANNING.md)**: Roadmap and Backlog.
* **[PRODUCT.md](docs/PRODUCT.md)**: Vision and "Why?" (The problem space).
* **[examples/](examples/)**: Runnable recipes (`basic`, `hooks`, `termio`).

## Key Commands

Ensure dependencies are synced:

```bash
go mod tidy
```

### Running Tests

```bash
go test -v ./...
# or
make test
```

### Running Examples

```bash
go run ./examples/hooks/main.go
```

## Development Philosophy

* **Zero Global State**: Use `Context` propagation, never global handlers.
* **Leak-Free**: Every resource (goroutine, file handle) must close on shutdown.
* **Platform Agnostic**: Windows `CONIN$` handling is a first-class citizen, not an afterthought.
* **Observability**: Internal state changes must be visible via `pkg/metrics` interfaces.
