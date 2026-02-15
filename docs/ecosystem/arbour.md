# Ecosystem Analysis: Arbour (The Indirect Consumer)

**Arbour** represents the "Library of Libraries" consumption model. It does not depend on `lifecycle` directly, but inherits its benefits through **Trellis**.

## Current State

- **Dependency**: Indirect (via `github.com/aretw0/trellis`).
- **Signal Handling**: Delegated to Trellis Runner.

## The Validation

Arbour validates the architectural decision to keep `lifecycle` as a foundational, transitive dependency.

- **Zero Config**: Arbour developers don't need to configure `lifecycle`. It "just works" because Trellis handles the `SignalContext`.
- **Version Propagation**: With `lifecycle` v2.0 now available, when Trellis upgrades, Arbour will automatically gain "Suspend/Resume" capabilities for its WhatsApp connections without a single line of code change in Arbour itself.

## Strategic Value

Arbour is a testament to the "Core" philosophy: Solve the problem once in the foundation (Lifecycle/Trellis), and solved it everywhere (Arbour, etc).
