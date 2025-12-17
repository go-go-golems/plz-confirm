---
Title: Port agent-ui-system CLI + backend to Go using Glazed framework
Ticket: DESIGN-PLZ-CONFIRM-001
Status: completed
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
    - Path: agent-ui-system/client/src/components/WidgetRenderer.tsx
      Note: Widget routing logic
    - Path: agent-ui-system/client/src/services/mockData.ts
      Note: Example request payloads to guide schema-first DSL
    - Path: agent-ui-system/client/src/services/websocket.ts
      Note: |-
        WebSocket client implementation
        Frontend WS protocol expectations (new_request/request_completed)
    - Path: agent-ui-system/client/src/store/store.ts
      Note: Redux store structure
    - Path: agent-ui-system/client/src/types/schemas.ts
      Note: TypeScript type definitions for all widget types
    - Path: agent-ui-system/demo_cli.py
      Note: Python CLI demonstration showing usage patterns
    - Path: agent-ui-system/server/index.ts
      Note: Backend server implementation - Express + WebSocket
    - Path: agent-ui-system/vite.config.ts
      Note: |-
        Dev proxy rules (3000->3001) explain CLI base URL and must be preserved
        Dev proxy contract (3000->3001) used by current tmux harness
    - Path: cmd/plz-confirm/main.go
      Note: |-
        Go CLI entrypoint (Cobra + Glazed) and serve command
        Go CLI entrypoint + serve subcommand
        Registers select command into root cobra
        Registers form/table/upload commands into root cobra
    - Path: internal/cli/form.go
      Note: Glazed form command implementation
    - Path: internal/cli/select.go
      Note: Glazed select command implementation
    - Path: internal/cli/table.go
      Note: Glazed table command implementation
    - Path: internal/cli/upload.go
      Note: Glazed upload command implementation
    - Path: internal/server/embed.go
      Note: Embedded filesystem for production static assets
    - Path: internal/server/generate.go
      Note: go:generate directive for building and embedding frontend
    - Path: internal/server/generate_build.go
      Note: Go program that builds Vite frontend and copies to embed directory
    - Path: internal/server/server.go
      Note: |-
        Go backend REST implementation (net/http, manual routing)
        Go REST handlers for /api/requests*
        Updated to serve embedded static files with SPA fallback
    - Path: internal/server/ws.go
      Note: |-
        Go backend WebSocket broadcast implementation (C2 map+mutex, no-session)
        Go WS handler and broadcaster (accept/ignore sessionId)
    - Path: internal/store/store.go
      Note: |-
        In-memory request store with event-driven wait (F2)
        In-memory request store with event-driven Wait
    - Path: ttmp/2025/12/15/DESIGN-PLZ-CONFIRM-001--port-agent-ui-system-cli-backend-to-go-using-glazed-framework/scripts/test-all-commands.sh
      Note: End-to-end test script exercising all widget commands
    - Path: ttmp/2025/12/15/DESIGN-PLZ-CONFIRM-001--port-agent-ui-system-cli-backend-to-go-using-glazed-framework/scripts/test-form-schema.json
      Note: Test JSON Schema for form widget
    - Path: ttmp/2025/12/15/DESIGN-PLZ-CONFIRM-001--port-agent-ui-system-cli-backend-to-go-using-glazed-framework/scripts/test-table-data.json
      Note: Test data (array of row objects) for table widget
    - Path: ttmp/2025/12/15/DESIGN-PLZ-CONFIRM-001--port-agent-ui-system-cli-backend-to-go-using-glazed-framework/scripts/tmux-up.sh
      Note: tmux harness entrypoint (control/server/vite)
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-15T15:35:20.981402078-05:00
---








# Port agent-ui-system CLI + backend to Go using Glazed framework

## Overview

Successfully ported the CLI and backend components of `agent-ui-system` from Node.js/TypeScript to Go, using the Glazed framework. The React frontend was preserved unchanged. The Go implementation includes:

- **Complete backend server**: REST API and WebSocket server using `net/http` and `gorilla/websocket`
- **All widget commands**: confirm, select, form, table, upload implemented as Glazed commands
- **Production deployment**: Frontend assets embedded in Go binary via `go generate`
- **Comprehensive documentation**: Embedded help system with usage guide
- **Test infrastructure**: E2E test scripts and tmux development harness

The system enables AI agents to request user feedback through web-based dialogs, with agents using CLI commands and users interacting via browser.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **completed**

**Completion date:** 2025-12-17

All core functionality has been successfully ported from Node.js/TypeScript to Go. The system is fully functional with all five widget types (confirm, select, form, table, upload), production-ready embedding, and comprehensive documentation.

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
