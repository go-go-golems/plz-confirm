---
Title: Go Backend + Glazed CLI Design Options (keep React frontend)
Ticket: DESIGN-PLZ-CONFIRM-001
Status: active
Topics:
    - go
    - glazed
    - cli
    - backend
    - porting
    - agent-ui-system
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-15T15:45:35.994689927-05:00
---

# Go Backend + Glazed CLI Design Options (keep React frontend)

## Executive Summary

We want to **port the Node/TypeScript backend + CLI integration** of `agent-ui-system` to **Go**, using **Glazed** for the CLI portion, while keeping the existing **React + Redux Toolkit** frontend unchanged.

This design document intentionally **does not make final decisions yet**. Instead, it enumerates the **design surface area**, lists **credible options**, and documents **tradeoffs and risks**.

**Update (current working direction):** we will **defer schema codegen until later**. For the initial Go port we will **duplicate the widget type definitions** in Go and in the React app (manual sync), then revisit schema-first + codegen at the end.

## Problem Statement

The current system is implemented as:
- Backend: Node.js (Express) + `ws` WebSocket server
- Frontend: React + Redux Toolkit + WebSocket client
- CLI demo: Python calling REST endpoints and long-polling

We want to:
- **Port backend + CLI** to Go (for distribution, operational fit, integration with existing Go tooling, and reuse via a Go library).
- **Preserve the frontend contract** (same paths, same WebSocket message types, same UI expectations), so the React app can remain unchanged.
- **Avoid type drift** by introducing schema-first definitions and generated types.

Key constraints from the existing system (must remain true until explicitly changed):
- **Dev wiring**: Vite runs on `localhost:3000` and proxies:
  - `/api` → `http://localhost:3001`
  - `/ws` → `ws://localhost:3001`
- **Backend default port**: `3001`
- **WebSocket contract**: `/ws?sessionId=<id>`; server emits `new_request` and `request_completed`.
- **REST contract**: `/api/requests`, `/api/requests/:id`, `/api/requests/:id/response`, `/api/requests/:id/wait`.

## Proposed Solution

### “Target shape” (without committing to specific libraries yet)

At a high level, the Go port will likely consist of:

- **Go server** implementing the same REST+WS contract as `server/index.ts`.
- **Go client library** (package) used by the CLI (and reusable by other Go programs).
- **Glazed-based CLI** that provides ergonomic commands per widget type (and/or a generic command), using Glazed processors to output structured results.
- **Schema-first widget DSL**, stored as JSON Schema files in the repo, with a **Go generator** that produces:
  - Go types (structs + JSON marshal/unmarshal helpers where needed)
  - TypeScript types (interfaces/unions) for the frontend

### Compatibility envelope (frontend stays untouched)

The Go backend should initially behave as a drop-in replacement for the Node backend:

- Serve the same endpoints under `/api/*`.
- Provide a WebSocket endpoint at `/ws`.
- Emit the same message envelope used by the frontend today:
  - `{ "type": "new_request", "request": <UIRequest> }`
  - `{ "type": "request_completed", "request": <UIRequest> }`

In production mode, the Go server can additionally serve built frontend assets (equivalent to `dist/public` in this repo’s Vite config), but there are multiple viable approaches (see alternatives).

### Schema/codegen direction (deferred)

For now, we will **not** introduce shared JSON Schema + codegen. Instead:
- The Go server/CLI will define Go structs equivalent to `vibes/2025-12-15/agent-ui-system/client/src/types/schemas.ts`.
- The React app remains unchanged and continues using its existing `schemas.ts`.

We will revisit schema-first + codegen later (see end of this document for sketches and options).

### Schema sketch (drafts; no commitment yet)

This section sketches multiple JSON Schema modeling approaches for the widget DSL. The goal is to identify which patterns are easiest to validate and easiest to generate code for.

#### Sketch 1: Discriminated `oneOf` on `type` (const)

This is a common JSON Schema modeling technique for tagged unions:

- `UIRequest` is a `oneOf` over `{ type: const("confirm"), input: ConfirmInput, output?: ConfirmOutput }`, etc.
- Each variant schema fixes `type` and refines `input`/`output`.

Pros:
- Natural TS output: `type UIRequest = ConfirmRequest | SelectRequest | ...`.
- Allows schema validation to ensure “confirm requests always have ConfirmInput”.

Cons/Risks:
- Go codegen support for `oneOf` varies widely; some generators produce weak or awkward Go types.
- Requires careful handling of optional `output` (only present when completed).

#### Sketch 2: “Envelope + payload” with `type` + `input`/`output` as raw JSON + per-type sub-schemas

Model:
- The top-level `UIRequest` schema validates only shared envelope fields.
- `input`/`output` are left as `object`/`any` at the top-level.
- Separate schemas exist for `ConfirmInput`, `ConfirmOutput`, etc, but the “binding” between `type` and payload schemas is enforced by code (server/CLI), not by JSON Schema.

Pros:
- Go types are simple (`json.RawMessage` or `map[string]any`), easy to codegen.
- Avoids `oneOf` codegen pitfalls.

Cons:
- We lose schema-level guarantees that payloads match their declared `type`.
- More runtime logic / tests required to ensure correctness.

#### Sketch 3: Split schemas by lifecycle stage (create vs stored vs completed)

Model separate schema documents:
- `CreateRequest` (what `/api/requests` accepts): no `id`, no timestamps; has `type`, `sessionId`, `input`, `timeout?`.
- `UIRequest` (stored/returned): includes `id`, timestamps, `expiresAt`, etc.
- `CompletedRequest` (returned by `/wait` when done): guarantees `status=completed` and `output` present.

Pros:
- Stronger contracts on each endpoint.
- Better codegen and validation (fewer optional fields per schema).

Cons:
- More schema files and more generated types; may be “heavy” for a small project.

### Generated type shape options (Go + TypeScript)

We want generated types that:
- keep the frontend unchanged (at runtime),
- make it hard to accidentally mismatch request types and payloads,
- preserve the existing wire format.

#### Go type-shape options

- **Go shape 1: “Raw payloads + helpers”**
  - Generate `UIRequest` with `Input json.RawMessage` and `Output json.RawMessage` (or `any`).
  - Provide helper methods (handwritten or generated) like `ParseConfirmInput()` / `ParseConfirmOutput()`.
  - Good when schema uses `oneOf` but generator can’t make sum types.

- **Go shape 2: Explicit per-type request structs**
  - `ConfirmRequest`, `SelectRequest`, etc with typed `Input`/`Output`.
  - A wrapper `AnyRequest` that custom-unmarshals based on `type` (manual or generated switch).
  - Best type safety, but requires custom marshaling/unmarshaling logic.

- **Go shape 3: Interface-based “sum type”**
  - Define `type Request interface{ RequestType() WidgetType }` and implement per-type structs.
  - Useful for internal code; more complex to expose as a stable public API.

#### TypeScript type-shape options

- **TS shape 1: Discriminated union for requests**
  - `type UIRequest = ConfirmRequest | SelectRequest | ...` with `type: "confirm" | ...`.
  - Lowest friction for the frontend.

- **TS shape 2: Keep current `schemas.ts` and only generate for Go initially**
  - Avoids touching the frontend until the Go backend stabilizes.
  - Risks type drift unless we add a “schema parity check” in CI.

#### Runtime validation options (TS + Go)

- **TS validation**:
  - generate zod/ajv validators from JSON Schema (or validate using JSON Schema directly).
- **Go validation**:
  - validate incoming payloads against JSON Schema at ingress (server) and before sending (CLI).

We should treat runtime validation as optional until we decide the desired strictness and performance impact.

### CLI UX sketch (Glazed; no final commands yet)

Candidate CLI surface area (examples, not decisions):

- `agentui serve --addr :3001 [--static-dir dist/public]`
- `agentui confirm --title ... [--message ...] [--approve-text ...] [--reject-text ...]`
- `agentui select --title ... --option us-east-1 --option us-west-2 [--multi] [--searchable]`
- `agentui form --title ... --schema @schema.json`
- `agentui table --title ... --data @rows.json [--columns name,email] [--multi-select]`
- `agentui upload --title ... --file ./a.log --file ./b.log [--max-size ...]`

Common flags (layered via Glazed):
- `--base-url` (default might be `http://localhost:3000` for dev proxy, or `http://localhost:3001` for direct backend)
- `--session-id`
- `--timeout` (request expiration) and `--wait-timeout` (how long to block waiting)
- Glazed output flags (`--output json|yaml|table`, `--fields`, etc.)

Glazed output shape ideas:
- For simple outputs (confirm/select): emit a single row with typed columns (`approved`, `selected`, etc.)
- For complex outputs (form/table/upload): either:
  - emit a row with an `output_json` column (JSON string), or
  - emit multiple rows (e.g., one per uploaded file) depending on UX preference.

## Design Decisions

The initial implementation direction is now constrained by explicit choices:

- **C2**: WebSocket “minimal model” (simple map+mutex; no hub abstraction)
- **D1**: `net/http` with manual routing (no router framework)
- **E1**: In-memory store only
- **F2**: Event-driven `/wait` (block on per-request completion signal, not polling)
- **G**: No session concept (treat all WS clients as global subscribers; accept/ignore any `sessionId` fields for frontend compatibility)
- **H2**: Embed frontend assets in the Go server binary for production (`embed`)

We still keep this section’s “decision point” structure for future scope, but the above choices are the baseline for the rollout.

### Decision point A: Where does schema live and how is it versioned?

Options:
- **A1.** Keep schemas in a dedicated directory (e.g. `agent-ui-schema/` or `schema/agent-ui/`) with explicit versioning in filenames and `$id`.
- **A2.** Keep schemas inside the Go module (e.g. `internal/schema/`) and copy/export them for frontend generation.
- **A3.** Store schema alongside the frontend (`client/`) and treat backend as consumer.

Things to evaluate:
- How do we publish or pin schema versions for external clients?
- How do we keep schema changes from breaking old clients (if we ever need compatibility)?

### Decision point B: How do we generate Go + TS code from schema?

We need a **Go-based generator**. Options include:
- **B1. Orchestrator generator**: a Go program that runs:
  - a Go-schema→Go-types generator (or custom Go templates), and
  - a TS generator (possibly invoking a Node tool), and
  - writes outputs into stable locations (`internal/gen/...`, `client/src/gen/...`).
- **B2. Pure-Go generator**: a single Go program that parses JSON Schema and emits both Go and TS via templates.
- **B3. OpenAPI 3.1 as source of truth**: treat the DSL as part of an OpenAPI spec and generate everything from OpenAPI tooling (still JSON Schema under the hood), keeping the schema-first spirit but using OpenAPI infra.
- **B4. Go types as source of truth** (reverse direction): define Go structs and generate JSON Schema + TS from Go (likely easier, but violates “schema-first” as the true canonical source).

Key evaluation criteria:
- Support for `oneOf`/discriminator patterns for `UIRequest` variants.
- Stable output (diff-friendly, deterministic) for generated code.
- Handling of `any`/unknown/dynamic fields (table rows, form schema).
- Developer UX: `go generate` integration, CI checks, ergonomics.

### Decision point C: WebSocket implementation library and connection lifecycle model

Options:
- **C1.** `net/http` + WebSocket library + “hub” pattern (register/unregister/broadcast channels).
- **C2.** `net/http` + WebSocket library + simple map+mutex (minimal concurrency abstraction).

Things to evaluate:
- Backpressure behavior (slow clients) and broadcast fan-out.
- Connection cleanup reliability.
- Ping/pong and keepalive policy.

### Decision point D: Server routing + middleware stack

Options:
- **D1.** Standard library `net/http` with manual routing.
- **D2.** Minimal router (`chi`) for path params and middleware composition.
- **D3.** Full framework (`gin`, `echo`) for convenience.

Given current backend is small, “less framework” may be good, but we should confirm with future needs (auth, persistence, metrics).

### Decision point E: Storage/persistence for requests and sessions

Options:
- **E1. In-memory only** (current behavior): maps + locks; requests die on restart.
- **E2. In-memory + optional persistence**: pluggable store interface, with an initial in-memory impl and later SQLite/Redis/etc.
- **E3. Persistent-by-default**: store requests in SQLite/Redis from day 1 to survive restarts and allow multiple server instances.

### Decision point F: `/wait` semantics

Options:
- **F1. Keep polling semantics** similar to Node (periodic checks until timeout).
- **F2. Event-driven long-poll**: block on a per-request completion signal (channel/condvar), returning immediately when completed; still returns 408 if timeout.
- **F3. Replace `/wait` with SSE/WebSocket for CLI** (but keep `/wait` for compatibility).

### Decision point G: Session identity (frontend + CLI)

Current behavior is a fixed session id in the demo Redux store. Options for Go design:
- **G1. Explicit `sessionId` everywhere**: keep query-param for WS and body field for request creation.
- **G2. Cookie-based session**: backend issues cookie with session id; frontend uses cookie; CLI may pass a header/cookie explicitly.
- **G3. Auth token / JWT**: session derived from token; still might require explicit “session” grouping for multiple UIs.

We should not decide yet; we should enumerate how each impacts:
- WS URL building and proxying
- multi-user scenarios
- security posture

### Decision point H: Serving the frontend in production

Options:
- **H1. Serve static assets from disk** (e.g. `dist/public`), similar to Node.
- **H2. Embed assets via `embed`** in the Go binary (single artifact deployment).
- **H3. Don’t serve frontend in Go**: run it separately (CDN / Nginx) and only keep API/WS in Go.

Each option affects deployment, caching, and how we handle “client-side routing” fallback to `index.html`.

## Alternatives Considered

This section intentionally lists alternatives **without rejecting them yet**. Rejections should only happen once we explicitly commit to a design.

### Alternative set 1: Schema source of truth

- **JSON Schema (schema-first)** (requested): best for shared DSL and validation; codegen complexity depends on schema features used.
- **OpenAPI 3.1**: unifies DSL + API contract; generation ecosystem is larger; may be heavier than needed.
- **Protocol Buffers**: strong codegen; great for RPC; but doesn’t match the “JSON Schema DSL” requirement and complicates browser-native debugging.
- **Go structs as canonical**: easiest for Go; but requires trusting Go definitions to remain canonical (contrary to requirement).

### Alternative set 2: CLI shape

- **Per-widget commands**: `agentui confirm`, `agentui select`, etc.
  - Pros: typed params, friendly UX, easier to document.
  - Cons: N commands, duplicated flags (base URL, session, timeout) unless layered.
- **Generic “send request” command**: `agentui request --type confirm --input @file.json`.
  - Pros: minimal surface area, flexible.
  - Cons: weaker type safety and help UX; more JSON authoring burden.
- **Dual-mode commands** (Glazed “bare + glaze”): allow human-oriented output by default and structured output with `--with-glaze-output`.
  - Pros: best of both worlds.
  - Cons: more code; might confuse users unless documented carefully.

### Alternative set 3: Server concurrency model

- **Hub pattern** with channels: robust for fan-out, but introduces concurrency machinery.
- **Simple locks + per-connection goroutines**: minimal, but must be careful with broadcasts and slow consumers.

## Implementation Plan

This is a staged plan with decision gates (not a commitment to a specific option yet).

### Phase 0: Lock down the contract (baseline tests)
- [ ] Define a “contract test suite” that asserts existing endpoints and WS message envelopes.
- [ ] Reuse/port the Python `verify_e2e.py` flow concept into a Go integration test harness (or keep Python as black-box contract test).

### Phase 1: Schema-first scaffolding
- [ ] Add initial JSON Schema files for the widget DSL (request variants + input/output types).
- [ ] Create a Go generator entrypoint (e.g. `cmd/agentui-gen`) that produces:
  - Go types into `internal/gen/...` (or similar)
  - TS types into `client/src/gen/...` (or similar)
- [ ] Decide (later) whether the frontend switches to the generated TS types immediately or we keep the existing `schemas.ts` until the backend stabilizes.

### Phase 2: Go backend “drop-in” implementation
- [ ] Implement `/api/requests` (create), `/api/requests/:id` (get), `/api/requests/:id/response` (submit), `/api/requests/:id/wait`.
- [ ] Implement `/ws` WebSocket endpoint with `sessionId` query param.
- [ ] Match message envelopes exactly (`new_request`, `request_completed`).

### Phase 3: Go client library + Glazed CLI
- [ ] Build a Go client package for the API contract (create/wait/submit/WS listen as needed).
- [ ] Implement Glazed commands for the key widget types and output structured results.

### Phase 4: Optional improvements (after parity)
- [ ] Add request expiration + cleanup strategy (timer wheel / sweeper / TTL map).
- [ ] Add validation of inputs/outputs against JSON Schema (if desired).
- [ ] Consider persistence and auth/session identity enhancements.

## Open Questions

### Contract and scope
- Do we need to preserve *only* the currently used endpoints, or also future OAuth/login scaffolding hinted at by `client/src/const.ts` and `ManusDialog`?
- Should the Go port implement request expiration semantics that don’t exist today (or keep parity first)?

### Schema and code generation
- How strict should schemas be about “dynamic” shapes (`table.data`, `table.selected`, `form.schema`, `form.data`)?
- What level of `oneOf`/discriminator sophistication is required for generated Go types (sum types)?
- Do we want runtime validation in Go/TS, or only compile-time types?

### Operational model
- Do we need multi-instance backend support (thus persistence + shared pubsub), or is single-instance OK?
- How important is serving embedded frontend assets (single binary) vs serving from disk?

## References

- Ticket analysis: `analysis/01-code-structure-analysis-agent-ui-system.md`
- Existing implementation:
  - `vibes/2025-12-15/agent-ui-system/server/index.ts`
  - `vibes/2025-12-15/agent-ui-system/vite.config.ts`
  - `vibes/2025-12-15/agent-ui-system/client/src/types/schemas.ts`
  - `vibes/2025-12-15/agent-ui-system/demo_cli.py`
- Glazed tutorial: `glazed/pkg/doc/tutorials/05-build-first-command.md`
