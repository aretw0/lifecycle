# Hot Reload Example

This example demonstrates **cross-platform configuration hot reloading** using `FileWatchSource`.

## What It Does

- Watches `config.json` for changes
- Automatically reloads configuration when the file is saved
- **No restart required** — the application continues running
- **Works on Windows, Linux, and macOS**

## Running the Example

```bash
cd examples/reload
go run main.go
```

You should see:

```log
=== Hot Reload Example (Cross-Platform) ===

✅ Configuration loaded: {AppName:LifecycleDemo Debug:true Port:8080 Message:Hello from lifecycle hot reload!}

✨ App started with hot reload enabled!
   Edit config.json to see configuration reload in real-time
   Press Ctrl+C to exit

📊 LifecycleDemo running on port 8080 (debug=true): Hello from lifecycle hot reload!
```

## Testing Hot Reload

While the app is running, edit `config.json`:

```json
{
  "app_name": "MyAwesomeApp",
  "debug": false,
  "port": 9000,
  "message": "Configuration updated without restart!"
}
```

Save the file, and you'll immediately see:

```
🔄 Config file changed!

✅ Configuration loaded: {AppName:MyAwesomeApp Debug:false Port:9000 Message:Configuration updated without restart!}

📊 MyAwesomeApp running on port 9000 (debug=false): Configuration updated without restart!
```

## Technical Details

### FileWatchSource

Uses `fsnotify` library for efficient, event-driven file watching:

- **Linux**: Uses `inotify`
- **Windows**: Uses `ReadDirectoryChangesW`
- **macOS/BSD**: Uses `kqueue`

### Thread-Safe Configuration

Configuration updates use `sync.RWMutex`:

- **Write lock** during reload (exclusive)
- **Read lock** during access (shared)
- Ensures atomic configuration swaps

## Comparison: FileWatch vs SIGHUP

| Aspect | FileWatch | SIGHUP |
|:---|:---|:---|
| **Cross-platform** | ✅ Works everywhere | ❌ Unix-only |
| **Trigger** | Automatic (file save) | Manual (`kill -SIGHUP PID`) |
| **DX** | Edit & save → reload | Find PID → terminal command |
| **Kubernetes** | ✅ ConfigMap mounts | ❌ Doesn't integrate |

**Verdict**: FileWatch is superior for configuration hot reload.

## Related Examples

- **[control](../control/)**: Webhook-triggered reload (HTTP POST `/reload`)
- **[supervisor](../supervisor/)**: Worker restart patterns

---

**Note**: For production use, consider adding:

- Configuration validation before applying
- Rollback on invalid config
- Audit logging of config changes
