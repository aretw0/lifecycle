# Control Plane Example

This example demonstrates the **v2.0 Control Plane** architecture, where multiple event sources (OS Signals, Webhooks) are routed to different reactions.

## How it works

1. **Router**: Acts as the central hub (`http.ServeMux` style).
2. **Sources**:
    - **OS Signals**: Listens for `SIGINT` (Ctrl+C).
    - **Webhook**: Listens on `:8080` for HTTP requests.
3. **Reactions**:
    - **Shutdown**: Cancels the main context.
    - **Reload**: Simulates a config reload without stopping the app.

## Running the Example

```bash
go run ./examples/control_plane/main.go
```

## Triggering Events

### 1. OS Signal (Shutdown)

Press `Ctrl+C` in the terminal.

**Output:**

```text
🛑 Received Signal: Signal(interrupt)
👋 Shutdown Complete
```

### 2. Webhook (Reload)

Open a new terminal and run:

```bash
curl -X POST http://localhost:8080/reload
```

**Output:**

```text
🔄 Reload triggered via Webhook!
✅ Configuration Reloaded
```

### 3. Webhook (Shutdown)

Run:

```bash
curl -X POST http://localhost:8080/stop
```

**Output:**

```text
🛑 Stop triggered via Webhook!
👋 Shutdown Complete
```
