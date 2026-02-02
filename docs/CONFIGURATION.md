# Configuration Reference

This document serves as the central reference for configuring `lifecycle` components.
We use the **Functional Options Pattern** to provide flexible and readable configuration.

## 1. Signal Context (`lifecycle.Run`)

Configures how the application reacts to OS signals (Shutdown Management).

```go
lifecycle.Run(job, 
    signal.WithInterrupt(true),         // Handle SIGINT (Ctrl+C)
    signal.WithForceExit(2),            // Force exit after 2nd signal
    signal.WithHookTimeout(5*time.Second), // Max time for cleanup hooks
)
```

| Option | Default | Description |
| :--- | :--- | :--- |
| `WithInterrupt(bool)` | `true` | If true, `SIGINT` (Ctrl+C) triggers graceful shutdown. If false, it is ignored (useful for shells). |
| `WithForceExit(count int)` | `2` | Number of signals required to force `os.Exit(1)`. Set to 0 to disable. |
| `WithHookTimeout(d time.Duration)` | `5s` | Maximum duration to wait for a single `OnShutdown` hook before logging a warning. |

## 2. Control Router (`lifecycle.NewRouter`)

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
