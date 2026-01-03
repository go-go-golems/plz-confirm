---
Title: Request metadata (cwd + process tree + env)
Ticket: 003-REQUEST-METADATA
Status: complete
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
    - Path: agent-ui-system/client/src/pages/Home.tsx
      Note: Display cwd/process label in history
    - Path: internal/client/client.go
      Note: |-
        Attach metadata on CreateRequest (protojson UIRequest body)
        Attach metadata on request creation
    - Path: internal/metadata/metadata.go
      Note: Collect best-effort provenance (cwd + process tree)
    - Path: internal/server/server.go
      Note: |-
        Optionally enrich metadata with remote_addr/user_agent
        Enrich metadata with remoteAddr/userAgent
    - Path: internal/store/store.go
      Note: |-
        Preserve metadata when cloning requests
        Preserve Metadata through storage
    - Path: pkg/doc/adding-widgets.md
      Note: |-
        Document new UIRequest.metadata field
        Document UIRequest.metadata
    - Path: proto/plz_confirm/v1/request.proto
      Note: |-
        Add RequestMetadata + ProcessInfo to UIRequest
        Add RequestMetadata + ProcessInfo and UIRequest.metadata
    - Path: scripts/curl-inspector-smoke.sh
      Note: |-
        Extend smoke test to assert metadata preserved
        Assert metadata preserved and server enrichment present
    - Path: ttmp/2026/01/03/001-QOL-HISTORY-TUI--history-pagination-metadata-defaults/analysis/01-history-and-metadata-architecture.md
      Note: Earlier end-to-end metadata/history analysis (pre-implementation)
    - Path: ttmp/2026/01/03/001-QOL-HISTORY-TUI--history-pagination-metadata-defaults/analysis/02-ws-rest-cli-protocols.md
      Note: Protocol analysis; note REST is now protojson-only
    - Path: ttmp/2026/01/03/001-QOL-HISTORY-TUI--history-pagination-metadata-defaults/reference/01-diary.md
      Note: Original research diary for metadata/history/defaults
    - Path: ttmp/2026/01/03/002-REMOVE-LEGACY-REST-JSON--remove-legacy-rest-json-use-protojson-uirequest/reference/01-diary.md
      Note: REST protojson cutover diary (impacts metadata insertion point)
    - Path: ttmp/2026/01/03/003-REQUEST-METADATA--request-metadata-cwd-process-tree-env/reference/01-diary.md
      Note: Implementation diary (commit 865bcf1...)
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-03T16:31:48.673491326-05:00
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
