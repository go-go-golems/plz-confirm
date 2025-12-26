---
Title: Unify backend↔frontend shared data with Protobuf + codegen
Ticket: 006-USE-PROTOBUF
Status: active
Topics:
    - backend
    - frontend
    - api
    - protobuf
    - codegen
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: plz-confirm/agent-ui-system/client/src/types/schemas.ts
      Note: TypeScript type definitions (manually duplicated from backend)
    - Path: plz-confirm/internal/cli/confirm.go
      Note: Example CLI command showing marshal/unmarshal pattern
    - Path: plz-confirm/internal/client/client.go
      Note: HTTP client for CLI
    - Path: plz-confirm/internal/server/server.go
      Note: HTTP server with REST API handlers
    - Path: plz-confirm/internal/server/ws.go
      Note: WebSocket broadcaster implementation
    - Path: plz-confirm/internal/types/types.go
      Note: Core type definitions (manually duplicated from frontend)
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-25T22:56:37.739490019-05:00
WhatFor: ""
WhenToUse: ""
---


# Unify backend↔frontend shared data with Protobuf + codegen

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- backend
- frontend
- api
- protobuf
- codegen

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
