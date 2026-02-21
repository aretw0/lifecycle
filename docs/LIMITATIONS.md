# Limitations & Known Issues

> **Last Updated**: February 17, 2026 (v1.6.2)
>
> This document lists known limitations, platform-specific constraints, and measured performance characteristics. **Transparency is a feature.**

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
| **Performance** | Linear search (no indexing) | O(n) route lookup for n routes | See benchmark results below |
| **Ambiguity** | First-match wins (no priority weights) | Overlapping patterns use definition order | Define more-specific patterns first |

**Pattern Examples**:

```go
// ✅ Valid glob patterns
router.HandleFunc("signal/*/handler", fn)        // Matches any single segment
router.HandleFunc("signal/[it]*", fn)            // Matches interrupt, terminate
router.HandleFunc("event/*/", fn)                  // Prefix matching

// ❌ Invalid (not supported - use exact routes instead)
router.HandleFunc("signal/(int|term)", fn)       // Regex alternation not supported
router.HandleFunc("signal/\\d+", fn)               // Regex character classes not supported
```

**Benchmark Results** (See `pkg/events/router_benchmark_test.go`):

| Routes | Exact Match | Glob Match (avg) | Worst Case |
|:-------|:------------|:----------------|:-----------|
| 1 (exact) | ~77ns | N/A | Fast map lookup |
| 10 (glob) | ~77ns | ~984ns (~1µs) | Linear scan |
| 100 (glob) | ~77ns | ~7µs | Linear scan |
| 1000 (glob) | ~77ns | ~81µs | Linear scan |

**Recommendation**:

- Interactive apps: <50 glob routes for <5µs latency
- High-throughput: <100 routes or use exact matching only
- Enterprise (>1000 routes): Consider external router (e.g., radix tree) before lifecycle

**Affected Code**: [pkg/events/router.go](../pkg/events/router.go#L192) — See inline comments for optimization notes.

### Observer Interface (v1.6.0)

| Feature | Status | Caveat |
|:--------|:-------|:--------|
| **OnGoroutinePanicked** | Stable (v1.6.0) | Stack capture is optional; auto-detect uses `slog.LevelDebug` |
| **Stack Bytes Format** | Stable | Uses `runtime/debug.Stack()` (text format, not parsed) |
| **Observer Ordering** | Not guaranteed | Multiple observers called serially; exception stops chain (TBD) |
| **Production Overhead** | ~0.5-1µs per panic | Only if observer is installed; no overhead if nil |

**Documented In**: [TECHNICAL.md §14 - Panic Reporting](TECHNICAL.md#14-panic-reporting)

### Stack Capture Behavior

| Mode | Behavior | Overhead (per panic) | Memory | Use Case |
|:-----|:---------|:--------------------|:-------|:---------|
| **Enabled** `WithStackCapture(true)` | Always capture stack bytes | +1-2µs | ~4-8 KB | Critical tasks, debugging |
| **Disabled** `WithStackCapture(false)` | Never capture (even if debug on) | Baseline (~2µs) | ~0 bytes | Performance-sensitive code |
| **Auto-Detect** (default) | Capture only if `slog.LevelDebug` enabled | Conditional | Conditional | Development (recommended) |

**Recommendation**: Leave unset (auto-detect) in most cases. Only use explicit `true` for critical worker lifecycle tracking.

**Implementation**: [pkg/core/runtime/task.go](../pkg/core/runtime/task.go) — Conditional stack capture logic.

---

## Performance Characteristics

> **Example Benchmarking Environment** (yours will vary):
>
> - Hardware: 11th Gen Intel i9-11900H @ 2.50GHz, 16GB RAM
> - OS: Windows 11, Go 1.22
> - Benchmarks run with `-benchtime=2s -benchmem`
>
> **Important**: These are reference numbers from one machine. Performance varies significantly across hardware, OS, and workload. Always benchmark on your target environment.

### Core Runtime Overhead

| Operation | Baseline | lifecycle Overhead | Notes |
|:----------|:---------|:------------------|:------|
| `go func()` + `wg.Wait()` | ~440ns | — | Raw goroutine creation |
| `lifecycle.Go(ctx, fn)` | ~1.4µs | **~3x** | Tracking + metrics + observer setup |
| `lifecycle.Do(ctx, fn)` | ~800ns-1.2µs | **~2x** | Recovery + metrics |

**Interpretation**: The overhead is acceptable for I/O-bound tasks (network, disk) but may matter for tight CPU loops spawning thousands of goroutines per second.

### Observer Overhead

| Observer Status | Overhead per `Go()` | Notes |
|:---------------|:--------------------|:------|
| No Observer (`nil`) | Baseline | Check is ~5ns |
| Observer Installed | +0.5-1µs | Only on panic; normal execution unaffected |

### Router Performance

See "Router Pattern Matching" section above for detailed route count scaling.

**Middleware Overhead** (per middleware, per event):

| Middleware Count | Overhead | Example |
|:----------------|:---------|:--------|
| 0 | Baseline (~100ns) | Direct handler invocation |
| 1 | +50-100ns | Logging |
| 5 | +200-400ns | Logging + Recovery + Metrics + Custom |
| 10 | +500-800ns | Complex chains |

### Supervisor Introspection

| Tree Size | `State()` Call | Memory Footprint | Notes |
|:----------|:--------------|:----------------|:------|
| 10 workers | ~5-10µs | ~5 KB | Fast; suitable for live dashboards |
| 100 workers | ~50-100µs | ~50 KB | Acceptable for periodic polling |
| 1000 workers | ~500-800µs | ~500 KB | Consider caching; avoid hot loops |

**Recommendation**: For large trees, cache `State()` results or poll at intervals (e.g., 1s) instead of per-request.

---

## Benchmark Methodology

### Running Benchmarks Locally

```powershell
# Full benchmark suite (runtime + router)
make benchmark

# Individual packages
go test -bench=. -benchmem -benchtime=5s ./pkg/core/runtime/
go test -bench=. -benchmem -benchtime=5s ./pkg/events/

# Specific benchmark with profiling
go test -bench=BenchmarkGoVsRawGoroutine -benchmem -cpuprofile=cpu.prof ./pkg/core/runtime/
go tool pprof cpu.prof
```

### Interpreting Results

```text
BenchmarkGoVsRawGoroutine/LifecycleGo-8    500000    5234 ns/op    256 B/op    4 allocs/op
                                            ^^^^^^    ^^^^^^^^^^^^  ^^^^^^^^^   ^^^^^^^^^^^^^
                                            iterations  time/op      bytes/op    allocations
```

- **ns/op**: Lower is better (nanoseconds per operation)
- **B/op**: Memory allocated per operation (lower = less GC pressure)
- **allocs/op**: Number of heap allocations (fewer = better)

### Known Benchmark Limitations

- **Synthetic Workloads**: Benchmarks use minimal work (`_ = 1 + 1`). Real workloads (I/O, DB queries) will mask overhead.
- **Cold Start**: First iteration may include JIT warmup. Results stabilize after ~1s.
- **CI Variance**: GitHub Actions runners show ±30% variance. Local benchmarks are more reliable.

---

## Measured vs. Unmeasured (v1.6.2)

### ✅ Measured (Baselined as of v1.6.2)

- [x] `lifecycle.Go` vs raw goroutines
- [x] Stack capture overhead (3 modes)
- [x] Observer invocation impact
- [x] Router scaling (10-1000 routes)
- [x] Supervisor introspection (`State()` calls)

### ❌ Still Unmeasured

- [ ] **Shutdown latency**: How long to stop 1000 workers gracefully?
- [ ] **Memory footprint**: Peak RSS with 10K+ goroutines tracked
- [ ] **Cross-platform variance**: Windows vs Linux vs macOS performance deltas
- [ ] **Real-world workloads**: HTTP servers, database workers, file watchers

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

### Stable as of v1.6.5

- ✅ `pkg/events/filewatch.FileWatchSource` — Event-based file watching
- ✅ `pkg/events/webhook.WebhookSource` — HTTP trigger source (1MB payload limit)
- ✅ `pkg/events/health.HealthCheckSource` — Health status source

### Not Yet Marked (Audit Pending)

- ⚠️ `pkg/core/worker/suspend.Suspend` — Experimental; covered but not marked

**Deprecation Policy**: None documented yet (planned for v1.7+).

---

## Test Coverage Status

### High Coverage (>80%)

```log
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

---

## Worker Locking Pattern: Exceptions and Limitations

> **Context**: As of v1.6.3, all critical workers have migrated to the `withLock`/`withLockResult` pattern to ensure concurrency safety and state consistency. However, some exceptions and limitations are important for ongoing maintenance and project evolution.

### Documented Exceptions

- **BaseWorker**: Does not use `withLock`/`withLockResult` directly, as it is a generic base and does not have a polymorphic lock wrapper. Manual locking (`mu.Lock`/`mu.Unlock`) remains to ensure compatibility and flexibility for custom workers.
- **Event workers and external components**: Components outside `pkg/core/worker` (e.g., event workers, custom handlers) may use their own locks or different patterns, and are not directly affected by this standardization.
- **Generic utility functions**: The `withLockAny`/`withLockResultAny` helpers were created to allow safe locking on any struct with a `mu sync.(R)WMutex` field, without requiring specific type dependencies.
- **Locking API**: The `withLock`/`withLockResult` pattern is recommended for all new workers and future maintenance, but is not mandatory for legacy code or cases where manual locking is more appropriate.

### References

- See [TECHNICAL.md](TECHNICAL.md#15-known-limitations) for technical details and coverage philosophy.
- See [PLANNING.md](PLANNING.md) for decision history and task tracking.
