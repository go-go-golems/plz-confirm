---
Title: Diary
Ticket: 003-REQUEST-METADATA
Status: active
Topics:
    - backend
    - cli
    - protobuf
    - observability
    - linux
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: internal/metadata/metadata_test.go
      Note: Metadata collection regression test
    - Path: internal/metadata/process_linux.go
      Note: Linux /proc-based parent chain collector
    - Path: internal/metadata/process_other.go
      Note: Non-Linux stub collector
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-03T16:23:22.375016029-05:00
WhatFor: ""
WhenToUse: ""
---


# Diary

## Goal

Track implementation of `003-REQUEST-METADATA` (attach request metadata like cwd + process tree to `UIRequest.metadata`, preserve it through server/store, and optionally display it in the UI).

## Context

The repo uses `protojson(UIRequest)` as the REST contract; request creation is driven by the Go CLI client, so metadata capture should happen client-side and be best-effort (never blocking request creation).

## Quick Reference

N/A

## Usage Examples

N/A

## Related

- Analysis: `ttmp/2026/01/03/003-REQUEST-METADATA--request-metadata-cwd-process-tree-env/analysis/01-request-metadata-architecture-post-protojson-cutover.md`

## Step 1: Add `UIRequest.metadata` and capture cwd + process info

This step adds a protobuf-backed metadata envelope to requests and populates it at request creation time. The key goal is best-effort provenance: if metadata capture fails for any reason, the request still gets created and behaves normally.

The impact is better observability and a foundation for richer history UX without changing the core request/response flow.

**Commit (code):** 865bcf1d4d7a1f862cce7dd3ce03a20c1ef1bd56 — "✨ metadata: attach request provenance to UIRequest"

### What I did
- Extended `proto/plz_confirm/v1/request.proto` with `RequestMetadata` + `ProcessInfo` and added `UIRequest.metadata`.
- Implemented `internal/metadata` to collect:
  - `cwd` via `os.Getwd()`
  - `self` and (Linux-only) `parents` via `/proc`
  - non-Linux stub uses `os.Getpid/os.Getppid` and `os.Args`
- Attached metadata in `internal/client.Client.CreateRequest` (best-effort).
- Preserved metadata through storage and API/WS emission by copying `Metadata` in `internal/store/store.go:Create`.
- Optionally enriched server-side `metadata.remoteAddr` and `metadata.userAgent` in `internal/server/server.go:handleCreateRequest`.
- Added a minimal UI display in history (`agent-ui-system/client/src/pages/Home.tsx`) showing `comm @ cwd` when present.
- Updated `scripts/curl-inspector-smoke.sh` to create a request with `metadata` and assert it is preserved.
- Updated docs: `pkg/doc/adding-widgets.md` request metadata section.

### Why
- Requests are otherwise anonymous; capturing provenance helps debug “what created this request?” (especially when multiple agents/terminals are involved).

### What worked
- `make ci` passes (buf lint, go test, TS check).

### What didn't work
- N/A

### What I learned
- Keeping metadata in the protobuf envelope keeps the REST+WS+UI types aligned and avoids ad-hoc JSON side channels.

### What was tricky to build
- Ensuring the Linux `/proc` collector is strictly best-effort and never becomes a hard dependency for request creation.

### What warrants a second pair of eyes
- Protobuf field naming and stability: confirm `remote_addr` / `user_agent` / `cwd` are the desired long-term fields and JSON names.

### What should be done in the future
- If we want “agent identity” beyond process info, add an explicit agent label field rather than overloading `argv`.

### Code review instructions
- Start with `proto/plz_confirm/v1/request.proto`, then `internal/metadata/*`, then follow the wiring in `internal/client/client.go` and `internal/server/server.go`.
- Validate by running `API_BASE_URL=http://localhost:3001 bash scripts/curl-inspector-smoke.sh`.

## Step 2: Close ticket 003

This step closed the ticket after all tasks were completed and the implementation was committed. The impact is ticket hygiene: it keeps the active ticket list meaningful.

### What I did
- Closed the ticket with a changelog entry referencing the implementation commit.

### Why
- Avoid leaving finished work in an “active” state.

### What worked
- `docmgr ticket close --ticket 003-REQUEST-METADATA` updated status to `complete`.

### What didn't work
- N/A

### What I learned
- N/A

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- N/A

### What should be done in the future
- N/A

### Code review instructions
- N/A
