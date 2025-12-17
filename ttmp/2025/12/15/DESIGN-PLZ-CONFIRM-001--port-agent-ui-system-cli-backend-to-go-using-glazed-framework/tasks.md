# Tasks

## TODO

### 1) Lock scope + repo placement (server + CLI first)

- [x] **Choose Go code location + module name** (recommend: new module `agentui/` at repo root, added to `go.work`), so we can build a standalone binary while using local `glazed` via workspace.
  - Ref: `go.work` at repo root (currently uses `./glazed`, `./go-go-labs`, `./bobatea`)
  - Constraints: H2 (embed frontend assets) influences where embedded files live (embed cannot use `..` in patterns).
- [x] **Decide binary shape**: one binary `agentui` with subcommands `serve` + widget commands (recommended), vs separate binaries (`agentui-server`, `agentui`).

### 2) Implement Go types (duplicated; schema codegen later)

- [x] **Create Go types mirroring the existing TS types** (manual duplication for now).
  - Ref: `vibes/2025-12-15/agent-ui-system/client/src/types/schemas.ts`
  - Include: `UIRequest`, status enum, and per-widget `*Input/*Output` structs (`confirm/select/form/table/upload`)
  - Note: Keep wire-compatible JSON field names and “loose” `any` fields as `map[string]any` / `json.RawMessage` where needed (`form.schema`, `form.data`, `table.data`, `table.selected`).

### 3) Implement Go in-memory store (E1) with event-driven wait (F2)

- [x] **Implement request store** (in-memory only; map+mutex), including per-request completion signal.
  - Ref behavior: `vibes/2025-12-15/agent-ui-system/server/index.ts` (in-memory `Map`, statuses, timestamps)
- [x] **Event-driven wait**: store should expose `Wait(ctx, id)` that blocks on a per-request `done` channel (close on completion), returning 408 when the ctx deadline hits.
  - Ref behavior: `/api/requests/:id/wait` in `server/index.ts` but replace polling with event-driven semantics.

### 4) Implement Go HTTP server (D1) + WebSocket fanout (C2) (no session)

- [x] **HTTP server skeleton** using `net/http` with manual routing (no router framework).
- [x] **Implement REST endpoints** with the same path contract as today:
- [x] `POST /api/requests` (create request)
- [x] `GET /api/requests/{id}` (fetch request)
- [x] `POST /api/requests/{id}/response` (complete request)
- [x] `GET /api/requests/{id}/wait?timeout=...` (long-poll, event-driven)
  - Ref: `vibes/2025-12-15/agent-ui-system/server/index.ts`
- [x] **Implement WebSocket endpoint** at `/ws`:
- [x] Accept `sessionId` query param for frontend compatibility but **ignore it** (G = no session).
    - Ref: frontend builds ws URL in `vibes/2025-12-15/agent-ui-system/client/src/services/websocket.ts`
- [x] Maintain global WS clients as `map[*Conn]struct{}` guarded by mutex (C2).
- [x] On connect: send all currently `pending` requests as `new_request` messages.
- [x] On create: broadcast `{type:\"new_request\", request:<UIRequest>}` to all clients.
- [x] On completion: broadcast `{type:\"request_completed\", request:<UIRequest>}` to all clients.
  - Ref message envelopes: `client/src/services/websocket.ts` and `server/index.ts`
- [x] **CORS + preflight**: implement minimal headers so dev proxy works cleanly (Vite proxy already handles most cases, but keep parity with Express’s `cors()`).
  - Ref: `vibes/2025-12-15/agent-ui-system/vite.config.ts` proxy rules.

### 5) Implement Glazed CLI (Cobra-based) to create requests + wait for responses

- [x] **Create CLI root command** using Cobra + Glazed command bridging.
  - Ref: `glazed/pkg/doc/tutorials/05-build-first-command.md`
- [x] **Implement client-side HTTP calls**:
  - `POST /api/requests` → receive request id
  - `GET /api/requests/{id}/wait` → wait for completion
  - Ref workflow: `vibes/2025-12-15/agent-ui-system/demo_cli.py`
- [ ] **Commands (first pass)**:
  - [ ] `agentui confirm ...` → outputs approved/timestamp (Glazed rows)
  - [ ] `agentui select ...` → outputs selected
  - [ ] `agentui form --schema @file.json` → outputs `data` (likely as JSON column initially)
  - [ ] `agentui table --data @rows.json` → outputs selected (JSON column)
  - [ ] `agentui upload ...` → outputs files (rows or JSON column)
- [x] `agentui serve` → starts the Go server (same binary; can be BareCommand)
- [x] **Common flags** (Glazed layer or shared Cobra persistent flags):
- [x] `--base-url` (default: `http://localhost:3000` for dev proxy; optionally `http://localhost:3001` for direct backend)
- [x] `--timeout` (request expiry seconds, maps to server `timeout` on create)
- [x] `--wait-timeout` (seconds for `/wait`)
- [x] No session flags (G = no session), but server should still tolerate `sessionId` being present.

### 6) Production mode: embed frontend assets (H2)

- [ ] **Embed built frontend assets into the Go binary** using `embed`.
  - Ref current build output: `vibes/2025-12-15/agent-ui-system/dist/public` (from `vibes/.../agent-ui-system/vite.config.ts`)
  - [ ] Decide how assets get into an embeddable path (copy into Go module at build time; avoid `..` in embed patterns).
- [ ] **Serve SPA**:
  - [ ] Serve embedded static files for `/` and asset paths
  - [ ] Fallback to `index.html` for client-side routing (but never shadow `/api/*` or `/ws`)

### 7) Parity validation + minimal tests

- [ ] **Manual parity runbook**:
  - Start Go server on 3001
  - Run Vite dev server on 3000 (proxy to Go)
  - Open UI and run `agentui confirm/select/form` flows
  - Ref: `vibes/2025-12-15/agent-ui-system/vite.config.ts`, `client/src/services/websocket.ts`
- [ ] **Automated smoke test**:
  - Create request, then programmatically submit response via `POST /api/requests/{id}/response`, assert CLI receives output.
  - Ref: `vibes/2025-12-15/agent-ui-system/verify_e2e.py` approach (but implement in Go or keep Python as black-box test).

### 8) Later (explicitly last): JSON Schema + codegen

- [ ] Introduce shared JSON Schema for widget DSL + generate Go/TS types (replacing the duplicated definitions).

