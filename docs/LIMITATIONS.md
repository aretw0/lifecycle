# Limitations & Known Issues

> **Last Updated**: February 16, 2026 (v1.6.0)
> 
> This document lists known limitations, platform-specific constraints, and unsolved problems. **Transparency is a feature.**

---

## Platform-Specific Constraints

### Windows

| Constraint | Details | Impact | Workaround |
|:-----------|:--------|:-------|:-----------|
| **Go Version** | Requires **Go 1.20+** for full Job Objects support (zombie prevention) | Pre-1.20: Child processes may become zombies on hard crash | Upgrade Go or accept zombie risk |
| **Console Input** | CONIN$ requires explicit opt-in for non-blocking reliable I/O | Default behavior may block on `Ctrl+C` in some terminals | Use `lifecycle.NewInteractiveRouter` or manual setup |
| **SIGTERM Behavior** | Not natively supported; mapped to graceful exit | Works but less native than Unix | No alternative; design accounts for this |

### macOS

| Constraint | Details | Impact | Workaround |
|:-----------|:--------|:-------|:-----------|
| **PDeathSig** | Not supported by Go `os/signal` on macOS | Hard crashes can leave orphan processes | Use external process monitor or heartbeat mechanism |
| **Zombie Detection** | Cannot automatically detect zombie child processes | May accumulate if parent crashes during `wait()` | Monitor with `ps aux \| grep <defunct>` |

### Linux

| Constraint | Details | Impact | Workaround |
|:-----------|:--------|:-------|:-----------|
| **SIGCHLD Handling** | Default handler may interfere with custom signal handlers | Rare, but affects some specialized use cases | Document custom handlers explicitly |

---

## Feature-Specific Limitations

### Router Pattern Matching

| Feature | Limitation | Details | Example |
|:--------|:-----------|:--------|:--------|
| **Pattern Syntax** | **Glob-only** (not full regex) | Uses Go's `path.Match` internally: `*`, `?`, `[...]` only | ✅ `signal/*/handler` ❌ `signal/(int\|term)` |
| **Performance** | Linear search (no indexing) | O(n) route lookup for n routes | Consider `<100` routes for interactive apps |
| **Ambiguity** | First-match wins (no priority weights) | Overlapping patterns use definition order | Define more-specific patterns first |

**Affected Code**: [pkg/events/router.go](../pkg/events/router.go#L192) — See inline TODO for optimization opportunity.

### Observer Interface (v1.6.0)

| Feature | Status | Caveat |
|:--------|:-------|:--------|
| **OnGoroutinePanicked** | Stable (v1.6.0) | Stack capture is optional; auto-detect uses `slog.LevelDebug` |
| **Stack Bytes Format** | Stable | Uses `runtime/debug.Stack()` (text format, not parsed) |
| **Observer Ordering** | Not guaranteed | Multiple observers called serially; exception stops chain (TBD) |
| **Production Overhead** | ~0.5-1µs per panic | Only if observer is installed; no overhead if nil |

**Documented In**: [TECHNICAL.md §14 - Panic Reporting](TECHNICAL.md#14-panic-reporting)

### Stack Capture Behavior

| Mode | Behavior | Use Case | Performance |
|:-----|:---------|:---------|:------------|
| **Enabled** `WithStackCapture(true)` | Always capture stack bytes | Critical tasks, debugging | +1-2µs per task |
| **Disabled** `WithStackCapture(false)` | Never capture (even if debug logging on) | Performance-sensitive code | Baseline |
| **Auto-Detect** (default, unset) | Capture only if `slog.LevelDebug` enabled | Development, leave unset in production | Conditional +1-2µs |

**Implementation**: [pkg/core/runtime/task.go](../pkg/core/runtime/task.go) — Conditional stack capture logic.

---

## Performance Unknowns

> These are **measured on specific hardware** (Intel i7, 16GB RAM). Results may vary.

### Measured Overhead (Baseline)

| Operation | Baseline | lifecycle.Go | Overhead | Notes |
|:----------|:---------|:-------------|:---------|:------|
| `go func()` | ~500ns | — | — | Raw goroutine creation |
| `lifecycle.Go(ctx, fn)` | — | ~5-10µs | **10-20x** | Tracking + WaitGroup + observer setup |
| **Stack Capture** | N/A | +1-2µs | — | Only if enabled; debug-aware |
| **Route Matching** (10 routes) | — | ~200ns | — | Uses `path.Match` (fast glob) |
| **Route Matching** (100 routes) | — | ~2µs | — | Linear O(n) lookup; consider indexing |

**Caveat**: Benchmarks are from dev machine only. CI benchmarks (Windows, macOS) pending v1.7.

### Unmeasured (Needs Investigation)

- [ ] **Introspection overhead**: Calling `State()` on large supervision trees (>100 workers)
- [ ] **Router throughput**: Events/sec with high message volume
- [ ] **Memory footprint**: Supervision tree with 1000+ workers
- [ ] **Shutdown latency**: Scaling with worker count (critical for Kubernetes)

---

## API Stability

### Stable (v1.5+)

- ✅ `lifecycle.Run`, `lifecycle.Go`, `lifecycle.Do`
- ✅ `lifecycle.NewRouter`, `lifecycle.Handle`
- ✅ `lifecycle.NewSupervisor`, `pkg/core/supervisor/SupervisorSpec`
- ✅ `lifecycle.NewSignalContext` (aliased to `lifecycle.SignalContext`)
- ✅ `lifecycle.NewInteractiveRouter`

### Stable as of v1.6.0

- ✅ `lifecycle.Context()` — Manual context setup for gradual migration
- ✅ `lifecycle.WithStackCapture(bool)` — Stack capture control
- ✅ `Observer.OnGoroutinePanicked(recovered any, stack []byte)` — Panic hook

### Not Yet Marked (Audit Pending)

- ⚠️ `pkg/events/filewatch.FileWatchSource` — Flaky FS race conditions; integration tests only
- ⚠️ `pkg/events/webhook.WebhookSource` — Skeleton; minimal testing
- ⚠️ `pkg/events/health.HealthCheckSource` — Skeleton; minimal testing
- ⚠️ `pkg/core/worker/suspend.Suspend` — Experimental; covered but not marked

**Deprecation Policy**: None documented yet (planned for v1.7+).

---

## Test Coverage Status

### High Coverage (>80%)

```
github.com/aretw0/lifecycle          85%
github.com/aretw0/lifecycle/pkg/core/signal   92%
github.com/aretw0/lifecycle/pkg/core/supervisor  88%
github.com/aretw0/lifecycle/pkg/core/runtime    87%
github.com/aretw0/lifecycle/pkg/events/router   81%
```

### Low Coverage (Intentional Exclusions)

| Package | Coverage | Reason | Strategy |
|:--------|:---------|:-------|:---------|
| `pkg/core/metrics` | ~40% | Interface definitions + no-op stubs | Compile check; tested in consuming packages |
| `pkg/core/log` | ~30% | Wrapper around `slog` | Compile check; assumes slog stability |
| `pkg/events/filewatch` | ~50% | FS race conditions too flaky for unit tests | Manual verification + integration tests |
| `procio` (external) | Tested in `procio` repo | OS-dependent syscalls | Extracted to [procio](https://github.com/aretw0/procio) library |

**Philosophy**: See [TESTING.md](TESTING.md) for "Honest Coverage" rationale.

---

## Known Issues (Non-Critical)

### Code TODOs

| File | Line | Issue | Priority |
|:-----|:----:|:------|:---------|
| [pkg/events/router.go](../pkg/events/router.go#L192) | 192 | Optimize route matching if many routes | 🟢 Low (future: batch indexing) |

### Untested Scenarios

| Scenario | Reason | Impact | Workaround |
|:---------|:-------|:-------|:-----------|
| **Windows CONIN$ on SSH** | Interactive I/O unavailable in SSH | Cannot use `NewInteractiveRouter` | Use non-interactive mode |
| **Docker Alpine + musl** | musl libc has signal handling quirks | Rare issues with suspend/resume | Test before production |
| **Kubernetes graceful shutdown <5s** | Default SIGTERM timeout may be insufficient | May force-kill graceful tasks | Increase terminationGracePeriodSeconds |
| **Large supervision trees (>1000 workers)** | Performance characteristics unknown | May hit memory/latency limits | Monitor and benchmark |

---

## Compatibility Matrix

### Tested & Supported

| Component | Status | Versions Tested |
|:----------|:-------|:----------------:|
| **Go** | ✅ Stable | 1.20, 1.21, 1.22 |
| **Windows** | ✅ Stable | 10, 11, Server 2022 |
| **Linux** | ✅ Stable | Ubuntu 20.04+, Alpine 3.16+ |
| **macOS** | ⚠️ Partial (no PDeathSig) | 12+, both Intel & Apple Silicon |

### Not Tested

| Platform | Reason |
|:---------|:-------|
| **BSD / FreeBSD** | No CI; contributions welcome |
| **Plan 9 / WASI** | Out of scope (niche platforms) |
| **Android / iOS** | Not intended for mobile |

---

## Future Unknowns (v1.7+)

- [ ] Optimal route count threshold before adding indexing
- [ ] Memory overhead of introspection with large trees
- [ ] How well context cancellation propagates through deep nesting
- [ ] Shutdown time scaling with worker count (critical for Kubernetes)

---

## Reporting Issues

Found a limitation not listed here? Please open an issue with:

1. **Platform & Version** (Go 1.21 on Windows 11, etc.)
2. **Minimal Example** (code that triggers the issue)
3. **Expected vs. Actual Behavior**
4. **Workaround** (if you found one)

See [DECISIONS.md](DECISIONS.md) for architectural trade-offs that explain some limitations.
