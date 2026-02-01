# GEMINI.md

## Project Overview

**lifecycle** is a Go library for robust infrastructure signaling and interactive I/O.
It allows applications to differentiate between "User Interrupts" and "System Terminations" while strictly enforcing leak-free shutdowns.

For a detailed breakdown of the problems we solve (Zombie Processes, Blocking I/O, Dual Signals), see **[PRODUCT.md](docs/PRODUCT.md)**.

It is designed to be the foundational entry point for modern Go Applications (Services, Agents, CLIs).

## Project Structure & Documentation

* **[TECHNICAL.md](docs/TECHNICAL.md)**: Architecture (Foundation & Control Plane).
* **[PLANNING.md](docs/PLANNING.md)**: Roadmap (v2.0 Focus).
* **[PRODUCT.md](docs/PRODUCT.md)**: Vision and "Why?" (The problem space).
* **[examples/](examples/)**: Runnable recipes (`basic`, `hooks`, `termio`, `supervisor`).

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
