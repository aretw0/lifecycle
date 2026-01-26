# Planning: Lifecycle

## Roadmap

### v0.x: Foundation (Current)

- [x] **Signal Layer**: `pkg/signal` with `SignalContext` (Dual Signal support).
- [x] **I/O Layer**: `pkg/termio` with `InterruptibleReader` and Windows `CONIN$` support.
- [x] **Testing**: Unit tests for core logic.
- [ ] **Release Automation**: GoReleaser configuration.

### v1.0: Stability

- [ ] **Integration**: Adopted by `trellis` v0.8+.
- [ ] **BSD/Solaris Support**: Verify `termio.Open()` behavior on other Unixes.
- [ ] **Timeout Logic**: Add helpers for `Shutdown(ctx, timeout)`.

## Backlog

* **Raw Mode Helpers**: Consider wrapping `x/term` Raw Mode enter/restore logic if it becomes repetitive across projects? (Currently out of scope, aiming for minimalism).
- **Hook System**: A standard `OnShutdown(func())` registry?

## Technical Debt

* **Test Coverage**: `pkg/termio` relies heavily on manual verification for the "blocking read" scenarios due to lack of PTY in unit tests.
