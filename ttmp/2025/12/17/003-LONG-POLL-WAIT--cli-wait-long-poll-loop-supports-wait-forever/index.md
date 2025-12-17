---
Title: 'CLI wait: long-poll loop (supports wait forever)'
Ticket: 003-LONG-POLL-WAIT
Status: active
Topics:
    - cli
    - backend
    - go
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: plz-confirm/internal/cli/confirm.go
      Note: Documents --wait-timeout 0=forever
    - Path: plz-confirm/internal/cli/form.go
      Note: Documents --wait-timeout 0=forever
    - Path: plz-confirm/internal/cli/select.go
      Note: Documents --wait-timeout 0=forever
    - Path: plz-confirm/internal/cli/table.go
      Note: Documents --wait-timeout 0=forever
    - Path: plz-confirm/internal/cli/upload.go
      Note: Documents --wait-timeout 0=forever
    - Path: plz-confirm/internal/client/client.go
      Note: Implements long-poll loop and wait-forever semantics
    - Path: plz-confirm/internal/client/client_test.go
      Note: Unit tests for retry/loop behavior
    - Path: plz-confirm/internal/server/server.go
      Note: /wait handler returns 408 on poll timeout
ExternalSources: []
Summary: Switch CLI waiting to a long-poll loop so waits >30s are reliable and `--wait-timeout 0` can wait forever.
LastUpdated: 2025-12-17T17:01:08.843994048-05:00
---


# CLI wait: long-poll loop (supports wait forever)

## Overview

Make `plz-confirm` CLI waiting robust by switching from a single `/wait` request to a **long-poll loop**. This removes the effective 30s cap from the current HTTP client timeout and enables `--wait-timeout 0` to wait forever (until cancelled).

## Key Links

- Design doc: `design-doc/01-design-long-poll-wait-loop-wait-forever-semantics.md`

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- cli
- backend
- go

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
