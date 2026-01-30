# Zombie Prevention Example

This example demonstrates how `pkg/proc` ensures child processes are cleaned up when the parent process exits manually or crashes.

## How to Run

```bash
# Run the example (spawns a child process, then exits immediately)
go run examples/zombie/main.go
```

The output will look like (with `LogProvider` enabled):

```text
INFO successfully opened Windows CONIN$ for interruptible reads
INFO initialized Windows Job Object for process hygiene
Parent starting child via proc.Start...
DEBUG assigned process to job object pid=12345
DEBUG metric incremented name=lifecycle_processes_started_total
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

## Internal Mechanism

- **Windows**: Uses **Job Objects** with `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`. When the parent process handle to the Job Object closes (at death), the OS automatically kills all associated children.
- **Linux**: Uses `PR_SET_PDEATHSIG` to receive a `SIGKILL` when the parent dies, ensuring no residual processes.
- **Metrics**: Each successful start is recorded via `metrics.IncProcessStarted()`.
