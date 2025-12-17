---
Title: Diary
Ticket: 002-FIX-FORM-DISPLAY
Status: active
Topics:
    - plz-confirm
    - frontend
    - bug
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Investigation diary for missing schema/context display in form widget."
LastUpdated: 2025-12-17T16:23:02.910297519-05:00
---

# Diary

## Goal

Keep a chronological record of investigation steps for ticket `002-FIX-FORM-DISPLAY`, including what we inspected, what we learned, and what needs follow-up.

## Step 1: Reproduce report mentally + trace schema/context through the stack

This step focused on validating whether the “missing context” was lost in transit (CLI → server → WS → client) or simply not rendered. We used the DOM snippet (showing `schema-form`) as the anchor to locate the actual source component and then traced the request payload end-to-end.

### What I did
- Created ticket workspace + initial docs (bug report, root-cause analysis, diary).
- Located the real `FormDialog.tsx` in the repo (it lives under `plz-confirm/agent-ui-system/…`, matching `data-loc`).
- Read `FormDialog.tsx` and confirmed it renders only `input.title` and `schema.properties` keys.
- Traced the payload:
  - CLI reads schema file into `any` and sends it as `input.schema`.
  - Server stores request `Input` as `any` and broadcasts it via WebSocket.
  - Client stores the `UIRequest` and passes `active.input` to `FormDialog`.

### Why
- To determine whether the bug is:
  - transport/parsing (schema context gets dropped), or
  - rendering (context exists but isn’t displayed).

### What worked
- We confirmed the schema is transmitted and stored as `any` end-to-end.
- We identified a direct root cause in UI rendering: schema/field `title`/`description` aren’t used.

### What didn't work
- N/A (no runtime debugging performed yet; this was a static trace).

### What I learned
- The form widget UI currently behaves like a “barebones JSON schema properties editor” rather than a user-facing questionnaire:
  - labels are raw property keys,
  - schema descriptions aren’t shown,
  - placeholders are generic.

### What was tricky to build
- JSON Schema “context” is not a dedicated field; it typically lives in standard keywords (`description`, `title`) and needs intentional rendering.

### What warrants a second pair of eyes
- Whether key ordering matters for quiz schemas: Go’s `encoding/json` roundtrips via maps and may reorder object keys.
- Whether output typing (numbers as strings) is acceptable for downstream consumers; fixing display might surface data typing expectations.

### What should be done in the future
- Implement “Option A” rendering (schema + per-field `title`/`description`) and add a small visual spec for how instructions/help text should look.
- Decide if we also want explicit `FormInput.message` + CLI `--message/--instructions` support for Markdown instructions.

### Code review instructions
- Start in `plz-confirm/agent-ui-system/client/src/components/widgets/FormDialog.tsx`
- Then confirm payload shape in:
  - `plz-confirm/internal/cli/form.go`
  - `plz-confirm/internal/server/server.go`
  - `plz-confirm/agent-ui-system/client/src/services/websocket.ts`

### Technical details
- Reported command:
  - `plz-confirm form --title ... --schema @/tmp/doc-cleanup-quiz.json --base-url http://localhost:3000`
- Observed DOM anchor:
  - `<form id="schema-form" ...>` from `FormDialog.tsx`

## Step 2: Clean up accidental stray doc path (docmgr doctor)

This step fixed a documentation workspace hygiene issue: an accidentally created duplicate ticket folder broke `docmgr doctor` due to an empty file without frontmatter. The fix was to remove the stray folder so the ticket validates cleanly again.

### What I did
- Ran `docmgr doctor --ticket 002-FIX-FORM-DISPLAY` and found an `invalid_frontmatter` error referencing a stray path:
  - `.../002-FIX-FORM-DISPLAY--fix-form-display-missing-schema/context blocks in plz-confirm web form/reference/01-diary.md`
- Verified the file existed and was empty.
- Deleted the stray folder so only the intended ticket workspace remains:
  - `.../002-FIX-FORM-DISPLAY--fix-form-display-missing-schema-context-blocks-in-plz-confirm-web-form/`
- Re-ran `docmgr doctor` to confirm the error was gone.

### What warrants a second pair of eyes
- N/A (pure doc hygiene), but worth ensuring no other stray `ttmp` artifacts were created by tooling.

## Related
- See `analysis/01-bug-report.md`
- See `analysis/02-root-cause-analysis-missing-context-rendering.md`
