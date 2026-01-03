# plz-confirm: History, metadata, and default/timeout auto-completion (research doc)

## Table of contents

- Executive summary (what exists today)
- Source of truth: the “two contracts”
- Current architecture tour (schemas, server, store, UI)
- Feature: bounded/paginated history
- Feature: request metadata
- Feature: default result + timeout (server auto-complete)
- Feature: long-term storage + history API
- Concrete implementation plan (per feature, per layer)
- Optional: history TUI
- Documents and playbooks to read
- Appendix: canonical JSON payload shapes

## Audience and intent

This document is for a new developer joining `plz-confirm` who needs to understand:

1) how requests flow end-to-end today (CLI → server → WS → UI → response → CLI wait),
2) where “history” exists today (and what’s missing for long-term storage),
3) how to add request metadata end-to-end (wire shape, storage, UI),
4) how to add widget-level default+timeout behavior in a way that is reliable, observable, and debuggable.

It is intentionally verbose and cross-links the exact files and APIs involved.

## Executive summary (what exists today)

At a high level, `plz-confirm` is a “remote confirmation bus”:

- The **CLI** creates a request via `POST /api/requests`.
- The **server** stores it and pushes it to the browser via **WebSocket** `type: "new_request"`.
- The **web UI** renders exactly one “active” request at a time.
- When the user submits, the UI calls `POST /api/requests/{id}/response`.
- The **CLI** waits via long-poll `GET /api/requests/{id}/wait` until the request completes.

Today:

- The **Go backend’s store is in-memory** (`internal/store/store.go`), with no persistence and no completed-history listing API.
- The **web UI history is a Redux array** (`agent-ui-system/client/src/store/store.ts`) populated only from observed completion events.
- The **WS “replay” on connect sends only pending requests** (`internal/server/ws.go`), not completed history.

So “bounded/paginated history + long-term storage” requires backend work, not just UI work.

## Source of truth: the “two contracts”

The repo already has a canonical guide: `pkg/doc/adding-widgets.md`.

It frames the system around two stable contracts:

1) **REST request lifecycle**

```text
POST /api/requests
POST /api/requests/{id}/response
GET  /api/requests/{id}
GET  /api/requests/{id}/wait?timeout=<seconds>
```

2) **WebSocket realtime delivery**

```jsonc
{ "type": "new_request", "request": <UIRequest> }
{ "type": "request_completed", "request": <UIRequest> }
```

Everything in this ticket should ideally be expressed as an extension of these contracts:

- bounded/paged history: add a listing endpoint + UI fetch
- metadata: extend `<UIRequest>` schema and ensure it is propagated to WS + REST
- default+timeout: add an explicit “auto-complete” path that results in a `request_completed` broadcast

## Current architecture: component-by-component tour

### 1) Protobuf schemas (shared types)

**Files**

- `proto/plz_confirm/v1/request.proto`
  - Defines `UIRequest` envelope + `WidgetType` + `RequestStatus`.
- `proto/plz_confirm/v1/widgets.proto`
  - Defines widget-specific `*Input` / `*Output` messages.

**Why this matters**

Even though the server uses REST+JSON, it serializes and deserializes data using `protojson`:

- Go server writes responses using `protojson.MarshalOptions{UseProtoNames:false}` (camelCase JSON).
- Frontend uses generated TS types and normalizes enum values.

That means adding fields (metadata, completion reasons, defaults) should be done in protobuf first, and then translated through the REST/WS glue.

### 2) Go server: REST routes and WebSocket broadcast

**Entry**

- `internal/server/server.go` exposes:
  - `POST /api/requests` → `handleCreateRequest`
  - `POST /api/requests/{id}/response` → `handleSubmitResponse`
  - `GET /api/requests/{id}/wait` → `handleWait`
  - `GET /api/requests/{id}` → `store.Get`
  - `GET/POST /api/images` (image widget)
  - `GET /ws` → `handleWS` (WebSocket)

**Key traits**

- REST handlers accept a “legacy JSON shape”:
  - create body: `{ type, sessionId, input, timeout }`
  - response body: `{ output }`
- The server converts JSON to protobuf messages in `internal/server/proto_convert.go`.
- The server broadcasts WS events using `internal/server/ws_events.go` (`marshalWSEvent`).

**Data flow diagram (as implemented)**

```text
POST /api/requests
  -> decode JSON body (type/sessionId/input/timeout)
  -> proto_convert.createUIRequestFromJSON(...)
  -> store.Create(...)
  -> WS broadcast: marshalWSEvent("new_request", req)
  -> return UIRequest (protojson)

POST /api/requests/{id}/response
  -> decode JSON body (output)
  -> store.Get(id) to determine widget type
  -> proto_convert.createUIRequestWithOutput(type, output)
  -> store.Complete(id, outputReq)
  -> WS broadcast: marshalWSEvent("request_completed", completedReq)
  -> return completed UIRequest (protojson)

GET /api/requests/{id}/wait?timeout=60
  -> store.Wait(ctxTimeout, id)
  -> return completed UIRequest (protojson)
```

### 3) Go store: current request lifecycle state

**File**

- `internal/store/store.go`

**What it does**

- Holds a `map[id]*requestEntry` with:
  - `req *v1.UIRequest`
  - `done chan struct{}` closed on completion
- Implements:
  - `Create` (assign id, timestamps, status pending)
  - `Get`
  - `Pending` (used for WS replay)
  - `Complete` (sets output, status completed, completedAt)
  - `Wait` (blocks on `done` or ctx timeout)

**Important limitation**

There is no cleanup loop and no indexing by time/status beyond `Pending()`. There is no concept of “history” beyond “the requests still in the map”.

### 4) Frontend: WS client, Redux, and history panel

**Key files**

- `agent-ui-system/client/src/services/websocket.ts`
  - Connects to `/ws?sessionId=...`
  - On `new_request`: sets active request, shows browser notification
  - On `request_completed`: completes the active request or adds to history
- `agent-ui-system/client/src/store/store.ts`
  - `requestSlice` holds `{ active, history, loading }`
  - `completeRequest` and `addToHistory` unshift into `history`
- `agent-ui-system/client/src/pages/Home.tsx`
  - Renders “REQUEST_HISTORY” via `history.map(...)` inside a `ScrollArea`

**Critical behavior**

- History is *only* built from events observed while connected.
- There is no API fetch for older completed requests.
- The history array is unbounded (memory growth over long sessions).

## Feature 1: Bounded, scrollable, paginated history

### What “scrollable” means in this codebase

The history panel already uses `ScrollArea`, so it is *visually* scrollable today.

The missing pieces are:

- **bounded size** in Redux memory (avoid unbounded growth)
- **pagination** backed by server storage so the user can browse older entries

### Minimal bounded history (UI-only, no backend changes)

If you only want “limit in size”, you can implement this immediately:

- Add a constant like `MAX_HISTORY = 100` and, after `unshift`, truncate:

```ts
state.history.unshift(req);
state.history = state.history.slice(0, MAX_HISTORY);
```

This does not provide real pagination; it is just an in-memory cap.

### True pagination (requires server-side history API + storage)

To paginate, you need:

1) server stores completed requests durably (disk/DB)
2) server exposes list endpoint(s)
3) UI fetches pages and renders them (virtualized list optional)

**Proposed endpoint contract**

```text
GET /api/requests?status=completed&limit=50&before=<RFC3339Nano or cursor>
-> { items: UIRequest[], nextCursor: string | null }
```

Or cursor-based:

```text
GET /api/requests/history?limit=50&cursor=<opaque>
```

**Why cursor-based usually wins**

- stable ordering even when new requests arrive
- avoids time-based fencepost bugs

**Files impacted**

- Go server: `internal/server/server.go` (new handler)
- Store: likely replace/extend `internal/store/store.go` with a persistent store interface
- UI: add a data-fetch layer to `agent-ui-system/client/src/pages/Home.tsx` (or a dedicated service)

## Feature 2–4: Request metadata (origin + environment)

### What metadata should accomplish

Metadata should answer: “Where did this request come from?” and “How should I interpret it?”

Concrete examples:

- PWD (`/home/user/project`)
- parent PIDs (ppid chain) for correlating to the calling tool
- hostname, username
- command line / tool name (optional)
- git info (optional; can be expensive)
- “agent run id” / “conversation id” if plz-confirm is used by an LLM agent runner

### Where metadata lives in the current model

Right now, metadata is *implicit* and scattered:

- Some “origin context” is encoded in widget inputs (the `title`, `message`, etc.).
- Server-side request envelope fields (`createdAt`, `expiresAt`, `id`) give temporal correlation, but not process correlation.
- There is no structured field that records anything about:
  - the agent process that created the request
  - the working directory or repo context
  - the OS process tree
  - the program version / build that emitted the request

So if you want metadata, you will add it as a first-class field in the request envelope.

### The best extension point: the `UIRequest` envelope

For new features that are “about the request itself” (not “about a particular widget”), the correct home is `proto/plz_confirm/v1/request.proto` inside `message UIRequest`.

Why this is the best place:

- It is automatically delivered everywhere the request goes:
  - REST responses (Go server uses `protojson`)
  - WebSocket broadcasts (`marshalWSEvent` wraps `protojson(UIRequest)`)
  - CLI wait responses (`internal/client/client.go` unmarshals into `v1.UIRequest`)
- It lets you display metadata in a uniform way in history, without per-widget duplication.
- It keeps widget `*Input` / `*Output` messages focused on the user interaction itself.

### Proposed schema: `RequestMetadata`

There are many valid metadata shapes; the key is to pick a minimal core and allow extension. In protobuf, a good compromise is:

- typed, explicit “first-class” fields for the common things you know you’ll want
- plus a small “bag” for extra key/value tags (so experiments don’t require proto churn)

Pseudocode protobuf sketch (illustrative, not final):

```proto
// proto/plz_confirm/v1/request.proto
message RequestMetadata {
  // Process / invocation context
  optional string cwd = 1;
  optional int64 pid = 2;
  optional int64 ppid = 3;
  repeated int64 parent_pids = 4; // optional: full chain (linux via /proc)

  // Identity / host
  optional string hostname = 5;
  optional string username = 6;

  // Tooling
  optional string client_name = 7;     // "plz-confirm"
  optional string client_version = 8;  // e.g. v0.1.14
  optional string command = 9;         // "confirm", "select", ...

  // Transport perspective (can be set server-side too)
  optional string remote_addr = 10;
  optional string user_agent = 11;

  // Escape hatch
  map<string,string> tags = 100;
}

message UIRequest {
  ...
  optional RequestMetadata metadata = 21;
}
```

Design notes:

- PIDs are modeled as `int64` for simplicity. In JSON (`protojson`), 64-bit integers may appear as strings; that’s expected and already noted in existing playbooks (`ttmp/2025/12/25/.../playbook/01-test-inspection-playbook-post-protobuf-migration.md`).
- If you need “structured tags” later, you can introduce additional messages or use `google.protobuf.Struct`. A `map<string,string>` is intentionally constrained to keep the surface area small.

### Where metadata is captured and how it flows

This feature spans three layers:

1) **CLI collects client metadata** at request creation time.
2) **Server stores metadata** on the `UIRequest` record and emits it via WS/REST.
3) **UI displays metadata** in history (and optionally in the active widget header).

#### CLI capture points

The CLI creates requests in `internal/cli/*.go` by calling:

- `internal/client/client.go: CreateRequest(ctx, CreateRequestParams{...})`

So metadata capture belongs near `CreateRequestParams`, because:

- it avoids repeating capture logic in every widget command
- it lets you add “global CLI flags” later (e.g. `--meta-tag key=value`)

Candidate values and how to get them (Go):

```go
cwd, _ := os.Getwd()
pid := os.Getpid()
ppid := os.Getppid()
host, _ := os.Hostname()
user := os.Getenv("USER") // or os/user for richer info
```

For full parent PID chains, on Linux you can walk `/proc/<pid>/stat` until PID 1. (This is platform-specific; decide whether you want a best-effort implementation or a strict Linux-only implementation.)

#### Server capture points (and enrichment)

The Go server’s create handler is:

- `internal/server/server.go: handleCreateRequest`

It parses JSON into:

- `internal/server/server.go: createRequestBody`

and then calls:

- `internal/server/proto_convert.go: createUIRequestFromJSON(type, sessionId, input, timeout)`

To accept metadata, you would:

- extend `createRequestBody` to include an optional `metadata` field
- extend `createUIRequestFromJSON` to map that JSON object into `UIRequest.metadata`

Separately, the server can enrich metadata with transport details that the client cannot reliably know:

- `RemoteAddr`: `r.RemoteAddr`
- `User-Agent`: `r.Header.Get("User-Agent")`

If you do server-side enrichment, define a merge rule:

- server overwrites fields it considers authoritative (remote addr)
- server preserves client-provided fields (cwd, pid)

#### UI display points

The UI currently displays history entries in:

- `agent-ui-system/client/src/pages/Home.tsx`

You can display metadata:

- compactly (icons + tooltip, or a “details” drawer)
- or inline (show `cwd` and “client command” under the request title)

The payload is already delivered; once protobuf field exists and codegen runs, it will be present on the `UIRequest` object in TS as `request.metadata`.

## Feature 5: Default result + timeout (server-side auto-complete)

### What this feature means operationally

The request author (the agent/CLI) wants to say:

> “If the user hasn’t responded after N seconds, treat it as if they selected X.”

This is distinct from the existing flags:

- `--timeout` (server-side expiration / TTL hint) currently only sets `expiresAt`
- `--wait-timeout` (CLI patience) controls how long the agent waits before giving up locally

Today, the Go server does **not** enforce `expiresAt`, does **not** auto-complete, and does **not** broadcast expiry events. So a request can sit “pending” forever in the in-memory store unless someone submits a response.

### Why auto-completion belongs on the server

If you try to do “auto default selection” on the client side:

- the CLI process may exit or crash (no completion)
- the browser may not be open (no completion)
- the semantics become inconsistent across clients

If you do it server-side:

- there is one authoritative place that decides completion
- the result is broadcast via WS to all browsers (history stays consistent)
- the CLI long-poll unblocks reliably

### Proposed envelope fields: `auto_complete` + `completion`

You need two distinct concepts:

1) **What should happen if time passes** (policy)
2) **What actually happened** (audit trail)

In protobuf terms, this usually becomes:

- `AutoCompletePolicy` (stored on the request)
- `CompletionInfo` (set when request leaves pending)

Sketch:

```proto
enum CompletionKind {
  completion_kind_unspecified = 0;
  user_submitted = 1;
  auto_default = 2;
  expired = 3;
  server_error = 4;
}

message AutoCompletePolicy {
  // When to auto-complete. You can model this as:
  // - absolute timestamp, or
  // - duration from created_at.
  //
  // Absolute timestamps avoid “created_at parsing” on the server.
  optional string auto_complete_at = 1; // RFC3339Nano

  // Default output payload in widget-specific shape
  oneof default_output {
    ConfirmOutput confirm_output = 10;
    SelectOutput select_output = 11;
    FormOutput form_output = 12;
    UploadOutput upload_output = 13;
    TableOutput table_output = 14;
    ImageOutput image_output = 15;
  }
}

message CompletionInfo {
  CompletionKind kind = 1;
  optional bool used_default = 2;
  optional string note = 3;
}

message UIRequest {
  ...
  optional AutoCompletePolicy auto_complete = 22;
  optional CompletionInfo completion = 23;
}
```

Notes:

- Yes, this duplicates the output oneof fields. That’s normal: you need a place to store “default output” separately from “actual output”.
- Alternatively, you can model “default selection” in each `*Input` message, but that scatters logic across widgets and complicates generic history rendering.

### Wire contract: how the CLI supplies defaults

You have a few viable options:

#### Option A: Extend the existing create-request REST body

Current create body (Go server):

```json
{
  "type": "confirm",
  "sessionId": "global",
  "input": { ... },
  "timeout": 300
}
```

Proposed extension:

```jsonc
{
  "type": "confirm",
  "sessionId": "global",
  "input": { ... },

  // existing
  "timeout": 300,

  // new: metadata + auto-complete policy
  "metadata": { "cwd": "/home/...", "pid": 123, ... },
  "autoComplete": {
    "autoCompleteAt": "2026-01-03T18:15:00.000000000Z",
    "defaultOutput": { "approved": true, "timestamp": "..." }
  }
}
```

Implementation impact:

- extend `internal/server/server.go:createRequestBody`
- extend `internal/server/proto_convert.go:createUIRequestFromJSON`
  - parse `autoCompleteAt`
  - convert `defaultOutput` JSON into the widget’s output proto (`convertJSONOutputToProto`)

#### Option B: Add a “v2” endpoint that accepts protojson(UIRequest)

This is more invasive but conceptually cleaner:

```text
POST /api/v2/requests
body: protojson(UIRequest)  // including input + metadata + auto_complete
```

The tradeoff:

- fewer “legacy shape” adapters long-term
- but larger migration, and you now have two create paths until you delete v1

Given the repo’s current posture (“keep legacy REST shape”), Option A is a better fit.

### Server behavior: scheduling auto-complete

Today, `store.Store` is purely in-memory and only reacts to explicit `Complete` calls.

To auto-complete, the server needs a background scheduler. At a high level:

```text
on Create(request):
  if request.autoCompleteAt is set:
    schedule timer for that time
    when timer fires:
      if request still pending:
        complete request with default output (kind=auto_default)

periodic cleanup:
  if now > request.expiresAt and still pending:
    mark timeout (kind=expired, status=timeout)
```

There are two common implementation patterns:

1) **Timer per request**
   - simple
   - but can create many goroutines/timers if the system is busy

2) **Single scheduler loop**
   - maintain a min-heap by deadline
   - one goroutine wakes up at next deadline
   - scales better and is easier to persist across restarts if you have a DB

Given that you also want *long-term history* (persistence), the scheduler should probably be coupled with your persistent store: on restart, pending requests should be reloaded and their timers re-established.

### UI behavior: show default and show that it was used

The requirement explicitly says:

> “the default selection is show in the web UI”

There are two UX moments:

1) While pending: show “Default in N seconds: X”
2) After completion: show “Completed automatically (default)” in history

Where to implement:

- Active widget header: `agent-ui-system/client/src/components/WidgetRenderer.tsx`
  - It already renders a request header line (`REQ_ID`, `TYPE`)
  - This is a natural place to render a small countdown badge
- Widget body: optionally show the default choice inline
  - e.g. for ConfirmDialog, highlight the default button
- History list: `agent-ui-system/client/src/pages/Home.tsx`
  - show a label like `AUTO_DEFAULT` if `req.completion?.kind === auto_default`
  - optionally show a short “result summary” derived from `req.output`

## Feature 1 revisited: long-term storage for history (what to build on the backend)

### Current state

The Go server creates `st := store.New()` in `cmd/plz-confirm/main.go` when you run `plz-confirm serve`.

That store is:

- purely in-memory
- created fresh on server start

So, by definition, there is no long-term history today.

### What “long-term” means for plz-confirm

In practice you want:

- “recent history” survives page refresh (browser reload)
- “older history” survives server restart (disk persistence)
- “paging” works without loading everything into memory

This means you need a persistent storage engine.

### Storage design options (ranked by simplicity)

#### Option 1: Append-only JSONL file + in-memory index

Shape:

- on completion, append a single JSON line containing `protojson(UIRequest)` to a file
- keep a small in-memory index (offsets) for paging

Pros:

- trivial to implement
- no external dependencies

Cons:

- compaction/rotation needed eventually
- concurrent writes need careful locking
- querying/filtering becomes “scan-ish” unless you index more

#### Option 2: SQLite (recommended for durability + querying)

The repo already includes `github.com/mattn/go-sqlite3` indirectly in `go.mod`, which suggests SQLite is already an acceptable dependency in this ecosystem.

Pros:

- durable, queryable, easy paging (`ORDER BY created_at DESC LIMIT ? OFFSET ?`)
- easy to filter by status/type/time
- can store `protojson` as text or store key columns + blob/json

Cons:

- `go-sqlite3` uses CGO; cross-compilation and static builds need care

If CGO is a non-starter, consider a pure-Go SQLite driver (not currently in deps) or a key-value store.

#### Option 3: Embedded KV store (BoltDB/Badger)

Pros:

- fast, pure Go (depending on library)
- good for append-only-ish event logs

Cons:

- introduces a new dependency
- querying/paging requires custom indexing

### Proposed “store interface” to enable persistence

Even if you start with in-memory and later switch to SQLite, the cleanest path is to introduce an interface in `internal/store` and provide implementations.

Concept sketch:

```go
type ListParams struct {
  Status   *v1.RequestStatus
  Type     *v1.WidgetType
  Limit    int
  Cursor   string // opaque cursor (preferred) or timestamp/id composite
}

type ListResult struct {
  Items      []*v1.UIRequest
  NextCursor string
}

type Store interface {
  Create(ctx context.Context, req *v1.UIRequest) (*v1.UIRequest, error)
  Get(ctx context.Context, id string) (*v1.UIRequest, error)
  Pending(ctx context.Context) []*v1.UIRequest
  Complete(ctx context.Context, id string, output *v1.UIRequest) (*v1.UIRequest, error)
  Wait(ctx context.Context, id string) (*v1.UIRequest, error)

  // New for history/paging
  List(ctx context.Context, p ListParams) (ListResult, error)
}
```

Then:

- `internal/store/memory` implements the current behavior
- `internal/store/sqlite` implements persistence + paging

### Proposed history endpoints

There are two reasonable choices:

1) Add `GET /api/requests` with query params
2) Add a dedicated `GET /api/history` endpoint

Choice (1) fits REST conventions and reuses the `/api/requests` collection.

Proposed API reference:

```text
GET /api/requests?status=completed&limit=50&cursor=<opaque>

200 OK
{
  "items": [ <UIRequest>, ... ],
  "nextCursor": "<opaque>" | ""
}
```

Filtering extensions you’ll probably want quickly:

- `type=confirm|select|...`
- `createdBefore=<RFC3339Nano>` (if not using opaque cursors)
- `createdAfter=<RFC3339Nano>`
- `sessionId=...` (if you ever reintroduce sessions)

### UI paging strategy (infinite scroll)

The current UI has a natural “history panel” container. With a list endpoint, you can implement:

- fetch first page on load
- append pages as user scrolls
- keep only the latest N in memory (still bounded) while older pages can be re-fetched

Pseudocode:

```ts
// on mount:
dispatch(historyLoadFirstPage())

// on scroll near bottom:
if (hasNextPage && !loading) dispatch(historyLoadNextPage())
```

In Redux, you’d typically store:

```ts
history: UIRequest[]
historyNextCursor: string | null
historyLoading: boolean
```

### “Bounded history” UX: what to do when the list caps

Even with server paging, it’s still worth capping the in-memory list to prevent pathological memory growth in long-lived browser tabs.

Two patterns work well:

1) **Hard cap + fetch-on-demand**
   - Keep only the newest N entries in `state.history`
   - If user scrolls past the oldest, fetch older pages again from the server

2) **Two-tier storage**
   - `recentHistory` (bounded, in memory)
   - `pagedHistory` (append-only pages loaded while browsing)

Given the current simplicity of the UI, a hard cap is the easiest: keep `history` bounded, but show a banner like:

> “Showing newest 100. Load older…”

and implement “Load older” as a button that fetches from the server and replaces/extends the list.

### UI performance notes (why virtualization may matter)

Right now, history rows render as a simple `.map`. For small N this is fine, but once you introduce paging you can easily reach 1k+ items.

If you see jank:

- use list virtualization (`react-virtual`, `react-window`, etc.)
- or enforce a strict UI cap (e.g. never keep more than 500 rows client-side)

Because this codebase already uses a `ScrollArea` component, adding virtualization may require swapping the scroll container or wiring custom scroll events; plan for that complexity if you go beyond a few hundred rows.

## Concrete implementation plan (per feature, per layer)

This section is an “engineering checklist” that maps each feature to the concrete files and symbols you will touch.

### A) Bounded history (UI-only quick win)

**Goal:** Prevent unbounded history growth immediately, without changing backend APIs.

**Files**

- `agent-ui-system/client/src/store/store.ts`
  - reducers: `completeRequest`, `addToHistory`

**Change sketch**

```ts
const MAX_HISTORY = 100;
state.history.unshift(req);
state.history = state.history.slice(0, MAX_HISTORY);
```

**Behavior**

- UI remains scrollable.
- Old entries eventually fall off.
- No “load older” (because there is no source of truth to fetch from yet).

### B) Request metadata (end-to-end)

**Goal:** Capture metadata at creation time and render it in history and/or active request UI.

**Backend schema**

- `proto/plz_confirm/v1/request.proto`:
  - add `RequestMetadata` message
  - add `UIRequest.metadata`
- run codegen:
  - `make proto` (Go)
  - `make ts-proto` (TS)

**Go server**

- `internal/server/server.go`
  - extend `createRequestBody` to include `Metadata any` (or `json.RawMessage`)
- `internal/server/proto_convert.go`
  - extend `createUIRequestFromJSON` to populate `req.Metadata`
  - consider server-side enrichment with `RemoteAddr` and `User-Agent`

**Go CLI**

- `internal/client/client.go`
  - extend `CreateRequestParams` to include `Metadata *v1.RequestMetadata`
  - include it in the JSON body sent to `POST /api/requests`
- optionally: add CLI flags for extra tags (future work)

**Frontend**

- `agent-ui-system/client/src/pages/Home.tsx`
  - render metadata in each history row (e.g. show `cwd` or a tooltip)
- `agent-ui-system/client/src/components/WidgetRenderer.tsx`
  - render small metadata badges on active request (optional)

### C) Long-term storage + pagination (backend + UI)

**Goal:** History survives browser refresh and server restart, and UI can page through it.

**Go store refactor**

- Introduce a `Store` interface (as described earlier).
- Implement a persistent store (SQLite or file-based).
- Ensure `Pending()` is still efficient for WS connect replay.

**Go server endpoints**

- `internal/server/server.go`
  - extend `handleRequestsCollection` to support `GET`
    - `GET /api/requests?status=completed&limit=...&cursor=...`
  - add response wrapper type:
    - `{ items: UIRequest[], nextCursor: string }`

**Frontend**

- Add a “history loader” to fetch first page on app load.
- Add infinite scroll or “Load older” button.
- Decide merge strategy with WS events:
  - new WS completions should prepend to history even if you have paged older items

**Edge cases**

- deduplication: if you fetch a page that includes a request you already saw via WS, avoid duplicates
- ordering: define “newest first” and stick to it everywhere

### D) Default result + timeout (server scheduler + UI display)

**Goal:** Requests can complete automatically with a default output after a deadline, and the UI shows that default was used.

**Schema**

- `proto/plz_confirm/v1/request.proto`:
  - add `AutoCompletePolicy` and `CompletionInfo`
  - add `UIRequest.auto_complete` and `UIRequest.completion`

**Create request contract**

- Extend `POST /api/requests` JSON body:
  - `autoComplete.autoCompleteAt` (timestamp)
  - `autoComplete.defaultOutput` (widget-shaped JSON)

**Server**

- On create:
  - store the policy on the request
  - schedule auto-complete
- On timer fire:
  - if still pending:
    - set `completion.kind=auto_default`
    - set `completion.used_default=true`
    - set status completed
    - set output to default
    - broadcast `request_completed`
- On expiry (if you also enforce `expiresAt`):
  - set status timeout
  - set `completion.kind=expired`
  - broadcast state transition (either reuse `request_completed` or introduce `request_expired`)

**Frontend**

- Active request:
  - show countdown to `autoCompleteAt`
  - show default choice (best-effort summary)
- History:
  - show `AUTO_DEFAULT` label (or similar) when completion kind indicates it
  - show the output summary so the “default selection” is visible

## Optional: A History TUI (why the ticket name might include “TUI”)

There is no terminal UI in the repo today, but the Go module includes Bubble Tea (`github.com/charmbracelet/bubbletea`) indirectly. If you want a terminal history viewer, the cleanest approach is:

- build it *on top of the same history list API* that powers the web UI
- keep it read-only (no side effects), so it’s safe to run anywhere

Proposed command:

```text
plz-confirm history [--base-url ...] [--limit 50] [--type confirm] [--status completed]
```

Implementation sketch:

- new cobra/glazed command under `internal/cli/history.go`
- bubbletea model that:
  - fetches first page
  - supports next/prev page
  - shows per-row summary (title/type/status/completion kind)

This TUI becomes especially useful once “metadata” exists: it can show detailed request origin context without needing the browser.

## Documents and playbooks you should read before implementing

This repo already contains excellent “institutional memory” in ticket docs. For this ticket, the most relevant are:

- `pkg/doc/adding-widgets.md`
  - The best “request lifecycle” diagram and the authoritative list of core handlers.
- `ttmp/2025/12/25/006-USE-PROTOBUF--unify-backend-frontend-shared-data-with-protobuf-codegen/analysis/01-backend-frontend-architecture-current-protocols.md`
  - A precise statement of current REST + WS contracts and how protojson is used.
- `ttmp/2025/12/25/006-USE-PROTOBUF--unify-backend-frontend-shared-data-with-protobuf-codegen/playbook/01-test-inspection-playbook-post-protobuf-migration.md`
  - The “what shapes changed in JSON” reference, especially oneof output payloads.
- `ttmp/2025/12/15/DESIGN-PLZ-CONFIRM-001--port-agent-ui-system-cli-backend-to-go-using-glazed-framework/analysis/01-code-structure-analysis-agent-ui-system.md`
  - Good background on the original Node architecture and the porting decisions (also calls out missing persistence/history explicitly).

## Appendix: Current API and implementation map (quick index)

### REST endpoints (Go server)

- `POST /api/requests` → `internal/server/server.go: handleCreateRequest`
- `GET /api/requests/{id}` → `internal/server/server.go: handleRequestsItem` (store.Get)
- `POST /api/requests/{id}/response` → `internal/server/server.go: handleSubmitResponse`
- `GET /api/requests/{id}/wait?timeout=` → `internal/server/server.go: handleWait`
- `POST /api/images` → `internal/server/server.go: handleImagesCollection`
- `GET /api/images/{id}` → `internal/server/server.go: handleImagesItem`

### WebSocket events

- `/ws` upgrade + pending replay: `internal/server/ws.go`
- event wrapper serialization: `internal/server/ws_events.go`
- event broadcasts:
  - `"new_request"` on create: `internal/server/server.go`
  - `"request_completed"` on completion: `internal/server/server.go`

### Frontend integration points

- WS client: `agent-ui-system/client/src/services/websocket.ts`
- Request state: `agent-ui-system/client/src/store/store.ts`
- Active widget routing: `agent-ui-system/client/src/components/WidgetRenderer.tsx`
- History panel: `agent-ui-system/client/src/pages/Home.tsx`

## Appendix: Canonical JSON payload shapes (create, response, and oneof outputs)

This appendix exists because most implementation bugs in plz-confirm changes come from subtle JSON shape mismatches:

- protojson enum emission as strings (by name)
- oneof output fields being explicit (`selectedSingle`, `selectedMulti`, …)
- 64-bit integer fields potentially appearing as JSON strings

The best “shape authority” is the generated protobuf types + protojson behavior, but these examples are convenient to copy/paste while developing.

### 1) Create request (all widgets)

```jsonc
POST /api/requests
{
  "type": "confirm",     // WidgetType enum name (string)
  "sessionId": "global", // tolerated; ignored by Go backend today
  "input": { /* widget-specific */ },
  "timeout": 300         // seconds; becomes expiresAt hint
}
```

### 2) Submit response (all widgets)

```jsonc
POST /api/requests/{id}/response
{
  "output": { /* widget-specific output shape */ }
}
```

### 3) WebSocket event wrapper (server → browser)

```jsonc
{
  "type": "new_request" | "request_completed",
  "request": { /* protojson(UIRequest) */ }
}
```

### 4) Widget-specific output shapes (proto oneofs)

These are the payload shapes expected by `POST /api/requests/{id}/response` and the shapes that can be reused for “defaultOutput” if you implement auto-default.

#### Confirm

```jsonc
{
  "approved": true,
  "timestamp": "2026-01-03T18:15:00.000Z",
  "comment": "optional"
}
```

#### Select (single)

```jsonc
{ "selectedSingle": "us-west-2", "comment": "optional" }
```

#### Select (multi)

```jsonc
{ "selectedMulti": { "values": ["a", "b"] }, "comment": "optional" }
```

#### Table (single)

```jsonc
{ "selectedSingle": { "id": 1, "name": "Alice" }, "comment": "optional" }
```

#### Table (multi)

```jsonc
{ "selectedMulti": { "values": [ { "id": 1 }, { "id": 2 } ] }, "comment": "optional" }
```

#### Form

```jsonc
{ "data": { "host": "db", "port": 5432 }, "comment": "optional" }
```

#### Upload

```jsonc
{
  "files": [
    { "name": "app.log", "size": 1234, "path": "/tmp/app.log", "mimeType": "text/plain" }
  ],
  "comment": "optional"
}
```

#### Image

Image outputs are a oneof with multiple “selected” variants. Examples:

```jsonc
{ "selectedBool": true, "timestamp": "2026-01-03T18:15:00.000Z", "comment": "optional" }
```

```jsonc
{ "selectedNumber": 0, "timestamp": "2026-01-03T18:15:00.000Z" }
```

```jsonc
{ "selectedNumbers": { "values": [0, 2] }, "timestamp": "2026-01-03T18:15:00.000Z" }
```

```jsonc
{ "selectedString": "Wrong theme", "timestamp": "2026-01-03T18:15:00.000Z" }
```

```jsonc
{ "selectedStrings": { "values": ["A", "B"] }, "timestamp": "2026-01-03T18:15:00.000Z" }
```

### 5) Note on 64-bit integer JSON encoding

Protojson encodes some 64-bit integer fields as JSON strings to preserve precision. If you see values like `"0"` instead of `0`, that may be expected.

This matters most for:

- image selection indexes (`selectedNumber`)
- sizes (`maxSize`, upload file sizes)

Existing guidance: `ttmp/2025/12/25/006-USE-PROTOBUF--unify-backend-frontend-shared-data-with-protobuf-codegen/playbook/01-test-inspection-playbook-post-protobuf-migration.md`
