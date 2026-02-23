---
Title: 'Refactor Plan: Adopt go-go-goja RuntimeFactory in Script Engine'
Ticket: PC-03-USE-GOJA-RUNTIMEFACTORY
Status: active
Topics:
    - go
    - backend
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../go-go-goja/engine/factory.go
      Note: Builder/Factory/NewRuntime API used by proposed provider implementation.
    - Path: ../../../../../../../go-go-goja/engine/module_specs.go
      Note: RuntimeInitializer interface used for sandbox stripping initialization.
    - Path: ../../../../../../../go-go-goja/engine/runtime.go
      Note: Runtime ownership and required close lifecycle.
    - Path: internal/scriptengine/engine.go
      Note: Primary refactor target for provider abstraction and setup deduplication.
    - Path: internal/server/script.go
      Note: Consumer of engine errors and script lifecycle outputs; status mapping update target.
    - Path: internal/server/server.go
      Note: Script create path that calls InitAndView and must keep behavior parity.
    - Path: pkg/doc/js-script-development.md
      Note: Developer documentation that must be updated after migration.
ExternalSources: []
Summary: Phased implementation plan to migrate script runtime bootstrap to go-go-goja Factory/NewRuntime while preserving plz-confirm sandbox and HTTP behavior.
LastUpdated: 2026-02-23T11:10:00-05:00
WhatFor: Provide an execution-ready technical design and rollout plan for PC-03.
WhenToUse: Use when implementing and reviewing RuntimeFactory migration work.
---


# Refactor Plan: Adopt go-go-goja RuntimeFactory in Script Engine

This design doc is superseded by `design-doc/02-implementation-plan-factory-hard-cut-with-require-and-console-log-capture.md` after updated product decisions on `require` and console log behavior.

## Executive Summary

This design introduces a runtime provider seam inside `internal/scriptengine`, then migrates runtime creation from `goja.New()` to `go-go-goja` `Factory.NewRuntime(ctx)` in a controlled sequence.

Primary constraints:
- Keep existing script contract unchanged (`describe/init/view/update`).
- Preserve sandbox invariants (`require` must remain unavailable to user scripts).
- Preserve server HTTP status behavior for timeout/cancel/validation/runtime faults.
- Avoid runtime lifecycle leaks by explicitly closing owned runtimes.

## Problem Statement

Current code has three limitations:

1. Runtime setup is duplicated in `InitAndView` and `UpdateAndView`.
2. Runtime construction is tightly coupled to raw `goja.New()`, making future runtime policy changes hard.
3. Error classification is text-based and brittle across refactors.

New `go-go-goja` runtime factory abstraction can improve construction consistency and lifecycle ownership, but direct adoption would expose `require` by default and regress sandbox behavior.

## Proposed Solution

### 1) Introduce Runtime Provider Abstraction in `internal/scriptengine`

Create an internal interface and runtime handle type:

```go
type runtimeHandle struct {
    VM    *goja.Runtime
    Close func(context.Context) error
}

type runtimeProvider interface {
    NewRuntime(context.Context) (*runtimeHandle, error)
}
```

Initial provider: `directGojaProvider` (parity baseline).

Second provider: `factoryProvider` backed by `go-go-goja` factory.

### 2) Add a Single Runtime Session Helper

Unify runtime lifecycle and timeout plumbing in one helper:

```go
func (e *Engine) withRuntime(ctx context.Context, in *v1.ScriptInput, run func(*goja.Runtime) error) error {
    h, err := e.runtimeProvider.NewRuntime(ctx)
    if err != nil { return wrapSetupErr(err) }
    defer func() { _ = h.Close(ctx) }()

    return runWithTimeout(ctx, h.VM, timeoutFromInput(in), func() error {
        return run(h.VM)
    })
}
```

This centralizes close behavior and prevents missed teardown in error paths.

### 3) Factory Construction with Sandbox-Preserving Runtime Initializer

Construct factory without modules and strip host globals right after runtime creation:

```go
type stripHostGlobals struct{}

func (stripHostGlobals) ID() string { return "pc-strip-host-globals" }
func (stripHostGlobals) InitRuntime(ctx *ggjengine.RuntimeContext) error {
    _, err := ctx.VM.RunString(`
      delete globalThis.require;
      delete this.require;
      delete globalThis.console;
      delete this.console;
    `)
    return err
}

factory, err := ggjengine.NewBuilder().
    WithRuntimeInitializers(stripHostGlobals{}).
    Build()
```

Important: do not call `WithModules(...)` for script runtime.

### 4) Typed Script Errors

Introduce sentinel/classified errors in `internal/scriptengine` and map by `errors.Is` in `statusForScriptError`.

Examples:
- `ErrScriptTimeout`
- `ErrScriptCancelled`
- `ErrScriptValidation`
- `ErrScriptRuntime`
- `ErrScriptSetup`

This removes reliance on string heuristics.

### 5) Keep Script Bootstrap Logic but Extract Helpers

Refactor duplicated setup into reusable helpers:
- `loadAndBindScript(vm, script)`
- `prepareContext(vm, props)`
- `requireExports(vm, names...)`

Optional follow-up in same ticket if low risk: move inline JS bootstrap to embedded `bootstrap.js`.

## Design Decisions

### Decision 1: Use a provider seam instead of hard direct swap

Rationale:
- Supports controlled rollout and rollback.
- Keeps implementation diff smaller and more reviewable.

### Decision 2: Preserve sandbox contract exactly

Rationale:
- Existing tests/docs/user expectations assume no `require` host bridge.
- Runtime abstraction should not silently alter user script capability model.

### Decision 3: Enforce explicit runtime close on all paths

Rationale:
- Factory runtime owns event loop and runtime owner resources.
- Prevents goroutine/resource leaks.

### Decision 4: Introduce typed errors in same refactor window

Rationale:
- Migration changes error wrapping layers.
- If not addressed, HTTP status behavior becomes fragile.

## Alternatives Considered

### A) Keep direct `goja.New()` and skip factory migration

Pros:
- Lowest risk.

Cons:
- Misses abstraction alignment with updated `go-go-goja`.
- Keeps duplication and weak lifecycle abstraction.

Status: rejected.

### B) Direct hard cut to factory without provider seam

Pros:
- Fewer files.

Cons:
- Larger blast radius.
- Harder rollback.
- Higher chance of sandbox/resource regressions.

Status: rejected.

### C) Enable factory + modules (`WithModules(DefaultRegistryModules())`)

Pros:
- Feature-rich runtime.

Cons:
- Explicitly violates current script sandbox goals.
- Expands attack/complexity surface.

Status: rejected.

## Implementation Plan

### Phase 0: Guardrails and Baseline

1. Add a runtime behavior test that asserts:
- `typeof require === "undefined"`
- `typeof process === "undefined"`
- chosen `console` policy.
2. Record baseline script engine test pass.
3. Add/refresh benchmark scaffold (optional but recommended).

Exit criteria:
- Existing `scriptengine` + `server` tests green.
- Sandbox contract codified in tests.

### Phase 1: Internal Runtime Provider + No Behavior Change

1. Add provider interface and direct-goja provider.
2. Route `InitAndView` and `UpdateAndView` through `withRuntime(...)`.
3. Keep current behavior exactly.

Exit criteria:
- No behavior diff.
- Tests unchanged except minor internal adjustments.

### Phase 2: Add Factory Provider (Not Yet Default)

1. Add factory provider implementation and constructor.
2. Add strip-host-globals runtime initializer.
3. Wire factory provider behind explicit opt-in env/config (for test gating).

Exit criteria:
- Both providers pass scriptengine tests.
- No leaked resources in stress test smoke run.

### Phase 3: Error Classification Hardening

1. Introduce typed errors in engine.
2. Update `statusForScriptError` to use `errors.Is` first, retain short-term fallback only if required.
3. Add status mapping tests for setup and wrapped runtime errors.

Exit criteria:
- HTTP status mapping stable and explicit.

### Phase 4: Default Cutover and Cleanup

1. Switch default provider to factory provider.
2. Remove temporary fallback/config toggle if no longer needed.
3. Update docs (`pkg/doc/js-script-development.md`).

Exit criteria:
- Factory is default.
- Documentation aligned.

### Phase 5: Optional Follow-ups

1. Extract JS bootstrap helpers into embedded file.
2. Add benchmark comparison artifact to ticket.
3. Evaluate whether to keep/remove `console` in sandbox policy.

## Testing and Validation Plan

Mandatory commands:

```bash
go test ./internal/scriptengine ./internal/server -count=1
```

Recommended additional checks:

```bash
go test ./internal/scriptengine -run TestSandboxHasNoHostBridge -count=1
go test ./internal/scriptengine -run TestTimeout -count=1
go test ./internal/server -run TestScriptCreateTimeoutMapsTo504 -count=1
```

Optional benchmark:

```bash
go test ./internal/scriptengine -run '^$' -bench BenchmarkInitAndView -benchmem -count=5
```

## Risks and Mitigations

- Risk: goroutine leaks due to missed runtime close.
  - Mitigation: single `withRuntime` helper; always defer close immediately after creation.

- Risk: sandbox regression through factory defaults.
  - Mitigation: strip-host-globals initializer + invariant tests.

- Risk: hidden HTTP status changes from wrapped errors.
  - Mitigation: typed errors + `errors.Is` mapping.

- Risk: accidental module enablement in future edits.
  - Mitigation: centralize factory construction in one private function with explicit comments and tests.

## Rollout Strategy

- Start in test-only dual-provider mode.
- Enable factory provider in local/dev first.
- Run script lifecycle integration tests.
- Switch default after parity evidence.

Rollback:
- Swap provider back to direct-goja implementation (single constructor/config pivot).

## Open Questions

1. Should `console` remain unavailable (strict parity) or be intentionally exposed and documented?
2. Should benchmark evidence be a merge requirement or informational only?
3. Should provider selection remain configurable after rollout, or be removed to simplify codepath?

## References

- `analysis/01-runtimefactory-migration-implications-for-plz-confirm-script-engine.md`
- `internal/scriptengine/engine.go`
- `internal/server/script.go`
- `go-go-goja/engine/factory.go`
- `go-go-goja/engine/module_specs.go`
- `go-go-goja/engine/runtime.go`
