---
Title: Request metadata (cwd + process tree + env)
Ticket: 003-REQUEST-METADATA
Status: active
Topics:
    - backend
    - cli
    - protobuf
    - observability
    - linux
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: internal/client/client.go
      Note: Attach metadata on CreateRequest (protojson UIRequest body)
    - Path: internal/server/server.go
      Note: Optionally enrich metadata with remote_addr/user_agent
    - Path: internal/store/store.go
      Note: Preserve metadata when cloning requests
    - Path: pkg/doc/adding-widgets.md
      Note: Document new UIRequest.metadata field
    - Path: proto/plz_confirm/v1/request.proto
      Note: Add RequestMetadata + ProcessInfo to UIRequest
    - Path: scripts/curl-inspector-smoke.sh
      Note: Extend smoke test to assert metadata preserved
    - Path: ttmp/2026/01/03/001-QOL-HISTORY-TUI--history-pagination-metadata-defaults/analysis/01-history-and-metadata-architecture.md
      Note: Earlier end-to-end metadata/history analysis (pre-implementation)
    - Path: ttmp/2026/01/03/001-QOL-HISTORY-TUI--history-pagination-metadata-defaults/analysis/02-ws-rest-cli-protocols.md
      Note: Protocol analysis; note REST is now protojson-only
    - Path: ttmp/2026/01/03/001-QOL-HISTORY-TUI--history-pagination-metadata-defaults/reference/01-diary.md
      Note: Original research diary for metadata/history/defaults
    - Path: ttmp/2026/01/03/002-REMOVE-LEGACY-REST-JSON--remove-legacy-rest-json-use-protojson-uirequest/reference/01-diary.md
      Note: REST protojson cutover diary (impacts metadata insertion point)
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-03T14:49:27.593301108-05:00
WhatFor: ""
WhenToUse: ""
---



# Request metadata (cwd + process tree + env)

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- backend
- cli
- protobuf
- observability
- linux

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
