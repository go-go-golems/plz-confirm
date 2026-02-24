---
Title: Diary
Ticket: PC-13-EXTRACT-409-RECONCILIATION
Status: active
Topics:
    - architecture
    - frontend
    - javascript
    - ux
    - bug
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/apps/inventory/src/App.tsx
    - /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/runtime
ExternalSources: []
Summary: Diary for extracting duplicated 409 reconciliation logic.
LastUpdated: 2026-02-24T09:42:00-05:00
WhatFor: Preserve rationale and edge-case behavior for conflict reconciliation.
WhenToUse: Use when reviewing or extending 409 behavior.
---

# Diary

## Goal

Document extraction of duplicate host-level 409 conflict logic into reusable runtime helper APIs.

## Step 1: Scope and baseline

I identified the exact duplicate blocks in inventory app submit handlers and confirmed they are semantically identical except for request ID source. That made this a clean extraction candidate with minimal behavior change risk.

The step produced a concrete implementation plan, API shape, and test matrix before touching behavior.

### Prompt Context

**User prompt (verbatim):** (see PC-12 Step 1)

**Assistant interpretation:** Implement a dedicated P3 ticket to remove 409 reconciliation redundancy with technical guidance and then execute it.

**Inferred user intent:** Keep host integrations thin and reusable from day one.

**Commit (code):** N/A (planning step)

### What I did

- Created ticket `PC-13-EXTRACT-409-RECONCILIATION`.
- Documented duplicated logic in `apps/inventory/src/App.tsx`.
- Wrote shared-helper implementation plan + planned tests.

### Why

- Centralized reconciliation avoids divergence and improves maintainability for future host apps.

### What worked

- The duplicated branches align enough to extract without adapter/API redesign.

### What didn't work

- N/A in this step.

### What I learned

- The existing reconciliation policy is sound; placement is the problem.

### What was tricky to build

- N/A yet; extraction implementation pending.

### What warrants a second pair of eyes

- Decision for latest status `pending` fallback (current plan keeps existing remove-and-close behavior).

### What should be done in the future

- Consider making fallback policy configurable if UX requirements differ by host.

### Code review instructions

- Inspect both submit handler catch blocks in `App.tsx` for pre-extraction parity.
- Verify helper tests cover completed/pending/fetch-fail/non-409 branches.

### Technical details

- Current duplicate sequence:
  1. if `error.status===409`, fetch latest;
  2. if latest non-pending -> complete + close;
  3. else remove + close.

## Step 2: Implement shared 409 reconciliation helper and adopt it in inventory host

I implemented `reconcileSubmitConflict409` in confirm-runtime and changed both inventory submit catch branches to call that single helper. This preserves existing behavior while removing host-level duplication.

The helper now owns the policy: detect 409, refetch latest, complete-and-close if non-pending, else remove-and-close fallback. I added focused tests for each branch.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Complete the extraction so host submit handlers stop duplicating reconciliation code.

**Inferred user intent:** Make conflict handling reusable and consistent for current and future host apps.

**Commit (code):** `5b373ac` — "confirm-runtime: add reconnect policy and 409 conflict helper"

### What I did

- Added `packages/confirm-runtime/src/runtime/reconcileSubmitConflict409.ts`.
- Added `packages/confirm-runtime/src/runtime/reconcileSubmitConflict409.test.ts`.
- Exported helper via `packages/confirm-runtime/src/index.ts`.
- Replaced duplicate `ConfirmApiError(409)` branches in `apps/inventory/src/App.tsx` with helper calls.

### Why

- Keep policy logic in one place.
- Avoid drift between submit-response and submit-script handlers.

### What worked

- Helper tests passed:
  - non-409 returns `false`
  - completed latest triggers `completeRequestById + close`
  - pending/refetch-fail trigger `removeRequest + close`

### What didn't work

- `npm run typecheck` and `npm run typecheck -w packages/confirm-runtime` fail due broad pre-existing engine/react typing baseline errors unrelated to this ticket.

### What I learned

- The existing fallback policy is appropriate; extraction was mostly a placement and API-boundary cleanup.

### What was tricky to build

- Preserving exact fallback semantics (`pending` latest still closes/removes local request) while generalizing into a reusable helper.

### What warrants a second pair of eyes

- Confirm expected UX for the `latest.status === pending` branch (current behavior keeps remove+close parity with existing code).

### What should be done in the future

- Consider adding a host-extensible fallback strategy for UIs that want to keep the window open on conflict.

### Code review instructions

- Start with `reconcileSubmitConflict409` helper implementation + test matrix.
- Verify both submit handlers in inventory call helper and only log on unhandled errors.

### Technical details

- Validation command:
  - `npx vitest run packages/confirm-runtime/src/runtime/reconcileSubmitConflict409.test.ts`
