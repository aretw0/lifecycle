# Examples

This directory contains examples demonstrating how to use `lifecycle` across different complexity levels.

## Learning Path

We recommend exploring these examples in order:

### Level 0: The Problem

* **[zombie/](zombie/)**: Demonstrates the problem `lifecycle` solves. A parent process crashes, leaving a child process (Zombie) running forever. `lifecycle` solves this by default.

### Level 1: Foundations

* **[basic/](basic/)**: The "Hello World".
  * Setup `lifecycle.Run` to manage the main context.
  * Use `lifecycle.Go` for safe, tracked concurrency.
  * Handle standard signals (`SIGINT`, `SIGTERM`).

### Level 2: Intermediate

* **[hooks/](hooks/)**: Graceful Shutdown.
  * Registering `OnShutdown` hooks.
  * Managing hook timeouts and dependencies.
* **[termio/](termio/)**: Safe Input/Output.
  * Handling `Ctrl+C` interrupt during blocking `Read()` calls on Windows/Linux.
  * Preventing "Batch Job Terminate?" prompts on Windows.

### Level 3: Advanced

* **[control/](control/)**: The Control Plane (v2.x).
  * Event-Driven architecture (`Router`).
  * Handling `Webhook` events (e.g., Hot Reload).
  * Introspection and custom Event sources.
* **[supervisor/](supervisor/)**: Process Supervision.
  * Managing a tree of child processes (Workers).
  * Restart strategies (OneForOne, etc.).

## Running Examples

```bash
# Basic
go run ./examples/basic

# Control Plane
go run ./examples/control
# In another terminal: curl -X POST http://localhost:8080/reload
```
