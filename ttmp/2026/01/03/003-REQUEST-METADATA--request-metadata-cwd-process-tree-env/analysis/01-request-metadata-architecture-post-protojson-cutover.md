---
Title: Request metadata architecture (post protojson cutover)
Ticket: 003-REQUEST-METADATA
Status: active
Topics:
    - backend
    - cli
    - protobuf
    - observability
    - linux
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-03T14:49:31.488764565-05:00
WhatFor: ""
WhenToUse: ""
---

# Request metadata architecture (post protojson cutover)

## Why this ticket exists

Now that the REST API has been cut over to accept `protojson(UIRequest)` directly (no wrapper JSON), the request envelope (`UIRequest`) is the single canonical “thing on the wire” for:

- REST request creation (`POST /api/requests`)
- REST response submission (`POST /api/requests/{id}/response`)
- WebSocket broadcasts (`new_request`, `request_completed`)
- CLI wait responses (`GET /api/requests/{id}/wait`)

This is exactly the moment to add **request metadata**: once metadata is a field on `UIRequest`, it automatically propagates everywhere with no extra wrapper-layer work.

The goal of this ticket is to add structured metadata such as:

- the caller’s working directory (`cwd`)
- the process tree (pid/parent chain with **name + args + pid**, Linux-only implementation via `/proc`)
- optional server-enriched transport metadata (remote address, user-agent)

## Current state (post-cutover) — what exists today

### REST contracts (current)

The server currently expects protojson payloads.

Create request:

```jsonc
POST /api/requests
{
  "type": "form",
  "sessionId": "global",
  "formInput": { "title": "...", "schema": { /* JSON schema */ } },
  "expiresAt": "2026-01-03T18:15:00Z"
}
```

Submit response:

```jsonc
POST /api/requests/{id}/response
{
  "type": "form",
  "formOutput": { "data": { "host": "db" }, "comment": "..." }
}
```

WebSocket events already wrap `protojson(UIRequest)`:

```jsonc
{ "type": "new_request", "request": { /* UIRequest */ } }
{ "type": "request_completed", "request": { /* UIRequest */ } }
```

### Storage (current)

The server store (`internal/store/store.go`) clones a subset of fields when creating a request:

- `Type`, `SessionId`, `Input` oneof
- and server-generated fields (`Id`, `CreatedAt`, `ExpiresAt`, `Status`)

There is no metadata field today, so this cloning is currently fine. Once metadata exists, the store must preserve it (and avoid silently dropping it).

## Proposed schema: metadata on UIRequest

### High-level model

We want metadata to answer:

1) who created the request (process-wise)
2) from where (cwd)
3) using what (binary/args), and what the parent process chain looked like

The cleanest place is:

- `proto/plz_confirm/v1/request.proto`: add `RequestMetadata metadata = ...;` to `message UIRequest`.

### Proposed protobuf sketch (for planning)

```proto
message ProcessInfo {
  int64 pid = 1;
  optional string name = 2;      // e.g. /proc/<pid>/comm
  repeated string argv = 3;      // e.g. /proc/<pid>/cmdline
}

message RequestMetadata {
  optional string cwd = 1;
  optional ProcessInfo self = 2;
  repeated ProcessInfo parents = 3; // nearest parent first; include init last if desired

  // Optional server-enriched (authoritative) fields:
  optional string remote_addr = 10;
  optional string user_agent = 11;

  map<string,string> tags = 100; // escape hatch for extra key/value metadata
}

message UIRequest {
  ...
  optional RequestMetadata metadata = 21;
}
```

Notes:

- Keep `argv` as `repeated string` so it’s easy to render and filter.
- The “self vs parents” structure makes it obvious what PID the request was created by, without forcing the UI to treat the first parent as the creator.

## Where metadata is captured and how it flows (after this change)

### Capture at the CLI

The CLI is the authoritative creator of most requests. It can reliably capture:

- `cwd` (`os.Getwd()`)
- `self` process info (`os.Getpid()`, argv via `os.Args`)
- parent chain (Linux-only, best effort)

Implementation should live in a small helper package so it’s reused across widget commands:

- likely under `internal/metadata/` or `internal/client/metadata/`

Then `internal/client/client.go:CreateRequest` can attach `reqProto.Metadata = Collect(...)`.

### Server enrichment (optional)

The server can add transport-layer metadata at create time:

- `remote_addr = r.RemoteAddr`
- `user_agent = r.Header.Get("User-Agent")`

Since the server decodes `protojson(UIRequest)` already, it can:

- create/merge `reqProto.Metadata` before storing
- preserve client-provided metadata fields
- overwrite only server-authoritative fields

### Persistence/display

Once metadata is a protobuf field on `UIRequest`:

- it is automatically present in WS and REST responses
- the UI can render it in history and/or active request header
- future history persistence work can store it as part of the request record

## Linux-only “full parent process info” (build tags)

The requirement is: “Get full parent process (name, args, pid) for linux with a linux build tag.”

Design:

- `internal/metadata/process_linux.go`:
  - `//go:build linux`
  - Read `/proc/<pid>/comm` for a stable process name.
  - Read `/proc/<pid>/cmdline` (NUL-separated) for argv.
  - Read `/proc/<pid>/stat` or `/proc/<pid>/status` to find `ppid`.
  - Walk parent chain until `pid <= 1` or a cycle/parse failure.
- `internal/metadata/process_other.go`:
  - `//go:build !linux`
  - Return minimal metadata (cwd + self pid/name/argv), no parent chain.

Important behavioral choice:

- This should be **best effort**: metadata collection failure should never prevent request creation.

Pseudocode:

```go
func CollectProcessTree() (self ProcessInfo, parents []ProcessInfo) {
  self = ProcessInfo{pid: os.Getpid(), argv: os.Args, name: readComm(os.Getpid())}
  ppid := readPPID(os.Getpid())
  for ppid > 1 {
    parents = append(parents, ProcessInfo{pid: ppid, name: readComm(ppid), argv: readCmdline(ppid)})
    ppid = readPPID(ppid)
  }
  return
}
```

## Implementation map (files that will change)

### Protobuf / codegen

- `proto/plz_confirm/v1/request.proto` (add metadata messages + field)
- run `make codegen` (updates Go + TS generated types)

### Go server

- `internal/server/server.go`:
  - optionally merge server-side transport metadata into `reqProto.Metadata` during create
- `internal/store/store.go`:
  - preserve `.Metadata` when cloning `UIRequest` in `Create`

### Go client/CLI

- `internal/client/client.go`:
  - populate `reqProto.Metadata` automatically on create
- add new helper package:
  - `internal/metadata/` (or similar) with `process_linux.go` / `process_other.go`

### Frontend

- Optional UI display work:
  - `agent-ui-system/client/src/pages/Home.tsx` (history rows)
  - `agent-ui-system/client/src/components/WidgetRenderer.tsx` (active request header)

## “No backwards compatibility” concerns

Adding new optional protobuf fields is typically non-breaking for protojson consumers, but note:

- The store cloning behavior can effectively “drop” metadata unless updated.
- If any external tooling depends on the exact JSON keys present in `UIRequest`, adding new keys may affect strict decoders (unlikely in JS, but possible in some scripts).

## Related docs

- Ticket `002-REMOVE-LEGACY-REST-JSON` for the cutover details.
- `pkg/doc/adding-widgets.md` for the core contracts and where to hook new fields.
