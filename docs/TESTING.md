# Testing & Quality Strategy

> **Philosophy**: "Honest Coverage" over "Test Theater".

This document defines how we test `lifecycle` to ensure reliability without wasting time on performative metrics.

## 1. Honest Coverage

We target **High Coverage (>80%)** for:

* **Core Logic**: State machines, Supervisors, Router matching.
* **Concurrency**: Race conditions, context propagation, panic recovery.
* **Public API**: The `lifecycle` facade.

We accept **Low Coverage** for:

* **Platform Specifics**: `termio` (requires manual Windows verification).
* **Boilerplate**: `metrics` interfaces, `log` wrappers.
* **Impossible Errors**: Syscall failures that can't be safely mocked without extreme complexity.

## 2. Exclusions ("Test Theater")

The following packages are explicitly exempted from strict coverage targets:

| Package | Reason | Strategy |
| :--- | :--- | :--- |
| `pkg/core/metrics` | Interface definitions & no-op stubs. | Compile check. |
| `pkg/core/log` | Wrapper around `slog`. | Compile check. |
| `procio` (external) | Heavily OS-dependent (syscalls). Extracted to standalone library. | Manual verification on Windows. Tested in `procio` repo. |
| `pkg/events/filewatch` | FS race conditions are flaky. | Integration tests. |

## 3. Running Tests

### Standard Suite

Runs in CI/CD on Linux, macOS, and Windows.

```bash
go test -v -race ./...
```

### Coverage Report

Generates the "Honest Coverage" report.

```bash
go test -coverprofile="coverage.out" ./...
go tool cover -func="coverage.out"
```

### Manual Verification (Windows)

For `termio` and `CONIN$` handling:

1. Run `examples/interactive_dx/main.go`.
2. Pres `Ctrl+C`.
3. **Expected**: "Suspended" message (if configured) or Clean Exit.
4. **Failure**: Immediate harsh termination or hanging.

## 4. Writing Tests

* **Avoid Sleep**: Use channels or `lifecycle.WaitForGlobal()` to synchronize.
* **Test Behavior, Not Mocks**: Prefer real implementations over excessive mocking.
* **Leak Check**: Ensure `goleak` or internal trackers confirm 0 goroutines left after test.
