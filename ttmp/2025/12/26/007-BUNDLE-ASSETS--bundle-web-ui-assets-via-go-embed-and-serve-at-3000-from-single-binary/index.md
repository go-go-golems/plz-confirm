---
Title: Bundle Web UI assets via go:embed and serve at :3000 from single binary
Ticket: 007-BUNDLE-ASSETS
Status: active
Topics:
    - backend
    - build
    - release
    - embed
    - assets
    - ui
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: .goreleaser.yaml
      Note: Release builds must compile with embed tags
    - Path: internal/server/embed.go
      Note: go:embed filesystem used when built with -tags embed
    - Path: internal/server/embed_none.go
      Note: Default build disables embedded FS
    - Path: internal/server/generate.go
      Note: go:generate entrypoint
    - Path: internal/server/generate_build.go
      Note: Build+copy Vite output into embed/public
    - Path: internal/server/server.go
      Note: Static + SPA serving handler (current behavior)
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-26T13:27:00.962290785-05:00
WhatFor: ""
WhenToUse: ""
---


# Bundle Web UI assets via go:embed and serve at :3000 from single binary

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- backend
- build
- release
- embed
- assets
- ui

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
