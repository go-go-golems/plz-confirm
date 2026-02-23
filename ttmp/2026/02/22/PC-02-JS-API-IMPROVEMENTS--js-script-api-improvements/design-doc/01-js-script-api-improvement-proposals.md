---
Title: JS Script API Improvement Proposals
Ticket: PC-02-JS-API-IMPROVEMENTS
Status: active
Topics:
    - backend
    - frontend
    - api
    - javascript
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/doc/js-script-api.md
      Note: Current API reference that would be updated as proposals are implemented
    - Path: pkg/doc/js-script-development.md
      Note: Contributor guide with codebase map and runtime internals
    - Path: internal/scriptengine/engine.go
      Note: Runtime contract implementation — most proposals require changes here
    - Path: internal/server/script.go
      Note: Server event lifecycle — several proposals add new event types or view fields
    - Path: agent-ui-system/client/src/components/WidgetRenderer.tsx
      Note: Frontend renderer — composite views, progress, back button all land here
    - Path: proto/plz_confirm/v1/widgets.proto
      Note: Proto schema — new widget types, view fields, and event types defined here
    - Path: agent-ui-system/client/src/services/websocket.ts
      Note: WebSocket client — sidebar display and event handling improvements
ExternalSources: []
Summary: "Comprehensive design proposals for extending the JS script API with new widget types, UX primitives, and flow control features."
LastUpdated: 2026-02-22T20:11:17.923026047-05:00
WhatFor: ""
WhenToUse: ""
---

# JS Script API Improvement Proposals

## Executive Summary

The current JS script API provides a working multi-step interaction model — scripts export `describe/init/view/update`, the server runs them in a sandbox, and the browser renders standard widgets. But real-world usage reveals gaps in three areas:

1. **Widget expressiveness** — spatial/visual interactions (like a game board or annotated image) can't be expressed with the current widget set, and feedback collection lacks common primitives like ratings and progress indicators.
2. **Flow control** — scripts must manually manage all state transitions, back-navigation, and branching logic, leading to boilerplate that obscures intent.
3. **UI integration** — script requests display poorly in the sidebar (showing as "UNKNOWN_REQUEST"), views can't combine read-only context with interactive widgets, and there's no way to show transient feedback to users.

This document proposes 15 improvements organized into four categories. Each proposal includes motivation, a concrete API shape, implementation scope, and priority. The proposals are designed to be additive — they extend the existing contract without breaking current scripts.

## Problem Statement

The script API was built as a minimal viable contract: four functions, a state machine, and existing widgets as the rendering layer. This was the right starting point, but it creates friction in two representative use cases:

**Interactive applications (e.g., tic-tac-toe).** A game needs a spatial grid, visual feedback on moves, and delayed responses (the computer "thinking"). Currently, a tic-tac-toe game must represent the board as a `select` dropdown with labels like "Top-Left" — functional but deeply unintuitive. There's no way to show the board state visually, animate the computer's response, or render a clickable grid.

**Structured feedback collection (e.g., multi-question surveys).** A feedback flow needs progress indicators, back-navigation, optional questions, rating scales, and a review/summary step. Currently, scripts must implement all of this as manual state management in `update`, with no visual progress, no back button, and rating questions shoehorned into `select` widgets.

Beyond these use cases, there's a baseline UX problem: script requests show up in the sidebar as "UNKNOWN_REQUEST" because the frontend doesn't know how to display script metadata (title, step, status) in the request list.

## Current Contract Shapes (for Reference)

These are the exact proto and runtime shapes that proposals extend. All proposals are designed to be backward-compatible additions to these structures.

**ScriptView (proto):**
```protobuf
message ScriptView {
  string widget_type = 1;
  google.protobuf.Struct input = 2;
  optional string step_id = 3;
  optional string title = 4;
  optional string description = 5;
}
```

**ScriptEvent (proto):**
```protobuf
message ScriptEvent {
  string type = 1;
  optional string step_id = 2;
  optional string action_id = 3;
  google.protobuf.Struct data = 4;
}
```

**view() return (runtime):** Must return a `map[string]any` with at least `widgetType` and `input`. Currently validated by `expectMap()` in `engine.go`.

**WidgetRenderer (frontend):** Reads `scriptView.widgetType`, switches on it, and renders the matching dialog component. Currently does not use `title`, `description`, or `stepId` from `scriptView`.

---

## Proposal 1: Fix Script Request Sidebar Display

**Priority: P0 — ship immediately**

### Motivation

Script requests currently appear in the sidebar as:

```
REQUEST_HISTORY
script  8:03 PM
UNKNOWN_REQUEST
COMPLETED
```

This happens because the sidebar renders based on `request.type` and widget-specific input fields (e.g., `confirmInput.title`). Script requests have `type: "script"` but the sidebar code doesn't know to look at `scriptInput.title` or `scriptDescribe.name` for display.

### Proposed Solution

The fix is purely frontend — no contract changes needed.

**Sidebar display logic in the request list component:**

1. For script requests, use `scriptInput.title` as the primary display text (this is always provided).
2. Show the current `scriptView.widgetType` as a secondary badge (e.g., "confirm", "select") so the user knows what kind of interaction is active.
3. For completed script requests, show `scriptDescribe.name` and `scriptDescribe.version` as metadata.
4. Use a distinct icon or color for script-type requests to differentiate them from one-shot widgets.

**What changes:**
- `agent-ui-system/client/src/components/` — sidebar/request list rendering logic.
- No proto, server, or engine changes.

### Implementation Scope

Small — likely 1-2 hours. Pure frontend change, no migration concerns.

---

## Proposal 2: Grid Widget Type

**Priority: P1 — high value, moderate effort**

### Motivation

Spatial interactions — game boards, seating charts, calendar pickers, pixel editors — can't be expressed with the current widget set. The tic-tac-toe example is the clearest illustration: representing a 3x3 board as a dropdown list of position names is technically functional but fundamentally wrong for the interaction pattern.

### Proposed API

**`view()` return:**

```javascript
return {
  widgetType: "grid",
  input: {
    title: "Your turn (X)",
    rows: 3,
    cols: 3,
    cells: [
      { value: "X", style: "filled" },
      { value: "",  style: "empty"  },
      { value: "O", style: "filled" },
      // ... 9 cells total
    ],
    // Optional: disable specific cells (already taken)
    // Optional: highlight specific cells (winning line)
    cellSize: "medium"  // "small" | "medium" | "large"
  }
};
```

Each cell in the `cells` array can have:

| Field | Type | Description |
|---|---|---|
| `value` | string | Display text or emoji for the cell |
| `style` | string | Visual style: `"empty"`, `"filled"`, `"highlighted"`, `"disabled"` |
| `disabled` | boolean | If true, cell is not clickable |
| `label` | string? | Optional tooltip or aria-label |
| `color` | string? | Optional CSS color override |

**`event.data` on cell click:**

```json
{ "row": 1, "col": 2, "cellIndex": 5 }
```

Both row/col (zero-indexed) and flat cellIndex are provided so scripts can use whichever addressing they prefer.

### Design Decisions

- Cells are a flat array (not nested arrays) for compatibility with `google.protobuf.Struct`, which doesn't have great support for nested repeated types. Row/column layout is computed from `rows` and `cols`.
- `style` uses semantic names rather than raw CSS to keep the sandbox boundary clean.
- Grid sizes larger than ~10x10 should still work but the UI may need scroll behavior; initial implementation can cap at a reasonable maximum.

### Implementation Scope

Medium:
- **Proto:** Add `GridInput` message to `widgets.proto`.
- **Frontend:** New `GridDialog` component under `components/widgets/`. Add `case "grid"` to `WidgetRenderer.tsx`.
- **Engine/server:** No changes needed — the grid is just another `widgetType` string and `input` shape.
- **Tests:** Frontend component test, one engine integration test with a grid script.

---

## Proposal 3: Composite / Multi-Widget Views

**Priority: P1 — high value, moderate effort**

### Motivation

A single `view()` call currently produces a single interactive widget. But many flows need to show context alongside an interactive element:

- A code diff above a confirm dialog ("approve this change?")
- A board state display above action buttons
- Previous answers displayed as a summary while asking the next question
- An image with a text form beside it

Currently, scripts must cram all context into the `title` or `message` string fields, losing formatting and interactivity.

### Proposed API

Allow `view()` to return a `sections` array instead of (or in addition to) the flat `widgetType`/`input` pair:

```javascript
return {
  sections: [
    {
      widgetType: "display",
      input: {
        content: "## Current Board\n\n```\n X | O | · \n---------\n · | X | · \n---------\n · | · | O \n```",
        format: "markdown"
      }
    },
    {
      widgetType: "select",
      input: { title: "Your move", options: ["Top-Center", "Mid-Left", "Mid-Right", "Bot-Left", "Bot-Center"] }
    }
  ],
  stepId: "move"
};
```

Rules:
- Exactly **one** section must be interactive (a widget that produces events). The rest must be `display` (read-only).
- The interactive widget's submit behavior works the same as today.
- If `sections` is absent, the current flat `widgetType`/`input` shape works as before (backward compatible).

### The `display` Widget Type

A new read-only widget type that renders formatted content:

```javascript
{
  widgetType: "display",
  input: {
    content: "Some **markdown** text with `code`",
    format: "markdown"  // "markdown" | "text" | "html"
  }
}
```

This also addresses the "rich text in messages" need — instead of adding markdown support to every widget's `message` field, you compose a `display` section before the interactive widget.

### Design Decisions

- **Single interactive widget per view** keeps the event model simple — there's exactly one submit target per step, and `event.data` has a single unambiguous shape.
- **`display` as a widget type** (not a special-case property) keeps the sections array homogeneous and lets us add other read-only widget types later (e.g., `chart`, `diff`).
- The `sections` field on `ScriptView` would be a `repeated ScriptViewSection` in proto, where each section has its own `widget_type` and `input`.

### Implementation Scope

Medium-large:
- **Proto:** Add `repeated ScriptViewSection sections` to `ScriptView`. Add `DisplayInput` message.
- **Frontend:** Update `WidgetRenderer.tsx` to handle sections layout. New `DisplayWidget` component. Layout stacks sections vertically.
- **Engine:** Update view validation to accept either flat `widgetType`/`input` or `sections` array.
- **Tests:** Multiple — section rendering, mixed display+interactive, backward compat.

---

## Proposal 4: Progress Indicators

**Priority: P1 — high value, small effort**

### Motivation

In a multi-step feedback flow (8 questions, say), the user has no idea how far along they are. This creates anxiety and dropout. Progress indicators are table stakes for survey/wizard UX.

### Proposed API

Add optional progress fields to the `view()` return:

```javascript
return {
  widgetType: "select",
  input: { title: "Rate the documentation", options: ["1", "2", "3", "4", "5"] },
  progress: {
    current: 3,
    total: 8,
    label: "Question 3 of 8"  // optional, auto-generated if absent
  }
};
```

The UI renders a progress bar or step indicator above the widget.

### Design Decisions

- `progress` is optional — scripts without it work exactly as before.
- Using `current`/`total` (not a 0-1 float) because it maps to natural language ("step 3 of 8") and avoids rounding confusion.
- The optional `label` lets scripts provide custom text like "Almost done!" for the last step.

### Implementation Scope

Small:
- **Proto:** Add optional `ScriptProgress progress` to `ScriptView` (with `int32 current`, `int32 total`, `optional string label`).
- **Frontend:** Render progress bar in `WidgetRenderer.tsx` when `scriptView.progress` is present.
- **Engine:** No validation changes needed — `progress` is just another field in the view map.

---

## Proposal 5: Back / Undo Navigation

**Priority: P1 — high value, moderate effort**

### Motivation

Feedback flows almost always need a "go back" button. Users make mistakes, change their minds, or want to review a previous answer before continuing. Currently there's no way to navigate backward — the flow is strictly forward-only.

### Proposed API

Two complementary pieces:

**1. Script declares back-support in `view()`:**

```javascript
return {
  widgetType: "form",
  input: { title: "Your details", schema: { ... } },
  allowBack: true  // shows a "Back" button in the UI
};
```

**2. The UI sends a `back` event:**

```json
{ "type": "back", "stepId": "details" }
```

**3. The script handles it in `update()`:**

```javascript
update: function (state, event) {
  if (event.type === "back") {
    state.step = state.previousStep;
    return state;
  }
  // normal forward handling...
}
```

### Design Decisions

- **Script-managed history** rather than engine-managed. The engine could maintain a state history stack and handle `back` events automatically, but this creates issues: scripts may have side effects in `update` that shouldn't be replayed, and automatic undo semantics get complicated with conditional branching. Letting the script manage its own back-navigation keeps it simple and predictable.
- **`allowBack: true`** in the view return controls UI rendering — the Back button only appears when the script says it should (e.g., not on the first step).
- **`event.type: "back"`** is a new event type alongside `"submit"`. The script must handle it or the engine ignores it with a no-op (return current state unchanged). This is safer than failing.

### Alternatives Considered

- **Engine-managed state stack:** The server could automatically store previous states and restore on `back`. Simpler for script authors but less flexible — doesn't handle computed-state or branching well. Could be offered as an opt-in via `describe.capabilities: ["auto-back"]` in the future.

### Implementation Scope

Small-medium:
- **Proto:** Add `optional bool allow_back` to `ScriptView`.
- **Frontend:** Render Back button when `scriptView.allowBack` is true. Send `{ type: "back" }` event on click.
- **Engine:** No changes — `back` is just another event type passed to `update()`.
- **Server:** No changes — event handling is type-agnostic.

---

## Proposal 6: Rating / Likert Scale Widget

**Priority: P2 — medium value, small effort**

### Motivation

"Rate this 1-5" is the most common feedback primitive, but currently requires either a `select` (which looks like a dropdown, not a rating) or a `form` with a number field (awkward). A dedicated widget would be more natural and visually clear.

### Proposed API

```javascript
return {
  widgetType: "rating",
  input: {
    title: "How would you rate the documentation?",
    scale: 5,              // number of points (default 5)
    labels: {
      low: "Poor",         // label for leftmost point
      high: "Excellent"    // label for rightmost point
    },
    style: "stars"         // "stars" | "numbers" | "emoji" | "slider"
  }
};
```

**`event.data` on submit:**

```json
{ "value": 4, "comment": "Good but could use more examples" }
```

### Implementation Scope

Small:
- **Proto:** Add `RatingInput` / `RatingOutput` messages.
- **Frontend:** New `RatingDialog` component. Star/number/slider variants.
- **Engine/server:** No changes.

---

## Proposal 7: Prefilled Defaults and Initial Values

**Priority: P2 — medium value, small effort**

### Motivation

Review-and-edit workflows are common in feedback collection: "here are the settings we detected, adjust if needed" or "here's your previous answer, update it." Currently, forms start empty and selects have no pre-selection. There's no way to seed widgets with initial values.

### Proposed API

Add a `defaults` field to widget inputs:

```javascript
// Pre-select an option
return {
  widgetType: "select",
  input: {
    title: "Deployment target",
    options: ["staging", "prod", "dev"],
    defaults: { selectedSingle: "staging" }
  }
};

// Pre-fill form fields
return {
  widgetType: "form",
  input: {
    title: "Review config",
    schema: { properties: { name: { type: "string" }, port: { type: "number" } } },
    defaults: { name: "api-server", port: 8080 }
  }
};
```

### Design Decisions

- `defaults` is placed inside `input` rather than as a sibling to it because defaults are widget-specific — the shape depends on the widget type.
- For `confirm`, a default doesn't make much sense (you don't pre-select yes/no), so it would be ignored.
- For `table`, `defaults.selectedSingle` or `defaults.selectedMulti` would pre-highlight rows.

### Implementation Scope

Small — mainly frontend widget components need to read `input.defaults` and set initial state. No engine or server changes.

---

## Proposal 8: Skip / Optional Steps

**Priority: P2 — medium value, small effort**

### Motivation

Not every question in a feedback flow is mandatory. Currently, the user must submit something (even a blank form) to advance. A "Skip" affordance lets optional questions be optional.

### Proposed API

```javascript
return {
  widgetType: "form",
  input: { title: "Any additional comments?", schema: { ... } },
  skippable: true,
  skipLabel: "Skip this question"  // optional, defaults to "Skip"
};
```

When the user clicks Skip, the UI sends:

```json
{ "type": "skip", "stepId": "comments" }
```

The script handles it in `update`:

```javascript
if (event.type === "skip") {
  state.step = "next";
  return state;
}
```

### Implementation Scope

Small:
- **Proto:** Add `optional bool skippable` and `optional string skip_label` to `ScriptView`.
- **Frontend:** Render Skip button when `skippable` is true. Send `{ type: "skip" }` event.
- **Engine/server:** No changes.

---

## Proposal 9: Toast / Flash Messages

**Priority: P2 — medium value, small effort**

### Motivation

After an action, scripts sometimes want to show brief feedback before the next step: "Saved!", "Computer is thinking...", "O takes center." Currently the only way to communicate between steps is by changing the widget title, which is jarring (a new widget flashes in).

### Proposed API

Add an optional `toast` field to the `view()` return:

```javascript
return {
  widgetType: "select",
  input: { title: "Your move", options: ["..."] },
  toast: {
    message: "Computer played O at center",
    duration: 2000,  // ms, default 3000
    style: "info"    // "info" | "success" | "warning" | "error"
  }
};
```

The toast appears briefly when the view transitions, then fades away. It doesn't block interaction.

### Design Decisions

- Toasts are part of the view, not a separate server event, so they're guaranteed to appear at the right time (when the new view renders).
- Duration is a hint to the frontend, not a hard contract — the UI may adjust based on message length or animation preferences.

### Implementation Scope

Small:
- **Proto:** Add optional `ScriptToast toast` to `ScriptView`.
- **Frontend:** Toast/snackbar component triggered on view transition.
- **Engine:** No changes.

---

## Proposal 10: Delayed / Timed Follow-Up Events

**Priority: P2 — medium value, moderate effort**

### Motivation

In the tic-tac-toe example, the computer's move happens synchronously inside `update` — the user never sees the board after their own move. The UI jumps from "user's board" to "board with computer's response" instantly. A real game would show the user's move, pause briefly, then show the computer's move.

More broadly, any flow that has a "processing" or "thinking" step needs a way to trigger a follow-up without user interaction.

### Proposed API

Allow `update()` to return a delayed follow-up event:

```javascript
update: function (state, event) {
  if (event.type === "submit") {
    state.board[idx] = "X";
    state.phase = "computer-thinking";
    return {
      state: state,
      delayedEvent: {
        type: "timer",
        data: {},
        delayMs: 800  // server calls update again after 800ms
      }
    };
  }
  if (event.type === "timer") {
    // Computer makes its move
    state.board[computerMove(state.board)] = "O";
    state.phase = "player-turn";
    return state;
  }
}
```

### Design Decisions

- The delayed event is server-managed — the server sets a timer and calls `update` again with the synthetic event. This keeps the sandbox simple (no setTimeout in the VM).
- The non-terminal return shape changes from "plain state object" to optionally `{ state, delayedEvent }`. The engine detects the shape and separates the two.
- Maximum delay should be capped (e.g., 30 seconds) to prevent indefinite resource holding.
- During the delay, the view from the current state is shown — the script can set a "thinking" view with a spinner or animation.

### Alternatives Considered

- **Exposing `setTimeout` in the VM:** Would work but breaks the "each call is a fresh VM" isolation model and opens resource management concerns (what if the script sets 1000 timers?).
- **Client-side timers:** The frontend could send a timer event after a delay. Simpler to implement but puts timer responsibility in the wrong layer and doesn't work for headless/API consumers.

### Implementation Scope

Medium:
- **Engine:** Detect `{ state, delayedEvent }` return shape from `update`. Return both to server.
- **Server:** After processing a delayed-event update, schedule a goroutine to sleep and call `update` again with the synthetic event.
- **Proto:** No changes needed — synthetic events use the existing `ScriptEvent` shape.

---

## Proposal 11: Declarative Branching

**Priority: P3 — nice to have, moderate effort**

### Motivation

Most multi-step scripts follow a pattern in `update`:

```javascript
if (state.step === "confirm") {
  if (event.data.approved) return { ...state, step: "details" };
  return { ...state, step: "reason" };
}
if (state.step === "details") { ... }
if (state.step === "reason") { ... }
```

This is boilerplate. For simple flows, a declarative routing table would be clearer.

### Proposed API

Allow `view()` to return optional routing hints:

```javascript
return {
  widgetType: "confirm",
  input: { title: "Continue?", approveText: "Yes", rejectText: "No" },
  stepId: "confirm",
  routes: {
    "approved": "details",    // if event.data.approved is truthy, go to step "details"
    "rejected": "reason",     // if event.data.approved is falsy, go to step "reason"
    "default": "fallback"     // catch-all
  }
};
```

When `routes` is present and `update` is not exported (or returns `undefined`), the engine applies the routing automatically by setting `state.step` to the matched route and calling `view` again.

### Design Decisions

- This is opt-in and complementary — scripts that export `update` can ignore `routes` entirely.
- Route matching would need widget-type-specific logic (what "approved" means for confirm vs select vs form). This is the main complexity.
- For scripts that use `routes`, `update` becomes optional, which is a contract change (currently all four exports are required).

### Implementation Scope

Medium-large — the route matching logic is non-trivial and needs to handle each widget type's output shape. Probably best deferred until the simpler proposals are shipped and we have more real-world scripts to validate the routing model.

---

## Proposal 12: Prefilled State History for Back Navigation

**Priority: P3 — nice to have, moderate effort**

### Motivation

Proposal 5 gives scripts manual back-navigation, but the script has to maintain its own history stack. For simple linear flows, the engine could manage this automatically.

### Proposed API

Scripts opt in via `describe`:

```javascript
describe: function () {
  return {
    name: "feedback-survey",
    version: "1.0.0",
    capabilities: ["auto-back"]
  };
}
```

When `auto-back` is declared:
- The engine stores a stack of `(state, view)` snapshots after each `update`.
- When a `back` event arrives, the engine pops the stack and restores the previous state/view without calling `update`.
- The script's `update` function never sees `back` events.

### Design Decisions

- Stored server-side (in the request's `scriptState`), not in the engine. The stack is part of the persisted state.
- Stack depth should be bounded (e.g., max 50 entries) to prevent memory issues.
- The script can still override by handling `back` events in `update` explicitly — in that case, `auto-back` is ignored for that event.

### Implementation Scope

Medium — requires server-side state stack management and integration with `PatchScript`.

---

## Proposal 13: Summary / Review Widget Type

**Priority: P3 — nice to have, small effort**

### Motivation

At the end of a multi-step flow, you typically want to show "here's everything you said, confirm or go back." Building this from a `table` or `confirm` message is awkward.

### Proposed API

```javascript
return {
  widgetType: "summary",
  input: {
    title: "Review your answers",
    entries: [
      { label: "Name", value: "Alice" },
      { label: "Rating", value: "4/5" },
      { label: "Comments", value: "Great docs, needs more examples" }
    ],
    confirmText: "Submit",
    editText: "Go back and edit"
  }
};
```

**`event.data` on submit:**

```json
{ "confirmed": true }
// or
{ "confirmed": false }  // user wants to edit
```

### Implementation Scope

Small — similar to a confirm dialog but with a key-value list display. Frontend component + proto messages.

---

## Proposal 14: Seeded Randomness via `ctx`

**Priority: P3 — nice to have, tiny effort**

### Motivation

Each script call runs in a fresh VM, so `Math.random()` isn't seeded consistently. For games or randomized surveys, scripts need a way to generate pseudo-random numbers that vary per request but are reproducible within a request's lifecycle.

### Proposed API

Add `ctx.seed` — a random number between 0 and 1, generated once per request and stable across all calls for that request:

```javascript
init: function (ctx) {
  // Use seed to shuffle question order
  var questions = shuffle(allQuestions, ctx.seed);
  return { questions: questions, current: 0 };
}
```

### Implementation Scope

Tiny:
- **Engine:** Generate a random float on request creation, store it in `scriptState`, pass it as `ctx.seed`.
- No proto changes — it's just another field in the ctx map.

---

## Proposal 15: Rich Select Options

**Priority: P3 — nice to have, small effort**

### Motivation

Select options are currently flat strings. For many use cases, options need descriptions, icons, or visual state indicators:

- Server list: "prod-us-east (healthy)" vs "staging-eu (degraded)"
- Permission levels with explanations
- Options with emoji or status indicators

### Proposed API

Allow options to be objects instead of strings:

```javascript
return {
  widgetType: "select",
  input: {
    title: "Select server",
    options: [
      { value: "prod-us", label: "Production US", description: "3 instances, healthy", badge: "healthy" },
      { value: "staging-eu", label: "Staging EU", description: "1 instance, degraded", badge: "warning" }
    ]
  }
};
```

`event.data.selectedSingle` would contain the `value` field (not the full object), keeping the output shape simple.

### Design Decisions

- Backward compatible — if `options` contains strings, they work as before. If they're objects, the frontend renders the richer display.
- `badge` uses semantic names (like grid `style`) rather than raw CSS.

### Implementation Scope

Small — frontend `SelectDialog` needs to handle object options. Proto stays the same (Struct handles both shapes).

---

## Priority Summary

| Priority | Proposal | Effort | Category |
|---|---|---|---|
| **P0** | 1. Fix sidebar display | Small | UI fix |
| **P1** | 2. Grid widget | Medium | New widget |
| **P1** | 3. Composite views | Medium-large | View model |
| **P1** | 4. Progress indicators | Small | UX |
| **P1** | 5. Back / undo | Small-medium | Flow control |
| **P2** | 6. Rating widget | Small | New widget |
| **P2** | 7. Prefilled defaults | Small | UX |
| **P2** | 8. Skip / optional | Small | Flow control |
| **P2** | 9. Toast messages | Small | UX |
| **P2** | 10. Delayed events | Medium | Flow control |
| **P3** | 11. Declarative branching | Medium-large | Flow control |
| **P3** | 12. Auto-back history | Medium | Flow control |
| **P3** | 13. Summary widget | Small | New widget |
| **P3** | 14. Seeded randomness | Tiny | Runtime |
| **P3** | 15. Rich select options | Small | UX |

## Recommended Implementation Order

**Phase 1 — Immediate (fixes + quick wins):**
1. Fix sidebar display (P0)
4. Progress indicators (P1, small)
5. Back / undo (P1, small-medium)
8. Skip / optional (P2, small)

**Phase 2 — New capabilities:**
2. Grid widget (P1, medium)
3. Composite views (P1, medium-large)
6. Rating widget (P2, small)
9. Toast messages (P2, small)

**Phase 3 — Refinements:**
7. Prefilled defaults (P2, small)
10. Delayed events (P2, medium)
15. Rich select options (P3, small)
14. Seeded randomness (P3, tiny)
13. Summary widget (P3, small)

**Phase 4 — Advanced (validate need first):**
11. Declarative branching (P3, medium-large)
12. Auto-back history (P3, medium)

## Open Questions

- **Composite views layout:** Should sections stack vertically only, or should we support side-by-side layouts (e.g., image left, form right)? Vertical-only is simpler and works on mobile. Side-by-side is better for annotation-style workflows.
- **Grid widget cell content:** Should cells support icons/images or only text? Text-only covers tic-tac-toe and most board games. Images would support richer use cases but increase component complexity.
- **Delayed event security:** Should there be a hard cap on delay duration? Unbounded delays could tie up server resources. A 30-second max seems reasonable but may need adjustment.
- **Rich select backward compatibility:** If a script sends object options to a client that only understands string options, what happens? The proto Struct handles both, but older frontends would render `[object Object]`. Need a migration strategy or version negotiation.

## References

- `js-script-api` — current API reference and contract documentation
- `js-script-development` — codebase map and runtime internals
- `adding-widgets` — guide for implementing new widget types
- PC-01-ADD-JS-API ticket — original implementation diary and design plan
