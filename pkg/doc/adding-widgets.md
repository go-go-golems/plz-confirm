---
Title: Adding New Widgets to plz-confirm
Slug: adding-widgets
Short: Developer guide for implementing new widget types across Go CLI/server and the React agent UI
Topics:
- developer
- guide
- architecture
- cli
- backend
- agent-ui-system
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

# Adding New Widgets to plz-confirm

## Overview

plz-confirm is intentionally built around a small number of stable contracts: a **request** is created by the CLI, stored by the server, rendered by the web UI, and completed by submitting an **output** payload. Adding a “complex widget” means you’re extending that contract across **four places** (types, server, CLI, UI), and then wiring in **docs + tests** so future changes don’t silently break it.

This guide is written for developers new to the repo. It mixes background (why the system is shaped this way) with a didactic, step-by-step implementation recipe. It uses the image widget (`type: "image"`) as the running example.

## Mental model: the two core contracts

Before touching code, internalize these two “shapes”. Almost everything in plz-confirm is built around them.

### 1) The request/response contract (`/api/requests`)

The CLI creates a request with JSON:

```text
POST /api/requests
{
  "type": "<widget-type>",
  "sessionId": "global",
  "input": { ...widget-specific... },
  "timeout": 300
}
```

The UI completes the request with JSON:

```text
POST /api/requests/{id}/response
{
  "output": { ...widget-specific... }
}
```

**Relevant code**
- **Server routes**: `internal/server/server.go`
  - `handleCreateRequest`
  - `handleSubmitResponse`
  - `handleWait`
- **CLI HTTP client**: `internal/client/client.go`
  - `CreateRequest`
  - `WaitRequest`

### 2) The realtime delivery contract (`/ws`)

The server broadcasts to the browser via WebSocket:

```text
{ "type": "new_request", "request": <UIRequest> }
{ "type": "request_completed", "request": <UIRequest> }
```

**Relevant code**
- **Server WebSocket**: `internal/server/ws.go`
- **Frontend WS client**: `agent-ui-system/client/src/services/websocket.ts`

## Architecture diagram (high level)

This shows the “happy path” lifecycle. The only time you need extra endpoints is when your widget needs something beyond JSON, like file uploads.

```text
CLI (agent)                       Go server                         Browser (React UI)
----------                       ---------                         ------------------
CreateRequest  POST /api/requests   -> store + WS broadcast  ->  WS "new_request" -> render widget
WaitRequest    GET  /api/requests/{id}/wait  <-------------------------------- user clicks submit
                                POST /api/requests/{id}/response  <- UI submits output
                                -> store completes + WS broadcast -> WS "request_completed"
CLI prints structured output
```

## Step-by-step: adding a new widget type

Each step starts with *why it exists*, then gives the mechanical checklist.

### Step 1: Define the wire schema (Go + TypeScript)

Typed schemas prevent “shape drift”. The server stores `input` and `output` as `any`, but the CLI and UI still need a clear contract so they can marshal/unmarshal reliably.

**What to change**
- `internal/types/types.go`
  - Add a new `WidgetType` constant (e.g. `WidgetImage`)
  - Add `XInput` / `XOutput` structs (and helper structs like `ImageItem`)
- `agent-ui-system/client/src/types/schemas.ts`
  - Extend the `UIRequest.type` union
  - Add matching TS interfaces

**Example (image widget)**
- Go: `internal/types/types.go` → `WidgetImage`, `ImageInput`, `ImageOutput`, `ImageItem`
- TS: `client/src/types/schemas.ts` → `type: ... | 'image'` + `ImageInput`, `ImageOutput`

### Step 2: Add server endpoints (only if the widget needs them)

Most widgets don’t need custom server logic: `/api/requests` already supports arbitrary JSON payloads.

You add server endpoints when you need:
- binary uploads (files, images)
- server-side derived data that the UI will fetch later
- any non-JSON side channel

**Example: image uploads**

The browser cannot fetch bytes from an agent’s local filesystem. So if the CLI accepts `--image ./file.png`, it must convert that file path into a URL the browser can load.

**Minimal API**

```text
POST /api/images        (multipart/form-data, field "file")
-> { id, url, mimeType, size }

GET /api/images/{id}
-> bytes with Content-Type
```

**Where it lives**
- `internal/server/server.go`
  - `handleImagesCollection` (POST)
  - `handleImagesItem` (GET/HEAD)
- `internal/server/images.go`
  - `ImageStore` (disk storage + in-memory index)

### Step 3: Extend the Go HTTP client (`internal/client`)

Keep the CLI logic small by putting transport details in the client package.

**Example (image widget)**
- `internal/client/client.go`
  - `UploadImage(ctx, filePath, ttlSeconds)`:
    - streams multipart upload (no huge buffering)
    - returns `{id,url,mimeType,size}`

### Step 4: Implement the CLI command (`internal/cli`) and register it

Every widget command follows the same pattern:
- parse flags into a settings struct (`glazed.parameter` tags)
- build an `Input` payload
- `CreateRequest` → `WaitRequest`
- print output as Glazed rows

**Where to implement**
- `internal/cli/<widget>.go` (new file)
- `cmd/plz-confirm/main.go` (register command)

**Pseudocode**

```pseudo
settings := parse flags
input := build XInput from settings
req := client.CreateRequest(type="x", input=input)
done := client.WaitRequest(req.id)
print done.output (as rows)
```

**Image widget nuance**

```pseudo
for each --image:
  if it's a local path:
    upload -> /api/images/{id}
    src = returned url
  else:
    src = url or data-uri
```

### Step 5: Implement the React widget + wire the renderer

The web UI renders exactly one “active request” at a time.

**Where to implement**
- Add a widget component under:
  - `agent-ui-system/client/src/components/widgets/`
- Route it in:
  - `agent-ui-system/client/src/components/WidgetRenderer.tsx`

**Pattern**
- Props: `{ requestId, input, onSubmit, loading }`
- Local state for selection / form inputs
- On submit: call `onSubmit(output)`

**Example: image widget**
- Component: `components/widgets/ImageDialog.tsx`
- Router: `components/WidgetRenderer.tsx` adds a `case 'image': ...`

### Step 6: Update user docs + add a smoke test

Shipping a widget without docs and a repeatable validation path makes it hard for others to trust or extend it.

**Docs to update**
- `pkg/doc/how-to-use.md` (end-user / agent developer docs; embedded in `plz-confirm help`)
- `README.md` (quick “what exists” overview)

**Validation tooling**
- Add a ticket-local smoke script under `ttmp/.../scripts/`.
- Prefer scripts that:
  - can be run repeatedly
  - log clearly
  - don’t require mental reassembly from chat history

## Dev workflow (tmux + logs)

In this repo, the most reliable development topology is:
- Go backend on `:3001`
- Vite UI on `:3000` proxying `/api` and `/ws` → `:3001`

**Run in two terminals (or tmux)**

```bash
# Terminal 1
go run ./cmd/plz-confirm serve --addr :3001

# Terminal 2
pnpm -C agent-ui-system dev --host
```

If you want to inspect behavior, tail the logs. For example, server logs include:

- `[API] Created request <id> (<type>)`
- `[API] Request <id> completed`

## Common pitfalls (things that bite new contributors)

### 1) Frontend assets embedding can break `go test` in fresh clones

If embedding expects generated files that aren’t present, compilation fails early. The fix pattern we use is:
- gate embedding behind a build tag
- provide a `!embed` stub that sets `embeddedPublicFS = nil`

### 2) `.gitignore` patterns can hide whole directories

Be careful with broad patterns like `plz-confirm` which can accidentally ignore `cmd/plz-confirm/`. Prefer anchored patterns like `/plz-confirm` if you mean “binary at repo root”.

### 3) “Local file path” needs an upload story

If a widget accepts a local path and the UI needs the bytes, you must convert it into a URL the browser can fetch (server-side upload + serve).

## Where to go next

- For end-user usage, run:

```bash
plz-confirm help how-to-use
```

- For the image widget design and UI sketches, see the ticket docs under `ttmp/2025/12/24/001-ADD-IMG-WIDGET--.../`.


