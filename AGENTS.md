# AGENTS.md

## Project Overview

**lifecycle** is a Go library for robust infrastructure signaling and interactive I/O.
It allows applications to differentiate between "User Interrupts" and "System Terminations" while enforcing leak-free shutdowns.

For a detailed breakdown of the problems we solve (Zombie Processes, Blocking I/O, Dual Signals), see **[PRODUCT.md](docs/PRODUCT.md)**.

It is designed to be the foundational entry point for modern Go Applications (Services, Agents, CLIs).

## Project Structure & Documentation

* **[TECHNICAL.md](docs/TECHNICAL.md)**: Architecture (Foundation & Control Plane).
* **[PLANNING.md](docs/PLANNING.md)**: Roadmap (v2.0 Focus).
* **[PRODUCT.md](docs/PRODUCT.md)**: Vision and "Why?" (The problem space).
* **[DECISIONS.md](docs/DECISIONS.md)**: Design Decisions.
* **[STATE_MACHINE.md](docs/STATE_MACHINE.md)**: Worker State Machine.
* **[CONFIGURATION.md](docs/CONFIGURATION.md)**: Configuration Philosophy.
* **[RECIPES.md](docs/RECIPES.md)**: Common Usage Patterns.
* **[TESTING.md](docs/TESTING.md)**: Testing Philosophy.
* **[examples/](examples/)**: Runnable recipes (`basic`, `hooks`, `termio`, `supervisor`).

## Key Commands

Ensure dependencies are synced:

```bash
go mod tidy
```

### Running Tests

```bash
go test -timeout 90s -race -v ./...
# or
make test
```

### Running Coverage

> Poweshell needs double quotes for file paths

```bash
go test -race -v -timeout 90s -coverprofile="coverage.out" ./...
go tool cover -func="coverage.out"
# or
make coverage
```

### Running Examples

```bash
go run ./examples/hooks/main.go
```

## Development Philosophy

* **Managed Global State**: We abstract the inevitable global state (OS Signals) into clean, context-aware usage. Prefer `Context` propagation, but enjoy `DefaultRouter` convenience.
* **Leak-Free**: Every resource (goroutine, file handle) must close on shutdown.
* **Platform Agnostic**: Windows `CONIN$` handling is a first-class citizen, not an afterthought.
* **Observability**: Internal state changes must be visible via `pkg/metrics` interfaces.
