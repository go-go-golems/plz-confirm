---
Title: RuntimeFactory Migration Implications for plz-confirm Script Engine
Ticket: PC-03-USE-GOJA-RUNTIMEFACTORY
Status: active
Topics:
    - go
    - backend
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../go-go-goja/engine/factory.go
      Note: Factory/NewRuntime behavior and runtime construction details.
    - Path: ../../../../../../../go-go-goja/engine/runtime.go
      Note: Owned runtime lifecycle and Close semantics.
    - Path: internal/scriptengine/engine.go
      Note: Current runtime bootstrap
    - Path: internal/scriptengine/engine_test.go
      Note: Sandbox
    - Path: internal/server/script.go
      Note: Status mapping
    - Path: internal/server/script_test.go
      Note: Integration tests that validate HTTP status behavior for script flows.
    - Path: pkg/doc/js-script-development.md
      Note: Current contributor-facing runtime documentation requiring updates.
ExternalSources: []
Summary: Deep impact analysis for migrating plz-confirm script runtime setup from direct goja.New() to go-go-goja Factory/NewRuntime while preserving script sandbox and HTTP behavior.
LastUpdated: 2026-02-23T11:05:00-05:00
WhatFor: Analyze architecture, risk, and cleanup opportunities before implementing RuntimeFactory migration.
WhenToUse: Use before and during implementation of PC-03 to avoid sandbox regressions and lifecycle leaks.
---


# RuntimeFactory Migration Implications for plz-confirm Script Engine

## Executive Summary

`plz-confirm` currently creates a fresh bare Goja runtime with `goja.New()` on every script call (`InitAndView` and `UpdateAndView`) and intentionally exposes no host bridge (`require` is expected to be undefined).

`go-go-goja` now provides a reusable `Factory` abstraction (`NewBuilder() -> Build() -> NewRuntime(ctx)`) that prebuilds runtime bootstrap state and gives explicit runtime ownership via `rt.Close(ctx)`.

The migration is feasible and valuable, but a direct swap is not safe: a default `Factory` runtime currently exposes `require` and `console`, which violates current script sandbox expectations. The migration must therefore include explicit sandbox scrubbing and runtime lifecycle management.

## Scope and Non-Goals

In scope:
- Runtime bootstrap path in `internal/scriptengine`.
- Server interactions that depend on script engine errors/status mapping.
- Test and doc implications.

Out of scope:
- New script API capabilities (modules, filesystem, timers, network access).
- Frontend rendering contract changes.
- Persistence model changes.

## Current Runtime Architecture (Baseline)

### End-to-End Flow

1. `POST /api/requests` hits `internal/server/server.go` and, for `WidgetType_script`, calls `s.scripts.InitAndView(...)`.
2. `POST /api/requests/{id}/event` hits `internal/server/script.go` and calls `s.scripts.UpdateAndView(...)`.
3. `internal/scriptengine/engine.go` creates a new `goja.Runtime` each call, injects context/state/event, executes script lifecycle functions, and returns plain `map[string]any`.
4. `internal/server/script.go` converts maps to protobuf structs/views and maps errors to HTTP status codes.

### Runtime Invariants Today

- Fresh VM per call: `goja.New()` in both `InitAndView` and `UpdateAndView`.
- No host bridge by default (`require` and `process` expected undefined by tests).
- Timeout/cancellation enforced by `vm.Interrupt(...)` in `runWithTimeout`.
- Script state is externalized in Go maps and persisted server-side; VM is intentionally disposable.

## go-go-goja Factory Behavior Relevant to Migration

Key facts from `go-go-goja/engine`:
- `FactoryBuilder.Build()` freezes module + initializer composition once.
- `Factory.NewRuntime(ctx)` creates VM + event loop + runtime owner and returns `*engine.Runtime` with explicit `Close(ctx)`.
- `NewRuntime` currently calls both `f.registry.Enable(vm)` and `console.Enable(vm)`.

Empirical check against current codebase:

```text
typeof require => function
typeof process => undefined
typeof console => object
```

This means default factory runtime changes sandbox surface compared to current `plz-confirm` behavior.

## Findings (Code Quality + Cleanup)

## Runtime Engine

### 1) Duplicated Bootstrap Logic in `InitAndView` and `UpdateAndView`

Problem: Both methods repeat script loading, export checks, context setup, and helper attachment with small differences.

Where to look:
- `internal/scriptengine/engine.go` (`InitAndView`, `UpdateAndView`).

Example:
```go
vm := goja.New()
if _, err := vm.RunString(buildExportsProgram(in.GetScript())); err != nil {
    return fmt.Errorf("script load failed: %w", err)
}
scriptCtx := defaultScriptContext(in.GetProps())
if err := vm.Set("__pc_ctx", scriptCtx); err != nil {
    return fmt.Errorf("set ctx failed: %w", err)
}
```

Why it matters:
- Higher maintenance cost and drift risk when runtime setup changes.
- Migration to factory path would require touching duplicated blocks twice.

Cleanup sketch:
```go
type runtimeSession struct {
    vm *goja.Runtime
    close func(context.Context) error
}

func (e *Engine) withSession(ctx context.Context, in *v1.ScriptInput, fn func(*runtimeSession) error) error
func (e *Engine) loadScriptAndContext(s *runtimeSession, in *v1.ScriptInput) error
func (e *Engine) validateExports(s *runtimeSession, required ...string) error
```

### 2) Runtime Lifecycle Is Implicit Today, but Must Become Explicit with Factory

Problem: Current runtime has no close lifecycle. Factory runtimes own event loops and must be closed.

Where to look:
- `internal/scriptengine/engine.go` (runtime creation points).
- `go-go-goja/engine/factory.go` (`NewRuntime`).
- `go-go-goja/engine/runtime.go` (`Close`).

Example:
```go
loop := eventloop.NewEventLoop()
go loop.Start()
...
func (r *Runtime) Close(ctx context.Context) error {
    if r.Owner != nil { _ = r.Owner.Shutdown(ctx) }
    if r.Loop != nil { r.Loop.Stop() }
}
```

Why it matters:
- Missing `Close` after migration can leak goroutines and runtime resources.
- Error paths (load failure, validation failure, timeout) must still close runtime.

Cleanup sketch:
```go
rt, err := e.runtimeProvider.NewRuntime(ctx)
if err != nil { return err }
defer func() { _ = rt.Close(ctx) }()

// proceed with script execution
```

### 3) Sandbox Contract Drift Risk (`require` Exposure)

Problem: `plz-confirm` tests currently require `require === undefined`, but factory runtime exposes `require` by default.

Where to look:
- `internal/scriptengine/engine_test.go` (`TestSandboxHasNoHostBridge`).
- `go-go-goja/engine/factory.go` (`f.registry.Enable(vm)`, `console.Enable(vm)`).

Example:
```go
if out.State["noRequire"] != true {
    t.Fatalf("expected require to be unavailable")
}
```

Why it matters:
- Security/sandbox regression.
- Existing tests and docs become incorrect.
- Potentially enables script authors to depend on behavior we do not want to support.

Cleanup sketch:
```go
type stripGlobalsInit struct{}
func (stripGlobalsInit) ID() string { return "pc-strip-host-globals" }
func (stripGlobalsInit) InitRuntime(ctx *engine.RuntimeContext) error {
    _, err := ctx.VM.RunString(`
      delete globalThis.require;
      delete globalThis.console;
      delete this.require;
      delete this.console;
    `)
    return err
}

factory := engine.NewBuilder().
    WithRuntimeInitializers(stripGlobalsInit{}).
    Build()
```

### 4) Error Classification Uses String Heuristics

Problem: HTTP mapping in server layer depends on string matching (`"timeout"`, `"cancel"`, `"must export"`, etc.).

Where to look:
- `internal/server/script.go` (`statusForScriptError`).

Example:
```go
msg := strings.ToLower(err.Error())
case strings.Contains(msg, "timeout"):
    return http.StatusGatewayTimeout
```

Why it matters:
- Refactors or wrapped errors can change text and silently alter HTTP status behavior.
- Migration will likely introduce new wrapped errors from runtime setup/close paths.

Cleanup sketch:
```go
var (
    ErrScriptTimeout = errors.New("script-timeout")
    ErrScriptCancelled = errors.New("script-cancelled")
    ErrScriptValidation = errors.New("script-validation")
)

// engine wraps with sentinel
return fmt.Errorf("...: %w", ErrScriptValidation)

// server maps by errors.Is
switch {
case errors.Is(err, ErrScriptTimeout): ...
}
```

### 5) Monolithic Embedded JS Helper Program in Go String

Problem: `buildExportsProgram` contains large helper JS (branching DSL) inline as string concatenation.

Where to look:
- `internal/scriptengine/engine.go` (`buildExportsProgram`).

Example:
```go
return `
var __pc_module = { exports: {} };
...
function __pc_branch(state, event, spec) { ... }
` + script + `
var __pc_exports = __pc_module.exports;
`
```

Why it matters:
- Harder to test helper logic separately.
- Harder to evolve runtime setup and bootstrap safely during migration.

Cleanup sketch:
```go
//go:embed bootstrap.js
var bootstrapJS string

func buildProgram(userScript string) string {
    return bootstrapJS + "\n" + userScript + "\nvar __pc_exports = __pc_module.exports;"
}
```

## Runtime/Performance Implications

- Factory prebuild can reduce repeated registry/bootstrap setup overhead when many runtimes are created.
- For `plz-confirm`, scripts are generally short and single-request scoped, so the bigger value is consistency and lifecycle abstraction rather than raw throughput.
- Factory runtime includes event loop/owner setup; this adds overhead versus bare `goja.New()`, so parity benchmarking should be run in `plz-confirm` context before default cutover.

## Behavioral Compatibility Matrix

| Concern | Current | Naive Factory Migration | Required Action |
|---|---|---|---|
| `require` availability | `undefined` | `function` | Strip `require` in runtime initializer and test it |
| `process` availability | `undefined` | `undefined` | Keep explicit test |
| `console` availability | likely undefined | object | Decide policy, then enforce (strip or document) |
| Timeout semantics | `vm.Interrupt` + timeout context | still possible | Keep `runWithTimeout` + assert status mapping |
| Runtime lifecycle | GC-only | explicit close needed | Always `defer rt.Close(ctx)` |
| Error -> HTTP mapping | text heuristics | fragile with wrapped errors | move to sentinel errors |

## Recommended Migration Strategy

Recommended: staged migration with runtime provider abstraction in `internal/scriptengine`.

Why:
- Allows introducing factory runtime without immediate behavior drift.
- Keeps rollback simple.
- Creates one seam for future runtime choices.

Proposed seam:
- `type runtimeProvider interface { NewRuntime(ctx context.Context) (scriptRuntime, error) }`
- `scriptRuntime` contains `VM` and `Close`.
- Start with direct-goja provider.
- Add factory provider behind config/flag.
- Make factory provider default after parity tests.

## Test Impact Plan

Tests to preserve/extend:

1. `internal/scriptengine/engine_test.go`
- Keep `TestSandboxHasNoHostBridge`.
- Add explicit check for `typeof console` policy.
- Add test that repeated calls do not leak resources (bounded runtime close behavior smoke test).

2. `internal/server/script_test.go`
- Keep timeout/cancel/validation/runtim-fault status tests.
- Add cases for setup failures from runtime provider (should map predictably).

3. Optional benchmark
- Add focused benchmark in `internal/scriptengine`:
  - `BenchmarkInitAndView_DirectGoja`
  - `BenchmarkInitAndView_FactoryRuntime`

## Documentation Impact

Docs requiring updates during implementation:
- `pkg/doc/js-script-development.md` currently states direct Goja and no wrapper; this must be revised to explain runtime provider/factory path.
- If sandbox policy keeps `require` unavailable, documentation must state this explicitly (and mention that internal runtime abstraction does not imply module access in user scripts).

## Implementation Risks and Mitigations

- Risk: runtime leaks from missed `Close` on early returns.
  - Mitigation: single `withRuntime(...)` helper handling close in one place.

- Risk: sandbox regression (`require` unexpectedly available).
  - Mitigation: runtime initializer strip + invariant tests.

- Risk: HTTP regression due to changed error text.
  - Mitigation: sentinel errors + `errors.Is` mapping.

- Risk: accidental module enablement in future builder usage.
  - Mitigation: enforce builder construction in one function with no `WithModules(...)` call and code comment explicitly warning against host bridge expansion.

## Cleanup Opportunities Beyond Migration

Low-risk:
- Extract shared setup helpers from `InitAndView`/`UpdateAndView`.
- Move bootstrap JS to embedded file.
- Add structured engine error types.

Medium-risk:
- Add runtime provider seam and factory-backed implementation.
- Add explicit policy toggles for sandbox globals.

Higher-risk (defer):
- Script compile caching across calls.
- Partial VM reuse.
- Any module/capability expansion.

## Conclusion

The RuntimeFactory abstraction is worth adopting for `plz-confirm`, but only with explicit sandbox-preserving controls and runtime lifecycle ownership. The highest-leverage path is a staged provider-based refactor that keeps current script contract stable while reducing bootstrap duplication and making runtime behavior easier to reason about.
