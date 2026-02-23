---
Title: Implementation Plan - Script console.log capture
Ticket: PC-03-SCRIPT-CONSOLE-LOG-CAPTURE
Status: active
Topics:
    - backend
    - javascript
    - observability
    - api
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: internal/scriptengine/engine.go
      Note: Script runtime where console bridge and log capture will be implemented
    - Path: internal/server/script.go
      Note: Script completion path that persists ScriptOutput.logs
    - Path: proto/plz_confirm/v1/widgets.proto
      Note: ScriptOutput.logs schema contract already exists and must stay stable
    - Path: pkg/doc/js-script-api.md
      Note: User-facing API docs for script output and runtime context
    - Path: internal/scriptengine/engine_test.go
      Note: Runtime tests for sandbox behavior and helper exposure
ExternalSources: []
Summary: Plan for exposing and capturing script console logs end-to-end.
LastUpdated: 2026-02-22T21:58:00-05:00
WhatFor: ""
WhenToUse: ""
---

# Implementation Plan - Script console.log capture

## Executive Summary

`ScriptOutput.logs` exists in the wire schema but the runtime does not expose `console`, so scripts currently crash when calling `console.log`. This ticket adds a bounded, safe console bridge in the Goja runtime and ensures logs are returned in script outputs without allowing logs to destabilize execution.

## Problem Statement

Today, script authors cannot inspect runtime state while debugging because:

- `console` is undefined in the VM.
- `ScriptOutput.logs` is usually empty even though the field exists.
- There is no byte/entry cap strategy, so naive logging could become a memory and payload risk once enabled.

This creates avoidable friction for script development, troubleshooting, and support.

## Proposed Solution

Implement runtime log capture in `internal/scriptengine` and propagate logs through normal completion responses.

1. Inject `console.log`, `console.warn`, and `console.error` into the VM before user code executes.
2. Route all console calls into a Go-side capture buffer with strict limits:
   - max entries
   - max bytes
   - max bytes per entry
3. Serialize JS arguments safely to strings (handle primitives, objects, `undefined`, cyclic values) with deterministic fallback text.
4. Prefix entries with severity (`log`/`warn`/`error`) and execution phase context (`describe`/`init`/`view`/`update`) for traceability.
5. Return captured entries in `InitAndViewResult.Logs` and `UpdateAndViewResult.Logs`, and ensure server completion paths preserve them in `ScriptOutput.logs`.
6. Update API docs to define behavior, limits, and non-goals.

## Design Decisions

### Decision 1: Bounded in-memory capture

Use strict caps instead of unbounded slices. If limits are reached, append one truncation sentinel and stop recording further entries.

Reasoning: scripts are untrusted inputs and logs are optional diagnostics, so hard resource bounds are required.

### Decision 2: Capture as plain strings

Do not add structured log objects to the wire format in this ticket. Keep compatibility with existing `repeated string logs`.

Reasoning: lowest-risk rollout that does not require protobuf changes or frontend shape changes.

### Decision 3: No live streaming in this ticket

Capture logs for request responses only; do not add WebSocket log streaming yet.

Reasoning: avoids protocol churn and keeps scope focused on core runtime behavior.

## Alternatives Considered

### Alternative A: Keep `console` disabled and document workaround

Rejected because it preserves current pain and ignores an existing logs field.

### Alternative B: Add a custom `ctx.log()` helper instead of `console`

Rejected because it is non-idiomatic for JS authors and duplicates existing expectations.

### Alternative C: Persist logs in `script_state` for every step

Deferred. It increases state payload size and replay complexity. We can add optional accumulation later if needed.

## Implementation Plan

1. Add a console bridge helper in `engine.go` that registers `console.log/warn/error` and writes into a bounded collector.
2. Implement safe argument formatting and truncation behavior.
3. Tag log entries with level and phase.
4. Populate `InitAndViewResult.Logs` and `UpdateAndViewResult.Logs` from the collector.
5. Verify `internal/server/script.go` continues to pass update logs to `ScriptOutput.logs` and define desired create-path behavior.
6. Add tests:
   - `console.log` no longer throws
   - multi-arg logging
   - limit/truncation behavior
   - cyclic object formatting fallback
7. Update `pkg/doc/js-script-api.md` and `pkg/doc/js-script-development.md` with runtime logging semantics.
8. Add a manual smoke script under this ticket's `scripts/` for verification.

## Open Questions

- Should non-terminal step logs be accumulated across the full request lifecycle or only returned for the final update call?
- Should truncation metadata be exposed separately in the future, or remain string-only?

## References

- `ttmp/2026/02/22/PC-01-ADD-JS-API--add-js-api-describe-extension/sources/local/plz-confirm-js.md`
- `proto/plz_confirm/v1/widgets.proto`
- `internal/scriptengine/engine.go`
