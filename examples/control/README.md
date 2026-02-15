# Control Plane Example

This example demonstrates the **Control Plane** features (v1.5+) of `lifecycle`.

## Features Demonstrated

1. **Control Router**: A mux-style event router (`control.Router`) that dispatches generic events to handlers.
2. **Event Sources**:
    - `OSSignalSource`: Turns `SIGINT`/`SIGTERM` into generic events.
    - `TickerSource`: Emits periodic "Tick" events (headless progress).
3. **Managed Concurrency**: Uses `lifecycle.Go(ctx, fn)` to spawn a background task that is automatically waited for during shutdown.
4. **Introspection**: You can print `router.Routes()` to see registered handlers (not shown in output but available).

## Run

```bash
go run ./examples/control/main.go
```

**Expected Output:**

```text
Application started. Press Ctrl+C to exit.
[Background] Task started (will run for 2s)
[Router] Tick(2023-10-27T10:00:00Z)
[Router] Tick(...)
[Background] Task finished
...
(Press Ctrl+C)
[Router] Received Signal: Signal(interrupt)
```
