---
Title: Implementation Plan - Script action_id wiring
Ticket: PC-04-SCRIPT-ACTION-ID-WIRING
Status: active
Topics:
    - backend
    - frontend
    - javascript
    - api
    - ux
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: proto/plz_confirm/v1/widgets.proto
      Note: ScriptView and ScriptEvent contracts for actions
    - Path: internal/server/script.go
      Note: View mapping/validation and event decoding already handle action_id
    - Path: agent-ui-system/client/src/components/WidgetRenderer.tsx
      Note: Script rendering path where action buttons and action events will be emitted
    - Path: agent-ui-system/client/src/services/websocket.ts
      Note: submitScriptEvent transport supports actionId and will be reused
    - Path: pkg/doc/js-script-api.md
      Note: API contract documentation for action events
ExternalSources: []
Summary: Plan for end-to-end action_id support from script UI actions to runtime branching.
LastUpdated: 2026-02-22T21:58:00-05:00
WhatFor: ""
WhenToUse: ""
---

# Implementation Plan - Script action_id wiring

## Executive Summary

`ScriptEvent.action_id` already exists in proto and is consumed by the runtime branch helper, but scripts currently lack a first-class way to declare UI actions that emit action events. This ticket adds a clear ScriptView action model and frontend wiring so scripts can declare secondary actions (skip, retry, apply-later) without overloading submit payloads.

## Problem Statement

Current behavior supports `submit` and `back` only from the main script widget path. Although `action_id` is parsed server-side, there is no standardized UI contract for script-defined action buttons. This limits branching ergonomics and forces awkward patterns in `event.data`.

## Proposed Solution

Add an additive action contract at script view level and wire it to runtime events.

1. Extend `ScriptView` with a repeated action descriptor (id, label, variant, optional confirm text).
2. Map and validate action descriptors in `internal/server/script.go`.
3. Render actions in `WidgetRenderer` consistently for single and section-based views.
4. On click, send `POST /api/requests/{id}/event` payload:
   - `type: "action"`
   - `stepId: current step`
   - `actionId: selected action id`
5. Keep back-button behavior intact and distinct from generic actions.
6. Document usage with `ctx.branch` patterns keyed by `actionId`.

## Design Decisions

### Decision 1: View-level actions, not widget-specific custom actions

Actions live on `ScriptView` so all interactive widget types can use the same mechanism.

Reasoning: one consistent contract and minimal duplication.

### Decision 2: Stable IDs required

Require non-empty, stable `id` values and treat labels as presentation-only.

Reasoning: branching must be deterministic and localization-safe.

### Decision 3: Preserve backward compatibility

No breaking changes to existing scripts; action arrays are optional.

Reasoning: current script flows should continue to work unchanged.

## Alternatives Considered

### Alternative A: Encode actions in each widget's input shape

Rejected because it fragments the contract and duplicates UI logic across widgets.

### Alternative B: Use only `event.type` variants (no actionId)

Rejected because multiple actions per step would require dynamic event type strings and weaker validation.

### Alternative C: Keep current model and rely on select/form for pseudo-actions

Rejected because it harms UX and makes intent less explicit.

## Implementation Plan

1. Extend protobuf schema with `ScriptAction` and `repeated ScriptAction actions` on `ScriptView`.
2. Regenerate Go and TypeScript protobuf artifacts.
3. Add server-side mapping/validation for action arrays (id/label required, optional style/confirm text enums if added).
4. Render action bar in `WidgetRenderer` for script views.
5. Emit `submitScriptEvent(..., { type: "action", actionId })` for action clicks.
6. Add frontend tests for action rendering and event payloads.
7. Add server integration tests for action event lifecycle and `ctx.branch` route selection via `actionId`.
8. Update JS API docs and add an example script in ticket `scripts/`.

## Open Questions

- Should actions support per-action confirmation dialogs in the first iteration?
- Should action buttons be allowed in display-only steps, or only when one interactive section is present?

## References

- `ttmp/2026/02/22/PC-01-ADD-JS-API--add-js-api-describe-extension/sources/local/plz-confirm-js.md`
- `internal/scriptengine/engine.go`
- `agent-ui-system/client/src/components/WidgetRenderer.tsx`
