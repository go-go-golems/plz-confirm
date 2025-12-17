---
Title: 'Root-cause analysis: missing context rendering'
Ticket: 002-FIX-FORM-DISPLAY
Status: active
Topics:
    - plz-confirm
    - frontend
    - bug
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Schema is transmitted end-to-end; missing context is due to FormDialog.tsx not rendering schema/field title/description and labeling inputs by property key."
LastUpdated: 2025-12-17T16:23:02.692446294-05:00
---

# Root-cause analysis — missing context rendering in form widget

## TL;DR

The schema is **successfully transmitted** from `plz-confirm` CLI → Go server → WebSocket → React app. The missing “context” is caused by the React component `FormDialog.tsx` rendering only:
- the dialog title (`input.title`), and
- raw `schema.properties` keys as field labels,

while ignoring JSON Schema context fields like:
- schema-level `title` / `description`,
- per-property `title` / `description`,
- and related usability fields (`default`, `examples`, `enum`, etc.).

## What we mean by “context”

For quiz-style schemas, the human-facing content usually lives in JSON Schema keywords:
- **Schema-level**: `title`, `description`
- **Property-level**: `title`, `description`, `default`, `examples`, `enum`

In the reported case, the UI shows identifiers like `q1_total_files` instead of the actual question text / instructions.

## End-to-end data flow (confirmed)

### CLI: reads JSON Schema and sends it verbatim
- `plz-confirm/internal/cli/form.go`
  - decodes `--schema @file.json` into `var schema any`
  - sends `Input: FormInput{ Title: settings.Title, Schema: schema }`

### HTTP client: posts JSON to `/api/requests`
- `plz-confirm/internal/client/client.go`
  - `CreateRequestParams{ Type, Input:any, TimeoutS, SessionID }`
  - `POST { "type": "form", "input": { "title": "...", "schema": <object> } }`

### Server/store: accepts and stores `Input` as `any`
- `plz-confirm/internal/server/server.go`
  - `createRequestBody{ Input any }`
  - `store.Create(... Input: body.Input ...)`
- `plz-confirm/internal/store/store.go`
  - stores `types.UIRequest{ Input:any }`

### WebSocket: broadcasts the stored request as JSON
- `plz-confirm/internal/server/ws.go`
  - `WriteJSON({ type: "new_request", request: req })`

### Client: puts request in Redux and renders widget
- `plz-confirm/agent-ui-system/client/src/services/websocket.ts`
  - parses `data.request` and dispatches `setActiveRequest(request)`
- `plz-confirm/agent-ui-system/client/src/components/WidgetRenderer.tsx`
  - renders `<FormDialog input={active.input} ... />`

Conclusion: The schema is *present in `active.input.schema`* when it reaches the form widget.

## Root cause: FormDialog ignores schema context fields

`plz-confirm/agent-ui-system/client/src/components/widgets/FormDialog.tsx`:
- builds `properties = schema.properties || {}`
- renders `Object.entries(properties)` as fields
- uses the **property key** (e.g. `q1_total_files`) as the label
- does **not** render:
  - `schema.description`
  - `fieldSchema.title`
  - `fieldSchema.description`
  - any other “help text” / “instructions” / examples

This exactly matches the reported behavior (“just a bunch of fields”).

## Contributing factors / secondary issues (not the main bug, but relevant)

### Property ordering may be unintuitive

Because the schema is decoded into Go maps and re-encoded through `encoding/json`, the order of object keys may not match the original file’s order. This can shuffle a quiz, even if labels are improved.

### Output typing is currently lossy (numbers as strings)

For text/number inputs the UI stores `e.target.value` (a string) into `formData`.
The output is therefore likely strings for numbers (unless a later normalization step exists).
This is separate from “context display”, but matters for correctness if downstream expects typed numbers.

## Fix options

### Option A (recommended): render standard JSON Schema context

Update `FormDialog.tsx` to:
- show schema-level description/instructions:
  - `schema.title` (optional)
  - `schema.description` (important)
- show per-field labels/help:
  - label: `fieldSchema.title ?? name`
  - helper text: `fieldSchema.description`
- improve placeholder:
  - prefer `fieldSchema.examples?.[0]`, then `fieldSchema.default`, then fallback placeholder

This is backwards compatible and immediately improves UX for the reported schema.

### Option B: support an explicit “message/instructions” field on FormInput

Add `message?: string` to `FormInput` (mirroring `ConfirmInput.message`) and render it above the form.
Then add a CLI flag like `--message` or `--instructions @file.md` to pass rich context.

This helps when instructions are not embedded in JSON Schema (or when we want Markdown).

### Option C: adopt a schema-driven form renderer library

Use a dedicated JSON Schema form renderer (e.g. react-jsonschema-form) to support more of the spec:
`oneOf`, `anyOf`, nested objects, arrays, enums, formats, validation messages, etc.

This is higher-effort and would need UI/theming decisions.

## Acceptance criteria (for “fixed”)

Given a schema like:
- schema-level `description`: visible at the top of the dialog
- property-level `title` and `description`: visible next to each field

The user can complete the quiz without needing to interpret internal field identifiers.

