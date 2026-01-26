# Zombie Prevention Example

This example demonstrates how `pkg/proc` ensures child processes are cleaned up when the parent process exits manually or crashes.

## How to Run

```bash
# Run the example (spawns a child process, then exits)
go run examples/zombie/main.go
```

The output will look like:

```text
Parent starting child via proc.Start...
Child PID: 12345. Parent exiting now.
```

## Verification

To verify that the child process (PID 12345) has been terminated, check the process list.

### Windows (PowerShell)

```powershell
Get-Process -Id 12345
# Expected Output:
# Get-Process : Cannot find a process with the process identifier 12345.
```

### Linux (Bash)

```bash
ps -p 12345
# Expected Output:
# PID TTY          TIME CMD
# (No lines output, meaning process is gone)
```
