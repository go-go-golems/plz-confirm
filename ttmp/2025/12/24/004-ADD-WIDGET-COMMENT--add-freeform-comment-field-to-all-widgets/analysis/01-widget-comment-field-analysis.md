---
Title: Analysis: Add Optional Freeform Comment Field to All Widgets
Ticket: 004-ADD-WIDGET-COMMENT
Slug: widget-comment-field-analysis
Short: Codebase analysis and implementation touchpoints for adding a folded “comment” textarea to all widgets and returning it via CLI outputs
Topics:
- analysis
- cli
- frontend
- backend
- ux
---

## Goal (what we’re building)

Add an **optional freeform comment** textarea to **every widget** (confirm/select/form/table/upload/image), **hidden/folded by default**, and include that comment in:

- the JSON stored as request output (`POST /api/requests/{id}/response`)
- the CLI command’s emitted rows (`--output json/yaml/table/csv`)

Conceptually: each widget output gets a new optional field:

```text
comment?: string
```

## Why this is straightforward in plz-confirm

The backend is widget-agnostic:

- Requests carry `input: any`
- Responses carry `output: any`

So “returning additional fields” is primarily a **schema + UI + CLI** concern.

## The key contracts (where comment must flow)

### REST contract: Submit response

Frontend submits:

```text
POST /api/requests/{id}/response
{ "output": { ... } }
```

Relevant implementation:
- `internal/server/server.go`
  - `handleSubmitResponse` unmarshals `submitResponseBody{ Output any }` and passes it into the store without validation.

### Frontend contract: `onSubmit(output)` from widget components

All React widgets follow a consistent pattern:

- Widget component gathers user input
- Calls `onSubmit({ ...widgetOutput... })`
- `WidgetRenderer` calls `submitResponse(active.id, output)` then `completeRequest(...)`

Relevant files/symbols:
- `agent-ui-system/client/src/components/WidgetRenderer.tsx`
  - `handleSubmit(output: any)`
- `agent-ui-system/client/src/services/websocket.ts`
  - `submitResponse(requestId: string, output: any)` → `fetch('/api/requests/:id/response', { body: JSON.stringify({ output }) })`

### CLI contract: “re-marshal any → typed struct → rows”

Each Go CLI command:

1. creates request
2. waits for completion
3. takes `completed.Output` (typed as `any`)
4. `json.Marshal(completed.Output)` then `json.Unmarshal(..., &TypedOutput)`
5. emits a `types.Row`

Key files:
- `internal/cli/confirm.go`
- `internal/cli/select.go`
- `internal/cli/form.go`
- `internal/cli/table.go`
- `internal/cli/upload.go`
- `internal/cli/image.go`

## Where to change types (Go + TS)

### Go types

`internal/types/types.go` defines per-widget output structs:
- `ConfirmOutput`
- `SelectOutput`
- `FormOutput`
- `UploadOutput`
- `TableOutput`
- `ImageOutput`

Implementation: add a common optional field to each output struct:

```go
Comment *string `json:"comment,omitempty"`
```

Notes:
- Use pointer to preserve “absent vs empty string”.
- No backend change needed for storage; it stores `any`.

### TypeScript types

`agent-ui-system/client/src/types/schemas.ts` mirrors the Go wire types.

Add `comment?: string` to:
- `ConfirmOutput`
- `SelectOutput`
- `FormOutput`
- `UploadOutput`
- `TableOutput`
- `ImageOutput`

## Where to change UI (React): folded comment input

### Existing UI primitives you can reuse

The UI already has Radix-based primitives:
- `agent-ui-system/client/src/components/ui/collapsible.tsx`
- `agent-ui-system/client/src/components/ui/accordion.tsx`

Preferred UX: a **collapsed-by-default** section, eg:

```text
[+] Add a comment (optional)
    (textarea...)
```

### Widget components to update

Add local state `comment` and include it in `onSubmit(...)` payload if non-empty.

Touchpoints:
- `agent-ui-system/client/src/components/widgets/ConfirmDialog.tsx`
  - currently submits `{ approved, timestamp }`
- `.../SelectDialog.tsx`
  - currently submits `{ selected }`
- `.../FormDialog.tsx`
  - currently submits `{ data }`
- `.../TableDialog.tsx`
  - currently submits `{ selected }`
- `.../UploadDialog.tsx`
  - currently submits `{ files: uploadedFiles }` (simulated upload)
- `.../ImageDialog.tsx`
  - currently submits `{ selected, timestamp }`

Implementation sketch (per widget):

```pseudo
state:
  comment = ""
  commentOpen = false

UI:
  Collapsible default closed
  Textarea bound to comment

on submit:
  output = { ...existingFields }
  if comment.trim() != "":
    output.comment = comment.trim()
  await onSubmit(output)
```

“Hidden/folded by default” = `commentOpen=false` initially.

## Where to change CLI outputs (add a column)

### Add `comment` (string) column everywhere

For each CLI command, when building the output row(s), add:

```go
types.MRP("comment", derefOrEmpty(out.Comment))
```

For multi-row outputs (upload), include the same comment value on each row (so the result is easily joinable by `request_id`).

### Edge cases

- Widgets that currently don’t include a timestamp (select/form/table/upload) can still carry a comment without changing semantics.
- If the UI includes comment but the CLI struct doesn’t, the comment will be silently dropped by `json.Unmarshal` — so types must be updated in both Go and TS.

## Minimal backend work

No changes required to `/api/requests/*` for functionality.

Optional (nice-to-have):
- Add backend validation / size limit for `comment` (but that changes the current “backend is widget-agnostic” philosophy).

## Testing strategy (recommended)

1. Extend the API-driven smoke script approach (submit responses with `"comment": "..."`) and verify the CLI prints it.
2. Manual UI validation:
   - confirm widget: approve + comment → CLI includes comment
   - image widget: select + comment → CLI includes comment
3. Add one small Go unit test per CLI output type (marshal/unmarshal roundtrip includes comment).

## Files/symbols checklist (quick reference)

- **Go types**: `internal/types/types.go`
  - `ConfirmOutput`, `SelectOutput`, `FormOutput`, `UploadOutput`, `TableOutput`, `ImageOutput`
- **TS types**: `agent-ui-system/client/src/types/schemas.ts`
  - `ConfirmOutput`, `SelectOutput`, `FormOutput`, `UploadOutput`, `TableOutput`, `ImageOutput`
- **React widgets**:
  - `agent-ui-system/client/src/components/widgets/ConfirmDialog.tsx`
  - `.../SelectDialog.tsx`
  - `.../FormDialog.tsx`
  - `.../TableDialog.tsx`
  - `.../UploadDialog.tsx`
  - `.../ImageDialog.tsx`
- **Submit plumbing**:
  - `agent-ui-system/client/src/services/websocket.ts` → `submitResponse`
  - `internal/server/server.go` → `handleSubmitResponse`
- **CLI row emission**:
  - `internal/cli/*.go` per widget command


