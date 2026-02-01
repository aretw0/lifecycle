# Control Plane Example

This example demonstrates the **v2.0 Control Plane** features:

1. **Event Router**: Wiring `Sources` (Signals, Webhooks) to `Reactions`.
2. **Managed Concurrency**: Using `lifecycle.Group` to manage goroutines with panic recovery and observability.

## Running

```bash
go run main.go
```

## Features Demonstrated

* **Signal Handling**: Press `Ctrl+C` to trigger the shutdown reaction.
* **Panic Recovery**: Uncomment the panic line in `main.go` to see the group recover and exit gracefully.
* **Observability**: Metrics are logged to `debug` level.
