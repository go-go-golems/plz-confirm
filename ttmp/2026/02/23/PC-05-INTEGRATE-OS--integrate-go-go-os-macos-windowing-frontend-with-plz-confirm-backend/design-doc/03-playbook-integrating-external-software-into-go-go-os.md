---
Title: 'Playbook: Integrating External Software into go-go-os'
Ticket: PC-05-INTEGRATE-OS
Status: active
Topics:
    - architecture
    - frontend
    - backend
    - go
    - javascript
    - ux
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main.go
      Note: Host mux composition reference for backend mounting patterns
    - Path: ../../../../../../../go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main_integration_test.go
      Note: Route coexistence and prefixed WS integration test pattern
    - Path: ../../../../../../../go-go-os/packages/confirm-runtime/src/runtime/createConfirmRuntime.ts
      Note: Runtime bootstrap pattern for WS event -> window lifecycle bridging
    - Path: ../../../../../../../go-go-os/packages/confirm-runtime/src/proto/confirmProtoAdapter.ts
      Note: Protocol translation boundary (decode/encode normalization)
    - Path: ../../../../../../../go-go-os/packages/confirm-runtime/src/state/confirmRuntimeSlice.ts
      Note: Queue state model and stale replay reconciliation pattern
    - Path: ../../../../../../../go-go-os/packages/confirm-runtime/src/components/ConfirmRequestWindowHost.tsx
      Note: Package-specific host that composes core widgets
    - Path: ../../../../../../../go-go-os/packages/engine/src/components/widgets/index.ts
      Note: Core reusable widget export surface for integration-first UI design
    - Path: ../../../../../../../go-go-os/packages/engine/src/theme/desktop/tokens.css
      Note: Token layer extension example for new integration UI vocabulary
    - Path: ../../../../../../../plz-confirm/pkg/backend/backend.go
      Note: Public embeddable backend facade pattern for cross-repo integration
    - Path: ../../../../../../../plz-confirm/internal/server/server.go
      Note: Request lifecycle contracts and oneof validation shape
    - Path: ../../../../../../../plz-confirm/internal/server/script.go
      Note: Script workflow update/complete lifecycle and concurrency guards
    - Path: ../../../../../../../plz-confirm/cmd/plz-confirm/ws.go
      Note: Prefix-aware websocket URL derivation example for embedded mode
    - Path: ../../../../../../../plz-confirm/ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/reference/01-diary.md
      Note: Source evidence trail used to derive this reusable playbook
    - Path: ../../../../../../../plz-confirm/ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/design-doc/02-postmortem-plz-confirm-integration-into-go-go-os.md
      Note: Worked example postmortem feeding lessons learned into this playbook
ExternalSources: []
Summary: Reusable end-to-end playbook for integrating external software into go-go-os, including architecture intake, package boundaries, protocol mapping, routing/runtime/UI/state patterns, testing, rollout, handoff, and postmortem standards.
LastUpdated: 2026-02-23T19:39:36-05:00
WhatFor: Provide a repeatable, intern-friendly, deeply technical process template for future go-go-os integrations so teams can ship faster with fewer regressions.
WhenToUse: Use as the primary guide before starting any external software integration into go-go-os or when auditing an in-flight integration for missing workstreams.
---

# Playbook: Integrating External Software into go-go-os

## Executive Summary

This playbook standardizes how to integrate external software into the go-go-os ecosystem without sacrificing package hygiene, protocol correctness, UI consistency, or operational reliability. It is intentionally detailed so a new engineer can run it as a procedural guide instead of relying on tribal memory.

The core principle is simple: treat integrations as system-boundary work, not as ad-hoc feature work. Every successful integration in go-go-os has the same durable shape:

1. establish legal package boundaries,
2. isolate protocol translation,
3. compose UI from reusable core primitives,
4. verify behavior under replay/race and prefix-mount conditions,
5. document deeply enough that handoff is reversible.

This document includes a complete table of contents, technical checklists, pseudocode, diagrams, validation commands, and an applied reference model based on the plz-confirm integration.

## How to Use This Playbook

Use this playbook in three modes:

1. Planning mode: fill the templates in Sections `1-9` before writing code.
2. Execution mode: implement in tranches using Sections `10-16` and commit checkpoints.
3. Postmortem mode: complete Sections `17-20` to preserve lessons and improve the next integration.

If you are an intern or first-time contributor, follow the numbered sections in order. Do not skip phase gates.

## Table of Contents

1. Integration Intake and Scope Definition
2. Baseline Architecture Reconnaissance
3. Boundary Legality and Package Extraction Strategy
4. Contract and Protocol Mapping
5. Router, Prefixing, and Transport Topology
6. Runtime Bridge and State Lifecycle Model
7. UI Composition Strategy (Core-first + Package-first)
8. Error Semantics and Reconciliation Rules
9. Security, Safety, and Operational Constraints
10. Implementation Tranches and Commit Rhythm
11. Testing Matrix and Validation Scripts
12. Observability and Debugging Runbook
13. Handoff Artifacts (Engineering + Design)
14. Rollout and Rollback Strategy
15. Definition of Done and Sign-off Checklist
16. Reference Project Skeleton and Pseudocode
17. Worked Example: plz-confirm -> go-go-os
18. Reusable Templates (ADR, Task List, Diary Entry)
19. Anti-patterns and Failure Signatures
20. Continuous Improvement Loop

## 1) Integration Intake and Scope Definition

Before any coding, produce a one-page integration brief.

Required fields:

- external system name and owner
- integration objective in one sentence
- what remains authoritative in external system
- what moves into go-go-os
- success criteria (user-facing and operator-facing)
- explicit non-goals

Template:

```markdown
# Integration Brief

External System: <name>
Owner: <team/person>
Ticket: <ticket-id>

Objective:
<single sentence>

Authoritative Source of Truth:
- Backend authority: <where>
- Frontend authority: <where>

In Scope:
1. ...
2. ...

Out of Scope:
1. ...
2. ...

Success Criteria:
1. ...
2. ...
```

Gate A (must pass before architecture work):

1. Objective and non-goals are approved.
2. Integration ticket and docs exist.
3. Owner/maintainer for each side is identified.

## 2) Baseline Architecture Reconnaissance

Do source reconnaissance before proposing architecture. At minimum, identify:

- host backend route ownership
- host frontend runtime/windowing entry points
- external backend API and WS contracts
- external runtime/model ownership
- persistence model and status transitions

Output artifact: a `Current State` section with concrete file/symbol references.

Recommended command pattern:

```bash
rg -n "Handle\(|/ws|/api|create.*Runtime|Reducer|slice|oneof|protojson" <repo-root>
```

Recon checklist:

1. Where is host mux composed?
2. Which routes are already reserved?
3. Where is window lifecycle controlled?
4. Where are WS events consumed and reduced?
5. Which schema format is authoritative (proto/json/openapi)?
6. Are there internal package boundaries that block direct imports?

Gate B:

1. Baseline architecture map exists.
2. At least 10 critical files are related in ticket docs.
3. At least one diagram captures as-is flow.

## 3) Boundary Legality and Package Extraction Strategy

This is the highest-risk decision point.

If host cannot legally import external internals, extract a public embeddable package first. Do not bypass by copying code into host.

Preferred extraction shape:

```text
external-repo/
  pkg/<integration-surface>/
    facade.go
    mount.go
    tests/
  internal/
    ...existing...
```

Public facade should expose minimal stable operations:

- `NewServer()`
- `Handler()`
- `Mount(mux, prefix)`
- optional `ListenAndServe()`

Extraction checklist:

1. No host code imports `internal/*` across module boundary.
2. Public package has tests for root mount and prefixed mount.
3. Existing external CLI/app can adopt same public package where possible.

Gate C:

1. Embeddable package exists.
2. Prefix mount behavior is tested.
3. Import path is workspace-resolvable and release plan is documented.

## 4) Contract and Protocol Mapping

Never decode protocol payloads directly in UI components.

Create one translation boundary file/module that handles:

- inbound decode normalization
- outbound encode normalization
- enum/widget type canonicalization
- timestamp/default fallback policy
- oneof/polymorphic mapping

Pattern:

```text
packages/<integration-runtime>/src/proto/<adapter>.ts
```

Pseudocode:

```text
function mapInbound(raw): InternalRequest {
  type = normalizeType(raw.type)
  input = extractOneofInput(type, raw)
  return normalizeRequest(raw.id, type, input, status, metadata)
}

function mapOutbound(request, payload): RawProtocol {
  switch request.type:
    case confirm: return { confirmOutput: ... }
    case select:  return { selectOutput: ... }
    ...
}
```

Contract checklist:

1. Mapping unit tests exist per request type.
2. Unknown types fail loudly with diagnostics.
3. Outbound mapping requires full request context.
4. Mapping module owns timestamp and default policy.

Gate D:

1. Adapter tests pass independently.
2. No protocol field parsing remains in UI host component except display-only formatting.

## 5) Router, Prefixing, and Transport Topology

Always mount external system under explicit namespace in host app unless there is a strong reason not to.

Recommended pattern:

- host existing routes remain unchanged
- mount external at `/<integration-prefix>`
- frontend base URL resolves to prefix path
- WS URL derivation preserves prefix path

Example topology:

```text
/chat
/ws
/api/*
/<integration-prefix>/api/*
/<integration-prefix>/ws
/
```

Failure mode to avoid: dropping prefix when deriving WS URL from base URL.

Pseudocode:

```text
basePath = trimTrailingSlash(parsedBaseURL.path)
if basePath == "": wsPath = "/ws"
else if endsWith(basePath, "/ws"): wsPath = basePath
else wsPath = basePath + "/ws"
```

Router checklist:

1. Route coexistence tests: host features + integrated features in same server.
2. WS connection test against prefixed endpoint.
3. Redirect behavior for no-trailing-slash prefix path is defined.
4. Dev proxy includes both prefix API and prefix WS entries.

Gate E:

1. Coexistence tests pass.
2. Prefix WS smoke test passes.

## 6) Runtime Bridge and State Lifecycle Model

Introduce a dedicated runtime package that bridges protocol events into host app lifecycle.

Recommended package responsibilities:

- connect/disconnect WS
- dispatch realtime events to store
- open/close windows or sessions via host adapters
- expose typed API client
- avoid host-app imports

Runtime package should accept host adapters only:

```ts
interface HostAdapters {
  resolveBaseUrl(): string;
  resolveSessionId(): string;
  openRequestWindow(payload): void;
  closeRequestWindow?(id: string): void;
  onError?(err: Error): void;
}
```

State model checklist:

1. connected status
2. active queue keyed by ID
3. stable ordering list
4. completion map/history
5. last error

Reconciliation rule (required):

- if inbound request status is non-pending, place into completion lane and evict from active queue immediately.

Gate F:

1. Runtime package can run in isolation with mocks.
2. Host app only wires adapters/store, not protocol details.

## 7) UI Composition Strategy (Core-first + Package-first)

Two-tier UI strategy is mandatory for maintainability:

1. Core tier (`packages/engine`): generic reusable widgets and tokens.
2. Integration tier (`packages/<integration-runtime>`): protocol-specific composition and behavior.

Do not place protocol-specific rendering logic inside core widgets.

Widget classification template:

```markdown
| Widget/Control | Layer | Rationale |
|---|---|---|
| SelectableList | Core | Generic list selection pattern |
| RequestActionBar | Core | Generic action/cancel/comment footer |
| ConfirmRequestWindowHost | Integration | Protocol-specific payload mapping |
```

Visual consistency pattern:

- extend token set for new interaction semantics
- use `data-part` vocabulary so design can target stable selectors
- add stories per primitive and per composed scenario

Gate G:

1. New generic widgets exported from engine.
2. Integration package imports widgets but does not fork style primitives.
3. Storybook coverage exists for both primitive and composite flows.

## 8) Error Semantics and Reconciliation Rules

Treat status codes and event timing as first-class architecture.

Minimum error policy table:

```markdown
| Case | Expected Cause | Frontend Behavior | Backend Behavior |
|---|---|---|---|
| 400 invalid payload | contract mismatch | show actionable error + log adapter dump | return precise validation message |
| 404 missing request | stale local id | close window and remove queue item | return not found |
| 409 already completed | stale replay/race | refetch canonical request and reconcile state | return conflict |
| WS disconnect | network/server restart | mark disconnected + retry policy | n/a |
```

Mandatory reconciliation pattern for 409:

```text
submit(request, payload):
  try API.submit(...)
  catch e:
    if e.status == 409:
      latest = API.getRequest(request.id)
      if latest.status == completed:
        markCompleted(latest)
        closeWindow(request.id)
        return
    rethrow
```

Timestamp policy:

1. frontend may set timestamp defaults when missing.
2. backend must enforce timestamp defaults for canonical completeness.

Gate H:

1. 409 path is explicitly tested.
2. timestamp fallback is covered by tests on at least one side.

## 9) Security, Safety, and Operational Constraints

Integration is not complete without safety review.

Security checklist:

1. route exposure reviewed (public/private)
2. session scoping explicit
3. input size limits on submission endpoints
4. script/runtime sandbox boundaries explicit
5. no leakage of internal debug data in user-facing payloads

Operational checklist:

1. sensible request timeouts
2. deterministic idempotency/retry expectations
3. keepalive/touch behavior defined if long-running interactions exist
4. concurrency control around stateful updates (locks/versioning)

If external runtime executes code (scripts/plugins), document:

- runtime engine
- timeout model
- deterministic random/seed semantics
- forbidden capabilities

Gate I:

1. basic threat model is documented.
2. runtime and endpoint guardrails are test-covered.

## 10) Implementation Tranches and Commit Rhythm

Use tranche execution with clear boundaries. Recommended sequence:

1. Tranche A: core widgets and tests
2. Tranche B: runtime package scaffold
3. Tranche C: host app integration
4. Tranche D: backend extraction + mount
5. Tranche E: protocol fixups and parity
6. Tranche F: polish and visual consistency

Commit rhythm guideline:

- one commit per tranche objective
- include tests with each commit where feasible
- update ticket tasks/changelog/diary after each checkpoint

Example commit map (from plz-confirm integration):

- core widgets
- runtime scaffold
- host wiring
- backend mount
- adapter fix
- stale queue/timestamp fix
- composite story coverage
- visual consistency pass

Gate J:

1. each tranche can be reviewed independently.
2. each tranche has explicit validation commands recorded in diary.

## 11) Testing Matrix and Validation Scripts

Required testing layers:

1. Unit: protocol adapter and widget parsing helpers
2. Integration: route coexistence and prefixed WS behavior
3. Functional: request lifecycle through UI
4. Regression scripts: CLI/API reproducible scripts under ticket `scripts/`
5. Storybook: component and composite scenarios

Test matrix template:

```markdown
| Layer | Artifact | Command | Status |
|---|---|---|---|
| Unit | proto adapter | npm exec vitest run ... | pass/fail |
| Integration | host server routes | go test ./cmd/... | pass/fail |
| E2E script | CLI roundtrip | ./scripts/e2e_...sh | pass/fail |
| Storybook taxonomy | stories index | npm run storybook:check | pass/fail |
```

Script management rule:

- every ad-hoc debugging script must be moved into `ttmp/.../scripts/` before ticket closure.

Gate K:

1. scripts are archived in ticket.
2. at least one full roundtrip script exists.
3. storybook and unit tests both pass for touched integration surfaces.

## 12) Observability and Debugging Runbook

For every integration, define quick triage commands.

Minimum runbook commands:

```bash
# Inspect server side route availability
curl -i http://127.0.0.1:<port>/<prefix>/api/health-or-list

# Create request payload
curl -sS -X POST http://127.0.0.1:<port>/<prefix>/api/requests -d '...'

# Subscribe to websocket stream
go run ./cmd/<cli> ws --base-url "http://127.0.0.1:<port>/<prefix>" --session-id global --pretty

# Fetch canonical request state
curl -sS http://127.0.0.1:<port>/<prefix>/api/requests/<id>
```

Failure signature catalog template:

```markdown
| Symptom | Likely Root Cause | First File to Inspect |
|---|---|---|
| Unsupported type undefined | inbound adapter mismatch | proto adapter file |
| 409 already completed | stale queue replay | runtime slice + submit handler |
| WS connects wrong path | prefix dropped in URL builder | cli ws builder / runtime ws URL |
| Empty timestamps | client omitted + no server fallback | adapter + backend submit handler |
```

Gate L:

1. triage runbook exists in ticket docs.
2. failure signatures include at least top 5 likely issues.

## 13) Handoff Artifacts (Engineering + Design)

A strong integration handoff requires both code and documentation artifacts.

Engineering handoff set:

1. integration blueprint
2. execution diary
3. changelog with tranche checkpoints
4. reproducible scripts
5. postmortem
6. reusable playbook (this doc)

Design handoff set:

1. widget inventory list
2. story inventory (primitive + composite)
3. token and `data-part` vocabulary
4. unresolved visual consistency scenarios

Handoff checklist:

1. every new widget appears in storybook.
2. every integration-specific scenario appears in composite stories.
3. design references exact story IDs and screenshots.

Gate M:

1. cross-functional handoff ticket exists (if needed).
2. design and engineering artifacts are linked from primary ticket index.

## 14) Rollout and Rollback Strategy

Rollout should be staged even for internal tooling integrations.

Recommended rollout phases:

1. Local-only integration behind development toggles
2. Internal pilot session IDs/users
3. Default-on for target app with fallback path retained briefly
4. Remove legacy path once stability criteria are met

Rollback strategy requirements:

- one command/config toggle to disable new runtime path
- keep backend API compatibility intact
- preserve old client behavior where possible

Rollback checklist:

1. rollback trigger conditions documented.
2. owner for rollback decision identified.
3. monitoring signals linked to trigger conditions.

Gate N:

1. rollback drill executed once before full rollout.

## 15) Definition of Done and Sign-off Checklist

Integration is done only if all categories are complete.

Technical DoD:

1. backend mount works with prefix and root coexistence
2. frontend runtime handles create/update/complete lifecycle
3. protocol adapter supports all in-scope request types
4. stale replay and 409 reconciliation behavior validated
5. timestamps and required output fields are consistently populated

Quality DoD:

1. unit + integration tests pass for touched surfaces
2. storybook stories added for new widgets and composites
3. regression scripts archived and runnable

Documentation DoD:

1. diary updated with commands and failures
2. changelog updated with tranche outcomes
3. postmortem and playbook present
4. ticket index links are current

Operational DoD:

1. runbook exists
2. rollback path documented
3. known risks and follow-ups recorded

Gate O (final):

- all DoD checkboxes complete and reviewed by at least one secondary engineer.

## 16) Reference Project Skeleton and Pseudocode

Use this skeleton when starting a new integration:

```text
go-go-os/
  packages/
    engine/
      src/components/widgets/<new-core-widgets>.tsx
    <integration-runtime>/
      src/api/<api-client>.ts
      src/ws/<ws-manager>.ts
      src/proto/<protocol-adapter>.ts
      src/state/<slice>.ts
      src/components/<request-host>.tsx
  apps/<host-app>/
    src/App.tsx
    src/app/store.ts
  <host-backend>/
    cmd/<server>/main.go
```

Backend pseudocode:

```text
func wireIntegration(appMux):
  integrationServer := integrationpkg.NewServer()
  integrationServer.Mount(appMux, "/integration")
```

Runtime pseudocode:

```text
runtime := createIntegrationRuntime({
  host: adapters,
  dispatch: store.dispatch,
})
runtime.connect()
```

Submit flow pseudocode:

```text
onSubmit(request, uiPayload):
  protoPayload = adapter.encode(request.type, uiPayload)
  api.submit(request.id, protoPayload)
```

## 17) Worked Example: plz-confirm -> go-go-os

This section demonstrates the playbook steps with the completed plz-confirm integration.

### Intake summary

- Goal: use go-go-os windows as frontend for plz-confirm flows.
- Constraint: keep script runtime on plz-confirm backend.
- Key blocker: `internal/*` import boundary.

### Boundary extraction

- Added `plz-confirm/pkg/backend` to expose embeddable server facade.
- Host imported and mounted under `/confirm`.

### Runtime bridge

- Added `@hypercard/confirm-runtime` package.
- Host app wired runtime adapters and reducer.

### Core-first UI

- Added reusable widgets to `packages/engine`.
- Integration-specific host composed those widgets.

### Major regression fixes that informed this playbook

1. protojson oneof mismatch (`Unsupported widget type: undefined`)
2. CLI flag decode regression
3. WS prefix path drop
4. stale queue -> 409 reconciliation
5. missing timestamps

### Validation shape

- route coexistence tests in host backend
- adapter unit tests
- storybook primitive + composite scenarios
- ticket scripts for ws/cli/e2e reproductions

## 18) Reusable Templates

### A) ADR template

```markdown
## ADR-<n>: <decision title>

Context:
<what forced this decision>

Decision:
<chosen approach>

Alternatives considered:
1. <option>
2. <option>

Rationale:
<why this option>

Tradeoffs:
<costs/risks>

Rollback plan:
<how to revert if needed>
```

### B) Task board template

```markdown
## Tranche A: Core primitives
- [ ] A1 ...
- [ ] A2 ...

## Tranche B: Runtime package
- [ ] B1 ...
- [ ] B2 ...

## Tranche C: Host integration
- [ ] C1 ...
- [ ] C2 ...
```

### C) Diary step template (short form)

```markdown
## Step N: <title>

Prompt Context
- User prompt: "..."
- Interpretation: ...
- Intent: ...

What I did
- ...

What worked
- ...

What failed
- ...

Validation
- command: result

Future follow-up
- ...
```

## 19) Anti-patterns and Failure Signatures

Avoid these anti-patterns:

1. App-first integration with no reusable package boundary.
2. Protocol parsing inside UI components.
3. Route mounting without explicit prefix strategy.
4. Assuming event order guarantees queue correctness.
5. Closing ticket without script archives and composite stories.

Common failure signatures:

- `Unsupported widget type: undefined`
- `invalid protojson ... oneof`
- `request already completed` on stale submit
- WS connected to `/ws` instead of `/<prefix>/ws`
- blank operator-facing timestamps

For each signature, attach one script and one test that catches it.

## 20) Continuous Improvement Loop

This playbook should evolve after every integration.

Process:

1. During implementation, keep diary high fidelity.
2. After shipment, write postmortem with root causes and prevention.
3. Fold new lessons into this playbook.
4. Mark version in changelog.

Playbook change checklist:

1. New failure signature added?
2. New test pattern added?
3. New template or command shortcut added?
4. Prior sections updated to prevent repeated mistakes?

## Suggested Phase Gates Summary

```markdown
A: Intake approved
B: Baseline mapped
C: Package legality solved
D: Protocol adapter tested
E: Router/prefix tests pass
F: Runtime bridge integrated
G: UI stories complete
H: Reconciliation semantics validated
I: Safety review complete
J: Tranche commits complete
K: Test matrix green
L: Debug runbook written
M: Handoff pack assembled
N: Rollback plan tested
O: DoD and sign-off complete
```

## Quick-Start Checklist (One Page)

Use this as an at-a-glance launch list when starting a new integration ticket:

1. Create ticket + blueprint + diary.
2. Recon host routes/runtime and external contracts.
3. Resolve package legality (`pkg/` extraction if needed).
4. Define prefix route namespace and WS derivation.
5. Build protocol adapter with oneof/enum tests.
6. Build runtime package (API/WS/slice/host adapters).
7. Add core widgets in engine; compose in integration package.
8. Wire host app with thin adapter only.
9. Add route coexistence, WS prefix, and adapter tests.
10. Add regression scripts to ticket `scripts/`.
11. Add primitive + composite story coverage.
12. Validate stale replay/409/timestamp behavior.
13. Update diary/changelog/index continuously.
14. Ship with rollback plan and runbook.
15. Publish postmortem and update this playbook.

## Closing Note

If this playbook feels heavy, that means it is doing its job. Integrations fail when teams optimize for short-term speed and leave boundary decisions implicit. This process forces the implicit decisions to become explicit, testable, and reviewable.

Use it rigorously for the first two integrations, then streamline where your team repeatedly demonstrates maturity. Do not streamline by removing boundary checks, protocol adapter tests, or handoff artifacts.

## Appendix A: Endpoint and Event Contract Worksheet

Fill this worksheet before implementing API clients.

```markdown
## Endpoint Worksheet

Base URL (host + prefix): <e.g. http://127.0.0.1:8091/confirm>

| Capability | Method | Path | Request Shape | Response Shape | Error Codes |
|---|---|---|---|---|---|
| Create request | POST | /api/requests | ... | ... | 400/500 |
| Fetch request | GET | /api/requests/{id} | n/a | ... | 404/500 |
| Submit response | POST | /api/requests/{id}/response | oneof output | updated request | 400/404/409/500 |
| Submit event | POST | /api/requests/{id}/event | event payload | updated/completed request | 400/404/409/500 |
| Touch | POST | /api/requests/{id}/touch | n/a | updated request | 404/409/500 |
| Wait | POST | /api/requests/{id}/wait | timeout opts | completed/pending | 404/500 |
| Realtime stream | GET | /ws?sessionId=... | n/a | WS event frames | connect/close/errors |
```

WS frame worksheet:

```markdown
## WS Event Worksheet

| Event Type | Required Fields | Optional Fields | Action in Runtime | Action in Host UI |
|---|---|---|---|---|
| new_request | type, request.id | ... | upsert active queue | open window |
| request_updated | type, request.id | scriptView | upsert active queue | rerender window |
| request_completed | type, request.id | output, completedAt | move to completion lane | close window |
```

Payload normalization worksheet:

```markdown
## Normalization Rules

1. Widget type canonicalization:
   - input values: confirm, Confirm, WIDGET_TYPE_CONFIRM
   - normalized value: confirm
2. Missing timestamp policy:
   - input: missing
   - normalized: current UTC RFC3339
3. Optional comment policy:
   - input: blank string
   - normalized: omitted
```

## Appendix B: Reference Commands by Phase

These commands are intended as copy/paste bootstrap for a new ticket.

### Ticket bootstrap

```bash
docmgr ticket create-ticket --ticket <TICKET-ID> --title \"<Title>\" --topics architecture,frontend,backend,go,javascript,ux
docmgr doc add --ticket <TICKET-ID> --doc-type design-doc --title \"Integration Blueprint\"
docmgr doc add --ticket <TICKET-ID> --doc-type reference --title \"Diary\"
```

### Reconnaissance

```bash
rg -n \"Handle\\(|/api|/ws|openWindow|create.*Runtime|Reducer|slice|oneof|protojson\" go-go-os plz-confirm
rg --files go-go-os/packages | head -n 200
```

### Backend extraction and mount validation

```bash
go test ./pkg/<public-backend-package> -count=1
go test ./cmd/<host-server> -count=1
```

### Frontend runtime validation

```bash
npm exec vitest run packages/<integration-runtime>/src/proto/*.test.ts
npm run storybook:check
```

### Roundtrip smoke

```bash
BASE=\"http://127.0.0.1:8091/<prefix>\"
SESSION=\"global\"

curl -sS -X POST \"$BASE/api/requests\" \
  -H 'content-type: application/json' \
  -d '{\"type\":\"confirm\",\"sessionId\":\"global\",\"confirmInput\":{\"title\":\"smoke\"}}'

go run ./cmd/<cli> ws --base-url \"$BASE\" --session-id \"$SESSION\" --pretty
```

### Documentation closure

```bash
docmgr doctor --ticket <TICKET-ID> --stale-after 30
docmgr changelog update --ticket <TICKET-ID> --entry \"...\" --file-note \"/abs/path:note\"
```

## Appendix C: Review Checklist for Senior Engineer Sign-off

Use this checklist during final review. A single unchecked item means the integration is not ready to close.

Architecture:

1. Integration boundary package is public and minimal.
2. No cross-module `internal/*` imports exist.
3. Prefix namespace and route ownership are explicit.
4. Adapter layer is the only place with protocol oneof logic.

Runtime/state:

1. WS disconnect and reconnect behavior is defined.
2. Active/completed request lanes are separated.
3. Non-pending replay does not re-enter active queue.
4. 409 conflict path reconciles against canonical backend state.

UI:

1. Generic widgets live in engine core.
2. Integration-specific composition stays in integration package.
3. New `data-part` names are documented and stable.
4. Composite stories cover at least one multi-step workflow.

Testing:

1. Adapter tests cover all in-scope widget/event types.
2. Host backend integration tests cover route coexistence.
3. Prefix WS test exists.
4. At least one e2e script exercises create -> complete.
5. Tests include at least one known regression signature.

Operations:

1. Runbook includes triage commands and expected outputs.
2. Rollback trigger and rollback owner are documented.
3. Release/versioning implications are tracked.

Documentation:

1. Diary includes failures and exact commands.
2. Changelog entries reference commits and files.
3. Ticket index links all key docs and scripts.
4. Postmortem and playbook were updated after completion.

## Appendix D: Versioning and Dependency Publication Guidance

Cross-repo integrations often look complete locally but fail for downstream users because publish/version state is unclear.

Required versioning notes per ticket:

1. Which new packages/symbols were added?
2. Which repository must cut a release first?
3. Which host dependency currently relies on local workspace resolution?
4. What is the minimum published version for stable consumption?

Template:

```markdown
## Dependency Release Notes

Producer repo: <repo that added public integration package>
New export path: <module path>
Local workspace dependency used: yes/no
Minimum release required: <tag/version>
Host repo update step:
1. bump dependency
2. run integration tests
3. remove temporary replace/workspace override if present
```

Do not mark integration complete until this section is filled; otherwise teammates can unknowingly depend on unpublished local-only behavior.
