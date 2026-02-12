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
| `lifecycle.SetMetricsProvider(p)` | Connects `lifecycle` internal metrics to your observability backend (Prometheus, OTel). |
