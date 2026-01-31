// Package worker defines interfaces and implementations for managed units of work.
//
// It provides a uniform `Worker` interface to manage the lifecycle (Start, Stop, Wait)
// of various execution units such as OS processes, Goroutines, or Containers.
//
// key features:
//   - **Worker Interface**: A standard contract for starting, stopping, and waiting for work.
//   - **Functional Worker**: Simple adapter `FromFunc` to turn any function into a Worker.
//   - **Process Worker**: A wrapper around `os/exec` that ensures process hygiene (Fail-Closed)
//     using `pkg/proc` logic (Job Objects on Windows, PDeathSig on Linux).
//   - **Container Worker**: A bridge to manage containerized workloads via `pkg/container`.
//   - **Handover Protocol**: Standard environment variables (`LIFECYCLE_RESUME_ID`, `LIFECYCLE_PREV_EXIT`)
//     to pass context across worker restarts.
//
// This package is foundational for the Supervisor pattern (v1.3+).
package worker
