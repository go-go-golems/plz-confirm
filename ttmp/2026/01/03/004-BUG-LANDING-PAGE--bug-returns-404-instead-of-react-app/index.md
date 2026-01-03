---
Title: 'BUG: / returns 404 instead of React app'
Ticket: 004-BUG-LANDING-PAGE
Status: active
Topics:
    - web
    - backend
    - static
    - bug
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: Makefile
      Note: Build/install no longer require -tags embed
    - Path: README.md
      Note: Document embedding behavior and build steps
    - Path: internal/server/embed.go
      Note: Embed SPA assets unconditionally so go run serve can serve /
    - Path: internal/server/server.go
      Note: Static file handler mounts at / with SPA fallback
    - Path: internal/server/server_static_test.go
      Note: 'Regression test: GET / returns SPA index.html'
    - Path: pkg/doc/adding-widgets.md
      Note: Update dev docs to match embedding contract
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-03T14:54:28.739987847-05:00
WhatFor: ""
WhenToUse: ""
---


# BUG: / returns 404 instead of React app

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- web
- backend
- static
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
