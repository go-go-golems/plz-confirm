---
Title: 'Implementation Plan: P2 Hardening'
Ticket: PC-10-P2-HARDENING
Status: complete
Topics:
    - architecture
    - frontend
    - backend
    - javascript
    - go
    - api
    - ux
    - bug
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../go-go-os/packages/engine/src/components/widgets/SchemaFormRenderer.tsx
      Note: Boolean schema mapping and uncontrolled state resync behavior
    - Path: ../../../../../../../go-go-os/packages/engine/src/components/widgets/FieldRow.tsx
      Note: Boolean form control rendering path
    - Path: ../../../../../../../go-go-os/packages/engine/src/components/widgets/FormView.tsx
      Note: Required value validation logic for false/zero semantics
    - Path: ../../../../../../../go-go-os/packages/engine/src/types.ts
      Note: FieldType extension for boolean support
    - Path: ../../../../../../../go-go-os/packages/confirm-runtime/src/components/ConfirmRequestWindowHost.tsx
      Note: Request-scoped reset strategy for action bar comment state
    - Path: ../../../../../../../go-go-os/packages/confirm-runtime/src/proto/confirmProtoAdapter.ts
      Note: Status/output mapping hardening for realtime events
    - Path: ../../../../../../../go-go-os/packages/confirm-runtime/src/proto/confirmProtoAdapter.test.ts
      Note: Adapter regression tests for P2 mapping behavior
    - Path: ../../../../../../../plz-confirm/proto/plz_confirm/v1/request.proto
      Note: Authoritative status enum contract for timeout/error names
ExternalSources: []
Summary: Implementation contract for addressing P2 findings 4-7 from the PC-05 inspector review (forms, comment state, status/output mapping).
LastUpdated: 2026-02-24T20:48:00-05:00
WhatFor: Execute medium-severity reliability/UX fixes with explicit tests and closure criteria before broader adoption.
WhenToUse: Use while implementing, reviewing, and validating PC-10 P2 hardening work.
---

# Implementation Plan: P2 Hardening

## Executive Summary

This ticket hardens four medium-severity integration gaps identified in the inspector audit:

1. boolean JSON schema fields not represented as boolean controls;
2. form required-check semantics that treat `false`/`0` as missing;
3. comment text leakage risk across sequential requests/steps;
4. lossy status and websocket completion-output mapping.

The implementation is intentionally constrained to targeted patches in `packages/engine` and `packages/confirm-runtime`, backed by focused regression tests and an explicit validation script.

## Problem Statement

Current runtime behavior can produce confusing UX or metadata loss:

- boolean schema fields degrade to text entry, increasing friction and coercion errors;
- required-value checks in form submit path rely on truthiness, invalidating legitimate values like `false` and `0`;
- comment text in action bars can persist between request/step contexts when component instances are reused;
- runtime status/event mapping misses proto contract values (`timeout`, `error`) and drops output payloads from `request_completed` websocket events.

## Proposed Solution

### S1. Boolean field support in engine form stack

- Extend `FieldType` with `boolean`.
- Teach schema inference to map `type: boolean` to `boolean` controls.
- Render checkbox input in `FieldRow` for boolean fields.

### S2. Required-field validation correctness

- Replace truthy checks in form submit gating with explicit "empty" checks (`undefined`, `null`, empty string).
- Keep `false` and `0` valid for required fields.

### S3. SchemaFormRenderer uncontrolled resync

- Add sync effect so uncontrolled internal values are refreshed when `initialValue` changes.
- Keep controlled mode (`onValueChange` present) unchanged.

### S4. Request-scoped comment reset

- Add deterministic request/step scoped reset key in confirm runtime host.
- Wire `RequestActionBar` instances with stable keying so comment state resets on request/step transitions.

### S5. Status/output mapping hardening

- Align runtime status mapping to proto names: `pending`, `completed`, `timeout`, `error`; map legacy `expired` to `timeout`.
- Extract and preserve completed request output oneof payload from websocket event request envelope.

## Design Decisions

1. **Fix boolean support in core widgets, not adapter-only coercion.**
   Reason: this is a UI semantics gap in shared engine components and should be solved at core form rendering level.

2. **Treat legacy `expired` as normalized `timeout`.**
   Reason: keeps backward tolerance while converging runtime state to proto contract vocabulary.

3. **Reset comment state via request-scoped keying in host layer.**
   Reason: avoids wider API churn; leverages React remount behavior to clear uncontrolled local state per request/step context.

4. **Capture completion output in adapter boundary.**
   Reason: mapper is the canonical place to preserve wire payload structure for runtime consumers.

## Alternatives Considered

1. **Leave `FieldRow` unchanged and rely on boolean string coercion only.**
   Rejected: still poor UX and high operator error risk.

2. **Force all action-bar comments to controlled mode globally.**
   Rejected: broader API migration than needed for P2 timeline.

3. **Ignore legacy `expired` entirely.**
   Rejected: could cause regressions against older payload emitters.

4. **Parse completion output in runtime reducer instead of adapter.**
   Rejected: leaks wire-format concerns into state layer.

## Implementation Plan

### Phase A: Engine form hardening

1. Add boolean field type support in `types.ts`, schema inference, and `FieldRow`.
2. Fix required-field check semantics in `FormView`.
3. Add uncontrolled resync effect in `SchemaFormRenderer`.
4. Extend engine tests for boolean mapping/coercion/required behavior.

### Phase B: Confirm runtime hardening

1. Add request/step scoped reset key logic for action bars in host.
2. Harden status mapping in adapter (`timeout`/`error` + legacy `expired` normalization).
3. Map completion output payload from websocket request envelopes.
4. Extend adapter/host tests for P2 scenarios.

### Phase C: Verification and closure

1. Run focused vitest suites for touched files.
2. Run engine package tests and plz-confirm backend tests.
3. Add ticket validation script and run it.
4. Update diary/changelog/tasks and close ticket.

## Open Questions

1. Should `RequestActionBar` expose a formal `resetKey` prop in a follow-up to generalize reset semantics outside confirm runtime?
2. Should runtime statuses eventually include a distinct "legacy-expired" marker for observability, or is normalization to `timeout` sufficient?

## References

- Inspector report section for findings 4-7:
  - `PC-05 design-doc/04-inspector-review-plz-confirm-integration-quality-audit.md`
