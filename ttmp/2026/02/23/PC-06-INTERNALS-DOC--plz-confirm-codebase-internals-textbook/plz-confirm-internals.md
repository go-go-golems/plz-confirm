---
title: "plz-confirm Codebase Internals: A Complete Guide"
ticket: PC-06-INTERNALS-DOC
date: 2026-02-23
type: reference
topics:
  - internals
  - architecture
  - onboarding
---

# plz-confirm Codebase Internals

**A Complete Guide for New Engineers**

---

## Table of Contents

1. [What Is plz-confirm?](#chapter-1-what-is-plz-confirm)
2. [Architecture Overview](#chapter-2-architecture-overview)
3. [Project Structure & Build System](#chapter-3-project-structure--build-system)
4. [The Protocol Buffer Contract](#chapter-4-the-protocol-buffer-contract)
5. [The Go Backend Server](#chapter-5-the-go-backend-server)
6. [The In-Memory Store](#chapter-6-the-in-memory-store)
7. [WebSocket Broadcasting](#chapter-7-websocket-broadcasting)
8. [CLI Commands & the Glazed Framework](#chapter-8-cli-commands--the-glazed-framework)
9. [The HTTP Client & Long-Polling](#chapter-9-the-http-client--long-polling)
10. [The React Frontend](#chapter-10-the-react-frontend)
11. [Redux State Management](#chapter-11-redux-state-management)
12. [Widget Components Deep Dive](#chapter-12-widget-components-deep-dive)
13. [The Script Engine (Goja)](#chapter-13-the-script-engine-goja)
14. [Image Handling](#chapter-14-image-handling)
15. [Request Lifecycle: End-to-End Walkthrough](#chapter-15-request-lifecycle-end-to-end-walkthrough)
16. [Expiry, Touch, and Timeout Mechanics](#chapter-16-expiry-touch-and-timeout-mechanics)
17. [Development Environment Setup](#chapter-17-development-environment-setup)
18. [Testing Strategy](#chapter-18-testing-strategy)
19. [CI/CD & Release Pipeline](#chapter-19-cicd--release-pipeline)
20. [Key Design Decisions & Trade-offs](#chapter-20-key-design-decisions--trade-offs)

---

## Chapter 1: What Is plz-confirm?

plz-confirm is a **human-in-the-loop confirmation tool** for AI agents and automated scripts. When an agent (Claude, a deployment script, a CI pipeline) needs to ask a human a question -- "Should I deploy to production?", "Which server should I target?", "Please fill in these credentials" -- it uses plz-confirm to present a rich UI dialog and wait for the response.

### The Core Idea

```
Agent/Script                     Human Operator
    |                                 |
    |-- plz-confirm confirm           |
    |   --title "Deploy?"             |
    |                                 |
    |       [HTTP POST]               |
    |   ---------------------->       |
    |                          [Browser shows dialog]
    |                                 |
    |                          [Human clicks "Approve"]
    |   <----------------------       |
    |       [HTTP Response]           |
    |                                 |
    |-- receives: approved=true       |
    |-- continues execution           |
```

### Widget Types

plz-confirm supports seven widget types, each designed for a different kind of human input:

| Widget | Purpose | Example Use Case |
|--------|---------|-----------------|
| **confirm** | Yes/No binary decision | "Deploy to production?" |
| **select** | Pick from a list | "Which environment?" |
| **form** | Fill in structured data | "Enter DB credentials" |
| **table** | Select rows from data | "Which servers to patch?" |
| **upload** | Provide files | "Upload the config file" |
| **image** | Visual selection/approval | "Which design do you prefer?" |
| **script** | Multi-step interactive workflow | Complex approval pipelines |

### The Single Binary

The final artifact is a **single Go binary** that bundles:
- The HTTP/WebSocket **server**
- The **CLI commands** for agents to call
- An embedded **React web application** for the browser UI

One binary does everything. No separate frontend server needed.

---

## Chapter 2: Architecture Overview

### Three-Layer Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                       CLI LAYER (Go)                        │
│  plz-confirm confirm/select/form/table/upload/image         │
│  Glazed framework for parsing + output formatting           │
│  HTTP client for API calls + long-poll waiting              │
└──────────────────────────┬──────────────────────────────────┘
                           │ HTTP (JSON/protojson)
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                     SERVER LAYER (Go)                       │
│  net/http router with REST API                              │
│  In-memory request store with done channels                 │
│  WebSocket broadcaster (gorilla/websocket)                  │
│  Script engine (goja JavaScript VM)                         │
│  Image store (temp files on disk)                           │
│  Embedded React SPA (go:embed)                              │
└──────┬─────────────────────────────────┬────────────────────┘
       │ WebSocket                       │ HTTP
       ▼                                 ▼
┌─────────────────────────────────────────────────────────────┐
│                   FRONTEND LAYER (React)                    │
│  React 19 + Redux Toolkit + Tailwind CSS                    │
│  WebSocket client for real-time updates                     │
│  Widget components (ConfirmDialog, SelectDialog, ...)       │
│  Cyberpunk-themed UI (shadcn/ui + Radix)                    │
└─────────────────────────────────────────────────────────────┘
```

### Communication Patterns

There are exactly three communication channels:

1. **CLI -> Server**: HTTP POST to create requests, HTTP GET to long-poll for results
2. **Server -> Browser**: WebSocket push for new requests and completions
3. **Browser -> Server**: HTTP POST to submit responses

The WebSocket is **unidirectional** (server-to-client). The browser never sends WebSocket messages -- it uses HTTP POST for all writes.

### The Protocol Buffer Contract

All data structures are defined in `.proto` files and code-generated for both Go and TypeScript. This is the **single source of truth** for the data model. If you change a proto file, you regenerate code for both sides.

---

## Chapter 3: Project Structure & Build System

### Directory Layout

```
plz-confirm/
├── cmd/plz-confirm/           # CLI entry point (main.go)
│   ├── main.go                # Cobra root command + all subcommands
│   └── ws.go                  # WebSocket debug utility command
│
├── internal/
│   ├── cli/                   # CLI command implementations
│   │   ├── confirm.go         # plz-confirm confirm
│   │   ├── select.go          # plz-confirm select
│   │   ├── form.go            # plz-confirm form
│   │   ├── table.go           # plz-confirm table
│   │   ├── upload.go          # plz-confirm upload
│   │   └── image.go           # plz-confirm image
│   │
│   ├── client/                # HTTP client for talking to server
│   │   └── client.go          # CreateRequest, WaitRequest, UploadImage
│   │
│   ├── server/                # HTTP/WebSocket backend
│   │   ├── server.go          # Main server: routes, handlers, lifecycle
│   │   ├── ws.go              # WebSocket broadcaster
│   │   ├── ws_events.go       # WebSocket event serialization
│   │   ├── images.go          # Image upload/serve store
│   │   ├── cors.go            # CORS middleware
│   │   ├── embed.go           # go:embed for frontend assets
│   │   └── embed_none.go      # No-op embed for dev builds
│   │
│   ├── store/                 # In-memory request store
│   │   └── store.go           # Create, Get, Wait, Complete, Expire
│   │
│   ├── metadata/              # Process metadata collection
│   │   └── metadata.go        # CWD, PID, parent process chain
│   │
│   └── scriptengine/          # JavaScript VM (goja)
│       └── engine.go          # InitAndView, UpdateAndView
│
├── proto/                     # Protocol Buffer definitions
│   ├── plz_confirm/v1/
│   │   ├── request.proto      # UIRequest envelope + metadata
│   │   ├── widgets.proto      # All widget input/output types
│   │   └── image.proto        # Image-specific types
│   └── generated/go/          # Generated Go protobuf code
│
├── agent-ui-system/           # React frontend (whole SPA)
│   ├── client/src/
│   │   ├── main.tsx           # React DOM entry point
│   │   ├── App.tsx            # Root: providers, router, error boundary
│   │   ├── index.css          # Tailwind + cyberpunk theme
│   │   ├── components/
│   │   │   ├── WidgetRenderer.tsx    # Dynamic widget dispatcher
│   │   │   ├── widgets/             # One file per widget type
│   │   │   ├── ui/                  # shadcn/ui component library
│   │   │   └── Layout.tsx           # Page chrome
│   │   ├── store/store.ts           # Redux store (3 slices)
│   │   ├── services/
│   │   │   ├── websocket.ts         # WS connection + submit helpers
│   │   │   └── notifications.ts     # Browser notification API
│   │   ├── hooks/                   # Custom React hooks
│   │   └── proto/generated/         # Generated TypeScript types
│   ├── package.json
│   ├── vite.config.ts
│   └── tsconfig.json
│
├── pkg/doc/                   # User-facing documentation
├── Makefile                   # Build automation
├── go.mod / go.sum            # Go dependencies
└── .goreleaser.yaml           # Release configuration
```

### Build System

The project uses a **Makefile** as the primary build orchestrator:

```makefile
# Key targets:
make build          # Full build: codegen + ui-build + go build (embed tag)
make dev-backend    # Run Go server on :3001
make dev-frontend   # Run Vite dev server on :3000
make dev-tmux       # Both in tmux panes
make codegen        # Regenerate all proto code (Go + TS)
make test           # Go tests
make ci             # Full CI: lint + test + frontend-check
```

**The build flow for a production binary:**

```
1. make codegen
   └── protoc generates Go code (proto/generated/go/)
   └── ts-proto generates TS code (agent-ui-system/client/src/proto/generated/)

2. make ui-build
   └── cd agent-ui-system && pnpm install && pnpm build
   └── Vite builds React app → agent-ui-system/dist/public/

3. go build -tags embed ./cmd/plz-confirm
   └── embed.go uses //go:embed to bundle dist/public/ into binary
   └── Single binary with frontend baked in
```

**Development mode** runs two processes:
- `make dev-frontend`: Vite on `:3000` with HMR, proxies `/api` and `/ws` to `:3001`
- `make dev-backend`: Go server on `:3001`

### Key Technology Choices

| Layer | Technology | Why |
|-------|-----------|-----|
| CLI Framework | Cobra + Glazed | Glazed adds structured output (JSON/YAML/CSV/table) |
| HTTP Server | net/http (stdlib) | Simple, no framework needed |
| WebSocket | gorilla/websocket | De facto standard Go WebSocket lib |
| Data Contract | Protocol Buffers v3 | Type-safe cross-language schema |
| JavaScript VM | Goja | In-process JS execution, no Node.js needed |
| Frontend | React 19 + Vite | Modern, fast HMR |
| State | Redux Toolkit | Predictable state for complex widget queue |
| Styling | Tailwind CSS + Radix | Utility-first + accessible primitives |
| Components | shadcn/ui | Copy-paste component library |

---

## Chapter 4: The Protocol Buffer Contract

### Why Protobuf?

Protocol Buffers are the **single source of truth** for every data structure that crosses a boundary (CLI -> Server, Server -> Frontend). By defining types once in `.proto` files, we get:

- Type-safe Go structs (generated by `protoc-gen-go`)
- Type-safe TypeScript interfaces (generated by `ts-proto`)
- JSON serialization via `protojson` (camelCase field names)
- Forward/backward compatibility with optional fields

### The Core Message: UIRequest

**File: `proto/plz_confirm/v1/request.proto`**

```protobuf
message UIRequest {
  string id = 1;                          // UUID, set by server
  WidgetType type = 2;                    // Which widget
  string session_id = 3;                  // Scoping for WS

  // Input: exactly ONE of these is set (the "question")
  oneof input {
    ConfirmInput confirm_input = 4;
    SelectInput select_input = 5;
    FormInput form_input = 6;
    UploadInput upload_input = 7;
    TableInput table_input = 8;
    ImageInput image_input = 9;
    ScriptInput script_input = 10;
  }

  // Output: exactly ONE of these is set (the "answer")
  oneof output {
    ConfirmOutput confirm_output = 11;
    SelectOutput select_output = 12;
    FormOutput form_output = 13;
    UploadOutput upload_output = 14;
    TableOutput table_output = 15;
    ImageOutput image_output = 16;
    ScriptOutput script_output = 17;
  }

  RequestStatus status = 18;             // pending -> completed/timeout/error
  string created_at = 19;                // RFC3339Nano
  optional string completed_at = 20;
  string expires_at = 21;                // Server-side deadline
  optional string error = 22;
  optional RequestMetadata metadata = 23; // Process info
  optional string touched_at = 24;        // First UI interaction
  optional bool expiry_disabled = 25;     // Touch disables expiry
}
```

**The `oneof` pattern is critical.** A UIRequest carries either an input (the question) or an output (the answer), never both simultaneously during creation. The `type` field tells you which oneof variant to expect.

### Widget Type Enum

```protobuf
enum WidgetType {
  widget_type_unspecified = 0;
  confirm = 1;
  select = 2;
  form = 3;
  upload = 4;
  table = 5;
  image = 6;
  script = 7;
}
```

### Widget Input/Output Pairs

Each widget type has a matching input/output pair:

**Confirm:**
```protobuf
message ConfirmInput {
  string title = 1;
  optional string message = 2;
  optional string approve_text = 3;
  optional string reject_text = 4;
}

message ConfirmOutput {
  bool approved = 1;
  string timestamp = 2;
  optional string comment = 3;
}
```

**Select:**
```protobuf
message SelectInput {
  string title = 1;
  repeated string options = 2;
  optional bool multi = 3;
  optional bool searchable = 4;
}

message SelectOutput {
  oneof selected {
    string selected_single = 1;
    SelectOutputMulti selected_multi = 2;
  }
  optional string comment = 3;
}
```

**Form:**
```protobuf
message FormInput {
  string title = 1;
  optional google.protobuf.Struct schema = 2;  // JSON Schema
}

message FormOutput {
  optional google.protobuf.Struct data = 1;    // Submitted values
  optional string comment = 2;
}
```

The `google.protobuf.Struct` type is how we represent **arbitrary JSON** in protobuf. The form schema is a JSON Schema document, and the form data is whatever the user submitted. Both travel as `Struct` values.

**Table, Upload, Image** follow the same input/output pattern. Each has their own message types in `widgets.proto`.

### JSON Wire Format

On the wire, protobuf messages are serialized using `protojson`, which produces **camelCase** field names:

```json
{
  "id": "abc123",
  "type": "confirm",
  "sessionId": "global",
  "confirmInput": {
    "title": "Deploy to production?",
    "message": "This will affect all users."
  },
  "status": "pending",
  "createdAt": "2026-02-23T10:00:00.000000000Z",
  "expiresAt": "2026-02-23T10:05:00.000000000Z"
}
```

---

## Chapter 5: The Go Backend Server

### File: `internal/server/server.go`

The server is the heart of the system. It handles HTTP requests, manages WebSocket connections, and orchestrates the request lifecycle.

### Server Struct

```go
type Server struct {
    store            *store.Store          // In-memory request storage
    ws               *wsBroadcaster        // WebSocket session manager
    images           *ImageStore           // Temporary image file store
    scripts          *scriptengine.Engine   // JavaScript VM
    scriptEventLocks *keyedLock            // Per-request locks for script events
}
```

### HTTP Route Table

```go
func (s *Server) Handler() http.Handler {
    mux := http.NewServeMux()

    mux.HandleFunc("/ws", s.handleWS)                    // WebSocket upgrade
    mux.HandleFunc("/api/images", s.handleImagesCollection)   // POST: upload
    mux.HandleFunc("/api/images/", s.handleImagesItem)        // GET: serve
    mux.HandleFunc("/api/requests", s.handleRequestsCollection) // POST: create
    mux.HandleFunc("/api/requests/", s.handleRequestsItem)      // GET/POST: various

    s.handleStaticFiles(mux)  // Embedded React SPA (production only)

    return withCORS(mux)
}
```

The request item handler dispatches on the sub-path:

| Path | Method | Handler | Purpose |
|------|--------|---------|---------|
| `/api/requests` | POST | `handleCreateRequest` | Create new request |
| `/api/requests/{id}` | GET | `handleRequestsItem` | Fetch request by ID |
| `/api/requests/{id}/response` | POST | `handleSubmitResponse` | Submit user's answer |
| `/api/requests/{id}/event` | POST | `handleScriptEvent` | Submit script widget event |
| `/api/requests/{id}/touch` | POST | `handleTouch` | Mark first interaction |
| `/api/requests/{id}/wait` | GET | `handleWait` | Long-poll for completion |

### Server Lifecycle

The server runs multiple goroutines via `errgroup`:

```go
func (s *Server) ListenAndServe(ctx context.Context, opts Options) error {
    g, gctx := errgroup.WithContext(ctx)

    // Goroutine 1: Expiry ticker (every 1 second)
    // Checks for expired requests and auto-completes them
    g.Go(func() error {
        t := time.NewTicker(1 * time.Second)
        for {
            select {
            case <-gctx.Done(): return nil
            case <-t.C:
                expired := s.store.Expire(time.Now().UTC())
                for _, req := range expired {
                    // Broadcast timeout completion to WS clients
                    s.ws.BroadcastRawJSON(req.SessionId, msg)
                }
            }
        }
    })

    // Goroutine 2: Image cleanup ticker (every 30 seconds)
    g.Go(func() error { ... })

    // Goroutine 3: HTTP server
    g.Go(func() error {
        return srv.ListenAndServe()
    })

    // Goroutine 4: Graceful shutdown on context cancellation
    g.Go(func() error {
        <-gctx.Done()
        srv.Shutdown(shutdownCtx)
        return nil
    })

    return g.Wait()
}
```

### Request Creation Handler

This is the most important handler. Here's what happens step by step:

```go
func (s *Server) handleCreateRequest(w http.ResponseWriter, r *http.Request) {
    // 1. Read body (limited to 1MB)
    bodyBytes, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))

    // 2. Unmarshal protojson into UIRequest
    reqProto := &v1.UIRequest{}
    protojson.Unmarshal(bodyBytes, reqProto)

    // 3. Validate: type must be set, input oneof must be populated
    //    and the input type must match the declared widget type
    inputType, _ := widgetTypeFromInputOneof(reqProto)
    if inputType != reqProto.Type { /* error */ }

    // 4. Special handling for script widgets:
    //    Run init() + view() in the JavaScript VM
    if reqProto.Type == v1.WidgetType_script {
        initResult, _ := s.scripts.InitAndView(r.Context(), seededInput)
        reqProto.ScriptState = scriptState
        reqProto.ScriptView = scriptView
    }

    // 5. Inject metadata (remote addr, user agent)
    if reqProto.Metadata != nil || r.RemoteAddr != "" {
        reqProto.Metadata.RemoteAddr = &r.RemoteAddr
        reqProto.Metadata.UserAgent = &r.UserAgent()
    }

    // 6. Store the request (generates UUID, sets timestamps)
    req, _ := s.store.Create(r.Context(), reqProto)

    // 7. Broadcast to WebSocket clients in the session
    s.ws.BroadcastRawJSON(req.SessionId, marshalWSEvent("new_request", req))

    // 8. Return created request as protojson (201 Created)
    writeProtoJSON(w, http.StatusCreated, req)
}
```

### Response Submission Handler

```go
func (s *Server) handleSubmitResponse(w http.ResponseWriter, r *http.Request, id string) {
    // 1. Fetch existing request to know the widget type
    existingReq, _ := s.store.Get(r.Context(), id)

    // 2. Parse incoming response body
    incoming := &v1.UIRequest{}
    protojson.Unmarshal(bodyBytes, incoming)

    // 3. Validate output type matches input type
    outputType, _ := widgetTypeFromOutputOneof(incoming)
    if outputType != existingReq.Type { /* error */ }

    // 4. Complete the request in the store
    req, _ := s.store.Complete(r.Context(), id, outputReq)

    // 5. Broadcast completion to WS clients
    s.ws.BroadcastRawJSON(req.SessionId, marshalWSEvent("request_completed", req))

    // 6. Return completed request (200 OK)
    writeProtoJSON(w, http.StatusOK, req)
}
```

### The Wait Handler (Long-Poll)

The wait handler is how the CLI blocks until a human responds:

```go
func (s *Server) handleWait(w http.ResponseWriter, r *http.Request, id string) {
    // Parse timeout from query string (default 60s)
    timeoutS := 60
    if raw := r.URL.Query().Get("timeout"); raw != "" {
        timeoutS, _ = strconv.Atoi(raw)
    }

    // Create a context with that timeout
    ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeoutS)*time.Second)
    defer cancel()

    // Block until request completes or timeout
    req, err := s.store.Wait(ctx, id)
    if errors.Is(err, store.ErrWaitTimeout) {
        // 408 tells the CLI to retry
        http.Error(w, "timeout waiting for response", http.StatusRequestTimeout)
        return
    }

    writeProtoJSON(w, http.StatusOK, req)
}
```

---

## Chapter 6: The In-Memory Store

### File: `internal/store/store.go`

The store is an **in-memory map** of request entries. It provides thread-safe CRUD operations and an event-driven wait mechanism.

### The requestEntry

```go
type requestEntry struct {
    req      *v1.UIRequest  // The protobuf request
    done     chan struct{}   // Closed when request completes
    doneOnce sync.Once      // Ensures done channel is closed exactly once
}
```

The `done` channel is the key to the wait mechanism. When a request is created, the channel is open. When the request completes, the channel is closed. Anyone blocked on `<-e.done` unblocks immediately.

### Store Operations

**Create**: Generates UUID, sets timestamps, creates done channel:
```go
func (s *Store) Create(_ context.Context, req *v1.UIRequest) (*v1.UIRequest, error) {
    id := uuid.NewString()
    reqCopy := &v1.UIRequest{
        Id:        id,
        Status:    v1.RequestStatus_pending,
        CreatedAt: now.Format(time.RFC3339Nano),
        ExpiresAt: now.Add(timeout).Format(time.RFC3339Nano),
        // ... copy input, metadata, etc
    }
    s.requests[id] = &requestEntry{
        req:  reqCopy,
        done: make(chan struct{}),
    }
    return reqCopy, nil
}
```

**Wait**: Blocks on the done channel with context cancellation:
```go
func (s *Store) Wait(ctx context.Context, id string) (*v1.UIRequest, error) {
    e := s.requests[id]
    // Already completed? Return immediately.
    if e.req.Status == v1.RequestStatus_completed { return e.req, nil }

    select {
    case <-e.done:      // Request completed
        return s.Get(ctx, id)
    case <-ctx.Done():  // Timeout or cancellation
        return nil, ErrWaitTimeout
    }
}
```

**Complete**: Sets output, closes the done channel:
```go
func (s *Store) Complete(_ context.Context, id string, output *v1.UIRequest) (*v1.UIRequest, error) {
    e := s.requests[id]
    e.req.Output = output.Output
    e.req.Status = v1.RequestStatus_completed
    e.doneOnce.Do(func() { close(e.done) })  // Unblocks all waiters
    return e.req, nil
}
```

**Expire**: Called every second by the server's ticker goroutine:
```go
func (s *Store) Expire(now time.Time) []*v1.UIRequest {
    var expired []*v1.UIRequest
    for _, e := range s.requests {
        if e.req.Status != v1.RequestStatus_pending { continue }
        if e.req.ExpiryDisabled { continue }
        if now.Before(expAt) { continue }

        // Auto-complete with default values
        setDefaultOutputFor(e.req, now, &autoComment)
        e.req.Status = v1.RequestStatus_completed
        e.doneOnce.Do(func() { close(e.done) })
        expired = append(expired, e.req)
    }
    return expired
}
```

**Touch**: Called on first user interaction, disables expiry:
```go
func (s *Store) Touch(_ context.Context, id string, now time.Time) (*v1.UIRequest, error) {
    e := s.requests[id]
    e.req.ExpiryDisabled = &true
    e.req.TouchedAt = &now.Format(time.RFC3339Nano)
    return e.req, nil
}
```

### Default Outputs on Timeout

When a request times out, it gets a **default output** that makes sense for the widget type:

- **Confirm**: `approved: false` (default deny)
- **Select (single)**: First option selected
- **Select (multi)**: Empty array
- **Form**: Empty object `{}`
- **Upload**: Empty files array
- **Table**: Empty selection
- **Image (confirm)**: `false`
- **Image (select)**: First image or first option

The comment field is set to `"AUTO_TIMEOUT"` so downstream code can detect that no human actually responded.

---

## Chapter 7: WebSocket Broadcasting

### File: `internal/server/ws.go`

### The wsBroadcaster

The broadcaster manages WebSocket connections grouped by **session ID**:

```go
type wsBroadcaster struct {
    mu               sync.Mutex
    writeMu          sync.Mutex
    clientsBySession map[string]map[*websocket.Conn]struct{}  // sessionID -> set of conns
    sessionByConn    map[*websocket.Conn]string               // conn -> sessionID
}
```

This double-map allows efficient lookup in both directions: broadcast to a session, or clean up when a connection drops.

### Connection Lifecycle

When a browser connects via WebSocket:

```go
func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
    sessionID := r.URL.Query().Get("sessionId")  // e.g., "global"

    // 1. Upgrade HTTP to WebSocket
    conn, _ := wsUpgrader.Upgrade(w, r, nil)

    // 2. Register in broadcaster
    s.ws.add(sessionID, conn)

    // 3. Send all currently pending requests (catch-up)
    pending := s.store.PendingForSession(r.Context(), sessionID)
    for _, req := range pending {
        s.ws.writeText(conn, marshalWSEvent("new_request", req))
    }

    // 4. Read loop (we don't expect client messages, but must read to detect close)
    for {
        if _, _, err := conn.ReadMessage(); err != nil {
            conn.Close()
            s.ws.remove(conn)
            return
        }
    }
}
```

Step 3 is critical: when a browser tab reconnects (e.g., after a network blip), it receives all pending requests it might have missed.

### Broadcasting

Broadcasting is session-scoped. A message for session "deployment-123" only goes to browsers connected with that session ID:

```go
func (b *wsBroadcaster) BroadcastRawJSON(sessionID string, msg []byte) {
    conns := b.snapshot(sessionID)  // Copy slice to avoid holding lock
    for _, c := range conns {
        c.SetWriteDeadline(time.Now().Add(5 * time.Second))
        if err := b.writeText(c, msg); err != nil {
            c.Close()
            b.remove(c)  // Drop misbehaving clients
        }
    }
}
```

### WebSocket Event Format

```json
{
  "type": "new_request",
  "request": { /* full UIRequest as protojson */ }
}

{
  "type": "request_completed",
  "request": { /* full UIRequest with output populated */ }
}

{
  "type": "request_updated",
  "request": { /* full UIRequest with updated state (e.g., script view change) */ }
}
```

---

## Chapter 8: CLI Commands & the Glazed Framework

### The Glazed Framework

**Glazed** (from `github.com/go-go-golems/glazed`) is a CLI framework built on top of Cobra that adds:
- Declarative parameter definitions with types, defaults, and validation
- Automatic structured output formatting (JSON, YAML, CSV, ASCII table)
- Middleware pipeline for output processing

### Command Structure Pattern

Every CLI command follows this pattern:

```go
// 1. Settings struct with glazed tags
type ConfirmSettings struct {
    BaseURL     string  `glazed.parameter:"base-url"`
    SessionID   string  `glazed.parameter:"session-id"`
    TimeoutS    int     `glazed.parameter:"timeout"`
    WaitTimeout int     `glazed.parameter:"wait-timeout"`
    Title       string  `glazed.parameter:"title"`
    Message     *string `glazed.parameter:"message"`
}

// 2. Command description with Glazed field definitions
func NewConfirmCommand() (*ConfirmCommand, error) {
    return &ConfirmCommand{
        CommandDescription: cmds.NewCommandDescription(
            "confirm",
            cmds.WithShort("Request a yes/no confirmation"),
            cmds.WithFlags(
                fields.New("title", fields.TypeString,
                    fields.WithRequired(true),
                    fields.WithHelp("Dialog title")),
                fields.New("timeout", fields.TypeInteger,
                    fields.WithDefault(300),
                    fields.WithHelp("Timeout in seconds")),
                // ... more fields
            ),
        ),
    }, nil
}

// 3. RunIntoGlazeProcessor: the command body
func (c *ConfirmCommand) RunIntoGlazeProcessor(
    ctx context.Context,
    parsedValues *values.Values,
    gp middlewares.Processor,  // Output sink
) error {
    settings := &ConfirmSettings{}
    parsedValues.DecodeSectionInto(schema.DefaultSlug, settings)

    // Create HTTP client
    cl := client.New(settings.BaseURL)

    // Create request on server
    created, _ := cl.CreateRequest(ctx, client.CreateRequestParams{
        Type:      v1.WidgetType_confirm,
        SessionID: settings.SessionID,
        TimeoutS:  settings.TimeoutS,
        Input:     &v1.ConfirmInput{Title: settings.Title, Message: settings.Message},
    })

    // Wait for human response (blocks!)
    completed, _ := cl.WaitRequest(ctx, created.Id, settings.WaitTimeout)

    // Extract output
    out := completed.GetConfirmOutput()

    // Emit structured row
    row := types.NewRow(
        types.MRP("request_id", created.Id),
        types.MRP("approved", out.GetApproved()),
        types.MRP("timestamp", out.GetTimestamp()),
        types.MRP("comment", comment),
    )
    return gp.AddRow(ctx, row)
}
```

### Output Formatting

Glazed automatically handles output formatting based on flags:

```bash
# Default output (YAML for non-TTY, table for TTY)
plz-confirm confirm --title "Deploy?"

# Explicit JSON output
plz-confirm confirm --title "Deploy?" --output json

# CSV for piping
plz-confirm confirm --title "Deploy?" --output csv
```

The `gp.AddRow()` call feeds data into Glazed's middleware pipeline, which formats it according to the `--output` flag.

### All Commands at a Glance

| Command | Required Flags | Key Outputs |
|---------|---------------|-------------|
| `confirm` | `--title` | `approved` (bool), `comment` |
| `select` | `--title`, `--option` (repeatable) | `selected_json` (string or array) |
| `form` | `--title`, `--schema` (file or stdin) | `data_json` (object) |
| `table` | `--title`, `--data` (file or stdin) | `selected_json` (object or array) |
| `upload` | `--title` | `file_name`, `file_size`, `file_path`, `mime_type` |
| `image` | `--title`, `--image` (repeatable) | `selected_json` (varies by mode) |

Common flags shared by all: `--base-url`, `--session-id`, `--timeout`, `--wait-timeout`.

---

## Chapter 9: The HTTP Client & Long-Polling

### File: `internal/client/client.go`

The HTTP client is used by CLI commands to talk to the server.

### CreateRequest

```go
func (c *Client) CreateRequest(ctx context.Context, p CreateRequestParams) (*v1.UIRequest, error) {
    // 1. Build UIRequest proto with input oneof
    reqProto := &v1.UIRequest{
        Type:      p.Type,
        SessionId: p.SessionID,
    }
    // Set input based on type (e.g., confirmInput, selectInput)

    // 2. Collect process metadata
    metadata := metadata.Collect()  // CWD, PID, parent chain

    // 3. Marshal to protojson
    body, _ := protojson.Marshal(reqProto)

    // 4. POST /api/requests
    resp, _ := c.HTTPClient.Do(req)

    // 5. Parse response
    return out, nil
}
```

### WaitRequest (Long-Poll Loop)

The CLI can't use WebSocket (it's a simple CLI tool), so it **long-polls**:

```go
func (c *Client) WaitRequest(ctx context.Context, id string, waitTimeoutS int) (*v1.UIRequest, error) {
    pollTimeoutS := 25  // Each poll cycle is 25 seconds

    for {
        // Build request with poll timeout
        url := fmt.Sprintf("%s/api/requests/%s/wait?timeout=%d", c.BaseURL, id, pollTimeoutS)

        // HTTP timeout is poll timeout + 5s buffer
        httpCtx, cancel := context.WithTimeout(ctx, (pollTimeoutS+5) * time.Second)
        resp, err := c.HTTPClient.Do(req.WithContext(httpCtx))
        cancel()

        switch resp.StatusCode {
        case 200:
            // Got a result! Parse and return.
            return out, nil
        case 408:
            // Server-side poll timeout. Retry.
            continue
        default:
            return nil, errors.Errorf("unexpected status: %d", resp.StatusCode)
        }
    }
}
```

The long-poll pattern:
1. CLI sends GET `/api/requests/{id}/wait?timeout=25`
2. Server blocks for up to 25 seconds
3. If request completes during that window -> 200 with result
4. If timeout -> 408 (client retries immediately)
5. If overall `--wait-timeout` exceeded -> CLI exits with error

This is an efficient pattern: no busy-waiting, no polling every second. The server blocks the HTTP connection until there's something to report.

---

## Chapter 10: The React Frontend

### Entry Point: `agent-ui-system/client/src/main.tsx`

```tsx
ReactDOM.createRoot(document.getElementById("root")!).render(
  <ErrorBoundary>
    <Provider store={store}>
      <App />
    </Provider>
  </ErrorBoundary>
);
```

### App Component: `App.tsx`

The App component sets up providers and routing:

```tsx
function App() {
  useEffect(() => {
    connectWebSocket();                              // Start WS connection
    browserNotificationService.requestPermission();  // Ask for browser notifications
  }, []);

  return (
    <ThemeProvider defaultTheme="dark">
      <TooltipProvider>
        <Router>
          <Route path="/" component={Home} />
          <Route path="/:rest*" component={NotFound} />
        </Router>
        <Toaster />
      </TooltipProvider>
    </ThemeProvider>
  );
}
```

### Home Page: The Dashboard

The Home page has two main areas:

1. **WidgetRenderer**: Shows the currently active request widget
2. **Request History**: Shows completed/timed-out requests

The layout uses a responsive grid. On mobile, it's a single column. On desktop, the widget takes 8 columns and history takes 4.

### Cyberpunk Theme

The UI has a distinctive cyberpunk/terminal aesthetic, implemented entirely through CSS:

```css
/* Key design tokens (OKLch color space) */
:root.dark {
  --primary: oklch(0.75 0.18 145);     /* Terminal green */
  --background: oklch(0.1 0.01 145);   /* Deep dark */
  --foreground: oklch(0.9 0.02 145);   /* Light green-tinted */
  --border: oklch(0.3 0.02 145);       /* Subtle green border */
}

/* Fonts */
--font-mono: "IBM Plex Mono";
--font-display: "JetBrains Mono";

/* Custom component classes */
.cyber-card {
  border: 1px solid var(--border);
  position: relative;
  overflow: hidden;
}
.cyber-button {
  border-radius: 0;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-weight: bold;
}
.cyber-input {
  background: var(--input);
  border: none;
}
```

Visual effects: scanline overlay, vignette, grid background pattern, all in pure CSS.

---

## Chapter 11: Redux State Management

### File: `agent-ui-system/client/src/store/store.ts`

The Redux store has **three slices**:

### 1. Session Slice

Tracks WebSocket connection state:

```typescript
interface SessionState {
  id: string | null;      // From URL ?sessionId= or "global"
  connected: boolean;     // WebSocket connected?
  reconnecting: boolean;
  error: string | null;
}
```

### 2. Request Slice (The Important One)

Manages the request queue:

```typescript
interface RequestState {
  active: UIRequest | null;   // Currently displayed widget
  pending: UIRequest[];       // Queue of waiting requests
  history: UIRequest[];       // Completed requests (max 512)
  loading: boolean;
}
```

**Key reducers:**

**`enqueueRequest`**: Smart queue with deduplication:
```typescript
enqueueRequest: (state, action) => {
  const incoming = action.payload;

  // Deduplicate: skip if already active, pending, or in history
  if (state.active?.id === incoming.id) return;
  if (state.pending.some(r => r.id === incoming.id)) return;
  if (state.history.some(r => r.id === incoming.id)) return;

  // If nothing active, show immediately
  if (!state.active) {
    state.active = incoming;
    return;
  }
  // Otherwise queue it
  state.pending.push(incoming);
}
```

**`completeRequest`**: Moves to history, promotes next pending:
```typescript
completeRequest: (state, action) => {
  const completedReq = action.payload;

  // Remove from active
  if (state.active?.id === completedReq.id) {
    state.active = null;
  }
  // Remove from pending
  state.pending = state.pending.filter(r => r.id !== completedReq.id);

  // Add to history (most recent first)
  state.history.unshift(completedReq);

  // Promote next pending to active
  if (!state.active && state.pending.length > 0) {
    state.active = state.pending.shift() ?? null;
  }
}
```

**`patchRequest`**: In-place updates (e.g., script view changes):
```typescript
patchRequest: (state, action) => {
  const { id, ...patch } = action.payload;
  if (state.active?.id === id) {
    state.active = { ...state.active, ...patch };
  }
  // Also patch in pending and history arrays
}
```

### 3. Notification Slice

Simple list of UI notifications (toast messages).

### How State Flows

```
WebSocket receives "new_request"
  → normalizeUIRequest(data.request)
  → dispatch(enqueueRequest(request))
  → If no active widget: state.active = request
  → WidgetRenderer re-renders, showing the widget

User clicks "Approve" in ConfirmDialog
  → submitResponse(requestId, type, output)
  → HTTP POST /api/requests/{id}/response
  → Server broadcasts "request_completed"
  → WebSocket receives "request_completed"
  → dispatch(completeRequest(completedReq))
  → state.active = null, next pending promoted
  → WidgetRenderer re-renders with next widget (or SYSTEM_IDLE)
```

---

## Chapter 12: Widget Components Deep Dive

### File: `agent-ui-system/client/src/components/WidgetRenderer.tsx`

WidgetRenderer is the **orchestrator**. It reads `active` from Redux and dispatches to the right widget component:

```tsx
const renderWidget = () => {
  switch (active.type) {
    case WidgetType.confirm:
      return <ConfirmDialog input={active.confirmInput} onSubmit={handleSubmit} />;
    case WidgetType.select:
      return <SelectDialog input={active.selectInput} onSubmit={handleSubmit} />;
    case WidgetType.form:
      return <FormDialog input={active.formInput} onSubmit={handleSubmit} />;
    case WidgetType.table:
      return <TableDialog input={active.tableInput} onSubmit={handleSubmit} />;
    case WidgetType.upload:
      return <UploadDialog input={active.uploadInput} onSubmit={handleSubmit} />;
    case WidgetType.image:
      return <ImageDialog input={active.imageInput} onSubmit={handleSubmit} />;
    case WidgetType.script:
      return renderScriptView();  // Special multi-step handling
  }
};
```

### WidgetRenderer Also Handles:

1. **Expiry countdown**: Calculates remaining time, shows progress bar
2. **First interaction detection**: On first pointer/key event, calls `touchRequest()` to disable expiry
3. **Script progress**: Shows step progress bar for script widgets
4. **Script toasts**: Displays toast notifications from script steps
5. **SYSTEM_IDLE state**: Shows animated waiting screen when no active request

### Widget Component Contract

Every widget component receives:
- `input`: The proto input type (e.g., `ConfirmInput`)
- `onSubmit(output)`: Callback to submit the response
- `requestId`: The request UUID
- `loading`: Boolean for submit-in-progress state

### ConfirmDialog

The simplest widget. Two buttons: approve and reject.

```tsx
const ConfirmDialog = ({ input, onSubmit }) => {
  const [comment, setComment] = useState("");
  const [submitting, setSubmitting] = useState<string | null>(null);

  const handleSubmit = async (approved: boolean) => {
    setSubmitting(approved ? "approve" : "reject");
    await onSubmit({
      approved,
      timestamp: new Date().toISOString(),
      comment: comment || undefined,
    });
  };

  return (
    <div>
      <h2>{input.title}</h2>
      {input.message && <p>{input.message}</p>}
      <OptionalComment value={comment} onChange={setComment} />
      <Button onClick={() => handleSubmit(false)}>
        {input.rejectText || "REJECT"}
      </Button>
      <Button onClick={() => handleSubmit(true)}>
        {input.approveText || "APPROVE"}
      </Button>
    </div>
  );
};
```

### SelectDialog

Renders a list of options with optional search filtering:

- Single select: click an option, then submit
- Multi select: checkboxes for each option
- Searchable: text input filters the list

### FormDialog

The most complex static widget. Renders form fields from a JSON Schema:

1. Parses the `schema.properties` to determine fields
2. Renders appropriate input for each type:
   - `string` -> text input (or textarea, password, email based on format)
   - `number` -> number input with min/max
   - `boolean` -> checkbox
   - `string` with `enum` -> select dropdown
3. Client-side validation: required, min/maxLength, pattern, min/max
4. Collects all values into a single object on submit

### TableDialog

Displays tabular data with:
- Column headers (auto-derived from data keys or explicit)
- Sort by clicking headers
- Search filter across all columns
- Single or multi-row selection (checkboxes)

### UploadDialog

File upload with:
- Drag-and-drop zone
- File type validation against `accept` list
- File size validation against `maxSize`
- Single or multiple file support
- Progress display per file

### ImageDialog

Two modes:
- **Select mode**: Click images to select (supports multi-select)
- **Confirm mode**: View images, click approve/reject

Responsive grid layout adapts to image count (1-3 columns).

### OptionalComment

A shared sub-component used by all widgets. It's a collapsible textarea for adding an optional comment to any response. Uses IME composition handling for CJK input.

---

## Chapter 13: The Script Engine (Goja)

### File: `internal/scriptengine/engine.go`

The script engine runs **JavaScript code inside the Go process** using Goja (a Go implementation of ECMAScript 5.1). This enables complex multi-step workflows without requiring Node.js.

### The Script Contract

A script must export four functions:

```javascript
// 1. describe(): Return metadata about the script
exports.describe = function(ctx) {
  return {
    name: "deployment-workflow",
    version: "1.0.0",
    capabilities: ["confirmation", "selection"]
  };
};

// 2. init(): Return initial state
exports.init = function(ctx) {
  return {
    step: 0,
    selectedEnv: null,
    confirmed: false
  };
};

// 3. view(state): Return what widget to show
exports.view = function(state, ctx) {
  if (state.step === 0) {
    return {
      widgetType: "confirm",
      input: { title: "Start deployment?" },
      progress: { current: 1, total: 3, label: "Confirmation" }
    };
  }
  if (state.step === 1) {
    return {
      widgetType: "select",
      input: {
        title: "Choose environment",
        options: ["production", "staging", "development"]
      },
      progress: { current: 2, total: 3 }
    };
  }
  // Final step: return done
  return { done: true, result: { env: state.selectedEnv } };
};

// 4. update(state, event): Process user input, return new state
exports.update = function(state, ctx, event) {
  if (event.type === "submit") {
    return {
      ...state,
      step: state.step + 1,
      selectedEnv: event.data?.selectedSingle || state.selectedEnv
    };
  }
  if (event.type === "back") {
    return { ...state, step: Math.max(0, state.step - 1) };
  }
  return state;
};
```

### Engine Flow

**On request creation** (`InitAndView`):
1. Create a new Goja VM
2. Load the script source
3. Call `describe()` -> get metadata
4. Call `init()` -> get initial state
5. Call `view(state)` -> get first widget to show
6. Store state + view in the UIRequest

**On user interaction** (`UpdateAndView`):
1. Reload the script in a fresh VM
2. Call `update(state, event)` -> get new state
3. Call `view(newState)` -> get next widget
4. If `view()` returns `{done: true}` -> complete the request
5. Otherwise -> update state + view, broadcast to UI

### Script View Sections

A script view can contain multiple **sections**. Display sections (read-only markdown) precede the interactive widget:

```javascript
return {
  sections: [
    { widgetType: "display", input: { content: "## Instructions\nRead carefully." } },
    { widgetType: "confirm", input: { title: "Proceed?" } }
  ],
  progress: { current: 2, total: 5 }
};
```

The frontend renders all display sections, then the single interactive widget. Exactly one interactive section is allowed per step.

### Timeout Protection

Scripts run with a configurable timeout (default 2 seconds):
```go
func runWithTimeout(ctx context.Context, vm *goja.Runtime, timeout time.Duration, fn func() error) error {
    done := make(chan error, 1)
    go func() { done <- fn() }()

    timer := time.NewTimer(timeout)
    select {
    case err := <-done:
        timer.Stop()
        return err
    case <-timer.C:
        vm.Interrupt("execution timeout")
        return errors.New("script execution timed out")
    case <-ctx.Done():
        vm.Interrupt("context cancelled")
        return ctx.Err()
    }
}
```

---

## Chapter 14: Image Handling

### File: `internal/server/images.go`

### Image Store

The server maintains a temporary file store for uploaded images:

```go
type ImageStore struct {
    mu             sync.RWMutex
    dir            string              // Temp directory (auto-created)
    maxUploadBytes int64               // 50MB default
    images         map[string]StoredImage
}

type StoredImage struct {
    ID        string
    Path      string     // File path on disk
    MimeType  string     // e.g., "image/png"
    Size      int64
    CreatedAt time.Time
    ExpiresAt time.Time  // Auto-cleanup deadline
}
```

### Upload Flow

1. Client uploads via multipart POST to `/api/images`
2. Server sniffs first 512 bytes to detect MIME type
3. Validates it's `image/*` (rejects non-image uploads)
4. Streams to a temp file on disk (UUID filename)
5. Returns `{id, url, mimeType, size}`
6. URL format: `/api/images/{id}`

### Serving

Images are served with:
- Correct `Content-Type` header
- Conservative caching: `private, max-age=60`
- Lazy expiry check on access (deletes if expired)

### Cleanup

A background goroutine runs every 30 seconds, deleting expired images from both the map and disk.

### CLI Image Sources

The image command accepts three source types:
- **Local file path**: Auto-uploaded to `/api/images` via multipart POST
- **HTTP(S) URL**: Used directly (no upload needed)
- **Data URI**: `data:image/png;base64,...` used directly

There's special handling for data URIs: the CLI flag parser splits on commas, which breaks base64 data URIs. The code detects split pairs and re-joins them.

---

## Chapter 15: Request Lifecycle: End-to-End Walkthrough

Let's trace a complete confirm request from start to finish.

### Step 1: Agent Runs CLI Command

```bash
plz-confirm confirm \
  --title "Deploy v2.1?" \
  --message "This deploys to all prod servers" \
  --approve-text "DEPLOY" \
  --reject-text "ABORT" \
  --timeout 600
```

### Step 2: CLI Creates Request

The CLI command (`internal/cli/confirm.go`):
1. Parses flags into `ConfirmSettings`
2. Creates HTTP client pointing at `http://localhost:3000`
3. Builds `ConfirmInput` proto
4. Collects process metadata (CWD, PID, parent processes)
5. Sends `POST /api/requests`:

```json
{
  "type": "confirm",
  "sessionId": "global",
  "confirmInput": {
    "title": "Deploy v2.1?",
    "message": "This deploys to all prod servers",
    "approveText": "DEPLOY",
    "rejectText": "ABORT"
  },
  "expiresAt": "2026-02-23T10:10:00.000Z",
  "metadata": {
    "cwd": "/home/deploy/app",
    "self": { "pid": 12345, "comm": "plz-confirm", "argv": ["confirm", "--title", "..."] },
    "parents": [{ "pid": 12344, "comm": "bash" }]
  }
}
```

### Step 3: Server Processes Creation

The server (`internal/server/server.go`):
1. Validates type + input match
2. Injects remote_addr and user_agent from HTTP headers
3. Calls `store.Create()`:
   - Generates UUID: `"a1b2c3d4-..."`
   - Sets status: `pending`
   - Sets timestamps: `createdAt`, `expiresAt`
   - Creates done channel
4. Broadcasts to WebSocket:

```json
{
  "type": "new_request",
  "request": { /* full UIRequest */ }
}
```

5. Returns 201 with the created request

### Step 4: CLI Starts Long-Polling

The CLI (`internal/client/client.go`):
1. Receives the created request with its ID
2. Starts long-poll loop:
   - `GET /api/requests/a1b2c3d4-.../wait?timeout=25`
   - Server blocks for up to 25 seconds
   - Returns 408 if no response yet
   - CLI retries immediately

### Step 5: Browser Receives Request

The frontend (`services/websocket.ts`):
1. WebSocket `onmessage` fires with `new_request` event
2. Calls `normalizeUIRequest()` to fix protojson quirks
3. Dispatches `enqueueRequest(request)` to Redux
4. Since no widget was active: `state.active = request`
5. Shows browser notification: "Deploy v2.1? (confirm)"

### Step 6: Widget Renders

The WidgetRenderer:
1. Reads `active` from Redux
2. Sees `type: confirm`
3. Renders `<ConfirmDialog input={active.confirmInput} />`
4. Shows title, message, DEPLOY button, ABORT button

### Step 7: First Interaction (Touch)

When the user first clicks or presses a key:
1. `onPointerDownCapture` fires on the wrapper div
2. Calls `touchRequest(active.id)`
3. HTTP `POST /api/requests/{id}/touch`
4. Server sets `expiryDisabled = true`, `touchedAt = now`
5. Response patches Redux state
6. Expiry countdown disappears from UI

### Step 8: User Clicks DEPLOY

The ConfirmDialog:
1. Calls `onSubmit({ approved: true, timestamp: "...", comment: "" })`
2. WidgetRenderer's `handleSubmit`:
   - Calls `submitResponse(active.id, WidgetType.confirm, output)`
   - HTTP `POST /api/requests/{id}/response`:

```json
{
  "type": "confirm",
  "sessionId": "global",
  "confirmOutput": {
    "approved": true,
    "timestamp": "2026-02-23T10:02:15.123Z"
  }
}
```

### Step 9: Server Completes Request

The server:
1. Validates output type matches input type
2. Calls `store.Complete()`:
   - Sets output on the request
   - Sets status = `completed`
   - Closes the done channel (unblocks CLI)
3. Broadcasts `request_completed` to WebSocket
4. Returns 200 with completed request

### Step 10: CLI Receives Result

The client's WaitRequest:
1. The blocked `GET /wait` unblocks (done channel closed)
2. Returns 200 with the completed request
3. CLI extracts `ConfirmOutput`
4. Emits to Glazed processor:

```yaml
request_id: a1b2c3d4-...
approved: true
timestamp: "2026-02-23T10:02:15.123Z"
comment: ""
```

### Step 11: Agent Continues

The calling script receives the YAML/JSON output and continues execution based on the `approved` value.

### Step 12: Frontend Updates

1. WebSocket receives `request_completed`
2. Redux `completeRequest()`:
   - `active = null`
   - Request moves to history
   - Next pending promoted to active (if any)
3. WidgetRenderer shows `SYSTEM_IDLE` animation

---

## Chapter 16: Expiry, Touch, and Timeout Mechanics

### Three Timeout Concepts

1. **Server-side expiry** (`--timeout`): How long the request lives before auto-completing
2. **CLI wait timeout** (`--wait-timeout`): How long the CLI waits before giving up
3. **Touch**: First user interaction disables server-side expiry

### Expiry Flow

```
Request created with expiresAt = now + 300s
  │
  ├─ Every 1 second, server's ticker checks:
  │   Is now >= expiresAt AND status == pending AND !expiryDisabled?
  │     YES → Auto-complete with default output + comment "AUTO_TIMEOUT"
  │     NO  → Continue waiting
  │
  ├─ User opens browser, sees countdown: "EXPIRES_IN: 04:32"
  │
  ├─ User clicks/types (first interaction):
  │   POST /api/requests/{id}/touch
  │   → expiryDisabled = true
  │   → Countdown disappears from UI
  │   → Request lives until user submits or disconnects
  │
  └─ User submits response:
      POST /api/requests/{id}/response
      → status = completed, done channel closed
```

### Why Touch?

Without touch, a user might be reading a complex form when the timeout expires, auto-submitting empty data. Touch says "a human is looking at this -- don't time it out."

### Default Outputs

When a request auto-times out, it gets sensible defaults:
- Confirm: `approved: false` (safe default: don't proceed)
- Select: first option (or empty array for multi-select)
- Form: empty object
- Upload: no files
- Table: empty selection

All timeout outputs include `comment: "AUTO_TIMEOUT"` so callers can distinguish timeout from human rejection.

---

## Chapter 17: Development Environment Setup

### Prerequisites

- Go 1.25+ (the go.mod specifies the version)
- Node.js 20+ with pnpm 10.4+
- `protoc` (Protocol Buffer compiler)
- `protoc-gen-go` (Go code generator)
- `buf` CLI (Protocol Buffer linting)

### Quick Start

```bash
# Clone and enter
cd plz-confirm

# Install frontend dependencies
cd agent-ui-system && pnpm install && cd ..

# Generate protobuf code
make codegen

# Start development servers (requires tmux)
make dev-tmux
# OR start them separately:
make dev-backend   # Go server on :3001
make dev-frontend  # Vite on :3000 (proxies to :3001)

# Open browser to http://localhost:3000
```

### Development Architecture

```
Browser (:3000)
    │
    ├── Static files: Vite serves React app with HMR
    │
    ├── /api/*: Vite proxies to Go backend (:3001)
    │
    └── /ws: Vite proxies WebSocket to Go backend (:3001)

Vite Dev Server (:3000)        Go Backend (:3001)
    │                              │
    ├── Serves React              ├── REST API
    ├── Hot module reload         ├── WebSocket
    └── Proxies API/WS ──────────└── Request store
```

### Testing a Widget

```bash
# In one terminal: start servers
make dev-tmux

# In another terminal: send a confirm request
./dist/linux_amd64/plz-confirm confirm \
  --title "Test Dialog" \
  --message "Does this work?" \
  --base-url http://localhost:3000

# Open browser, see the dialog, click approve
# CLI unblocks and prints result
```

### Building for Production

```bash
make build
# Binary: ./dist/linux_amd64/plz-confirm

# Run with embedded UI:
./dist/linux_amd64/plz-confirm serve --addr :3000
```

---

## Chapter 18: Testing Strategy

### Go Tests

```bash
make test  # Runs: go test ./... -count=1
```

Key test files:
- `internal/server/script_test.go`: Script engine integration tests
- `internal/server/ws_test.go`: WebSocket handler tests
- `internal/server/images_test.go`: Image upload/serve tests
- `internal/server/server_static_test.go`: Embedded SPA serving tests

### Frontend Tests

```bash
cd agent-ui-system
pnpm run check    # TypeScript type checking (tsc --noEmit)
npx vitest        # Unit tests with Vitest
```

Key test files:
- `WidgetRenderer.test.ts`: Widget dispatch logic
- `FormDialog.test.tsx`: Form validation
- `SelectDialog.test.tsx`: Selection mechanics
- `TableDialog.test.tsx`: Sort/filter/select
- `homeRequestHistoryDisplay.test.ts`: History display logic

### Smoke Tests

```bash
# Start server, then run:
./scripts/curl-inspector-smoke.sh
# Sends various curl requests to test all widget types
```

### CI Pipeline

The `make ci` target runs:
1. `buf lint .` - Protocol Buffer schema linting
2. `go test ./...` - All Go tests
3. `make frontend-check` - TypeScript type checking

---

## Chapter 19: CI/CD & Release Pipeline

### CI Workflows (GitHub Actions)

**Push workflow** (`.github/workflows/push.yml`):
- Triggers on push to main and all PRs
- Installs Go, pnpm, Node, protoc, buf
- Verifies generated code is up-to-date: `make codegen && git diff --exit-code`
- Runs `make ci` (lint + test + type check)
- Runs `make build` (full production build)

**Lint workflow** (`.github/workflows/lint.yml`):
- Runs `golangci-lint` for Go code quality
- Must build UI first (go:embed requires the files to exist)

**Security workflows**:
- CodeQL analysis for code scanning
- Dependency vulnerability scanning
- Secret detection

### Release Pipeline

**Triggered by**: Git tag push (`v*`) or manual dispatch

**Three stages**:

```
Stage 1: UI Pre-build
  └── Build React app, upload as CI artifact "ui-embed"

Stage 2a: Linux Build (goreleaser)
  ├── Download ui-embed artifact
  ├── Build linux/amd64 with CGO (native)
  └── Build linux/arm64 with cross-compiler

Stage 2b: macOS Build (goreleaser)
  ├── Download ui-embed artifact
  ├── Build darwin/amd64
  └── Build darwin/arm64

Stage 3: Merge & Release
  ├── Merge linux + darwin artifacts
  ├── GPG sign checksums
  └── Publish to:
      ├── GitHub Releases (with checksums.txt)
      ├── Homebrew tap (go-go-golems/homebrew-go-go-go)
      ├── Fury.io package repository
      └── Linux packages (deb, rpm)
```

### Installing

```bash
# Homebrew
brew install go-go-golems/go-go-go/plz-confirm

# Or download from GitHub releases
```

---

## Chapter 20: Key Design Decisions & Trade-offs

### 1. Single Binary with Embedded Frontend

**Decision**: Bundle the React SPA into the Go binary via `go:embed`.

**Why**: Users get a single binary with zero dependencies. No Node.js, no separate frontend server, no static file directory.

**Trade-off**: Build is more complex (must build frontend before Go). Development requires two processes.

### 2. In-Memory Store (No Database)

**Decision**: All state lives in a Go map with mutexes.

**Why**: plz-confirm is a transient tool. Requests are short-lived (minutes, not days). There's no need for persistence, replication, or query capabilities.

**Trade-off**: State is lost on server restart. Not suitable for multi-node deployment without a shared store.

### 3. Long-Poll Instead of WebSocket for CLI

**Decision**: CLI uses HTTP long-poll, not WebSocket.

**Why**: CLIs are simpler with HTTP. Long-poll is easy to implement with `curl` for debugging. WebSocket adds complexity (reconnect logic, framing) that isn't needed for a simple "wait for response" pattern.

**Trade-off**: Slightly less efficient than WebSocket (new TCP connection per poll cycle), but the poll timeout (25s) keeps overhead minimal.

### 4. Protocol Buffers for Schema

**Decision**: Use proto3 as the data schema, with protojson for wire format.

**Why**: Type safety across Go and TypeScript. Single source of truth. Forward/backward compatible.

**Trade-off**: Extra build step (code generation). Developers must understand proto3 syntax.

### 5. Goja for Script Engine

**Decision**: Use Goja (Go-based JavaScript VM) instead of Node.js or WASM.

**Why**: In-process execution with no external dependencies. No need to install Node.js. Scripts run in milliseconds.

**Trade-off**: Only ES5.1 support (no async/await, arrow functions, etc.). Limited standard library.

### 6. Session-Scoped WebSocket

**Decision**: WebSocket connections are grouped by session ID.

**Why**: Multiple agents can run simultaneously without interfering. A deployment pipeline sees only its own requests.

**Trade-off**: The "global" default session means all unsessioned requests are visible to all browsers.

### 7. Touch to Disable Expiry

**Decision**: First user interaction disables the countdown timer.

**Why**: Prevents timeout while user is actively working on a complex form. Balances "don't wait forever" with "don't rush the human."

**Trade-off**: A touched request with no response stays pending indefinitely until the server restarts.

### 8. Default Deny on Timeout

**Decision**: Confirm widget defaults to `approved: false` on timeout.

**Why**: Safety first. If nobody responds, don't proceed with the dangerous action.

**Trade-off**: Agents must handle the timeout case and decide whether to retry.

---

## Appendix A: File Reference

### Most Important Files to Read First

| Priority | File | What You'll Learn |
|----------|------|-------------------|
| 1 | `proto/plz_confirm/v1/request.proto` | The entire data model |
| 2 | `internal/store/store.go` | How requests are stored and waited on |
| 3 | `internal/server/server.go` | HTTP handlers, the glue |
| 4 | `internal/client/client.go` | How CLI talks to server |
| 5 | `internal/cli/confirm.go` | Pattern for all CLI commands |
| 6 | `agent-ui-system/client/src/store/store.ts` | Frontend state management |
| 7 | `agent-ui-system/client/src/services/websocket.ts` | Frontend-server communication |
| 8 | `agent-ui-system/client/src/components/WidgetRenderer.tsx` | Widget dispatch |
| 9 | `internal/server/ws.go` | WebSocket session management |
| 10 | `internal/scriptengine/engine.go` | JavaScript VM integration |

### Dependency Graph

```
cmd/plz-confirm/main.go
  ├── internal/cli/*.go (imports)
  │     └── internal/client/client.go
  │           └── internal/metadata/metadata.go
  ├── internal/server/server.go (imports)
  │     ├── internal/store/store.go
  │     ├── internal/server/ws.go
  │     ├── internal/server/images.go
  │     └── internal/scriptengine/engine.go
  └── proto/generated/go/ (imports)
        └── proto/plz_confirm/v1/*.proto (source of truth)
```

## Appendix B: Common Developer Tasks

### Adding a New Widget Type

See `pkg/doc/adding-widgets.md` for the full guide. Summary:

1. Add proto messages: `FooInput`, `FooOutput` in `widgets.proto`
2. Add to `UIRequest` oneof fields in `request.proto`
3. Add enum value in `WidgetType`
4. Run `make codegen`
5. Add CLI command: `internal/cli/foo.go`
6. Register in `cmd/plz-confirm/main.go`
7. Add React component: `agent-ui-system/client/src/components/widgets/FooDialog.tsx`
8. Add case in `WidgetRenderer.tsx` switch
9. Add default output in `store.go`'s `setDefaultOutputFor()`

### Debugging a Request

```bash
# Watch WebSocket events
plz-confirm ws --pretty

# Check server logs (printed to stderr)
make dev-backend 2>&1 | grep '\[API\]\|\[WS\]'

# Inspect a request
curl -s http://localhost:3000/api/requests/{id} | jq .
```

### Testing Script Widgets

```bash
# Create a script request via curl
curl -X POST http://localhost:3000/api/requests \
  -H "Content-Type: application/json" \
  -d '{
    "type": "script",
    "sessionId": "test",
    "scriptInput": {
      "title": "Test Script",
      "script": "exports.describe = function() { return {name:\"test\",version:\"1.0\"} }; exports.init = function() { return {step:0} }; exports.view = function(s) { return {widgetType:\"confirm\",input:{title:\"Hello?\"}} }; exports.update = function(s,c,e) { return {done:true,result:{ok:true}} };"
    }
  }'
```

---

*This document was generated from a thorough analysis of the plz-confirm codebase at commit HEAD on 2026-02-23.*
