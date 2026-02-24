---
Title: 'Implementation Plan: P1 Stabilization'
Ticket: PC-09-P1-STABILIZATION
Status: complete
Topics:
    - architecture
    - frontend
    - backend
    - javascript
    - go
    - api
    - bug
    - ux
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../go-go-os/packages/confirm-runtime/src/proto/confirmProtoAdapter.ts
      Note: Primary target for mode-aware oneof encoding and numeric normalization fixes
    - Path: ../../../../../../../go-go-os/packages/confirm-runtime/src/proto/confirmProtoAdapter.test.ts
      Note: Contract regression tests for P1 scenarios
    - Path: ../../../../../../../go-go-os/packages/confirm-runtime/src/components/ConfirmRequestWindowHost.tsx
      Note: Table row-key fallback fix in host composition layer
    - Path: ../../../../../../../go-go-os/packages/engine/src/components/widgets/SelectableDataTable.tsx
      Note: Row-key resolution semantics used by table selection flow
    - Path: ../../../../../../../plz-confirm/proto/plz_confirm/v1/widgets.proto
      Note: Authoritative oneof and int64 contract reference
    - Path: ../../../../../../../plz-confirm/agent-ui-system/client/src/components/widgets/SelectDialog.tsx
      Note: Legacy behavior baseline for mode-aware output shape
    - Path: ../../../../../../../plz-confirm/agent-ui-system/client/src/components/widgets/TableDialog.tsx
      Note: Legacy row-key fallback and multi-select shape baseline
Summary: Focused stabilization plan to resolve all P1 findings from the inspector review: mode-aware oneof encoding, table selection key correctness, and upload max-size numeric normalization.
LastUpdated: 2026-02-24T20:37:00-05:00
WhatFor: Drive an end-to-end implementation and verification pass that closes the highest-severity contract/correctness gaps before wider adoption.
WhenToUse: Use while implementing and reviewing PC-09; treat as the execution contract for code changes, tests, and closure criteria.
---

# Implementation Plan: P1 Stabilization

## Executive Summary

This ticket resolves the three `P1` issues identified in the PC-05 inspector review:

1. mode-blind oneof output encoding for select/table/image;
2. table row-key collisions when `id` is absent;
3. upload `maxSize` client validation bypass due protojson int64 string decoding.

The implementation strategy is intentionally narrow and high-confidence: apply targeted fixes in `@hypercard/confirm-runtime`, add explicit regression tests, verify behavior with focused test runs, and close the ticket only when all stabilization acceptance criteria are met.

## Problem Statement

The current integration works for common paths but has three high-severity correctness/compatibility risks:

1. **Contract-shape drift**: when requests are multi-enabled but only one value is selected, adapter emits `selectedSingle` instead of mode-consistent `selectedMulti`, diverging from legacy behavior.
2. **Selection integrity risk**: table host default row key is fixed to `id`; rows without `id` can collapse to identical keys (`"undefined"`) and corrupt selected row output.
3. **Validation bypass risk**: upload `maxSize` often arrives as string (protojson int64), but host only honors numeric values; client-side max-size enforcement can silently fail.

## Proposed Solution

### S1. Mode-aware oneof encoding

Update `mapSubmitResponseToProto` and helper mappings to inspect request input mode flags:

- select: if `input.payload.multi === true`, always emit `selectOutput.selectedMulti`;
- table: if `input.payload.multiSelect === true`, always emit `tableOutput.selectedMulti`;
- image: if `input.payload.multi === true` (and non-confirm mode), emit `imageOutput.selectedStrings`.

This restores semantic parity with legacy frontend behavior while remaining valid under proto oneof constraints.

### S2. Safe table row-key fallback

In `ConfirmRequestWindowHost`:

- only pass `rowKey` to `SelectableDataTable` when payload explicitly provides it;
- otherwise allow widget fallback logic to use `row.id ?? index`.

This prevents `"undefined"` key collisions for rows lacking `id`.

### S3. Upload max-size normalization

In protocol adapter request mapping:

- parse known numeric-string fields to numbers where needed for UI constraints (`maxSize`);
- leave raw payload structure otherwise unchanged.

This makes `FilePickerDropzone.maxSizeBytes` enforcement reliable for protojson int64 values.

### S4. Regression test expansion

Add adapter and host-focused tests for each fixed behavior:

1. select multi-mode one-item output shape;
2. table multi-mode one-item output shape;
3. image multi-mode one-item output shape;
4. request mapping converts upload `maxSize` string to number;
5. table selection remains distinct for rows without `id` (host-level helper test path).

## Design Decisions

1. **No backend contract changes in this ticket**.
   Reason: P1 findings are integration-side semantics mismatches and can be solved client-side without altering server API behavior.

2. **Mode semantics override selection-count heuristics**.
   Reason: output shape should be deterministic from request contract, not inferred from transient selection cardinality.

3. **Host avoids forcing `rowKey='id'` by default**.
   Reason: dynamic table data is contract-valid without `id`; forcing `id` creates avoidable collisions.

4. **Normalization stays in adapter boundary**.
   Reason: keep protocol-specific coercion centralized and testable.

## Alternatives Considered

1. **Server-side strict mode validation only** (reject mismatched oneof output shape).
   Rejected for this ticket because it would break existing clients immediately and does not fix integration-generated mismatches by itself.

2. **Introduce a full typed request-input model in runtime package now**.
   Deferred: would improve type safety but is broader than P1 stabilization scope.

3. **Force callers to always provide `rowKey` for table widgets**.
   Rejected: too strict for current dynamic contracts and increases burden on script/request authors.

## Implementation Plan

### Phase A: Adapter contract fixes

1. Implement mode-aware output selection in `confirmProtoAdapter.ts`.
2. Add numeric-string parsing for upload `maxSize` during request mapping.
3. Extend adapter tests for new mode and normalization behaviors.

### Phase B: Host selection correctness

1. Update `ConfirmRequestWindowHost` table rowKey handling.
2. Add regression test coverage for no-id table rows behavior (suitable level based on existing test stack).

### Phase C: Verification and closure

1. Run focused tests (`vitest` adapter + engine/host coverage).
2. Run relevant Go tests where integration coupling could be affected.
3. Update diary/changelog/tasks and close ticket status once all tasks are done.

## Acceptance Criteria

1. Multi-mode select/table/image requests always emit multi oneof variants regardless of selected count.
2. Table request flows with rows lacking `id` produce correct, distinct selection outputs.
3. Upload `maxSize` sent as protojson string is enforced client-side in host widget path.
4. New regression tests covering each P1 scenario are present and passing.
5. Ticket docs (plan/tasks/diary/changelog) fully updated and consistent.

## Open Questions

1. Should a follow-up ticket add optional server-side mode-shape validation for defense-in-depth?
2. Should runtime types be expanded in a separate ticket to eliminate `Record<string, unknown>` payload access for core widget flags?

## References

- Inspector report: `PC-05 design-doc/04-inspector-review-plz-confirm-integration-quality-audit.md`
- Legacy baseline behavior:
  - `plz-confirm/agent-ui-system/client/src/components/widgets/SelectDialog.tsx`
  - `plz-confirm/agent-ui-system/client/src/components/widgets/TableDialog.tsx`
