# Configuration Reference

This document serves as the central reference for configuring `lifecycle` components.
We use the **Functional Options Pattern** to provide flexible and readable configuration.

## 1. Signal Context (`lifecycle.Run`)

Configures how the application reacts to OS signals (Shutdown Management).

```go
lifecycle.Run(job, 
    signal.WithForceExit(2),            // Force exit after 2nd signal
    signal.WithHookTimeout(5*time.Second), // Max time for cleanup hooks
)
```

| Option | Default | Description |
| :--- | :--- | :--- |
| `WithForceExit(count int)` | `2` | Number of signals required to force `os.Exit(1)`. Set to 0 to disable. |
| `WithHookTimeout(d time.Duration)` | `5s` | Maximum duration to wait for a single `OnShutdown` hook before logging a warning. |
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

## 3. Control Router (`lifecycle.NewRouter`)

Configures the Event Bus.

```go
router := lifecycle.NewRouter(
    control.WithEventBuffer(500),
)
```

| Option | Default | Description |
| :--- | :--- | :--- |
| `WithEventBuffer(size int)` | `100` | Size of the event channel buffer. Larger buffers handle bursty traffic better but consume more memory. |

## 3. Webhook Source (`sources.NewWebhookSource`)

Configures the HTTP Event Receiver.

```go
source := sources.NewWebhookSource(":8080",
    sources.WithWebhookBuffer(50),
)
```

| Option | Default | Description |
| :--- | :--- | :--- |
| `WithWebhookBuffer(size int)` | `10` | Size of the internal event buffer before dropping incoming HTTP requests with `503 Service Unavailable`. |

## 4. Health Check Source (`sources.NewHealthCheckSource`)

Configures periodic health monitoring.

```go
source := sources.NewHealthCheckSource("db", checkFunc,
    sources.WithInterval(10*time.Second),
    sources.WithStrategy(sources.TriggerLevel),
)
```

| Option | Default | Description |
| :--- | :--- | :--- |
| `WithInterval(d time.Duration)` | `30s` | How often to run the check function. |
| `WithStrategy(s TriggerStrategy)` | `TriggerEdge` | `TriggerEdge` (Change only) or `TriggerLevel` (Every check). |
