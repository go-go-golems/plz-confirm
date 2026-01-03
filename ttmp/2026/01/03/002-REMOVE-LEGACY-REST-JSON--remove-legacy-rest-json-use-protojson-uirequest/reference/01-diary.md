---
Title: Diary
Ticket: 002-REMOVE-LEGACY-REST-JSON
Status: active
Topics:
    - backend
    - api
    - protobuf
    - breaking-change
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: agent-ui-system/client/src/components/WidgetRenderer.tsx
      Note: Passes active request type into submitResponse
    - Path: agent-ui-system/client/src/services/websocket.ts
      Note: UI submitResponse now wraps output into UIRequest oneof
    - Path: internal/client/client.go
      Note: CLI CreateRequest now sends protojson(UIRequest) with input oneof
    - Path: internal/server/proto_convert.go
      Note: Deleted legacy wrapper conversion layer
    - Path: internal/server/server.go
      Note: Switched REST create/response to protojson(UIRequest) bodies
    - Path: pkg/doc/adding-widgets.md
      Note: Updated docs to describe protojson(UIRequest) REST contract
    - Path: scripts/curl-inspector-smoke.sh
      Note: Updated smoke script to new protojson body shapes
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-03T14:30:06.881796218-05:00
WhatFor: ""
WhenToUse: ""
---


# Diary

## Goal

Track the breaking cutover that removes the legacy REST wrapper JSON shapes and switches the REST API to accept `protojson(UIRequest)` directly, with no backwards compatibility.

## Step 1: Establish the cutover scope and identify all callers

This step pins down exactly what “legacy REST JSON” means in this repo and where it is used. The main risk is missing a caller: once the server stops accepting `{type,input,timeout}` and `{output}`, any leftover script or client will fail hard.

The concrete outcome is a checklist of every file (Go server, Go CLI client, web UI submission, and curl smoke scripts) that must be changed in the same patch series.

### What I did
- Created ticket `002-REMOVE-LEGACY-REST-JSON` and added a detailed task breakdown.
- Searched for `/api/requests` usage across `scripts/` and `ttmp/` to find all hardcoded legacy payloads.
- Confirmed current implementations:
  - Server wrapper structs: `internal/server/server.go` (`createRequestBody`, `submitResponseBody`)
  - Server conversion glue: `internal/server/proto_convert.go`
  - CLI create wrapper: `internal/client/client.go` (embeds `input` into legacy body)
  - UI submit wrapper: `agent-ui-system/client/src/services/websocket.ts` (sends `{ output: ... }`)
  - Smoke script: `scripts/curl-inspector-smoke.sh` (uses both legacy create and legacy response wrappers)

### Why
- This is intentionally a hard cutover (no compatibility). Missing a caller would create confusing partial breakage (some workflows work, others fail).

### What worked
- The repo already emits `protojson(UIRequest)` on WS and REST responses, so shifting request bodies to the same representation is conceptually aligned.

### What didn't work
- N/A

### What I learned
- The “legacy REST JSON” is not only used by the CLI; it’s also used by the web UI for submit-response (`{output: ...}`) and by curl/automation scripts.

### What was tricky to build
- N/A (planning step)

### What warrants a second pair of eyes
- The new request-body shape must be documented and updated everywhere at once; reviewers should specifically sanity-check scripts under `scripts/` and `ttmp/.../scripts/`.

### What should be done in the future
- N/A

### Code review instructions
- Start in `internal/server/server.go` (handlers), then `internal/client/client.go` (CLI create), then `agent-ui-system/client/src/services/websocket.ts` (UI submit), then update and run `scripts/curl-inspector-smoke.sh`.

## Step 2: Cut over REST request bodies to protojson(UIRequest) everywhere (server, CLI, UI, scripts)

This step implemented the actual breaking change: the server no longer accepts the legacy wrapper JSON for request creation and response submission. Instead, both endpoints now accept `protojson(UIRequest)` directly. That required coordinated changes across Go server, Go CLI HTTP client, the React UI submit function, and the repo’s curl/E2E smoke scripts.

The immediate outcome is a consistent contract across REST and WS: everything that flows over the wire is now in the `protojson(UIRequest)` shape (including the input/output oneofs). This removes the last “special-case JSON wrapper” in the request lifecycle.

### What I did
- Server:
  - Updated `internal/server/server.go`:
    - `POST /api/requests` now decodes `protojson(UIRequest)` (requires `type` + input oneof).
    - `POST /api/requests/{id}/response` now decodes `protojson(UIRequest)` (requires output oneof).
    - Added strict validation that the output oneof type matches the stored request type.
  - Deleted `internal/server/proto_convert.go` (legacy wrapper conversion layer).
- CLI:
  - Updated `internal/client/client.go:CreateRequest` to send `protojson(UIRequest)` with the correct input oneof field (no `{type,input,timeout}` wrapper).
  - Preserved the CLI `--timeout` behavior by setting `expiresAt` (RFC3339Nano) client-side.
- UI:
  - Updated `agent-ui-system/client/src/services/websocket.ts:submitResponse` to send `protojson(UIRequest)` with the correct output oneof field (no `{output: ...}` wrapper).
  - Updated `agent-ui-system/client/src/components/WidgetRenderer.tsx` to pass the active request type to `submitResponse`.
- Smoke scripts / automation:
  - Updated `scripts/curl-inspector-smoke.sh` to create requests and submit responses using `protojson(UIRequest)`.
  - Updated:
    - `ttmp/2025/12/24/001-ADD-IMG-WIDGET--add-image-widget-to-cli-and-web-interface/scripts/auto-e2e-cli-via-api.sh`
    - `ttmp/2025/12/24/004-ADD-WIDGET-COMMENT--add-freeform-comment-field-to-all-widgets/scripts/auto-e2e-comment-via-api.sh`
    to wrap outputs using the correct output oneof field.
- Docs:
  - Updated `pkg/doc/adding-widgets.md` to document the new REST contract and oneof field requirements.

### Why
- Keeping the “create” and “response” bodies as special wrapper shapes was the last remaining source of protocol drift between REST and WS.
- A single canonical payload shape (`protojson(UIRequest)`) reduces codepaths, removes conversion glue, and makes it easier to evolve the request envelope (metadata, timeouts, completion info).

### What worked
- `protojson` already matched the WS payload shape, so the cutover was mostly mechanical once all callers were identified.
- Existing smoke scripts were a good safety net once updated.

### What didn't work
- Initially hit `bind: address already in use` when running e2e scripts because an old server process was still holding `:3001`; killed it and reran successfully.

## Step 3: Keep sessionId and scope WebSocket traffic by it

This step implements “sessionId is real” in the Go server. Previously, the server accepted `sessionId` for compatibility but ignored it (broadcasting all `new_request`/`request_completed` events to all connected WS clients, and replaying all pending requests on connect).

Now, each WS connection subscribes to a specific `sessionId` (`/ws?sessionId=...`), pending replay is filtered by that sessionId, and broadcasts are sent only to clients subscribed to the request’s session.

**Commit (code):** TBD

### What I did
- Updated `internal/server/ws.go` to group WS clients by `sessionId` and to replay only pending requests for that session.
- Updated `internal/server/server.go` to broadcast `new_request` and `request_completed` only within the request’s session.
- Added `internal/store/store.go:PendingForSession` to support session-scoped pending replay.

### Why
- Without session scoping, multiple open UIs (or multiple agents) leak requests across sessions, and the demo UI’s `sessionId` query param is effectively misleading.

### What warrants a second pair of eyes
- Confirm the intended default session behavior when `sessionId` is missing (currently defaults to `global`).

### What I learned
- Even though the legacy “create” wrapper was CLI-only, the legacy “submit response” wrapper was used by the UI and by multiple repo scripts, so the cutover must remain coordinated.

### What was tricky to build
- Ensuring the server validates “output oneof matches stored request type” without relying on the client-provided `type` field (which may be missing or wrong).

### What warrants a second pair of eyes
- Review the new request-body validation errors in `internal/server/server.go` to ensure they’re actionable (bad oneof shapes are easy to trip over).

### What should be done in the future
- If we introduce new endpoints (history paging, touch, metadata), keep the “protojson-only” rule consistent and avoid reintroducing wrapper message shapes.

### Code review instructions
- Start with `internal/server/server.go` changes (create + response handlers), then verify the CLI request bodies in `internal/client/client.go`, then verify the UI submit wrapper in `agent-ui-system/client/src/services/websocket.ts`.
- Validate with:
  - `go test ./... -count=1`
  - `pnpm -C agent-ui-system run check`
  - `API_BASE_URL=http://localhost:3001 bash scripts/curl-inspector-smoke.sh`

### Technical details
- New create request body is `protojson(UIRequest)` with:
  - `type` set to `"confirm" | "select" | ...`
  - exactly one of `confirmInput`, `selectInput`, `formInput`, `uploadInput`, `tableInput`, `imageInput`
  - optional `expiresAt` RFC3339Nano string (server defaults if omitted)
- New submit response body is `protojson(UIRequest)` with:
  - exactly one of `confirmOutput`, `selectOutput`, `formOutput`, `uploadOutput`, `tableOutput`, `imageOutput`
