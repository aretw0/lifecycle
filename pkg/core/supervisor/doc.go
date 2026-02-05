// Package supervisor implements the Supervisor Pattern for managing process and worker lifecycles.
//
// A Supervisor is a special type of Worker that manages a collection of child Workers.
// It is responsible for starting (spawning), monitoring (watching), and restarting (healing)
// its children based on defined strategies.
//
// # Strategies
//
// The supervisor supports the following restart strategies:
//
//   - StrategyOneForOne: If a child process terminates, only that process is restarted.
//   - StrategyOneForAll: If a child process terminates, all other child processes are terminated,
//     and then all child processes are restarted.
//
// # Backoff Strategy
//
// To prevent rapid restart loops, the supervisor implements an exponential backoff strategy
// with jitter. This is configurable per-worker via `Backoff` struct in `Spec`.
//
// # Dynamic Topology
//
// Workers can be added or removed dynamically at runtime using `Add(Spec)` and `Remove(name)`.
//
// # Handover Protocol
//
// The supervisor implements a Handover Protocol to provide continuity across restarts.
// Each worker spec is assigned a persistent `ResumeID`. When a worker is restarted,
// the supervisor injects this ID and the previous exit status (via `EnvInjector`)
// to allow the new instance to resume work or report status accurately.
//
// Usage
//
//	sup := supervisor.New("my-sup", supervisor.StrategyOneForOne,
//	    supervisor.Spec{
//	        Name: "worker-1",
//	        Factory: func() (worker.Worker, error) {
//	            return worker.NewProcess("worker-1", "sleep", "10"), nil
//	        },
//	    },
//	)
//
//	sup.Start(ctx)
package supervisor



