---
Title: Implementation Plan - renderNode DSL renderer
Ticket: PC-06-SCRIPT-RENDERNODE-DSL
Status: active
Topics:
    - frontend
    - backend
    - javascript
    - api
    - ux
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: agent-ui-system/client/src/components/WidgetRenderer.tsx
      Note: Current script rendering switch to be migrated toward node-based rendering
    - Path: internal/server/script.go
      Note: Script view mapping and validation path that will parse/validate node trees
    - Path: internal/scriptengine/engine.go
      Note: Runtime context helpers where ui DSL factories can be added
    - Path: proto/plz_confirm/v1/widgets.proto
      Note: ScriptView schema extension for node-tree contracts
    - Path: pkg/doc/js-script-api.md
      Note: Public API docs that must cover node schema and compatibility
    - Path: pkg/doc/js-script-development.md
      Note: Contributor docs for runtime and renderer internals
ExternalSources: []
Summary: Detailed migration and architecture plan for node-based script rendering.
LastUpdated: 2026-02-23T00:02:00Z
WhatFor: ""
WhenToUse: ""
---

# Implementation Plan - renderNode DSL renderer

## Executive Summary

The current script UI contract is widget-centric: one interactive widget plus optional display sections. This model works for simple flows but it is too rigid for richer layouts (context panels, grouped controls, reusable layout containers, and mixed interaction surfaces).

This ticket introduces a node-tree contract and a `renderNode` renderer in the frontend, while preserving backward compatibility with current script views. The design intentionally supports incremental rollout: existing scripts continue working unchanged, while new scripts can opt into node trees.

## Problem Statement

Current limits in the script rendering contract:

1. Layout rigidity
   - We can render one interactive widget and optional display sections.
   - We cannot express reusable layout primitives (row/column/stack/panel) in a general way.

2. Fragmented extension path
   - Every new UI idea tends to become a new widget type.
   - This inflates widget surface area and multiplies validation code.

3. Renderer complexity concentration
   - `WidgetRenderer` is a growing switch statement that mixes request orchestration with presentation-specific logic.

4. Incomplete composability
   - Shared wrappers (title bars, action bars, contextual hints) are hard to represent consistently across widget types.

## Goals and Non-Goals

### Goals

- Add a JSON node-tree schema for script views.
- Add a recursive `renderNode(node, context)` renderer in frontend.
- Keep current `widgetType/input/sections` contract fully functional during migration.
- Enforce strict server-side validation and depth/size limits for safety.
- Standardize action and submit dispatch from node interactions.

### Non-Goals

- No arbitrary React component injection from scripts.
- No client-side execution of script code.
- No visual redesign of existing widgets.
- No removal of legacy widget rendering in the first release.

## Proposed Solution

### 1. Extend ScriptView with optional node root

Add additive fields in `widgets.proto`:

- `ScriptNode root`
- `string node_schema_version`

Proposed node shape:

```protobuf
message ScriptNode {
  string type = 1;
  google.protobuf.Struct props = 2;
  repeated ScriptNode children = 3;
  optional string key = 4;
}
```

`ScriptView` compatibility rules:

- Legacy mode: `widget_type/input` (current behavior).
- Node mode: `root` set.
- During migration, if both are provided, `root` takes precedence only when feature flag enabled; otherwise legacy path is used.

### 2. Runtime DSL in scripts

Expose a minimal `ctx.ui` builder set that emits plain node objects. Example:

```javascript
ctx.ui.page({ title: "Review" }, [
  ctx.ui.markdown({ content: "## Diff summary" }),
  ctx.ui.confirm({ id: "ok", title: "Approve?" })
])
```

The DSL is sugar only. Scripts may return raw node objects as long as they match schema.

### 3. Server-side node validation

In `internal/server/script.go`, add `validateScriptNode` with constraints:

- Required `type` string.
- `children` must be array when present.
- Max depth (default 16).
- Max total nodes (default 400).
- Type-specific required props for interactive nodes.
- Disallow unknown top-level unsafe node types unless explicitly enabled.

Validation should fail request with `400` if shape is invalid.

### 4. Frontend `renderNode`

Split responsibilities:

- `WidgetRenderer`: request lifecycle and event submission orchestration.
- `ScriptNodeRenderer`: recursive renderer for node trees.

Pseudo-API:

```ts
function renderNode(node: ScriptNode, ctx: RenderContext): React.ReactNode
```

`RenderContext` carries:

- request id and step id
- loading state
- submit handler
- action handler
- back handler

### 5. Node registry and adapters

Implement a registry mapping node type -> React adapter.

Node classes:

- Layout: `page`, `stack`, `row`, `panel`, `divider`.
- Content: `markdown`, `text`, `callout`, `code`, `diff`.
- Interactive adapters: `confirm`, `select`, `form`, `table`, `upload`, `image`, `grid`, `rating`.

Interactive adapters reuse existing widget components to avoid behavior drift.

### 6. Event semantics

Keep existing event envelope:

- submit: `{ type: "submit", stepId, data }`
- action: `{ type: "action", stepId, actionId }`
- back: `{ type: "back", stepId }`

Node adapters normalize output payloads to match current script expectations.

## Design Decisions

### Decision 1: Add node mode rather than replacing widget mode

Reasoning: low-risk migration and immediate compatibility.

### Decision 2: Server validates node trees, not frontend only

Reasoning: avoid malformed payloads reaching clients and preserve deterministic API behavior.

### Decision 3: Registry-based rendering instead of giant switch

Reasoning: modular extension path, easier tests, and clearer ownership per node type.

### Decision 4: Strict limits on depth and node count

Reasoning: prevent pathological rendering payloads and accidental denial-of-service in browser.

### Decision 5: Reuse existing widget components for interactions

Reasoning: preserve proven UX and validation behavior while only changing composition model.

## Implications and Tradeoffs

### API/Schema implications

- Protobuf changes require Go and TS codegen updates.
- Docs must define node schema stability guarantees and versioning policy.

### Frontend implications

- Renderer architecture becomes layered (orchestration + recursive rendering).
- Requires careful key strategy to preserve local widget state where expected.

### Backend implications

- Additional validation code paths in `script.go`.
- More detailed error reporting needed for nested node paths (`root.children[2].props.title`).

### Operational implications

- Payload sizes may grow for rich UIs.
- Need metrics for node count/depth and render errors.

## Alternatives Considered

### Alternative A: Keep extending `sections`

Rejected: sections become an ad hoc layout language without clear schema or composability semantics.

### Alternative B: Put raw HTML in display nodes for all rich layouts

Rejected: unsafe and not expressive for interactive composition.

### Alternative C: Render everything with a separate micro-frontend

Rejected: too heavy operationally and breaks current integration model.

## Migration Plan

### Phase 0: Foundation

- Add proto fields and generated types.
- Add server validators and feature flag `scriptNodeMode`.

### Phase 1: Frontend renderer

- Add `ScriptNodeRenderer` with core node registry.
- Keep legacy path as default fallback.

### Phase 2: Script runtime helpers

- Add `ctx.ui` builder helpers in runtime prelude.
- Add examples and smoke scripts.

### Phase 3: Incremental adoption

- Convert a few internal scripts to node mode.
- Track telemetry and error rates.

### Phase 4: Stabilization

- Expand node types only after stability metrics are healthy.

## Testing Strategy

- Unit tests for node validation (backend).
- Frontend tests for recursive rendering and action dispatch.
- Snapshot-like tests for stable tree outputs.
- Integration tests for create/event/update flows in node mode.
- Stress tests for depth/node count limits.

## Documentation Plan

- `pkg/doc/js-script-api.md`: node schema, examples, migration guidance.
- `pkg/doc/js-script-development.md`: renderer architecture and debugging notes.
- Ticket-local scripts for manual verification.

## Open Questions

- Should node schema include explicit style tokens now, or defer until theme system work?
- Should mixed mode (`root` plus legacy fields) be hard error after migration period?
- Which node types are required for v1 versus v2?

## References

- `ttmp/2026/02/22/PC-01-ADD-JS-API--add-js-api-describe-extension/sources/local/plz-confirm-js.md`
- `agent-ui-system/client/src/components/WidgetRenderer.tsx`
- `internal/server/script.go`
