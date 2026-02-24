---
Title: 'Inspector Review: plz-confirm Integration Quality Audit'
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
    - Path: ../../../../../../../go-go-os/packages/confirm-runtime/src/proto/confirmProtoAdapter.ts
      Note: Primary protocol adapter where several contract-shape findings originate
    - Path: ../../../../../../../go-go-os/packages/confirm-runtime/src/components/ConfirmRequestWindowHost.tsx
      Note: Request host composition and widget submit mapping risks
    - Path: ../../../../../../../go-go-os/packages/confirm-runtime/src/ws/confirmWsManager.ts
      Note: WS lifecycle and reconnection behavior audit
    - Path: ../../../../../../../go-go-os/packages/confirm-runtime/src/state/confirmRuntimeSlice.ts
      Note: Queue lifecycle and replay reconciliation behavior
    - Path: ../../../../../../../go-go-os/packages/engine/src/components/widgets/SchemaFormRenderer.tsx
      Note: Schema form type inference and state lifecycle findings
    - Path: ../../../../../../../go-go-os/packages/engine/src/components/widgets/RequestActionBar.tsx
      Note: Comment-state leakage risk in uncontrolled mode
    - Path: ../../../../../../../go-go-os/packages/engine/src/components/widgets/SelectableDataTable.tsx
      Note: Row key fallback behavior and selection correctness
    - Path: ../../../../../../../go-go-os/packages/engine/src/components/widgets/SelectableList.tsx
      Note: Selection semantics and keyboard behavior
    - Path: ../../../../../../../go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main.go
      Note: Router integration and mount composition audit
    - Path: ../../../../../../../go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main_integration_test.go
      Note: Current integration test coverage baseline
    - Path: ../../../../../../../plz-confirm/pkg/backend/backend.go
      Note: Public embeddable backend facade review
    - Path: ../../../../../../../plz-confirm/internal/server/server.go
      Note: Request lifecycle validation and timestamp fallback behavior
    - Path: ../../../../../../../plz-confirm/internal/server/ws.go
      Note: Broadcast model and contention risk
    - Path: ../../../../../../../plz-confirm/proto/plz_confirm/v1/widgets.proto
      Note: Authoritative contract for multi-select and int64 fields
    - Path: ../../../../../../../plz-confirm/proto/plz_confirm/v1/request.proto
      Note: Status enum contract mismatch review
    - Path: ../../../../../../../plz-confirm/agent-ui-system/client/src/components/widgets/SelectDialog.tsx
      Note: Legacy behavior baseline used for backward-compatibility comparison
    - Path: ../../../../../../../plz-confirm/agent-ui-system/client/src/components/widgets/TableDialog.tsx
      Note: Legacy behavior baseline for table multi-select output and row keys
Summary: Exhaustive inspector-style code review of the plz-confirm integration into go-go-os, covering architecture, API contracts, runtime behavior, widget design, testing depth, operational resilience, and backward compatibility.
LastUpdated: 2026-02-23T19:56:39-05:00
WhatFor: Provide a critical, high-fidelity quality assessment and remediation roadmap before broadening adoption of the integrated confirm-runtime stack.
WhenToUse: Use before declaring integration production-ready, when prioritizing stabilization tasks, or when onboarding engineers to maintain and harden the integration.
---

# Inspector Review: plz-confirm Integration Quality Audit

## Executive Summary

This review audited the full integration delivered under `PC-05`: public backend embedding, `/confirm` router mount, `@hypercard/confirm-runtime`, inventory host wiring, core widget additions, protocol adapters, stale-queue reconciliation, and visual/storybook handoff work.

The integration is directionally strong and already solves the original strategic objective: plz-confirm remains backend/script authority while go-go-os provides desktop-window frontend UX. The package-first and core-first architectural pivots were correct and materially improved reuse and maintainability.

However, the audit found several non-trivial correctness risks and contract mismatches that should be addressed before treating the integration as hardened:

1. multi-select output encoding is shape-heuristic and not mode-aware, causing backward-compatibility drift versus the legacy plz-confirm UI;
2. table row-key defaulting can collapse distinct rows when `id` is absent, producing incorrect selections;
3. upload `maxSize` client enforcement is currently bypassed due protojson `int64` string decoding;
4. form boolean fields are not represented as booleans in rendered UI controls;
5. uncontrolled comment state can leak between requests/steps;
6. websocket lifecycle has no reconnect/backoff strategy, and broadcaster write lock is globally serialized.

These are fixable within the current architecture. The report includes a prioritized remediation plan (C0/C1/C2), concrete test additions, and code-level recommendations.

## Scope and Methodology

### Scope reviewed

This review covered all integration-related code from the implementation tranche set:

- `go-go-os` commits from `48c2724` through `6559031`
- `plz-confirm` commits from `56e40ec` through `850b79c`
- associated integration tests, stories, and runtime APIs

### Inspection method

1. Static source review of changed integration files.
2. Contract comparison against authoritative protobuf schema.
3. Behavioral comparison against legacy frontend implementation (`agent-ui-system/client`) using `git` history.
4. Targeted test execution on touched areas.
5. Architectural and API design review (coupling, boundary hygiene, resiliency, observability).

### Commands executed

Backend:

```bash
cd plz-confirm
go test ./pkg/backend ./internal/server ./cmd/plz-confirm -count=1
```

Frontend/runtime/unit:

```bash
cd go-go-os
npm exec vitest run \
  packages/confirm-runtime/src/proto/confirmProtoAdapter.test.ts \
  packages/engine/src/__tests__/schema-form-renderer.test.ts \
  packages/engine/src/__tests__/selectable-data-table.test.ts \
  packages/engine/src/__tests__/selectable-list.test.ts
```

All above passed; findings below therefore focus primarily on semantic correctness gaps and untested edge cases rather than currently failing tests.

## Findings (Ordered by Severity)

### Criticality Legend

- `P0` Critical: likely incorrect behavior with user-visible or contract-breaking impact.
- `P1` High: significant correctness or compatibility risk; should be addressed in stabilization.
- `P2` Medium: quality/usability/reliability issue; plan in near-term hardening.
- `P3` Low: polish, maintainability, or scale concern.

---

## `P1` Finding 1: Multi-select outputs are encoded heuristically, not by declared widget mode

### Evidence

Mode-blind response mapping:

- [confirmProtoAdapter.ts](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/proto/confirmProtoAdapter.ts:214) uses selection length to choose `selectedSingle` vs `selectedMulti` for `select`.
- [confirmProtoAdapter.ts](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/proto/confirmProtoAdapter.ts:242) does the same for `table`.
- [confirmProtoAdapter.ts](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/proto/confirmProtoAdapter.ts:281) does similar fallback for `image` non-confirm mode.

Legacy behavior (baseline) is mode-aware, not length-aware:

- [SelectDialog.tsx](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/agent-ui-system/client/src/components/widgets/SelectDialog.tsx:133) emits `selectedMulti` whenever `input.multi` is true, even for one selected item.
- [TableDialog.tsx](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/agent-ui-system/client/src/components/widgets/TableDialog.tsx:162) emits `selectedMulti` whenever `input.multiSelect` is true.

Protocol supports both oneof forms, but mode semantics should remain stable across frontends:

- [widgets.proto](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/proto/plz_confirm/v1/widgets.proto:31)

### Impact

For multi-enabled requests with exactly one selected item, the new integration can emit single-selection oneof variants. This is a contract-shape drift relative to legacy behavior and can break downstream assumptions in automation scripts or post-processing code expecting `selectedMulti` when `multi=true`/`multiSelect=true`.

### Risk level

`P1` (high compatibility risk)

### Recommendation

Make output encoding explicitly mode-aware:

- For `select`: inspect request input `multi` and emit `selectedMulti` whenever true.
- For `table`: inspect request input `multiSelect` and emit `selectedMulti` whenever true.
- For `image`: inspect `multi`/`mode` to preserve multi variant semantics independent of count.

Suggested pseudocode:

```text
if request.widgetType == select:
  if request.input.payload.multi == true:
    emit selectedMulti(values=selectedIds)
  else:
    emit selectedSingle(value=selectedIds[0] || "")
```

### Required tests

Add adapter tests for:

1. multi=true with one selected -> `selectedMulti`
2. multi=false with one selected -> `selectedSingle`
3. multi=true with zero selected -> explicit policy (error or empty multi)

---

## `P1` Finding 2: Table selection keying can collapse distinct rows when `id` is absent

### Evidence

In host component, default row key is hardcoded to `id`:

- [ConfirmRequestWindowHost.tsx](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/components/ConfirmRequestWindowHost.tsx:275)

In table widget, string rowKey resolves via direct property lookup:

- [SelectableDataTable.tsx](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/SelectableDataTable.tsx:31)

When row lacks `id`, `String(row["id"])` becomes `'undefined'`, causing key collisions and potentially incorrect selected row sets.

Legacy baseline uses fallback row identity for missing `id`:

- [TableDialog.tsx](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/agent-ui-system/client/src/components/widgets/TableDialog.tsx:25)

### Impact

For tables whose rows do not carry an `id` field (valid under proto dynamic struct model), selecting one row can be interpreted as selecting multiple/all rows sharing `'undefined'` key semantics, corrupting submitted output.

### Risk level

`P1` (high correctness risk for arbitrary table data)

### Recommendation

Use safe key strategy in host:

1. prefer payload `rowKey` when provided;
2. if missing, omit `rowKey` so component fallback logic uses `row.id ?? index`;
3. optionally provide host-level fallback function akin to legacy JSON-string identity when no stable id exists.

### Required tests

1. table rows without `id`: selecting one row should only select that row.
2. table rows with explicit `rowKey` should preserve expected selections.

---

## `P1` Finding 3: Upload max-size constraint is not honored due protojson int64 string decoding

### Evidence

`max_size` is `int64` in contract:

- [widgets.proto](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/proto/plz_confirm/v1/widgets.proto:104)

In browser runtime, `mapUIRequestFromProto` leaves raw payload unnormalized:

- [confirmProtoAdapter.ts](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/proto/confirmProtoAdapter.ts:144)

Host only accepts numeric `maxSize`:

- [ConfirmRequestWindowHost.tsx](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/components/ConfirmRequestWindowHost.tsx:460)

Protojson int64 fields arrive as strings in JS JSON parsing, so `typeof payload.maxSize === 'number'` is false and the limit is effectively ignored in client-side filtering.

### Impact

UI-side file size constraints are silently disabled for common payloads, weakening expected validation behavior and risking user confusion (backend may still reject later or accept inconsistent metadata).

### Risk level

`P1` (high validation correctness risk)

### Recommendation

Normalize numeric string fields in adapter:

```text
maxSizeRaw = rawInput.maxSize
if typeof maxSizeRaw == "string" and numeric:
  payload.maxSize = Number(maxSizeRaw)
```

Also normalize other int64 fields that matter for UI logic.

### Required tests

1. upload input with `maxSize` as string should enforce size limit.
2. upload input with invalid numeric string should fail predictably (or ignore with warning).

---

## `P2` Finding 4: Boolean schema fields are rendered as text inputs, not boolean controls

### Evidence

Boolean types are not handled in `inferFieldType`:

- [SchemaFormRenderer.tsx](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/SchemaFormRenderer.tsx:26)

Field type system currently lacks boolean control type:

- [types.ts](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/types.ts:16)

`FieldRow` only renders `text`, `number`, `select`, `readonly`, `tags`:

- [FieldRow.tsx](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/FieldRow.tsx:20)

### Impact

Form parity is lower than expected for JSON schema (booleans appear as text), increasing operator friction and error likelihood.

### Risk level

`P2` (medium UX/correctness)

### Recommendation

Add boolean support end-to-end:

1. extend `FieldType` with `boolean` (or `checkbox`);
2. map schema `type: boolean` accordingly;
3. render checkbox/toggle in `FieldRow`.

### Required tests

1. schema boolean field maps to checkbox type.
2. submit coercion with checkbox value remains boolean.

---

## `P2` Finding 5: Uncontrolled comment state can leak between sequential requests/steps

### Evidence

`RequestActionBar` keeps internal comment state without request/step reset hook:

- [RequestActionBar.tsx](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/RequestActionBar.tsx:32)

Host resets many per-widget states on request/step changes but does not control `RequestActionBar` comment values:

- [ConfirmRequestWindowHost.tsx](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/components/ConfirmRequestWindowHost.tsx:151)

### Impact

Previous comments may unintentionally carry into subsequent requests or script steps, which can cause incorrect submissions or accidental data leakage.

### Risk level

`P2` (medium data hygiene/UX risk)

### Recommendation

Either:

1. make comment field controlled from host and reset with request/step key, or
2. add reset key/prop to `RequestActionBar` and clear internal state on key changes.

### Required tests

1. open request A, type comment, then request B -> comment should start empty.
2. script step transition should not inherit prior step comment unless explicitly designed.

---

## `P2` Finding 6: Status mapping and completion payload mapping are partially lossy

### Evidence

Status mapper includes non-contract `expired` and omits proto enum `timeout`:

- [confirmProtoAdapter.ts](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/proto/confirmProtoAdapter.ts:44)
- [request.proto](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/proto/plz_confirm/v1/request.proto:26)

Realtime event mapper discards completion output:

- [confirmProtoAdapter.ts](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/proto/confirmProtoAdapter.ts:194)

### Impact

Runtime completion records cannot capture output payloads from WS events, limiting downstream debugging/audit UIs. Status semantics can become inconsistent if backend starts emitting `timeout` in future paths.

### Risk level

`P2` (medium observability/compatibility risk)

### Recommendation

1. align status mapping strictly with proto enum names.
2. map completion output from event/request oneof into `event.output` when available.

### Required tests

1. status `timeout` maps deterministically.
2. `request_completed` event with output preserves output in runtime completion map.

---

## `P2` Finding 7: Form renderer internal state does not resync on schema/value changes in uncontrolled mode

### Evidence

`useState(initialValue)` initialized once, no effect to sync on `initialValue` changes:

- [SchemaFormRenderer.tsx](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/SchemaFormRenderer.tsx:114)

### Impact

When same renderer instance is reused across request changes with differing schemas/defaults, stale values can persist unexpectedly.

### Risk level

`P2` (medium lifecycle correctness risk)

### Recommendation

Add sync effect:

```ts
useEffect(() => {
  if (onValueChange === undefined) setInternalValues(initialValue);
}, [initialValue, onValueChange]);
```

Or key the component by request+step in host to guarantee remount.

---

## `P3` Finding 8: WS manager lacks reconnect/backoff strategy for transient disconnects

### Evidence

`ConfirmWsManager` only connects once and exposes manual disconnect; no retry policy:

- [confirmWsManager.ts](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/ws/confirmWsManager.ts:21)

### Impact

Transient network/server restarts can leave runtime disconnected until app refresh or manual reconnect path.

### Risk level

`P3` (reliability hardening)

### Recommendation

Introduce optional reconnect policy with exponential backoff and max interval cap.

---

## `P3` Finding 9: WS broadcaster serializes writes globally across all clients

### Evidence

Broadcaster uses a single `writeMu` for all connections:

- [ws.go](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/internal/server/ws.go:14)
- [ws.go](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/internal/server/ws.go:99)

### Impact

One slow client write path can reduce throughput for all sessions due global lock contention.

### Risk level

`P3` (scale/performance)

### Recommendation

Use per-connection write locking (or a dedicated write pump per connection) instead of global lock.

## Architecture Assessment

### Strong architectural choices

1. Public backend extraction (`pkg/backend`) was the right legal boundary solution.
2. Prefix mount under `/confirm` cleanly avoids host route collisions.
3. Package-first runtime (`@hypercard/confirm-runtime`) creates reusable integration surface.
4. Core-first widget additions in `engine` avoid app-specific duplication.
5. Conflict reconciliation (`409`) in host app is a practical robustness improvement.
6. Backend timestamp fallback is a solid canonicalization guardrail.

### Architectural risks to monitor

1. Protocol adapter now carries significant policy; must remain heavily tested.
2. Integration package has thin runtime tests (mostly adapter-only), leaving lifecycle paths under-tested.
3. Story coverage is broad for design handoff but not a substitute for contract/integration tests.

## API Contract Review

### REST/WS topology

The embedded route strategy is coherent:

- host routes unchanged
- plz-confirm mounted under `/confirm`
- prefixed WS endpoint `/confirm/ws`

This is validated by host integration tests and by real CLI manual flows.

### Schema fidelity

Most oneof mapping is correct, but the audit identified mode-aware output regressions and numeric-string normalization gaps.

Key rule to enforce going forward:

- outbound oneof choice should be based on declared request semantics, not inferred by selected count.

## Integration Design Review

### Backend-host integration

`main.go` mount placement is correct and non-invasive:

- [main.go](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main.go:146)

### Frontend host integration

Inventory app remains a thin adapter that wires runtime into windowing contributions. This is clean.

Potential improvement:

- move 409 reconciliation logic into runtime package to avoid host duplication and enforce consistency across future apps.

## Testing Depth Assessment

### What is covered well

1. Backend mount basic create-request flow.
2. Host route coexistence and prefixed WS endpoint.
3. Adapter basic decode/encode happy-path scenarios.
4. Core widget helper unit tests.

### What is not covered (gaps)

1. mode-aware oneof regression tests for single-item multi-select.
2. upload max-size as numeric-string normalization.
3. table rows without `id` selection correctness.
4. comment reset lifecycle across request transitions.
5. schema renderer state reset across value/schema changes.
6. runtime reconnect/disconnect behavior tests.
7. completion event output mapping tests.
8. root-prefix (`--root`) + `/confirm` combined integration tests.

## Recommended Remediation Plan

### C0 (Immediate, correctness/compatibility)

1. Fix mode-aware output encoding in adapter for select/table/image.
2. Fix table row-key fallback behavior in request host.
3. Normalize upload `maxSize` numeric strings in adapter.
4. Add tests for all three above.

### C1 (Near-term hardening)

1. Add boolean field support in schema form renderer + field row.
2. Reset/request-scope comment state in action bar.
3. Add sync behavior for form internal state or key-based remount.
4. Align status mapping with proto enum and preserve completion outputs.

### C2 (Resilience/scale)

1. Add optional WS reconnect strategy.
2. Evaluate per-connection write lock in server broadcaster.
3. Add runtime-level integration tests around reconnect/replay.

## Proposed Task Breakdown

```markdown
- [ ] CR-01: Make select/table/image output encoding mode-aware (adapter + tests)
- [ ] CR-02: Fix table row key fallback when `id` missing (host + tests)
- [ ] CR-03: Parse upload maxSize int64 string to number (adapter + tests)
- [ ] CR-04: Add boolean schema field type and UI control support
- [ ] CR-05: Reset comment state per request/step in RequestActionBar usage
- [ ] CR-06: Sync SchemaFormRenderer uncontrolled state on schema/value changes
- [ ] CR-07: Map timeout status + completion outputs in realtime adapter
- [ ] CR-08: Add WS reconnect policy in runtime manager
- [ ] CR-09: Add integration tests for root-prefix + /confirm route coexistence
```

## Suggested Regression Test Additions (Concrete)

### Adapter tests (`confirmProtoAdapter.test.ts`)

1. `select` request with `multi=true` and single selected id -> `selectedMulti`.
2. `table` request with `multiSelect=true` and one row -> `selectedMulti`.
3. `image` request with `multi=true` and single id -> `selectedStrings`.
4. upload `maxSize: "10485760"` becomes number in mapped payload.
5. status `timeout` maps to runtime enum value.

### Host/widget tests

1. Table rows without `id` produce distinct selection keys.
2. RequestActionBar comment clears on request switch.
3. SchemaFormRenderer boolean field renders boolean control.
4. SchemaFormRenderer resets internal defaults on schema swap.

### Runtime tests

1. WS disconnect then reconnect with pending replay recovers queue.
2. completion event payload is captured in `completionsById` output.

## Inspector Notes on Positive Quality

Despite findings, the integration quality is substantially above typical first-pass cross-repo integration work:

1. route namespace strategy is clean;
2. backend packaging solution is pragmatic and test-backed;
3. runtime host boundary is conceptually sound;
4. diary/changelog discipline enabled high-confidence auditability;
5. composite story coverage is unusually strong for handoff quality.

This means stabilization can focus on targeted correctness fixes rather than architectural rewrites.

## Risk Register (Post-Review)

| Risk | Likelihood | Impact | Priority | Mitigation |
|---|---|---|---|---|
| Multi-select shape mismatch breaks consumers | Medium | High | C0 | Mode-aware encoder + regression tests |
| Table selection corruption without row ids | Medium | High | C0 | Row-key fallback fix + tests |
| Upload size guard bypass in UI | Medium | Medium | C0 | Numeric-string normalization |
| Boolean forms produce wrong UX/input | High | Medium | C1 | Boolean control support |
| Stale comment leakage | Medium | Medium | C1 | Controlled/reset comment state |
| WS transient disconnect requires manual recovery | Medium | Medium | C2 | Reconnect strategy |

## Acceptance Criteria for Stabilization Closure

Stabilization should be considered complete when all are true:

1. C0 items merged with passing tests.
2. No adapter contract regressions against legacy behavior for select/table/image mode semantics.
3. Table selections are correct for rows lacking explicit id.
4. Upload max-size UI validation works for protojson int64 string payloads.
5. Form boolean controls and state reset behavior validated.
6. Runtime status mapping reflects proto enum names.
7. Diary/changelog updated with each stabilization tranche.

## Open Questions

1. Should mode-shape strictness be enforced server-side (reject single variant when request was multi), or only normalized client-side?
2. For table row identity, do we want deterministic hash fallback or payload-provided `rowKey` as mandatory in future schema?
3. Should reconnect policy live in runtime package by default, or injected via host adapters?
4. Do we want to persist completion outputs in a dedicated history view for operators?

## Reviewer Conclusion

The integration is a strong architectural foundation with a manageable stabilization backlog. The most important work now is contract-shape correctness and edge-case lifecycle hardening, not broad redesign.

If C0/C1 items are executed with the recommended test additions, this integration should be robust enough for wider adoption and can serve as a reliable template for future external-system onboarding into go-go-os.

## Appendix: Key Evidence References

### Core integration files

- [backend.go](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/pkg/backend/backend.go)
- [main.go](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main.go)
- [createConfirmRuntime.ts](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/runtime/createConfirmRuntime.ts)
- [confirmProtoAdapter.ts](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/proto/confirmProtoAdapter.ts)
- [ConfirmRequestWindowHost.tsx](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/components/ConfirmRequestWindowHost.tsx)
- [confirmRuntimeSlice.ts](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/state/confirmRuntimeSlice.ts)

### Legacy behavior comparison

- [SelectDialog.tsx](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/agent-ui-system/client/src/components/widgets/SelectDialog.tsx)
- [TableDialog.tsx](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/agent-ui-system/client/src/components/widgets/TableDialog.tsx)
- [ImageDialog.tsx](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/agent-ui-system/client/src/components/widgets/ImageDialog.tsx)

### Contracts

- [widgets.proto](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/proto/plz_confirm/v1/widgets.proto)
- [request.proto](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/proto/plz_confirm/v1/request.proto)

## Appendix B: Architectural Scorecard

This section scores each integration plane from `1-5` for current quality posture, where `5` means stable and production-ready with strong test confidence.

| Plane | Score | Notes |
|---|---|---|
| Package boundaries | 4.5 | Public backend extraction is strong; dependency publication policy still needs stricter closure checks |
| Router composition | 4.0 | Prefix mount design is clean; root-prefix combinatorics need one extra integration test |
| Protocol adapter | 3.0 | Good abstraction, but mode-shape and int64 normalization issues are significant |
| Runtime lifecycle | 3.5 | Core lifecycle works; reconnect and output preservation are currently weak |
| Widget primitives | 4.0 | Strong foundation, broad stories; boolean form support gap remains |
| Host integration | 3.5 | Thin app adapter is good; duplicated reconciliation logic indicates missing runtime abstraction |
| Test depth | 3.0 | Healthy unit baseline but important contract and lifecycle edges remain untested |
| Observability/debuggability | 4.0 | Scripts + diary + changelog are excellent; completion payload visibility is limited |
| Documentation quality | 5.0 | Unusually high quality for an integration project |

Net score: **3.8 / 5** (good integration foundation, needs stabilization pass before calling fully hardened).

## Appendix C: Endpoint and Semantics Audit

### `POST /confirm/api/requests`

What is correct:

1. backend validates oneof/input-type congruence;
2. script requests run `init/view` server-side and persist view/state metadata;
3. metadata enrichment (`remoteAddr`, `userAgent`) is applied.

Potential risk:

1. no explicit payload-size telemetry or structured validation error schema (text-only errors today).

Recommendation:

1. standardize error response envelope for easier frontend error handling.

### `POST /confirm/api/requests/{id}/response`

What is correct:

1. type congruence checks between request type and output oneof are strict;
2. timestamps are now server-populated when omitted for confirm/image;
3. completion event is broadcast reliably.

Potential risk:

1. server does not currently enforce mode-sensitive shape (`selectedMulti` vs `selectedSingle`) by request input mode, so frontend contract drift can persist silently.

Recommendation:

1. optionally add server-side validation for mode/shape consistency if backward-compatibility requirements demand strictness.

### `POST /confirm/api/requests/{id}/event`

What is correct:

1. per-request lock prevents concurrent update races;
2. pending-only guard is enforced;
3. done vs updated branching is explicit and event emission matches state.

Potential risk:

1. frontend script-event payload validation is light; malformed payloads currently surface as backend validation errors without richer local diagnostics.

Recommendation:

1. optionally add lightweight frontend event payload validation before submit.

### `GET /confirm/ws?sessionId=...`

What is correct:

1. session-scoped connections are maintained;
2. on-connect pending replay behavior is useful for recovery;
3. connection cleanup is explicit on read error.

Potential risk:

1. global broadcaster write lock can become cross-session bottleneck;
2. no server-side ping/pong strategy beyond read loop could delay dead-connection detection under some network conditions.

Recommendation:

1. move to per-conn write serialization and consider ping ticker for better liveness detection.

## Appendix D: Detailed Runtime Lifecycle Trace

This trace documents expected behavior and current weak points.

### Expected nominal path

```text
new_request WS frame ->
  adapter maps request ->
  reducer upserts active queue ->
  runtime host opens window ->
  user submits ->
  API submit returns updated/completed ->
  WS request_completed ->
  reducer marks completion ->
  host closes window
```

### Observed edge paths and behavior

1. **Stale completion replay**: currently handled well by status-aware `upsertActiveRequest` plus 409 reconcile.
2. **Reconnect without retry**: currently not handled automatically.
3. **Completion output preservation**: currently dropped (`event.output` not mapped), reducing inspection value of completion lane.

### Suggested lifecycle hardening pseudocode

```text
onWsClose():
  setConnected(false)
  if reconnectEnabled:
    scheduleReconnect(backoff.next())

onReconnectOpen():
  backoff.reset()
  setConnected(true)

onRequestCompleted(event):
  output = decodeOutputFromEvent(event.request || event.output)
  completeRequest(id, completedAt, output)
```

## Appendix E: UI/UX and Accessibility Observations

### Positive observations

1. New widgets consistently use `data-part` vocabulary, enabling theming and design handoff.
2. Keyboard navigation exists in `SelectableList` (arrow keys + enter/space).
3. Focus-visible styles were added for key interactive elements.

### Accessibility concerns

1. `ImageChoiceGrid` does not define explicit `aria-pressed` for selected toggle cards.
2. `GridBoard` and rating buttons rely on visual state; richer ARIA labels could improve screen reader behavior.
3. `RequestActionBar` textarea has placeholder but no explicit label/`aria-label`.

These are not immediate blockers but should be tracked in the polish backlog.

## Appendix F: Backward-Compatibility Comparison Notes

Using older/legacy frontend code as reference, this audit compared output semantics:

1. Legacy `SelectDialog` uses mode-driven oneof output (`selectedMulti` if multi).
2. Legacy `TableDialog` also uses mode-driven oneof output.
3. Legacy table row identity uses `row.id || JSON.stringify(row)` fallback, preventing collisions for rows without id.

Current integration diverges on these points, which justifies prioritizing C0 fixes.

## Appendix G: Test Debt Heatmap

This matrix maps risk areas to current and target test depth.

| Risk Area | Current Tests | Needed Additions |
|---|---|---|
| Mode-aware output encoding | Partial adapter happy-path tests | Multi-with-single-item shape tests |
| Table row identity without id | None | Unit + host-level submit snapshot |
| Upload maxSize int64 string | None | Adapter normalization test + host behavior test |
| Comment leakage | None | Host interaction test across request transitions |
| Form boolean controls | None | Schema render and submit tests |
| Reconnect resilience | None | WS manager retry tests with mocked socket |
| Completion output persistence | None | Adapter+slice tests for completion payload retention |
| Root prefix + /confirm | None | Backend integration test with `--root` simulation |

## Appendix H: Recommended Stabilization Sprint Plan

### Sprint 1 (2-3 days): Contract and correctness

1. Fix adapter mode-aware output encoding.
2. Fix table row-key fallback logic.
3. Normalize upload max size numeric-string conversion.
4. Add targeted unit tests for all above.

Exit criteria:

1. all C0 tests pass;
2. no regressions in existing adapter tests.

### Sprint 2 (2-4 days): Form and UI state hygiene

1. Add boolean control support to schema forms.
2. Resolve uncontrolled comment-state leakage.
3. Add form-state resync on schema/value changes.
4. Add interaction tests for sequential request transitions.

Exit criteria:

1. form boolean scenarios are storybook-covered;
2. request-step transitions do not preserve stale input unexpectedly.

### Sprint 3 (2-3 days): Runtime resilience and observability

1. Add WS reconnect policy (configurable).
2. Preserve completion outputs in runtime completion lane.
3. Add runtime reconnect and completion payload tests.
4. Evaluate broadcaster lock model and create follow-up ticket if structural change deferred.

Exit criteria:

1. runtime survives WS restart in local integration test;
2. completion payload visible in state/debug output.
