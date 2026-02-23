---
Title: 'Implementation Plan: Factory Hard-Cut with require and Console Log Capture'
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
      Note: Factory/NewRuntime implementation used as new runtime source.
    - Path: ../../../../../../../go-go-goja/engine/module_specs.go
      Note: RuntimeInitializer hooks for console capture bootstrap policy.
    - Path: ../../../../../../../go-go-goja/engine/runtime.go
      Note: Owned runtime close lifecycle required to avoid leaks.
    - Path: internal/scriptengine/engine.go
      Note: Primary hard-cut runtime bootstrap target and console capture integration point.
    - Path: internal/server/script.go
      Note: Event handling path that must attach update and final run logs and map script errors.
    - Path: internal/server/server.go
      Note: Create-request script path that must attach init run logs to responses.
    - Path: internal/store/store.go
      Note: Patch and persistence path for latest script run logs.
    - Path: proto/plz_confirm/v1/request.proto
      Note: Planned contract addition for top-level scriptLogs field.
ExternalSources: []
Summary: Detailed no-compat implementation plan for migrating plz-confirm script runtime to go-go-goja factory, enabling require, and returning collected console logs in script HTTP responses.
LastUpdated: 2026-02-23T11:32:00-05:00
WhatFor: Provide execution-level plan after updated product decisions (allow require and capture console logs in responses).
WhenToUse: Use as the canonical implementation plan for PC-03 execution.
---


# Implementation Plan: Factory Hard-Cut with require and Console Log Capture

## Executive Summary

This plan performs a full no-compat cutover of `plz-confirm` script runtime internals to `go-go-goja` factory/runtime ownership APIs.

New requirements now locked in:
- `require` is allowed in script sandbox.
- `console.log` (and related console methods) must be captured by backend and returned in HTTP responses for script runs.
- No backwards compatibility path is required; implement as a direct drop-in refactor.

The implementation will:
1. Replace direct `goja.New()` paths with factory-owned runtimes.
2. Install deterministic console capture in runtime bootstrap.
3. Thread captured logs through create/update/complete responses.
4. Replace brittle string-only error mapping with typed engine errors.

## Requirement Delta (Supersedes Prior Assumptions)

This document supersedes earlier PC-03 assumptions that sandbox should hide `require` and `console`.

New policy:
- `require`: explicitly available.
- `console`: available; `console.*` output captured and returned as structured response data.
- No compatibility toggles: factory path becomes the only runtime creation path.

## Problem Statement

Current implementation gaps:

1. `internal/scriptengine/engine.go` uses duplicated direct `goja.New()` setup code.
2. `InitAndViewResult.Logs` and `UpdateAndViewResult.Logs` exist but are not populated from runtime console output.
3. Non-terminal script responses currently have no dedicated log channel.
4. Error-to-HTTP mapping relies heavily on string matching and becomes fragile as runtime setup changes.

## Proposed Solution

## 1) Hard-Cut Runtime Construction to Factory

Use `go-go-goja` factory in `internal/scriptengine` as the only runtime bootstrap mechanism.

Target behavior:
- Build factory once per `Engine` instance.
- Create fresh runtime per `InitAndView` / `UpdateAndView` call via `factory.NewRuntime(ctx)`.
- Always call `rt.Close(ctx)` via defer in a shared runtime helper.

Key files:
- `internal/scriptengine/engine.go`
- `go.mod` (add direct dependency if needed in module graph)

## 2) Console Capture Layer

Install a runtime initializer that provides capture-aware `console` methods:
- `console.log`
- `console.info`
- `console.warn`
- `console.error`

Collection model:
- Per script run (one `InitAndView` or one `UpdateAndView` invocation).
- Captured lines stored in collector and copied into result `.Logs`.
- Bounded memory: hard caps by line-count and total bytes.

Proposed limits:
- max lines: 200
- max bytes: 64 KiB
- if exceeded: append one sentinel line (for example `[system] log output truncated`)

Formatting rule:
- Each entry includes level prefix and argument join.
- Example: `[log] deploy step started env=prod`

Implementation sketch:

```go
type runLogCollector struct {
    lines []string
    bytes int
    truncated bool
}

func (c *runLogCollector) Add(level string, args ...goja.Value)
func installConsoleCapture(vm *goja.Runtime, c *runLogCollector) error
```

## 3) Response Contract for Logs

`ScriptOutput.logs` already exists, but only terminal outputs carry `script_output`.
To satisfy "logs returned in HTTP response to script runs" for create and non-terminal updates, add top-level request field:

```proto
// request.proto
repeated string script_logs = 29;
```

Semantics:
- `script_logs` = logs from latest script engine run that produced this response.
- On create (`POST /api/requests`): set to init/view run logs.
- On non-terminal event (`POST /api/requests/{id}/event`): set to update/view run logs.
- On terminal completion: set both:
  - `script_output.logs` for final output contract
  - `script_logs` for consistency of top-level response shape

Storage behavior:
- Persist latest run logs on request object (overwrite each run).
- Do not append unbounded log history to avoid request growth.

## 4) Store and Server Wiring

Required API/threading changes:

1. `internal/store/store.go`
- include `ScriptLogs` in `Create` copy.
- extend `PatchScript(...)` signature to accept logs:
  - from `(state, view)` to `(state, view, logs []string)`.
- set `e.req.ScriptLogs = logs` in patch path.
- preserve logs on completion path.

2. `internal/server/server.go`
- during script create flow, set `reqProto.ScriptLogs = initResult.Logs` before store create.

3. `internal/server/script.go`
- non-terminal event: pass `updateResult.Logs` to `PatchScript`.
- terminal event: set `output.ScriptOutput.Logs = updateResult.Logs` and ensure request `ScriptLogs` mirrors latest logs.

## 5) Error Model Hardening

Add typed sentinel errors in script engine package:
- `ErrScriptSetup`
- `ErrScriptValidation`
- `ErrScriptRuntime`
- `ErrScriptTimeout`
- `ErrScriptCancelled`

Use `errors.Is` mapping in `statusForScriptError` first; keep string fallback minimal and temporary.

This prevents status regressions when error messages change due to new runtime stack layers.

## Design Decisions

### Decision A: No compatibility mode

Rationale:
- User explicitly requested full drop-in.
- Reduces complexity and split-path testing overhead.

### Decision B: Keep `require` enabled by default

Rationale:
- Product requirement changed.
- Factory already exposes `require`; no stripping initializer needed.

### Decision C: Add explicit top-level `script_logs` instead of overloading `script_state`

Rationale:
- Clean transport contract.
- Keeps script state semantic (business state) separate from runtime diagnostics.

### Decision D: Overwrite latest logs per run, do not maintain full history by default

Rationale:
- Bounds response size.
- Avoids in-memory bloat for long-lived script sessions.

### Decision E: Capture `console.info/warn/error` alongside `console.log`

Rationale:
- Developers expect all console methods to behave consistently.

## Alternatives Considered

### 1) Keep logs only in final `ScriptOutput.logs`

Rejected because it does not satisfy requirement for logs in all script run responses (create and non-terminal update).

### 2) Embed logs inside `script_state` under reserved key

Rejected due to state pollution and risk of scripts depending on internal transport keys.

### 3) Dual runtime paths (old direct goja + new factory)

Rejected due to no-compat directive and extra maintenance/testing burden.

### 4) Keep default console behavior and scrape stdout logs

Rejected because it is non-deterministic, process-global, and hard to associate with a specific request/run.

## Implementation Plan

## Phase 1: Schema and generated types

Files:
- `proto/plz_confirm/v1/request.proto`
- generated Go/TS protobuf outputs

Steps:
1. Add `repeated string script_logs = 29;` to `UIRequest`.
2. Run codegen (`make codegen`).
3. Verify Go + TS generated code includes `scriptLogs`.

Acceptance criteria:
- Build compiles with new field.
- Existing non-script paths unaffected.

## Phase 2: Script engine factory hard-cut and console collector

Files:
- `internal/scriptengine/engine.go`
- (optional) new helper file `internal/scriptengine/console_capture.go`

Steps:
1. Replace raw `goja.New()` call sites with factory runtime creation.
2. Add engine-owned factory initialization in `New()`.
3. Install per-run console capture before script execution.
4. Populate `InitAndViewResult.Logs` and `UpdateAndViewResult.Logs` from collector.
5. Ensure runtime close in all code paths.

Acceptance criteria:
- No direct `goja.New()` remains in script engine run path.
- Logs populate result struct on both init and update execution.

## Phase 3: Server/store response plumbing

Files:
- `internal/server/server.go`
- `internal/server/script.go`
- `internal/store/store.go`

Steps:
1. Set `script_logs` on create response from init logs.
2. Extend `PatchScript` to accept and persist latest logs.
3. Set `script_logs` on update responses and complete responses.
4. Keep `ScriptOutput.logs` populated for terminal completion.

Acceptance criteria:
- `POST /api/requests` script create response includes `scriptLogs`.
- `POST /api/requests/{id}/event` response includes latest `scriptLogs` for both non-terminal and terminal runs.

## Phase 4: Error classification hardening

Files:
- `internal/scriptengine/engine.go`
- `internal/server/script.go`

Steps:
1. Introduce typed script errors.
2. Wrap engine failures with sentinels.
3. Update `statusForScriptError` to `errors.Is` first.

Acceptance criteria:
- Timeout/cancel/validation/runtime fault tests keep expected status behavior under wrapped errors.

## Phase 5: Tests

Files:
- `internal/scriptengine/engine_test.go`
- `internal/server/script_test.go`
- optional frontend/client tests for `scriptLogs` consumption if UI wiring is added.

Test additions:
1. Script engine:
- `require` is available (`typeof require === "function"`).
- `console` methods capture logs.
- log truncation behavior when cap exceeded.

2. Server integration:
- create script response includes `scriptLogs` with console output.
- non-terminal update response includes `scriptLogs`.
- terminal response includes both `scriptLogs` and `scriptOutput.logs`.

3. Regression:
- timeout and cancellation status mapping unchanged.

Commands:

```bash
go test ./internal/scriptengine ./internal/server ./internal/store -count=1
```

## Phase 6: Docs and operational updates

Files:
- `pkg/doc/js-script-development.md`
- `pkg/doc/js-script-api.md` (add section about `require` + `scriptLogs`)
- ticket docs/changelog

Steps:
1. Remove outdated statement that `require` is unavailable.
2. Document how console output is surfaced in API responses.
3. Add warning about log-size caps/truncation.

Acceptance criteria:
- contributor docs and API docs match new behavior.

## API Behavior Matrix (Post-Refactor)

| Endpoint | Script status | Response log fields |
|---|---|---|
| `POST /api/requests` | pending | `scriptLogs` populated from describe/init/view run |
| `POST /api/requests/{id}/event` | pending | `scriptLogs` populated from update/view run |
| `POST /api/requests/{id}/event` | completed | `scriptLogs` populated and `scriptOutput.logs` populated |
| `GET /api/requests/{id}` | any | returns persisted latest `scriptLogs` |
| `GET /api/requests/{id}/wait` | completed | includes final `scriptOutput.logs` and latest `scriptLogs` |

## Risks and Mitigations

- Risk: larger response payload due to logs.
  - Mitigation: strict collector caps and truncation sentinel.

- Risk: runtime leaks if close omitted.
  - Mitigation: centralize runtime ownership in one helper with immediate defer.

- Risk: `require` misuse for filesystem/module loading.
  - Mitigation: document behavior explicitly; optionally add future loader restrictions if required by policy.

- Risk: test fragility due to string formatting differences in logs.
  - Mitigation: assert contains/prefix patterns, not full exact strings where unnecessary.

## Rollout Plan

Single-branch no-compat rollout:
1. Implement all phases in one ticket branch.
2. Keep CI green for script engine + server test suites.
3. Merge as one coordinated change with docs and codegen outputs.

No fallback runtime path will be maintained.

## Open Questions

1. Should `console.debug` be captured too, or only log/info/warn/error?
2. Should frontend display `scriptLogs` live, or remain API-only initially?
3. Should `require` be unrestricted now, or should we immediately constrain loader roots in same ticket?

## References

- `design-doc/01-refactor-plan-adopt-go-go-goja-runtimefactory-in-script-engine.md` (earlier plan, superseded assumptions)
- `analysis/01-runtimefactory-migration-implications-for-plz-confirm-script-engine.md`
- `/home/manuel/workspaces/2026-02-22/plz-confirm-js/plz-confirm/internal/scriptengine/engine.go`
- `/home/manuel/workspaces/2026-02-22/plz-confirm-js/plz-confirm/internal/server/server.go`
- `/home/manuel/workspaces/2026-02-22/plz-confirm-js/plz-confirm/internal/server/script.go`
- `/home/manuel/workspaces/2026-02-22/plz-confirm-js/plz-confirm/internal/store/store.go`
- `/home/manuel/workspaces/2026-02-22/plz-confirm-js/plz-confirm/proto/plz_confirm/v1/request.proto`
- `/home/manuel/workspaces/2026-02-22/plz-confirm-js/plz-confirm/proto/plz_confirm/v1/widgets.proto`
- `/home/manuel/workspaces/2026-02-22/plz-confirm-js/go-go-goja/engine/factory.go`
