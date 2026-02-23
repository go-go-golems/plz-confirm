---
Title: 'Implementation Plan: JS Describe Extension'
Ticket: PC-01-ADD-JS-API
Status: active
Topics:
    - backend
    - cli
    - go
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../go-go-goja/engine/runtime.go
      Note: Current engine bootstrap and blank-import module behavior used for dependency tradeoff analysis
    - Path: ../../../../../../../go-go-goja/modules/common.go
      Note: Native module adapter/registration pattern referenced by runtime design
    - Path: ../../../../../../../go-go-goja/pkg/runtimeowner/runner.go
      Note: Owner-thread runtime scheduling model and timeout/cancellation guidance
    - Path: agent-ui-system/client/src/components/WidgetRenderer.tsx
      Note: Frontend widget switch and submit flow integration point for script dialog
    - Path: agent-ui-system/client/src/services/websocket.ts
      Note: WS event handling and submit/touch API helpers
    - Path: internal/server/server.go
      Note: REST lifecycle and widget completion routing that must gain event/update semantics
    - Path: internal/store/store.go
      Note: Pending/completion storage model and timeout defaults
    - Path: proto/plz_confirm/v1/request.proto
      Note: Current UIRequest and WidgetType schema to extend for script/describe support
    - Path: proto/plz_confirm/v1/widgets.proto
      Note: Current widget message contracts and oneof output/input patterns
    - Path: ttmp/2026/02/22/PC-01-ADD-JS-API--add-js-api-describe-extension/tasks.md
      Note: Implementation backlog checklist derives directly from this design doc
ExternalSources:
    - local:plz-confirm-js.md
Summary: Detailed, intern-oriented implementation plan for adding a JS-driven widget API and describe extension across plz-confirm and go-go-goja.
LastUpdated: 2026-02-22T20:58:00Z
WhatFor: Plan the full architecture, contracts, and phased implementation for script/describe support with concrete file-level guidance.
WhenToUse: Use when implementing or reviewing JS describe extension work in plz-confirm and reusable runtime support in go-go-goja.
---



# Implementation Plan: JS Describe Extension

## Executive Summary

This document proposes how to add a JavaScript-driven interaction API to `plz-confirm`, centered on a new **describe extension** plus a script execution lifecycle. The goal is to let an agent submit a JavaScript program that:

1. declares its contract via `describe(ctx)`;
2. initializes state via `init(ctx)`;
3. renders UI trees via `view(state, ctx)`;
4. consumes user events via `update(state, event, ctx)`;
5. either returns an updated state (continue) or a final result (complete).

The implementation spans both repositories:

- `plz-confirm`: wire protocol, server lifecycle, store/event model, CLI command, and React renderer.
- `go-go-goja`: reusable runtime and module patterns to keep the JS bridge maintainable and safe.

The recommended path is **phased**:

- Phase 1: add protocol/runtime foundations with strict sandbox defaults and no async requirements.
- Phase 2: add frontend generic renderer for script nodes and event loop endpoint.
- Phase 3: harden with validation, tests, and observability.
- Phase 4: optional extraction/reuse improvements in `go-go-goja`.

## Problem Statement

### What is missing today

`plz-confirm` is currently optimized for one-shot widgets (confirm/select/form/upload/table/image):

- create request (`POST /api/requests`),
- render one known widget type,
- submit final output (`POST /api/requests/{id}/response`),
- mark completed.

This works well for static interactions, but fails for dynamic multi-step flows where the next screen depends on user input and intermediate control decisions.

### Why a describe extension is needed

The imported source (`sources/local/plz-confirm-js.md`) defines a script widget concept. However, to safely operationalize this in production, we need a reliable preflight contract that states what the script is and expects before execution. That contract is the **describe extension**:

- defines API version/compatibility,
- declares expected props and output shape,
- declares optional capabilities (files, secondary actions, etc.),
- gives the server/frontend enough metadata to validate behavior before runtime errors.

Without `describe`, onboarding and debugging become difficult because behavior is implicit in code only.

### Existing architectural constraints

The current codebase imposes concrete constraints we must design around:

- `UIRequest` is typed by protobuf oneofs (`request.proto`, `widgets.proto`) and serialized with `protojson`.
- Server lifecycle is completion-oriented (`internal/server/server.go` + `internal/store/store.go`).
- Frontend `submitResponse` path always implies completion (`WidgetRenderer.tsx`, `services/websocket.ts`).
- Store has one done-channel per request and no intermediate event stream.
- Goja runtime access must remain single-owner and time-bounded.

## Current Architecture Map (What Exists Now)

This section is intentionally explicit so a new intern can navigate confidently.

### plz-confirm wire model

Primary files:

- `proto/plz_confirm/v1/request.proto`
- `proto/plz_confirm/v1/widgets.proto`

Current `WidgetType` values are `confirm/select/form/upload/table/image` (no script type yet). `UIRequest` has typed input/output oneofs and request metadata/status fields.

### plz-confirm server lifecycle

Primary file:

- `internal/server/server.go`

Current request endpoints:

- `POST /api/requests` -> create pending request
- `GET /api/requests/{id}` -> fetch request
- `POST /api/requests/{id}/response` -> submit final output
- `POST /api/requests/{id}/touch` -> disable expiry on first interaction
- `GET /api/requests/{id}/wait` -> long-poll until completion

WebSocket events currently broadcast:

- `new_request`
- `request_completed`

No `request_updated` event currently exists.

### plz-confirm state store

Primary file:

- `internal/store/store.go`

Store model:

- in-memory map of request entries,
- per-request `done chan struct{}` used to unblock waiters on completion,
- helper `setDefaultOutputFor` for timeout completion defaults.

Important limitation:

- the store can complete or touch a request, but has no first-class event stream/state-version updates for ongoing flows.

### plz-confirm CLI pattern

Primary files:

- `cmd/plz-confirm/main.go`
- `internal/cli/*.go`
- `internal/client/client.go`

Each command follows the same shape:

1. parse flags into settings,
2. build typed input payload,
3. call `CreateRequest`,
4. call `WaitRequest`,
5. decode typed output and print row.

### plz-confirm frontend pattern

Primary files:

- `agent-ui-system/client/src/components/WidgetRenderer.tsx`
- `agent-ui-system/client/src/services/websocket.ts`
- `agent-ui-system/client/src/store/store.ts`
- `agent-ui-system/client/src/proto/normalize.ts`

Rendering is a `switch(active.type)` into explicit dialog components. Submission calls `submitResponse`, which posts typed oneof output and then transitions request to completed/history.

### go-go-goja runtime and modules

Primary files:

- `go-go-goja/engine/runtime.go`
- `go-go-goja/engine/factory.go`
- `go-go-goja/modules/common.go`
- `go-go-goja/modules/exports.go`
- `go-go-goja/pkg/runtimeowner/runner.go`

Key patterns we should reuse:

- `modules.NativeModule` adapter shape (`Name`, `Doc`, `Loader`),
- `modules.Register` + `EnableAll` registry workflow,
- exported Go values via `modules.SetExport`,
- owner-thread execution discipline via `runtimeowner.Runner` where async is involved.

### Experiment findings used in this plan

Ticket-local experiments validated:

1. `module.exports` init/view/update invocation can return stable `map[string]any` trees after `Export()`.
2. `goja.Runtime.Interrupt()` reliably aborts runaway scripts with `*goja.InterruptedError`.
3. `protojson` shape matches existing conventions (enum strings + oneof lower-camel fields).

Experiment scripts:

- `scripts/goja-flow/main.go`
- `scripts/goja-interrupt/main.go`
- `scripts/protojson-shape/main.go`

## Proposed Solution

## 1) JS contract: add the describe extension and state-machine lifecycle

### Script exports

Recommended JS module contract:

```js
module.exports = {
  describe: function (ctx) {
    return {
      apiVersion: "plz-confirm.script.v1",
      name: "deploy-approval",
      summary: "Review plan and approve deployment",
      propsSchema: { /* JSON-Schema-ish object */ },
      resultSchema: { /* JSON-Schema-ish object */ },
      capabilities: {
        needsFs: false,
        supportsActions: true,
        maxStepsHint: 5,
      }
    };
  },

  init: function (ctx) {
    return { step: "start", answers: {} };
  },

  view: function (state, ctx) {
    return { kind: "page", title: "...", body: [] };
  },

  update: function (state, event, ctx) {
    // either return next state or done envelope
    return { done: true, result: { approved: true } };
  }
};
```

### Why `describe` is required

`describe` is not decorative. It enables:

- server-side preflight validation before first render,
- clearer runtime errors when required exports are missing,
- compatibility checks as API evolves,
- safer onboarding and observability (name/summary/capabilities).

## 2) plz-confirm protocol: add script input/output + runtime state/view

### Proto additions

Files to update:

- `proto/plz_confirm/v1/request.proto`
- `proto/plz_confirm/v1/widgets.proto`

Additions:

- `WidgetType.script = 7` (or `flow = 7`, pick one and keep naming consistent).
- `ScriptInput` and `ScriptOutput` messages.
- `UIRequest` oneof entries:
  - `script_input`
  - `script_output`
- New optional fields for pending runtime:
  - `script_state` (`google.protobuf.Struct`)
  - `script_view` (`google.protobuf.Struct`)
  - optional `script_describe` (`google.protobuf.Struct`) for surfaced metadata.

`ScriptInput` minimum fields:

- `title`
- `script` (source string)
- `props` (Struct)
- optional `mounts` (read-only fs capabilities)
- optional `limits` (timeouts/size guards)

`ScriptOutput` minimum fields:

- `result` (Struct)
- optional `logs` (captured console lines)

### Why state/view live on request

Storing current state/view in `UIRequest` makes multi-client behavior deterministic and debuggable. WS snapshots and REST fetches remain truthful without hidden server-local runtime context.

## 3) plz-confirm server: add event/update loop for pending script requests

### New endpoint

Add:

- `POST /api/requests/{id}/event`

Expected payload:

```json
{
  "type": "submit",
  "stepId": "review",
  "data": {"approve": true},
  "actionId": null
}
```

Server flow:

1. Load request by ID.
2. Ensure type is `script` and status pending.
3. Re-run/update JS engine with current `script_state` + event.
4. If done: set `script_output`, mark completed (existing completion path).
5. Else: persist updated `script_state` + `script_view`, keep pending.
6. Broadcast WS `request_updated`.

### WS event extension

Add event type:

- `request_updated`

Wire serialization reuses existing `marshalWSEvent` pattern in `internal/server/ws_events.go`.

### Store extension

Add store operations in `internal/store/store.go`:

- `UpdatePending(id, patchFn)` or explicit methods:
  - `SetScriptStateAndView(...)`
  - `CompleteScript(...)`

Keep lock discipline and status guards identical to existing completion methods.

## 4) Script runtime in plz-confirm: local package using go-go-goja patterns

### Recommendation

Implement runtime package **inside `plz-confirm`** first (for velocity and tight API evolution), while copying stable adapter patterns from `go-go-goja`.

Suggested package:

- `plz-confirm/internal/scriptengine`

Suggested files:

- `engine.go` (compile + invoke describe/init/view/update)
- `limits.go` (timeouts/output-size checks)
- `ui_dsl.go` (builders or direct pass-through)
- `sandbox_fs.go` (capability-based file reads)
- `errors.go` (typed user-facing errors)

### Why not import go-go-goja engine directly (initially)

`go-go-goja/engine/runtime.go` blank-imports multiple modules (`database`, `exec`, `fs`, `glazehelp`) and drags a larger dependency surface than needed for plz-confirm runtime execution. The feature requires a narrower and safer runtime profile.

### What to reuse from go-go-goja

Reuse design patterns, not necessarily direct imports:

- module adapter conventions (`modules/common.go`, `modules/exports.go`),
- export naming and conversion discipline,
- owner-thread and async safety concepts (`pkg/runtimeowner`).

## 5) Frontend: add ScriptDialog + generic node renderer

### Renderer integration

File:

- `agent-ui-system/client/src/components/WidgetRenderer.tsx`

Add new branch:

- `case WidgetType.script` -> `<ScriptDialog ... />`

### WebSocket integration

File:

- `agent-ui-system/client/src/services/websocket.ts`

Add new event handling:

- `request_updated` -> `patchRequest(...)` for active/pending entries.

### Script dialog responsibilities

`ScriptDialog` should:

- render `scriptView` node tree,
- collect local form/select/table selections for current step,
- submit to `/api/requests/{id}/event` (not `/response`) unless done-path semantics require `/response`.

### UI node strategy

Do not build a whole new UI library first.

Phase 1 renderer should map script nodes onto existing dialog subcomponents/patterns:

- display: `markdown`, `callout`, `code`, `diff`
- inputs: `confirm`, `select`, `form`, `table`, `image`, `upload`
- controls: `submit`, `action`

This avoids duplicating validation/UX behavior already implemented in existing widgets.

## 6) CLI: add `plz-confirm script`

### Command file

- `internal/cli/script.go`

### Main registration

- `cmd/plz-confirm/main.go`

### Core flags

- `--title`
- `--script @file.js` (and/or `--script -`)
- `--props @props.json`
- `--mount name:path:ro` (repeatable)
- `--timeout`, `--wait-timeout`, `--session-id`, `--base-url`

CLI output should report final `script_output.result` in existing Glazed output formats.

## 7) Sandbox defaults and security posture

Default capabilities for v1:

- allowed: `describe/init/view/update`, `console` capture, explicit read-only mounts.
- denied: arbitrary process execution, unrestricted filesystem, network, env mutation.

Implement fs safety with mount names + relative paths only. No absolute path reads from script code.

Use bounded limits from `ScriptInput.limits` with safe server defaults:

- max script size,
- max execution time per call,
- max output size,
- max file read bytes.

## Design Decisions

### Decision 1: require `describe()` in script contract

Reason:

- creates explicit compatibility and validation surface,
- aligns with “describe extension” requirement,
- prevents runtime guesswork about script intent.

Tradeoff:

- slightly more authoring overhead for script writers.

### Decision 2: keep `init/view/update` event-state machine

Reason:

- matches Goja constraints and existing imported proposal,
- avoids async/generator requirements,
- deterministic and testable.

Tradeoff:

- more explicit state plumbing in scripts.

### Decision 3: persist `script_state/script_view` in `UIRequest`

Reason:

- simplifies multi-client consistency and debugging,
- leverages existing request fetch + WS distribution model.

Tradeoff:

- larger request payloads for complex views.

### Decision 4: add `/event` + `request_updated` instead of overloading `/response`

Reason:

- clearer semantics (intermediate vs final),
- aligns with prior internal analysis patterns,
- minimizes accidental completion bugs.

Tradeoff:

- one additional endpoint and frontend code path.

### Decision 5: local runtime package in plz-confirm first; optional extraction later

Reason:

- fastest delivery with tight feature iteration,
- avoids pulling unrelated runtime modules by default,
- still informed by go-go-goja architecture patterns.

Tradeoff:

- some temporary duplication until extraction.

## Alternatives Considered

### Alternative A: static wizard spec only (no update function)

Pros:

- simpler server logic,
- no intermediate event/update lifecycle.

Cons:

- cannot support dynamic branching from user responses,
- weak fit for requested complex flows.

Decision: rejected.

### Alternative B: overload `/response` with `kind: hint|final`

Pros:

- fewer new endpoints.

Cons:

- muddier semantics,
- higher risk of accidental completion,
- harder to reason about in clients.

Decision: rejected.

### Alternative C: import go-go-goja engine wholesale in plz-confirm

Pros:

- reuse existing runtime bootstrap quickly.

Cons:

- broader dependency and module surface than needed,
- security posture harder to keep minimal by default.

Decision: rejected for v1; maybe revisit after runtime extraction work.

### Alternative D: keep all script state server-local only

Pros:

- smaller wire payloads.

Cons:

- harder WS parity/debugging,
- weaker recoverability from reconnects.

Decision: rejected for initial implementation.

## Implementation Plan

This section is a practical execution sequence for an intern.

### Phase 0: Contract alignment and scaffolding

1. Confirm naming choices:
   - widget type string (`script` vs `flow`),
   - required exported function names (`describe/init/view/update`).
2. Add/update ticket tasks in `tasks.md` with checkboxes per phase.
3. Add dev notes in ticket `scripts/README.md` for reproducibility.

### Phase 1: Proto + codegen + core request plumbing

Files:

- `proto/plz_confirm/v1/request.proto`
- `proto/plz_confirm/v1/widgets.proto`
- generated outputs in:
  - `proto/generated/go/plz_confirm/v1/*.pb.go`
  - `agent-ui-system/client/src/proto/generated/**`

Steps:

1. Add `WidgetType.script`.
2. Add `ScriptInput`, `ScriptOutput`, helper messages.
3. Extend `UIRequest` oneofs for script input/output.
4. Add optional runtime fields (`script_state`, `script_view`, `script_describe`).
5. Run codegen:
   - `make proto`
   - `pnpm -C agent-ui-system run proto`

Validation:

- compile generated Go and TS types,
- verify enum/string keys in generated TS include `script`.

### Phase 2: Server/store lifecycle for script events

Files:

- `internal/server/server.go`
- `internal/server/ws_events.go`
- `internal/store/store.go`

Steps:

1. Extend create/response widget type routing to include script where needed.
2. Add `POST /api/requests/{id}/event` handler.
3. Add store update operation for pending script state/view writes.
4. Add WS broadcast type `request_updated`.
5. Ensure `/wait` continues to return on completion; event updates should not complete.

Validation:

- unit tests for handler behavior (pending update vs completion),
- WS payload tests for `request_updated` shape.

### Phase 3: Script runtime package (plz-confirm)

Files (new):

- `internal/scriptengine/engine.go`
- `internal/scriptengine/sandbox_fs.go`
- `internal/scriptengine/limits.go`
- `internal/scriptengine/errors.go`

Steps:

1. Build compile+invoke path:
   - load script source,
   - resolve `module.exports`,
   - assert `describe/init/view/update` callability.
2. Implement `describe` preflight validation.
3. Implement bounded execution wrapper using `goja.Interrupt()` timeout pattern.
4. Implement minimal capability-safe fs wrapper.
5. Return typed Go maps/structs that convert cleanly to protobuf Struct.

Validation:

- unit tests for:
  - missing export errors,
  - timeout interruption,
  - invalid return shape handling,
  - done vs continue update envelopes.

### Phase 4: Frontend script renderer and event submit flow

Files:

- `agent-ui-system/client/src/components/WidgetRenderer.tsx`
- `agent-ui-system/client/src/services/websocket.ts`
- `agent-ui-system/client/src/store/store.ts` (likely minor)
- new files under `agent-ui-system/client/src/components/widgets/`:
  - `ScriptDialog.tsx`
  - optional node renderer helpers.

Steps:

1. Add `WidgetType.script` branch in renderer.
2. Add `/event` client call and `request_updated` WS handling.
3. Implement generic node-to-component mapping using existing UI primitives.
4. Keep user interaction semantics consistent with current widgets.

Validation:

- manual e2e with a sample script,
- UI behavior for multi-step flows,
- no regression to existing widgets.

### Phase 5: CLI command and operator workflow

Files:

- `internal/cli/script.go` (new)
- `cmd/plz-confirm/main.go`
- optional docs:
  - `pkg/doc/how-to-use.md`
  - `README.md`

Steps:

1. Implement `plz-confirm script` command.
2. Parse script/props/mounts input robustly.
3. Create request and wait for final completion.
4. Print `script_output.result` with Glazed row output.

Validation:

- command runs against local server,
- output formats include JSON/YAML/table as expected.

### Phase 6: Tests, hardening, and docs

Test coverage goals:

- server integration tests for create/event/response lifecycle,
- runtime unit tests for interruption and shape validation,
- frontend tests for renderer branches and request_updated handling,
- at least one end-to-end smoke script in ticket `scripts/`.

Docs:

- add script widget section to `adding-widgets` style docs,
- include one complete script example with describe/init/view/update,
- document safe defaults and limits.

## File-by-File Change Map

### plz-confirm (must change)

- `proto/plz_confirm/v1/request.proto`
- `proto/plz_confirm/v1/widgets.proto`
- `internal/server/server.go`
- `internal/server/ws_events.go`
- `internal/store/store.go`
- `internal/client/client.go` (if adding event helper methods)
- `internal/cli/script.go` (new)
- `cmd/plz-confirm/main.go`
- `agent-ui-system/client/src/components/WidgetRenderer.tsx`
- `agent-ui-system/client/src/services/websocket.ts`
- `agent-ui-system/client/src/components/widgets/ScriptDialog.tsx` (new)
- `agent-ui-system/client/src/proto/generated/...` (generated)

### go-go-goja (recommended follow-up changes)

- Add reusable package for script contract validation/runtime orchestration (new package; naming TBD).
- Optionally add lightweight engine bootstrap path that does not force full default module set.
- Add runtime integration tests for describe/init/view/update invocation.

## Testing Strategy

## Unit tests (minimum)

### plz-confirm runtime

- `describe` missing/invalid returns clear error.
- `init/view/update` missing yields clear error.
- update returning `{done:true,result:...}` completes request.
- update returning next state updates pending request.
- infinite loop interrupts on deadline.

### plz-confirm server/store

- `/event` on non-script request returns validation error.
- `/event` on completed request returns conflict.
- `/event` emits `request_updated` when not done.
- `/event` emits `request_completed` when done.

### frontend

- `WidgetRenderer` handles script branch.
- websocket handler processes `request_updated` and patches active request.
- `ScriptDialog` emits event payload, not final response, for intermediate submits.

## Integration tests

- submit script request with simple 2-step flow,
- verify first event updates view,
- verify final event completes and unblocks wait.

## Operational Risks and Mitigations

### Risk: script hangs or heavy CPU usage

Mitigation:

- strict per-call timeout,
- interrupt-based cancellation,
- bounded input/output sizes.

### Risk: unsafe file access

Mitigation:

- mount-based capability model,
- read-only default,
- reject absolute paths and traversal patterns.

### Risk: protocol complexity/regressions

Mitigation:

- keep existing `/response` semantics for non-script widgets untouched,
- add dedicated `/event` path and explicit WS event type,
- snapshot tests for payload shape.

### Risk: dependency bloat from runtime reuse

Mitigation:

- implement plz-confirm local runtime first,
- reuse patterns from go-go-goja rather than pulling whole engine path initially.

## Rollout Plan

1. Guarded path:
   - enable script widget only when rollout gate is on (env/config toggle) OR session is in explicit allowlist.
2. Internal rollout:
   - enable for internal development sessions only.
   - run smoke flow (create -> updated -> completed) on each deploy candidate.
3. Limited external rollout:
   - enable per-team/session allowlist.
   - keep non-script widgets untouched and monitor status-code/error ratios.
4. General availability:
   - enable by default after timeout/fault rates are within target and security checklist passes.

### Post-rollout observability checks

- API status mix for script endpoints:
  - monitor `400`, `408`, `422`, `504` rates independently.
- Lifecycle progression:
  - monitor counts and ratio of `new_request`, `request_updated`, `request_completed`.
- Timeout and runtime fault watchpoints:
  - alert if timeout rate exceeds baseline threshold.
  - alert if runtime fault (`422`) rate spikes after releases.
- Per-script metadata watchpoints:
  - include `scriptDescribe.name` + `scriptDescribe.version` in logs/metrics labels where feasible.
- Queue/latency watchpoints:
  - track time from request creation to completion for script flows.

## Open Questions

1. Naming finalization:
   - `script` vs `flow` in `WidgetType`.
2. `describe` strictness:
   - required fields in v1 (minimal vs rich metadata).
3. Storage strategy:
   - do we persist script logs on request object or keep best-effort ephemeral logs only?
4. FS scope defaults:
   - should no mounts be allowed by default unless explicitly set in input?
5. Cross-repo extraction timing:
   - when to move plz-confirm runtime package into reusable go-go-goja package.

## Intern Start Checklist

Use this exact sequence:

1. Read this file top-to-bottom once.
2. Read `proto/plz_confirm/v1/request.proto` and `proto/plz_confirm/v1/widgets.proto`.
3. Read `internal/server/server.go` request handlers.
4. Read `internal/store/store.go` create/complete/wait behavior.
5. Read frontend `WidgetRenderer.tsx` and `services/websocket.ts`.
6. Run ticket experiments:
   - `go run ./plz-confirm/ttmp/2026/02/22/PC-01-ADD-JS-API--add-js-api-describe-extension/scripts/goja-flow`
   - `go run ./plz-confirm/ttmp/2026/02/22/PC-01-ADD-JS-API--add-js-api-describe-extension/scripts/goja-interrupt`
   - `go run ./plz-confirm/ttmp/2026/02/22/PC-01-ADD-JS-API--add-js-api-describe-extension/scripts/protojson-shape`
7. Implement Phase 1 only; commit.
8. Implement Phase 2; commit.
9. Continue phase-by-phase with tests before each commit.

## References

- Imported source proposal:
  - `plz-confirm/ttmp/2026/02/22/PC-01-ADD-JS-API--add-js-api-describe-extension/sources/local/plz-confirm-js.md`
- Existing widget extension guide:
  - `plz-confirm/pkg/doc/adding-widgets.md`
- Prior protocol/event analysis:
  - `plz-confirm/ttmp/2025/12/25/005-HINT-PROMPT--add-non-closing-hint-prompt-events-request-updates/analysis/01-hint-prompt-events-and-request-updates-analysis.md`
- go-go-goja runtime/module references:
  - `go-go-goja/engine/runtime.go`
  - `go-go-goja/engine/factory.go`
  - `go-go-goja/modules/common.go`
  - `go-go-goja/pkg/runtimeowner/runner.go`
