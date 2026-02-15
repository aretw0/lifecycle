# Ecosystem Analysis: Fiscus (The Clean Slate)

**Fiscus** is currently a stub project (`main.go` only). This presents a unique opportunity to use it as the **Reference Implementation** for `lifecycle` best practices from day one.

## Current State

- **Stage**: Greenfield / Stub.
- **Compliance**: N/A.

## The Proposal

Fiscus should be built **Lifecycle-First**.

1. **Main Entrypoint**: Start immediately with `lifecycle.Run` instead of `func main()`.
2. **Domain Design**: Design the "Financial Transaction" domain events to implement `lifecycle.Introspectable` interfaces.
3. **IO**: strict usage of `lifecycle.Context` for all DB and API calls.

Unlike Loam (Retrofit) or Arbour (Indirect), Fiscus will be the **Native** implementation, showcasing how to build a sovereign application correctly from the first commit.
