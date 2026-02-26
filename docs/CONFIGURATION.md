# Configuration Reference

This document serves as the central reference for configuring `lifecycle` components.
We use the **Functional Options Pattern** to provide flexible and readable configuration.

## 1. Signal Context (`lifecycle.Run`)

Configures how the application reacts to OS signals (Shutdown Management) and coordinates the root context.

```go
lifecycle.Run(job, 
    lifecycle.WithForceExit(2),            // Force exit after 2nd signal
    lifecycle.WithResetTimeout(1*time.Second), // Reset signal count after 1s
    lifecycle.WithHookTimeout(5*time.Second), // Max time for cleanup hooks
)
```

| Option | Default | Description |
| :--- | :--- | :--- |
| `WithForceExit(count int)` | `1` | Number of signals required to force `os.Exit(1)`. <br>• `1`: Instant shutdown on first signal.<br>• `>1`: Escalation mode (e.g. 2 = first signal cancels context, second forces exit).<br>• `0`: Unsafe mode (never force exits). |
| `WithResetTimeout(d time.Duration)` | `0` (Disabled) | Duration after which the internal signal counter resets. Useful to prevent accidental force exits from spaced-out signals. |
| `WithHookTimeout(d time.Duration)` | `5s` | Maximum duration to wait for a single `OnShutdown` hook before logging a warning. Does not abort the hook, only warns. |
| `WithCancelOnInterrupt(enabled bool)` | `true` | If `true`, the context is cancelled immediately on the first signal. If `false`, the context remains valid until `ForceExit` threshold is reached or manual cancellation. |
| `WithLogger(l *slog.Logger)` | `slog.Default()` | Sets the global logger for the runtime. |
| `WithMetrics(p metrics.Provider)`| `NoOp` | Sets the global metrics provider. |

## 2. Interactive Router (`lifecycle.NewInteractiveRouter`)

Pre-configured router for CLI applications with built-in signal and input handling.

```go
router := lifecycle.NewInteractiveRouter(suspendHandler,
    lifecycle.WithShutdown(func() { ... }),
    lifecycle.WithCommand("status", statusHandler),
)
```

| Option | Default | Description |
| :--- | :--- | :--- |
| `WithInput(bool)` | `true` | Enables/Disables reading commands from Stdin. |
| `WithSignal(bool)` | `true` | Enables/Disables OS signal handling (Interrupt/Term). |
| `WithCommand(name, handler)` | - | Registers a custom command (e.g., `command/status`). |
| `WithShutdown(func())` | `No-Op` | Convenience to handle `q`/`quit` commands. |
| `WithInputBackoff(d)` | `100ms` | Duration to wait before retrying after interrupts or errors. |
| `WithInputBufferSize(size)` | `1024` | Size of the internal I/O read buffer (in bytes). |
| `WithInputEventBuffer(size)` | `10` | Size of the event channel buffer. |

## 3. Control Router (`lifecycle.NewRouter`)

Configures the Event Bus for internal communication.

```go
router := lifecycle.NewRouter(
    lifecycle.WithEventBuffer(500),
)
```

| Option | Default | Description |
| :--- | :--- | :--- |
| `WithEventBuffer(size int)` | `100` | Size of the event channel buffer. Larger buffers handle bursty traffic better but consume more memory. |

## 4. Supervisors (`lifecycle.NewSupervisor`)

Configures the supervision tree for managing child workers.

```go
sup := lifecycle.NewSupervisor("root", lifecycle.SupervisorStrategyOneForOne,
    lifecycle.SupervisorSpec{
        Name: "worker-1",
        Factory: myWorkerFactory,
        RestartPolicy: lifecycle.RestartAlways,
        Backoff: lifecycle.Backoff{Min: 1*time.Second, Max: 5*time.Second},
    },
)
```

### Strategies

| Strategy | Description |
| :--- | :--- |
| `SupervisorStrategyOneForOne` | If a child process terminates, only that process is restarted. |
| `SupervisorStrategyOneForAll` | If a child process terminates, all other child processes are terminated and then restarted. |

### Restart Policies

| Policy | Description |
| :--- | :--- |
| `RestartAlways` | Always restart the worker, regardless of exit reason. |
| `RestartOnFailure` | Restart only if the worker returns an error. |
| `RestartNever` | Never restart. |

## 5. Probes & Sources

Configures event sources for health checks, webhooks, and file watching.

### Health Checks (`lifecycle.NewHealthCheckSource`)

```go
source := lifecycle.NewHealthCheckSource("db", checkFunc,
    lifecycle.WithHealthInterval(10*time.Second),
    lifecycle.WithHealthStrategy(lifecycle.TriggerLevel),
)
```

| Option | Default | Description |
| :--- | :--- | :--- |
| `WithHealthInterval(d time.Duration)` | `30s` | How often to run the check function. |
| `WithHealthStrategy(s TriggerStrategy)` | `TriggerEdge` | `TriggerEdge` (Emit only on status change) or `TriggerLevel` (Emit event on every check). |

### Webhooks (`lifecycle.NewWebhookSource`)

```go
source := lifecycle.NewWebhookSource(":8080",
    lifecycle.WithWebhookBuffer(50), // Note: Check pkg/events specific options if needed
)
```

## 6. Global Overrides

These settings affect the entire library state.

| Function | Description |
| :--- | :--- |
| `lifecycle.SetLogger(l *slog.Logger)` | Overrides the default logger used by all components if not explicitly provided. |
| `lifecycle.NewNoOpLogger()` | Returns a logger that discards all output (use with `WithLogger`). |
| `lifecycle.SetObserver(o lifecycle.Observer)` | Routes logs, process events, and panic callbacks to a custom observer (disables default slog when set). |
| `lifecycle.SetMetricsProvider(p)` | Connects `lifecycle` internal metrics to your observability backend (Prometheus, OTel). |
| `lifecycle.SetMetricsLabelPolicy(p *metrics.LabelPolicy)` | Sanitizes metric labels and enforces cardinality rules. |

> [!NOTE]
> Passing `nil` to `SetLogger` resets to the default logger — it does **not** silence logs.
> To disable logging entirely, use `NewNoOpLogger()`.

### Disabling Logs

```go
lifecycle.Run(job, lifecycle.WithLogger(lifecycle.NewNoOpLogger()))
```

### Observer Bridge (lifecycle + procio)

When using both `lifecycle` workers and `procio` processes, you can unify telemetry with a single adapter that implements both `Observer` interfaces.

> [!NOTE]
> The observer affects **log routing**, **process events**, and **panic callbacks**.
> Starting with **v1.7.2**, `lifecycle` automatically bridges its observer to `procio`. You do **not** need to call `procio.SetObserver` manually if your observer implements the I/O hooks.
> For behavior and stack capture semantics, see [docs/TECHNICAL.md](TECHNICAL.md#14-observability).

**Type definition:**

```go
import (
	"log/slog"

	"github.com/aretw0/lifecycle"
	"github.com/aretw0/lifecycle/pkg/core/metrics"
	"github.com/aretw0/procio"
)

// ObserverBridge implements both lifecycle.Observer and procio.Observer,
// routing logs to slog and process events to the metrics provider.
type ObserverBridge struct {
	Logger   *slog.Logger
	Provider metrics.Provider
}

// Compile-time interface checks (Recommended).
var _ lifecycle.Observer = (*ObserverBridge)(nil)

// Optional: Implement these to receive low-level I/O events from procio.
// lifecycle's Discovery Bridge will auto-detect these methods.
func (b *ObserverBridge) OnIOError(op string, err error) { ... }
func (b *ObserverBridge) OnScanError(err error)           { ... }

func (b *ObserverBridge) OnProcessStarted(pid int) {
	b.Logger.Info("process started", "pid", pid)
	b.Provider.IncProcessStarted()
	// Alternative: metrics.GetProvider().IncProcessStarted()
}

func (b *ObserverBridge) OnProcessFailed(err error) {
	b.Logger.Error("process failed", "error", err)
	b.Provider.IncProcessFailed()
}

func (b *ObserverBridge) OnGoroutinePanicked(recovered any, stack []byte) {
    if len(stack) == 0 {
        b.Logger.Error("goroutine panicked", "recover", recovered)
        return
    }
    // Only log stack when provided by the runtime.
    b.Logger.Error("goroutine panicked", "recover", recovered, "stack", string(stack))
}

func (b *ObserverBridge) LogDebug(msg string, args ...any) { b.Logger.Debug(msg, args...) }
func (b *ObserverBridge) LogInfo(msg string, args ...any)  { b.Logger.Info(msg, args...) }
func (b *ObserverBridge) LogWarn(msg string, args ...any)  { b.Logger.Warn(msg, args...) }
func (b *ObserverBridge) LogError(msg string, args ...any) { b.Logger.Error(msg, args...) }
```

**Setup:**

```go
bridge := &ObserverBridge{
	Logger:   slog.Default(),
	Provider: myMetricsProvider,
}

// 1. Set the lifecycle observer.
// 2. That's it! The internal Discovery Bridge handles procio integration.
lifecycle.SetObserver(bridge)
```
