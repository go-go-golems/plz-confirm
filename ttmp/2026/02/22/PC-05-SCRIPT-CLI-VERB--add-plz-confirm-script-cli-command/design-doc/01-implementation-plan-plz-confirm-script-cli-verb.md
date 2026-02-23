---
Title: Implementation Plan - plz-confirm script CLI verb
Ticket: PC-05-SCRIPT-CLI-VERB
Status: active
Topics:
    - cli
    - backend
    - javascript
    - api
    - ux
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/plz-confirm/main.go
      Note: Root command registration for new script subcommand
    - Path: internal/cli
      Note: New command implementation package and settings struct
    - Path: internal/client/client.go
      Note: Existing script request create/wait support reused by CLI command
    - Path: pkg/doc/how-to-use.md
      Note: End-user command documentation that must include script verb
    - Path: pkg/doc/js-script-api.md
      Note: API docs currently note script is API-first; this changes with CLI support
ExternalSources: []
Summary: Plan for adding a first-class script command to the CLI.
LastUpdated: 2026-02-22T21:58:00-05:00
WhatFor: ""
WhenToUse: ""
---

# Implementation Plan - plz-confirm script CLI verb

## Executive Summary

Script requests are currently API-first and require manual curl or custom wrappers to run scripts. This ticket adds a first-class `plz-confirm script` command that creates script requests, waits for completion, and outputs script results in normal glazed formats.

## Problem Statement

Without a dedicated CLI command, users must:

- Build protojson payloads manually.
- Handle script source loading themselves.
- Manage session/timeouts in ad-hoc shell snippets.

This creates unnecessary friction and prevents parity with other widget commands.

## Proposed Solution

Add a new CLI command in `internal/cli` and wire it in `cmd/plz-confirm/main.go`.

Command goals:

1. Accept script source from file or stdin.
2. Accept optional props JSON.
3. Create script request via existing client path (`WidgetType_script`).
4. Wait for completion using existing long-poll behavior.
5. Print structured output including request id and result payload.

Proposed flags:

- `--base-url`
- `--session-id`
- `--timeout` (request expiration)
- `--wait-timeout`
- `--title` (required)
- `--script-file` (path or `-` for stdin; required unless `--script` provided)
- `--script` (inline source; optional alternative)
- `--props-file` (JSON object file)
- `--props-json` (inline JSON object)
- `--script-timeout-ms` (maps to `scriptInput.timeoutMs`)

## Design Decisions

### Decision 1: File-first script input with explicit stdin support

Prefer `--script-file` for reproducibility while allowing `-` for piping generated scripts.

### Decision 2: Dual props inputs (`--props-file` and `--props-json`)

Support both automation and quick interactive use.

### Decision 3: Reuse existing client request/wait flows

Do not introduce script-specific HTTP handling in CLI command code.

Reasoning: minimize divergence and reduce test surface.

## Alternatives Considered

### Alternative A: Keep API-only and ship helper shell scripts

Rejected because discoverability and UX stay poor.

### Alternative B: Add only `--script` inline string

Rejected because real scripts are multiline and easier to manage as files.

### Alternative C: Implement a separate script transport client

Rejected because `internal/client` already handles script create/wait.

## Implementation Plan

1. Add `ScriptCommand` in `internal/cli/script.go` with glazed flag definitions.
2. Implement script source loading and mutual-exclusion validation (`--script` vs `--script-file`).
3. Implement props loading/validation and conversion to `structpb.Struct`.
4. Build `v1.ScriptInput` and call `client.CreateRequest` with `WidgetType_script`.
5. Wait for completion via `client.WaitRequest` and validate completed status.
6. Emit output rows with request id, script describe metadata if present, and final `scriptOutput.result`.
7. Register command in `cmd/plz-confirm/main.go`.
8. Add tests for flag parsing, invalid inputs, successful completion path, and timeout/error behavior.
9. Update docs (`pkg/doc/how-to-use.md`, `pkg/doc/js-script-api.md`) with command examples.
10. Add a smoke script under ticket `scripts/` for manual verification.

## Open Questions

- Should command support `--watch` mode to stream intermediate `request_updated` states?
- Should script logs be printed by default or only under a verbose flag once log capture lands?

## References

- `internal/cli/confirm.go`
- `internal/client/client.go`
- `pkg/doc/js-script-api.md`
