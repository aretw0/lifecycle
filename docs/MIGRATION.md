# Migration Notes (v1.4.x → v1.5+)

This document is now a **historical summary** of the v1.5 Control Plane migration.
The goal is to preserve context without maintaining a full upgrade manual.

## Why No v2?

The Control Plane changes were originally planned as “v2,” but the project stayed on v1
to avoid breaking the module path. This kept `go.mod` stable at the cost of API churn.
For the rationale and decision trail, see:

* [docs/DECISIONS.md](DECISIONS.md)
* [docs/TECHNICAL.md](TECHNICAL.md)
* [docs/PLANNING.md](PLANNING.md)

## What Changed (Short Form)

* Signals evolved into an **Event Router** Control Plane (see [docs/TECHNICAL.md](TECHNICAL.md#11-event-router-source---handler)).
* Goroutines are now **tracked** via `lifecycle.Go`.
* Workers gained **Suspend/Resume** and are managed by a Supervisor.
* Shutdown uses **hooks** and structured phases.
* Context propagation is now enforced across all core operations.

## Migration Entry Points (Current)

* **Preferred**: `lifecycle.Run` for full lifecycle wiring.
* **Manual**: `lifecycle.Context()` for gradual adoption.

> [!NOTE]
> `lifecycle.Context` is now a function. The signal context type alias is `lifecycle.SignalContext`.

## Getting Help

* **Documentation**: See [TECHNICAL.md](TECHNICAL.md) for architecture details.
* **Examples**: The `examples/` directory contains runnable migration patterns.
* **Issues**: Report problems at <https://github.com/aretw0/lifecycle/issues>

## Version History Reference

* **v1.0 - v1.2**: Signal handling and terminal I/O primitives.
* **v1.3**: Introduction of Worker and Supervisor.
* **v1.4**: Reliability primitives (Critical Sections, Introspection).
* **v1.5**: Event-Driven Control Plane (this release).
* **v1.6+**: Planned ecosystem integrations (see [PLANNING.md](PLANNING.md)).
