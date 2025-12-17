---
Title: 'Fix form display: missing schema/context blocks in plz-confirm web form'
Ticket: 002-FIX-FORM-DISPLAY
Status: active
Topics:
    - plz-confirm
    - frontend
    - bug
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: plz-confirm/agent-ui-system/client/src/components/WidgetRenderer.tsx
      Note: Routes active request.type to FormDialog
    - Path: plz-confirm/agent-ui-system/client/src/components/widgets/FormDialog.tsx
      Note: Form widget UI; currently ignores schema context (title/description)
    - Path: plz-confirm/agent-ui-system/client/src/services/websocket.ts
      Note: Receives new_request payload and stores request.input used by widgets
    - Path: plz-confirm/agent-ui-system/client/src/types/schemas.ts
      Note: Frontend wire types; FormInput currently only title+schema
    - Path: plz-confirm/internal/cli/form.go
      Note: CLI reads schema file and sends FormInput{Title
    - Path: plz-confirm/internal/client/client.go
      Note: HTTP client POST /api/requests with Input:any
    - Path: plz-confirm/internal/server/server.go
      Note: CreateRequest handler stores Input:any and broadcasts via WS
    - Path: plz-confirm/internal/server/ws.go
      Note: Broadcasts pending/new requests to clients
    - Path: plz-confirm/internal/store/store.go
      Note: In-memory storage of UIRequest Input:any (schema should survive)
    - Path: plz-confirm/internal/types/types.go
      Note: Backend wire types; FormInput has Title+Schema only
ExternalSources: []
Summary: "Form widget currently omits schema-provided context (instructions, field titles/descriptions) and labels fields by raw property keys."
LastUpdated: 2025-12-17T16:23:02.477849842-05:00
---


# Fix form display: missing schema/context blocks in plz-confirm web form

## Overview

Fix the **form widget UI** so it renders schema-provided context (instructions + per-field question text/help), instead of only showing raw field keys.

This ticket was opened after `plz-confirm form --schema @/tmp/doc-cleanup-quiz.json` produced a browser form that contained “just a bunch of fields” (e.g. `q1_total_files`) without the quiz/instructions.

## Key Links

- Bug report: `analysis/01-bug-report.md`
- Root-cause analysis: `analysis/02-root-cause-analysis-missing-context-rendering.md`
- Diary: `reference/01-diary.md`

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- plz-confirm
- frontend
- bug

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
