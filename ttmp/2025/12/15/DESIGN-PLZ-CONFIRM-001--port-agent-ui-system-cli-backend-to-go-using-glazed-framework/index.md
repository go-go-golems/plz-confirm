---
Title: Port agent-ui-system CLI + backend to Go using Glazed framework
Ticket: DESIGN-PLZ-CONFIRM-001
Status: active
Topics:
    - go
    - glazed
    - cli
    - backend
    - porting
    - agent-ui-system
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: vibes/2025-12-15/agent-ui-system/client/src/components/WidgetRenderer.tsx
      Note: Widget routing logic
    - Path: vibes/2025-12-15/agent-ui-system/client/src/services/mockData.ts
      Note: Example request payloads to guide schema-first DSL
    - Path: vibes/2025-12-15/agent-ui-system/client/src/services/websocket.ts
      Note: WebSocket client implementation
    - Path: vibes/2025-12-15/agent-ui-system/client/src/store/store.ts
      Note: Redux store structure
    - Path: vibes/2025-12-15/agent-ui-system/client/src/types/schemas.ts
      Note: TypeScript type definitions for all widget types
    - Path: vibes/2025-12-15/agent-ui-system/demo_cli.py
      Note: Python CLI demonstration showing usage patterns
    - Path: vibes/2025-12-15/agent-ui-system/server/index.ts
      Note: Backend server implementation - Express + WebSocket
    - Path: vibes/2025-12-15/agent-ui-system/vite.config.ts
      Note: Dev proxy rules (3000->3001) explain CLI base URL and must be preserved
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-15T15:35:20.981402078-05:00
---



# Port agent-ui-system CLI + backend to Go using Glazed framework

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- go
- glazed
- cli
- backend
- porting
- agent-ui-system

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
