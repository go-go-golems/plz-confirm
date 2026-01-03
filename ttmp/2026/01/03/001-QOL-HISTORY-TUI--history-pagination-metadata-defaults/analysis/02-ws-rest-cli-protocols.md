# WebSocket + REST protocols in plz-confirm (and how CLI + UI use them)

## Purpose

This document explains:

- the **WebSocket protocol** (`/ws`) used to deliver requests to the browser in realtime,
- the **legacy REST API contract** (`/api/requests/*`) used by both CLI and UI,
- how the **CLI** uses REST and why it does *not* use WebSocket,
- how the **web UI** uses WebSocket to receive events and REST to submit user responses.

It is intended to be a “single mental model” reference for developers implementing features like:

- bounded/paginated history (requires list endpoints),
- `expiresAt` enforcement and UI countdown (requires server-driven transitions + UI display),
- metadata propagation (needs to survive protojson and WS payloads).

## The key architectural choice: realtime to browsers, polling to CLIs

In this repo, realtime delivery is **WebSocket server → browser** only.

- Browsers keep a WebSocket open to receive “new_request” and “request_completed” events.
- The CLI creates requests via REST, and then waits using **HTTP long-poll** (`GET /wait`) until the request completes.

This isn’t an accident: HTTP long-poll works reliably in environments where outbound WS is inconvenient, and it keeps the CLI transport simple and script-friendly.

## Protocol overview

At runtime you have three “channels”:

```text
Browser  <-- WebSocket (/ws) ---  Go server  <--- REST (/api/...) --- CLI
Browser  --- REST (/api/...) ---> Go server
```

The browser does two things:

1) **listen** via WebSocket for incoming work,
2) **submit** via REST when the user responds.

The CLI does two things:

1) **create** requests via REST,
2) **wait** via REST long-poll.

## WebSocket protocol (`/ws`)

### Endpoint

`GET /ws?sessionId=<id>` upgrades to WebSocket.

- Go server: `internal/server/ws.go:handleWS`
- Frontend client: `agent-ui-system/client/src/services/websocket.ts:connectWebSocket`

### SessionId semantics (compatibility only)

The frontend always includes a query parameter `sessionId`.

The Go backend **accepts it but ignores it**:

- `internal/server/ws.go` reads it and discards it.
- On connect, the server sends **all pending requests** (global), regardless of session.

This is a legacy compatibility holdover from the Node implementation where requests were scoped per session.

### Message shape (server → client)

All WS messages are JSON objects of this shape:

```jsonc
{
  "type": "new_request" | "request_completed",
  "request": { /* protojson(UIRequest) */ }
}
```

Implementation:

- wrapper: `internal/server/ws_events.go`
- broadcast call sites:
  - create: `internal/server/server.go` broadcasts `"new_request"`
  - completion: `internal/server/server.go` broadcasts `"request_completed"`

### “request” payload encoding: protojson(UIRequest)

The `request` field is a raw JSON object produced by `protojson`:

- camelCase field names (`createdAt`, `expiresAt`, `sessionId`, `confirmInput`, …)
- enum values emitted as their **enum value names** (strings), because that’s how protojson emits enums
  - e.g. `"type": "confirm"`, `"status": "pending"`

This is encoded in:

- `internal/server/ws_events.go` uses `protojson.MarshalOptions{UseProtoNames:false}`

### Initial replay behavior

On a new WS connection, the server pushes all **currently pending** requests as `new_request` events:

- `internal/server/ws.go` calls `s.store.Pending(...)` and sends each request as `"new_request"`.

Important: there is no equivalent replay of completed history today.

### Client → server WS messages

The Go server does **not** expect any WebSocket messages from the client:

- It reads until close and discards content (`conn.ReadMessage()` loop).
- All interactions are over REST.

## REST API (legacy contract, still the primary write path)

REST is where all state mutations happen:

- request creation: CLI → server
- response submission: browser → server

The Go server implements the REST surface in:

- `internal/server/server.go`
- conversion glue in `internal/server/proto_convert.go` (JSON ↔ protobuf)

### 1) Create request

`POST /api/requests`

Body:

```jsonc
{
  "type": "confirm" | "select" | "form" | "upload" | "table" | "image",
  "sessionId": "global",
  "input": { /* widget-specific */ },
  "timeout": 300
}
```

Behavior:

1) decode JSON body (`createRequestBody` in `internal/server/server.go`)
2) convert `{type,input,timeout}` into protobuf `UIRequest`:
   - `internal/server/proto_convert.go:createUIRequestFromJSON`
3) store in memory:
   - `internal/store/store.go:Create` assigns:
     - `id` (uuid)
     - `status=pending`
     - `createdAt`
     - `expiresAt` (now + timeout, default 300)
4) broadcast WS `"new_request"`
5) return JSON `protojson(UIRequest)` with HTTP 201

### 2) Get request by id

`GET /api/requests/{id}`

Behavior:

- reads from store and returns `protojson(UIRequest)` (status pending/completed).

### 3) Submit response (complete request)

`POST /api/requests/{id}/response`

Body:

```jsonc
{ "output": { /* widget-specific output */ } }
```

Behavior:

1) server loads existing request to determine widget type (`store.Get`)
2) server converts `output` JSON into typed protobuf `*Output` message:
   - `internal/server/proto_convert.go:createUIRequestWithOutput`
3) server marks request completed (`store.Complete`)
4) server broadcasts WS `"request_completed"`
5) returns `protojson(UIRequest)` for the completed request

### 4) Wait for completion (long-poll)

`GET /api/requests/{id}/wait?timeout=<seconds>`

Key semantics:

- `timeout` query parameter bounds **the duration of this single HTTP request**, not the request’s `expiresAt`.
- If the request is still pending when this poll times out, the server returns:
  - HTTP 408 and text `"timeout waiting for response"`.

Implementation:

- server: `internal/server/server.go:handleWait`
- store wait primitive: `internal/store/store.go:Wait` blocks on a per-request channel

## How the CLI uses the backend (REST only)

The CLI implementation pattern for widgets is:

1) build a widget input protobuf message (`ConfirmInput`, `SelectInput`, …)
2) `CreateRequest` → returns a `UIRequest` with `id`
3) `WaitRequest` → long-poll loop until completed
4) print output as Glazed rows

### CLI request creation: REST + protojson input embedding

The CLI client in `internal/client/client.go` has:

- `CreateRequest(ctx, CreateRequestParams)`

It uses protobuf types internally, but must speak the legacy JSON create-request contract externally. The way it does this is:

1) marshal the protobuf widget input to JSON via `protojson` (camelCase)
2) unmarshal that JSON into `any` so it can be embedded in the legacy body `{type,input,timeout,sessionId}`
3) send JSON to `POST /api/requests`
4) parse the server’s response back into protobuf `v1.UIRequest` via `protojson.Unmarshal`

That translation is in:

- `internal/client/client.go` (`CreateRequest`, `createRequestBody`)

### CLI waiting: long-poll loop with per-poll timeouts

The CLI does not hold a WebSocket open. Instead, it long-polls:

```text
repeat:
  GET /api/requests/{id}/wait?timeout=25
  if 200: done
  if 408: retry
  else: error
until overall wait-timeout expires
```

Implementation details:

- `internal/client/client.go:WaitRequest`
  - `waitTimeoutS` is an *overall* deadline (CLI patience); `0` means “wait forever”.
  - Each iteration uses a poll timeout (`defaultPollTimeoutS = 25`) and clamps it to remaining overall time.
  - The client treats HTTP 408 as “poll timed out, keep waiting”.

This pattern means:

- you can safely set long overall waits without keeping a single HTTP request open forever,
- the server can scale per-request waits because the store uses a channel-based wait, not a polling loop.

### CLI never uses WebSocket

There is no WebSocket client in Go CLI code. All CLI/backend interaction is via `net/http`.

This is intentional:

- CLIs are often run in restricted network contexts where long-lived WS connections are fragile.
- Long-poll is simple and interoperable.

## How the web UI uses the backend (WebSocket read + REST write)

### Receiving requests and completion events: WebSocket

The UI connects to `/ws?sessionId=...` and listens for:

- `type === "new_request"`: set the request as active
- `type === "request_completed"`: move it into history (or complete active)

Implementation:

- `agent-ui-system/client/src/services/websocket.ts`

The UI also triggers browser notifications on `new_request`.

### Submitting responses: REST

Widgets submit responses by calling:

`POST /api/requests/{id}/response` with `{ output: ... }`.

Implementation:

- `agent-ui-system/client/src/services/websocket.ts:submitResponse`
- invoked by `agent-ui-system/client/src/components/WidgetRenderer.tsx` in `handleSubmit`

So despite living in `services/websocket.ts`, the submission path is pure REST.

### Does the UI ever send messages over WebSocket?

No. The UI does not use WS for any mutation:

- it never writes messages on `ws.send(...)`
- all state changes (completions) go through REST endpoints

This mirrors the Go server’s WS posture (“server-only events”).

## “Legacy REST API” and protobuf: why both exist

From an implementation perspective, there are two “shapes”:

1) **Create/submit REST shapes** (legacy):
   - create: `{ type, sessionId, input, timeout }`
   - submit: `{ output }`

2) **Wire payloads returned to clients**:
   - `protojson(UIRequest)` (used in both REST responses and WS event `request` field)

So the server and CLI both run a conversion layer:

- JSON input/output payloads ↔ typed protobuf messages

The primary conversion choke points are:

- server: `internal/server/proto_convert.go`
- CLI: `internal/client/client.go`

## Practical diagrams

### End-to-end happy path

```text
CLI                                  Server                                  Browser
---                                  ------                                  -------
POST /api/requests (type,input,timeout)
  -> Store.Create (pending)  -------> WS: {type:new_request, request:UIRequest} ---> render widget

GET /api/requests/{id}/wait?timeout=25  (repeat until done)
                                        Browser POST /api/requests/{id}/response (output)
                                        -> Store.Complete (completed)
                                        -> WS: {type:request_completed, request:UIRequest} ---> add to history
<------------------------------- 200 + UIRequest (completed)  --------------------- CLI unblocks
```

### What happens if the browser refreshes?

Because WS connects are “stateless”, the server compensates by replaying pending requests on connect:

```text
Browser connects WS
Server sends all pending requests as "new_request"
Browser sets active request
```

Completed history is not replayed today.

## Protocol risks and invariants (things to keep stable)

### Invariants

- WS event wrapper shape stays stable:
  - `{ type: string, request: <UIRequest json> }`
- REST `POST /api/requests` continues to accept:
  - `type` as a string matching proto enum names (`confirm`, `select`, …)
  - `input` as a JSON object in protojson field-name style
- REST `POST /api/requests/{id}/response` continues to accept:
  - `{ output: <protojson output message> }` (with explicit oneof fields)

### Common pitfalls

- **Oneof JSON shapes**: outputs like select/table/image require explicit fields (`selectedSingle`, `selectedMulti`, etc.). If you send a “simplified” JSON shape, `protojson.Unmarshal` will fail.
- **Int64 JSON**: some 64-bit fields may appear as JSON strings. Don’t “fix” this in the protocol layer; normalize in UI if needed.
- **Sessions**: if you later reintroduce session scoping, the WS initial replay behavior and history fetching semantics will change significantly.

## What to change if you want “UI timeouts” or “history paging”

This protocol map implies:

- UI-only countdowns are not authoritative: you need a server-side transition (and a WS broadcast) for correctness.
- Bounded history is easy UI-only; pagination needs a new REST list endpoint and persistence.

See `analysis/01-history-and-metadata-architecture.md` for implementation strategies and schema proposals.
