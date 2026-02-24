---
Title: Use go-go-goja RuntimeFactory in script engine
Ticket: PC-03-USE-GOJA-RUNTIMEFACTORY
Status: complete
Topics:
    - go
    - backend
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-02-22/plz-confirm-js/plz-confirm/internal/scriptengine/engine.go
      Note: Current script runtime bootstrap and lifecycle implementation.
    - Path: /home/manuel/workspaces/2026-02-22/plz-confirm-js/plz-confirm/internal/server/script.go
      Note: HTTP mapping and script update lifecycle integration.
    - Path: /home/manuel/workspaces/2026-02-22/plz-confirm-js/plz-confirm/internal/server/server.go
      Note: Script create path where init logs will be attached to response payload.
    - Path: /home/manuel/workspaces/2026-02-22/plz-confirm-js/plz-confirm/internal/store/store.go
      Note: Request persistence layer that will store latest script run logs.
    - Path: /home/manuel/workspaces/2026-02-22/plz-confirm-js/plz-confirm/proto/plz_confirm/v1/request.proto
      Note: Planned API addition for top-level per-run script logs.
    - Path: /home/manuel/workspaces/2026-02-22/plz-confirm-js/plz-confirm/pkg/doc/js-script-development.md
      Note: Developer docs currently describing old sandbox assumptions.
    - Path: /home/manuel/workspaces/2026-02-22/plz-confirm-js/go-go-goja/engine/factory.go
      Note: RuntimeFactory creation and runtime ownership semantics.
    - Path: /home/manuel/workspaces/2026-02-22/plz-confirm-js/go-go-goja/engine/module_specs.go
      Note: RuntimeInitializer contract for console-capture bootstrap hooks.
    - Path: /home/manuel/workspaces/2026-02-22/plz-confirm-js/go-go-goja/engine/runtime.go
      Note: Owned runtime close semantics (event loop plus owner teardown).
ExternalSources: []
Summary: Ticket workspace for RuntimeFactory hard-cut plan with require-enabled sandbox and console log capture returned in script HTTP responses.
LastUpdated: 2026-02-24T10:02:31.275885708-05:00
WhatFor: Track planning and implementation assets for PC-03 RuntimeFactory migration.
WhenToUse: Use when implementing or reviewing script runtime bootstrap and script-log response refactor work.
---


# Use go-go-goja RuntimeFactory in script engine

## Overview

This ticket now plans a no-compat hard cut of `plz-confirm` script runtime setup from direct `goja.New()` calls to `go-go-goja` factory-owned runtimes.

Product decisions now locked in for this ticket:
- `require` is available in script sandboxes.
- `console` is available and backend captures console output.
- Script run HTTP responses must include captured logs.

## Key Links

- Analysis: `analysis/01-runtimefactory-migration-implications-for-plz-confirm-script-engine.md`
- Earlier design: `design-doc/01-refactor-plan-adopt-go-go-goja-runtimefactory-in-script-engine.md`
- Current implementation plan: `design-doc/02-implementation-plan-factory-hard-cut-with-require-and-console-log-capture.md`
- Tasks: `tasks.md`
- Changelog: `changelog.md`

## Status

Current status: **active**

## Topics

- go
- backend

## Scope Summary

- Define full hard-cut migration to factory-only runtime setup.
- Design and implement console capture pipeline for script runs.
- Return logs in create/update/complete HTTP response payloads.
- Replace brittle error mapping with typed runtime error categories.

## Structure

- `analysis/` - Impact and cleanup analysis.
- `design-doc/` - Architecture and implementation plans.
- `reference/` - Optional supplementary notes.
- `playbooks/` - Optional runbooks/test procedures.
- `scripts/` - Ticket-local helper scripts.
- `various/` - Working notes.
- `archive/` - Deprecated artifacts.
