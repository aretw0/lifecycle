# Deprecation Policy

> **Effective Date**: February 2026 (v1.7.0)

As `lifecycle` matures into a stable control plane for modern Go applications, ensuring backward compatibility and smooth transition paths for breaking changes is paramount. This document outlines the formal deprecation lifecycle for APIs, behaviors, and features.

## 1. The Deprecation Lifecycle (3 Phases)

We adhere to a strict 3-phase lifecycle for any public API identified for removal or significant structural change.

### Phase 1: Notice (Documentation)

* **Trigger**: A better alternative is introduced, or an architectural flaw is identified.
* **Action**:
  * The API is marked with the `// Deprecated: Use [Alternative] instead.` directive in the source code.
  * The `LIMITATIONS.md` or Release Notes clearly call out the upcoming change.
* **Impact**: Zero runtime impact. Linters (like `staticcheck` или `go vet`) will begin warning developers at compile time.

### Phase 2: Active Warning (Runtime)

* **Trigger**: One minor version after Phase 1 (e.g., Notice in v1.7, Warning in v1.8).
* **Action**:
  * The deprecated code paths will emit a `slog.Warn` or `log.Printf` message indicating that the function is deprecated and will be removed in the next major/minor milestone.
* **Impact**: Logs will become noisy for consumers still using the old API, acting as an active push to migrate, but the feature will continue to function exactly as before.

### Phase 3: Removal / Hard Breaking

* **Trigger**: The next major version (`v2.0`), or after at least two minor versions of warning (e.g., Warning in v1.8, Removal in v1.10).
* **Action**:
  * The API is completely removed from the codebase.
  * Alternative: If keeping the API signature is necessary but the behavior is unsafe, it may be stubbed to `panic("deprecated")` if removal breaks critical interfaces, though outright removal is preferred.
* **Impact**: Code will fail to compile.

## 2. Exceptions to the Rule

The deprecation policy **does not apply** to:

1. **Experimental Features**: Any API formally documented or commented as `Experimental` or `Unstable` can be modified or removed at any time without a deprecation cycle.
2. **Security Vulnerabilities**: If an API is inherently insecure and cannot be fixed without breaking changes, it may be removed or fundamentally altered in a patch release.
3. **Internal Packages**: Anything under an `internal/` directory is exempt from stability guarantees.

## 3. Communication Strategy

All deprecations will be listed in the **[Release Releases](https://github.com/aretw0/lifecycle/releases)** page under a dedicated `Deprecated` header. When Phase 3 (Removal) is reached, it will be listed under `Breaking Changes`.
