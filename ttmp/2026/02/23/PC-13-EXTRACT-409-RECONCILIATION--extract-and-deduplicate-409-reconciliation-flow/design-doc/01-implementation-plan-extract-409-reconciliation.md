---
Title: 'Implementation Plan: Extract 409 Reconciliation'
Ticket: PC-13-EXTRACT-409-RECONCILIATION
Status: active
Topics:
    - architecture
    - frontend
    - javascript
    - ux
    - bug
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/apps/inventory/src/App.tsx
    - /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/api/confirmApiClient.ts
    - /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/state/confirmRuntimeSlice.ts
ExternalSources: []
Summary: Remove duplicate host submit conflict handling by centralizing 409 reconciliation in confirm-runtime.
LastUpdated: 2026-02-24T09:42:00-05:00
WhatFor: Reduce drift and ensure consistent stale-window cleanup semantics.
WhenToUse: Use when adjusting submit conflict behavior for confirm/script requests.
---

# Implementation Plan: Extract 409 Reconciliation

## Executive Summary

The inventory app currently has two near-identical `catch` branches for `ConfirmApiError(status=409)` in request submit handlers. This duplicates policy and increases drift risk.

The fix is to move conflict reconciliation into a shared helper inside `@hypercard/confirm-runtime`, then call it from both host submit paths.

## Problem Statement

### Current duplication

`apps/inventory/src/App.tsx` has duplicate logic for:

- submit response conflict handling, and
- submit script event conflict handling.

Both branches refetch latest request, complete-and-close when no longer pending, else remove-and-close stale local request.

### Risk

Bug fixes can land in one branch and not the other. New host apps may copy/paste a third variant.

## Proposed Solution

Add a helper (runtime-level export) that:

1. checks whether error is `ConfirmApiError` with status `409`;
2. refetches latest request;
3. if latest is non-pending, dispatches `completeRequestById` and closes by latest ID;
4. on refetch failure (or still-pending latest), falls back to local `removeRequest + closeWindow`;
5. returns boolean (`handled`) so callers can decide whether to log/propagate.

## API Sketch

```ts
await reconcileConflict409({
  requestId,
  error,
  apiClient,
  dispatch,
  closeRequestWindow,
});
```

Return:

- `true`: error was a 409 and reconciliation logic ran.
- `false`: not a 409 (caller handles normally).

## Design Decisions

1. Keep helper in `confirm-runtime` package (not app-specific).
2. Keep host-specific window-closing operation injected callback-style.
3. Do not suppress non-409 errors.

## Alternatives Considered

1. Keep duplicate branches in app.
Rejected: known drift already present and will worsen with additional hosts.

2. Push conflict handling into API client.
Rejected: API client should remain transport-focused and unaware of runtime state/windowing actions.

## Implementation Plan

1. Add helper module + tests in `packages/confirm-runtime`.
2. Export helper from package index.
3. Refactor `App.tsx` submit handlers to call helper.
4. Run tests/typechecks.
5. Update diary/changelog and commit.

## Open Questions

1. Should helper be expanded later to support custom fallback policy (e.g., toast + keep window open)?

## References

- Inspector recommendation in `PC-05` review and deep-dive Q&A.
