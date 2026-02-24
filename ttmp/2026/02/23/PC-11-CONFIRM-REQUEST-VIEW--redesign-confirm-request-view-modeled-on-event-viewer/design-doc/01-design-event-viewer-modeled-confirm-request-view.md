---
Title: 'Design: Event Viewer–modeled Confirm Request View'
Ticket: PC-11-CONFIRM-REQUEST-VIEW
Status: active
Topics:
    - architecture
    - frontend
    - javascript
    - ux
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../go-go-os/apps/inventory/src/App.tsx
      Note: Current ConfirmQueueWindow implementation and window routing for request views
    - Path: ../../../../../../../go-go-os/packages/engine/src/chat/debug/EventViewerWindow.tsx
      Note: Reference interaction model and visual structure to emulate
    - Path: ../../../../../../../go-go-os/packages/confirm-runtime/src/components/ConfirmRequestWindowHost.tsx
      Note: Existing request-detail renderer that queue/list view should integrate with
    - Path: ../../../../../../../go-go-os/packages/confirm-runtime/src/state/confirmRuntimeSlice.ts
      Note: Runtime queue/completion data model used by redesigned view
    - Path: ../../../../../../../go-go-os/packages/confirm-runtime/src/state/selectors.ts
      Note: Existing selectors and expected extensions for richer list diagnostics
Summary: Detailed UI/UX and architecture proposal for redesigning the confirm request queue/view to follow Event Viewer ergonomics while preserving confirm-runtime boundaries.
LastUpdated: 2026-02-24T21:08:00-05:00
WhatFor: Define implementation-ready design for a richer, debuggable, and operator-friendly confirm request view patterned after Event Viewer interaction principles.
WhenToUse: Use before implementing queue/request UX changes in inventory host and when evaluating reusable widget extraction opportunities.
---

# Design: Event Viewer–modeled Confirm Request View

## Executive Summary

Current confirm queue UX is intentionally minimal and functional: a count, a flat list of active requests, and an `Open` button. That was correct for initial integration speed, but now it is the weakest UX surface compared to the sophistication already available in the Event Viewer.

This design proposes a new confirm request view modeled on Event Viewer principles:

1. filter-first toolbar;
2. compact row summaries with quick triage signals;
3. expandable detail panels for metadata and payload previews;
4. explicit control states (follow stream, hold, pause updates, clear completed);
5. clear transition paths from queue rows to detailed request windows.

The redesign should remain host-thin and runtime-driven: `@hypercard/confirm-runtime` continues to own request semantics; app shell composes presentation and window orchestration.

## Problem Statement

The existing `ConfirmQueueWindow` in `apps/inventory/src/App.tsx` has three limitations:

1. **Low information density**
   - It shows title/type/id but no meaningful metadata (age, status badge, session, widget mode hints, script step context).
2. **No triage controls**
   - Operators cannot filter/sort/pin by state or type.
3. **No embedded observability**
   - There is no quick way to inspect request metadata/payload without opening each request window.

By contrast, Event Viewer already demonstrates an effective local pattern for high-volume real-time data:

- top toolbar controls;
- visibility toggles;
- expandable rows;
- copy/export affordances;
- explicit follow/hold behavior.

We should apply that pattern to confirm queue workflows so operators can process multiple simultaneous requests with less context-switching and less accidental misses.

## Design Goals

1. Preserve current runtime boundaries and avoid embedding backend logic into host UI.
2. Improve queue triage speed and operator confidence under bursty request flow.
3. Keep visual/interaction parity with existing debug windows (Event Viewer family).
4. Enable future extraction into reusable `engine` components where practical.
5. Remain keyboard-accessible and robust for long-running sessions.

## Non-Goals

1. Replacing `ConfirmRequestWindowHost` widget renderer in this ticket.
2. Introducing backend API changes for queue data.
3. Full theme overhaul of desktop shell.
4. Cross-session global moderation workflows.

## Proposed Solution

### 1. Introduce a new host component: `ConfirmRequestViewerWindow`

Replace current inline `ConfirmQueueWindow` with a component that mirrors Event Viewer structure:

- Toolbar (filters + controls)
- Scrollable request list
- Expandable rows
- Footer status/metrics

High-level layout:

```text
+--------------------------------------------------------------+
| Toolbar: [Type filters] [Status filters] [Search] [Hold/Follow] ...
+--------------------------------------------------------------+
| Row: timestamp | type badge | title | summary | state chevron |
|   expanded: metadata + payload summary + actions              |
| Row ...                                                       |
| Row ...                                                       |
+--------------------------------------------------------------+
| Showing X/Y active requests | paused/follow state            |
+--------------------------------------------------------------+
```

### 2. Data model adaptation in host layer (read-only projection)

Use runtime selectors to derive a view-model per request (no mutation in projector):

```ts
type ConfirmRequestListItem = {
  id: string;
  widgetType: string;
  title: string;
  sessionId: string;
  createdAt?: string;
  status?: 'pending' | 'completed' | 'timeout' | 'error' | 'unknown';
  ageMs?: number;
  summary: string;
  metadataPreview: string;
  inputPreview: string;
};
```

This keeps renderer simple and testable while preserving source-of-truth in runtime state.

### 3. Toolbar controls (Event Viewer inspired)

#### Filter chips

- Widget type chips: `confirm`, `select`, `form`, `table`, `upload`, `image`, `script`.
- Status chips: `pending`, `completed`, `timeout`, `error`.

#### Search input

Search by title/id/session/widget summary.

#### Update controls

- `Pause` / `Resume`: freeze list updates for investigation.
- `Hold` / `Follow`: equivalent concept to Event Viewer auto-scroll behavior.

#### Queue maintenance actions

- `Open Next` (oldest pending)
- `Clear Completed` (local completion panel only, not backend mutation)

### 4. Row model and expansion

Each row header should include:

- time/age;
- widget type badge;
- title;
- compact summary (e.g., script step or select/table cardinality hint);
- quick actions: `Open`, `Copy ID`.

Expanded panel should include:

- metadata block (remoteAddr/userAgent/session);
- normalized payload preview;
- script context (stepId/title/progress if present);
- actions:
  - `Open Request Window`
  - `Copy JSON`
  - `Copy cURL template`.

### 5. Action strategy

Primary action remains opening a dedicated request window (`confirm-request:<id>`). Queue viewer is triage/orchestration, not full interaction surface.

Secondary convenience actions avoid context-switch:

- copy ID;
- copy payload;
- quick open oldest/newest.

### 6. Optional split panel (Phase 2)

If initial row expansion is insufficient, introduce a two-pane layout:

- left: filterable list
- right: detail inspector for selected row

This can be deferred and should only be added if row expansion proves limiting.

## Design Decisions

1. **Model on Event Viewer interaction grammar, not visual clone.**
   - Keep controls and behavior familiar, but maintain confirm-domain semantics and labels.

2. **Do not embed request widget execution in queue list.**
   - Full interactions remain in dedicated request windows to avoid queue-view complexity explosion.

3. **Keep runtime source-of-truth unchanged.**
   - Queue viewer reads from runtime selectors and dispatches only window/navigation actions.

4. **Start in host app, then extract reusable pieces.**
   - Initial implementation can live in inventory app for speed, with extraction candidates identified after stabilization.

## Component-Level Architecture

### Existing

- `App.tsx` contains inline `ConfirmQueueWindow` (simple list).
- `ConfirmRequestWindowHost` handles per-request widget interaction.
- runtime slice tracks active/completed requests.

### Proposed

- `features/confirm/ConfirmRequestViewerWindow.tsx` (new)
- `features/confirm/confirmRequestViewModel.ts` (projection helpers)
- `features/confirm/confirmRequestFilters.ts` (filter/search helpers)
- keep `ConfirmRequestWindowHost` unchanged

Potential reusable extraction targets later:

- generic filter-toolbar primitives;
- generic expandable “log-row” list component.

## UX Behavior Specification

### Default state

- show pending requests sorted by newest first;
- follow mode enabled;
- all widget types visible.

### Paused state

- no auto-append visual reorder;
- new incoming count indicator shown in toolbar (`+N unseen`).

### Filter state

- chips persist while window open;
- optional reset-all button.

### Empty state

- if no requests: “No active requests” with hint action “Follow stream”.

### Error state

- if projection/parsing fails for a row, row remains visible with safe fallback summary and warning badge.

## Accessibility and Keyboard Model

1. Arrow keys navigate rows.
2. Enter toggles expansion.
3. `o` opens selected request.
4. `f` focuses search input.
5. `space` toggles pause/resume.

All actions should be available through buttons with visible labels and focus styles.

## Data/State Flow

```text
WS/HTTP -> confirm runtime state (activeById + activeOrder)
        -> selectors/projectors -> ConfirmRequestViewerWindow rows
        -> user action open(row) -> openWindow(confirm-request:<id>)
        -> ConfirmRequestWindowHost handles actual submit flow
```

No direct backend mutations originate from queue viewer except potential future utility actions (out of scope now).

## Performance Considerations

1. Project row model with memoization to avoid full recompute on every render.
2. Cap visible entries or virtualize if active queue can exceed hundreds.
3. Avoid deep JSON stringify on every render; compute payload preview lazily on expansion.
4. Keep derived time/age refresh throttled (e.g., every 1s) instead of per-frame.

## Test Plan

### Unit tests

1. Filter helpers by widget type/status/search.
2. Projection helper generates stable summary fields for each widget type.
3. Pause/follow state transitions.

### Component tests

1. Row expansion toggles and detail panel rendering.
2. `Open` action dispatches expected window payload.
3. Keyboard navigation and action shortcuts.

### Manual smoke

1. Burst-create mixed request types; verify triage speed and ordering.
2. Toggle pause, create new requests, resume, verify unseen indicator and catch-up.
3. Expand script requests and inspect step metadata summary.

## Rollout Plan

### Phase 1 (MVP redesign)

- New viewer component with filters/search/expand/open.
- Replace old `ConfirmQueueWindow` wiring in app.
- Add basic unit/component tests.

### Phase 2 (hardening)

- Add pause/follow and unseen counters.
- Add copy actions and payload preview safeguards.
- Improve keyboard navigation.

### Phase 3 (extraction)

- Evaluate extraction of reusable viewer primitives into engine package.

## Risks and Mitigations

1. **Risk:** UI complexity grows too fast.
   - Mitigation: maintain strict scope boundaries; queue viewer is triage, not submit surface.

2. **Risk:** frequent re-renders with large queues.
   - Mitigation: memoized projections, lazy expanded payload rendering, optional virtualization.

3. **Risk:** divergence from Event Viewer behavior semantics.
   - Mitigation: explicit checklist of shared interaction patterns (toolbar/filter/expand/follow).

## Implementation Task Breakdown

```markdown
- [ ] RV-01: Add ConfirmRequestViewerWindow component scaffold
- [ ] RV-02: Add request projection + filter helpers (unit-tested)
- [ ] RV-03: Implement toolbar chips/search/reset controls
- [ ] RV-04: Implement expandable row summaries and detail panels
- [ ] RV-05: Wire open/copy actions and keyboard shortcuts
- [ ] RV-06: Replace old ConfirmQueueWindow routing in App.tsx
- [ ] RV-07: Add component tests and manual smoke checklist doc
```

## Open Questions

1. Should completed requests appear in the same viewer by default, or behind a toggle?
2. Do we want “Open Next” policy to be oldest pending or highest-priority by widget type?
3. Should payload preview include redaction for potentially sensitive metadata fields?

## References

- [`App.tsx`](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/apps/inventory/src/App.tsx)
- [`EventViewerWindow.tsx`](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/chat/debug/EventViewerWindow.tsx)
- [`ConfirmRequestWindowHost.tsx`](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/components/ConfirmRequestWindowHost.tsx)
- [`confirmRuntimeSlice.ts`](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/state/confirmRuntimeSlice.ts)
