---
Title: QOL: History + Metadata + Defaults/Timeouts
Ticket: 001-QOL-HISTORY-TUI
Status: active
Topics:
  - frontend
  - backend
  - storage
  - ux
  - telemetry
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
  - Path: pkg/doc/adding-widgets.md
    Note: Canonical doc for request lifecycle + API/WS contracts
  - Path: proto/plz_confirm/v1/request.proto
    Note: UIRequest envelope (likely place to add metadata + completion reasons)
  - Path: proto/plz_confirm/v1/widgets.proto
    Note: Widget input/output messages (likely place to add per-widget defaults/timeouts)
  - Path: internal/server/server.go
    Note: HTTP REST API for requests (/api/requests/*) and wait long-poll
  - Path: internal/server/ws.go
    Note: WebSocket broadcaster + initial pending replay on connect
  - Path: internal/server/proto_convert.go
    Note: JSON<->protobuf conversion (legacy REST shape) for inputs/outputs
  - Path: internal/store/store.go
    Note: Current in-memory request store (no long-term history)
  - Path: internal/client/client.go
    Note: CLI HTTP client that creates requests and long-polls for completion
  - Path: agent-ui-system/client/src/services/websocket.ts
    Note: Frontend WS client; dispatches active request + history updates
  - Path: agent-ui-system/client/src/store/store.ts
    Note: Redux history is unbounded; limit/pagination likely starts here
  - Path: agent-ui-system/client/src/pages/Home.tsx
    Note: History panel rendering (ScrollArea + full list)
  - Path: Makefile
    Note: Dev targets for running Go backend + Vite frontend
  - Path: scripts/tmux-up.sh
    Note: tmux session helper to run backend (:3001) and Vite (:3000)
  - Path: ttmp/2026/01/03/001-QOL-HISTORY-TUI--history-pagination-metadata-defaults/scripts/seed-requests-with-metadata.sh
    Note: Ticket-local script to seed multiple pending requests via CLI (for reproducing history/queue behavior)
ExternalSources: []
Summary: "Research + design notes to add bounded/paginated request history, per-request metadata, and widget-level defaults with timeouts."
LastUpdated: 2026-01-03T00:00:00Z
---

# 001-QOL-HISTORY-TUI — History + metadata + defaults

## Overview

This ticket is a codebase tour and implementation plan for:

- Bounded and scrollable/paginated request history in the web UI.
- Request-level metadata (origin + environment) carried end-to-end.
- Long-term storage for request history (server-side).
- Widget-level “default + timeout” so non-response auto-selects a default and is visibly recorded in history.

## Key Links

- See `analysis/01-history-and-metadata-architecture.md` for the deep dive.
- See `reference/01-diary.md` for the research diary (frequent steps + commands run).
- See `tasks.md` for an actionable checklist.
 - For manual UI repro/verification, run `make dev-tmux` and then seed requests with `ttmp/2026/01/03/001-QOL-HISTORY-TUI--history-pagination-metadata-defaults/scripts/seed-requests-with-metadata.sh`.
