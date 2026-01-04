# Diary

## Goal

Capture the research trail for implementing bounded/paginated history, request metadata, long-term storage, and widget-level default+timeout auto-completion in `plz-confirm`.

## Step 1: Read the doc workflows and locate the system’s core contracts

I started by reading the local workflow docs for documentation management and diaries, because this ticket is primarily a research deliverable (with a lot of cross-file linking). Then I did a quick repo scan to identify the minimum set of files that define “how a request flows” through CLI → server → WS → UI.

The immediate outcome is a short list of “source of truth” files for contracts: the protobuf schemas (`proto/...`), the Go server handlers (`internal/server/...`), the in-memory store (`internal/store/...`), and the UI’s WebSocket + Redux state wiring (`agent-ui-system/client/src/...`).

### What I did
- Read `~/.cursor/commands/docmgr.md` and `~/.cursor/commands/diary.md` (workflow expectations).
- Skimmed `pkg/doc/adding-widgets.md` to confirm the intended contract boundaries.
- Located key files for request creation, completion, and WS broadcasting.

### Why
- This work spans CLI, backend, and UI. Without the correct map of contracts, it’s easy to propose changes that don’t fit the system’s shape (especially because there is both a Go backend and a legacy Node backend).

### What worked
- `pkg/doc/adding-widgets.md` already provides an accurate “mental model” diagram for the request lifecycle and points to the core handlers.

### What didn't work
- N/A

### What I learned
- The repo already treats `/api/requests` and `/ws` as the “stable” contracts; most new features should be layered by extending the protobuf envelope and keeping the REST/WS shapes consistent.

### What was tricky to build
- N/A (research only)

### What warrants a second pair of eyes
- N/A (research only)

### What should be done in the future
- N/A

### Code review instructions
- Start with `pkg/doc/adding-widgets.md`, then follow the file links in the ticket `index.md` frontmatter.

### Technical details
- Core contract doc: `pkg/doc/adding-widgets.md`
- Envelope schema: `proto/plz_confirm/v1/request.proto`
- Widget schemas: `proto/plz_confirm/v1/widgets.proto`

## Step 2: Identify how history currently exists (and where it does not)

Next I traced “history” in the web UI to see what is actually stored and rendered today. The frontend maintains a `history: UIRequest[]` array in Redux and renders it in a right-hand “REQUEST_HISTORY” panel using a scroll container. History is populated only by events observed while the browser is connected: a request is pushed into history when it completes (either locally or via a `request_completed` WS broadcast).

The key discovery is that, in the Go backend path, the server store is explicitly in-memory and currently has no API for listing completed requests, no paging, and no persistence. So “long term storage” is not present in the Go server today; achieving it requires a persistence layer and list endpoints.

### What I did
- Read `agent-ui-system/client/src/store/store.ts` (history state structure).
- Read `agent-ui-system/client/src/pages/Home.tsx` (history UI rendering).
- Read `agent-ui-system/client/src/services/websocket.ts` (how history is populated).
- Read `internal/store/store.go` (server-side request lifecycle state).
- Checked `agent-ui-system/server/index.ts` (legacy Node server) for comparison.

### Why
- Before designing bounded/paginated history, it’s critical to know whether history is already persisted server-side (so UI pagination could just page over an API), or whether we need to add persistence first.

### What worked
- The UI is already visually scrollable (`ScrollArea`), so the “scrollable” piece is largely a state/size management + pagination problem rather than pure CSS.

### What didn't work
- N/A (research only)

### What I learned
- Current Go store: in-memory only; only *pending* requests are replayed on WS connect.
- Current UI history: unbounded in-memory array; no fetch of older history exists.

### What was tricky to build
- N/A (research only)

### What warrants a second pair of eyes
- Confirm which backend is actually used in your deployments (Go server vs legacy Node server), because “history persistence” needs to land in the authoritative backend.

### What should be done in the future
- Add explicit “history list/paging” endpoints in the chosen backend, otherwise UI pagination can only ever be local/in-memory.

### Code review instructions
- Start in `agent-ui-system/client/src/services/websocket.ts` and follow the Redux actions into `agent-ui-system/client/src/store/store.ts`.
- Then read `internal/server/ws.go` and `internal/store/store.go` for server-side behavior.

## Step 3: Trace the JSON↔protobuf “glue layer” to find the right extension points for metadata and defaults

I then traced the exact code paths that turn incoming REST JSON bodies into `UIRequest` protobuf messages, and the reverse path that serializes `UIRequest` back into JSON for REST responses and WebSocket events. This is the “choke point” for extending the request envelope: anything we add (metadata, completion reasons, auto-default policies) needs to survive (a) JSON decode on create/response, (b) protobuf conversion, (c) persistence/storage, and (d) protojson serialization back out to the browser and CLI.

The key insight is that the server intentionally preserves a *legacy REST shape* (separate `type`, `input`, and `timeout` fields) even though it stores and emits a protobuf `UIRequest`. That means envelope-level features should be added as *extra top-level JSON fields* on `POST /api/requests` (e.g. `meta`, `autoComplete`) and then explicitly mapped into the protobuf in `internal/server/proto_convert.go`.

### What I did
- Read `internal/server/server.go` request handlers:
  - `handleCreateRequest`
  - `handleSubmitResponse`
  - `handleWait`
- Read `internal/server/proto_convert.go`:
  - `createUIRequestFromJSON` (create path)
  - `createUIRequestWithOutput` (submit-response path)
- Read `internal/server/ws_events.go` (WS payload serialization) to confirm the exact JSON shape emitted to the browser.

### Why
- Metadata and auto-default behavior are “envelope features”. The safest and most maintainable place for them is the `UIRequest` envelope rather than widget-specific inputs (which would duplicate logic across widgets and complicate history rendering).

### What worked
- The repo already centralizes JSON→protobuf translation in a single file (`internal/server/proto_convert.go`), which makes it straightforward to extend the envelope in one place.

### What didn't work
- N/A (research only)

### What I learned
- REST create currently accepts `{ type, sessionId, input, timeout }` and converts `input` via `protojson.Unmarshal` into a widget-specific `*Input` message.
- REST submit-response accepts `{ output }` and uses the stored request’s `.Type` to unmarshal the widget-specific output message.
- WebSocket payload is `{ type: string, request: <protojson(UIRequest)> }`, where `request` is a raw JSON blob produced by `protojson` (camelCase field names).

### What was tricky to build
- N/A (research only)

### What warrants a second pair of eyes
- Decide whether envelope features should be introduced by extending the legacy REST JSON shape or by adding a new “v2” endpoint that accepts a full protojson `UIRequest` (both are valid; the former is less disruptive).

### What should be done in the future
- Explicitly document the create-request JSON shape in `pkg/doc/adding-widgets.md` once metadata/default fields are added so future widget work doesn’t accidentally drop them.

### Code review instructions
- Start at `internal/server/server.go` for REST+WS wiring, then read `internal/server/proto_convert.go` for the JSON↔protobuf mapping rules.

### Technical details
- WS serialization uses:
  - `protojson.MarshalOptions{EmitUnpopulated:true, UseProtoNames:false}`
- This means:
  - field names are camelCase (`sessionId`, `createdAt`, `expiresAt`, …)
  - enums are emitted as their *value names* (e.g. `"confirm"`, `"completed"`), not numeric values.

## Step 4: Confirm what “timeouts” mean today (and why default-on-timeout needs new semantics)

I specifically looked for existing timeout enforcement and “default response” semantics, because the CLI already exposes two flags that look timeout-ish (`--timeout` and `--wait-timeout`). The goal was to avoid accidentally reinterpreting a field that already has a meaning in the system (which would create confusing behavior for both agent authors and UI users).

The result: the only timeout that currently “does anything” is the long-poll timeout in `GET /api/requests/{id}/wait?timeout=...` (it bounds a *single poll*). The request expiration time (`expiresAt`) is currently set but not enforced. This means “default result after N seconds” is a new behavior that must be implemented explicitly server-side (scheduler + completion info), and cannot be achieved by tweaking existing flags alone.

### What I did
- Read `internal/store/store.go` to see what it does with `expiresAt`.
- Read `internal/server/server.go:handleWait` to understand the long-poll timeout behavior.
- Read `internal/client/client.go:WaitRequest` to see how the CLI’s overall wait timeout interacts with server polling.
- Checked `proto/plz_confirm/v1/request.proto` and saw `RequestStatus` already contains `timeout` and `error`, but no code currently sets those statuses.

### Why
- If we implement “auto default” by silently reusing `expiresAt`, we’ll end up with:
  - ambiguous semantics (“timeout” could mean TTL or auto-default)
  - hard-to-debug behavior (UI never sees a completion event unless server broadcasts it)

### What worked
- The CLI’s long-poll loop is already well structured: it distinguishes “poll timeout” from “overall wait timeout” and keeps retrying on 408.

### What didn't work
- There is no server-side expiration loop or cleanup: `expiresAt` is currently a passive timestamp.

### What I learned
- Existing time-related fields:
  - `expiresAt`: set at create time; not enforced today
  - `/wait?timeout=`: per-poll long-poll bound (server returns 408)
  - CLI `--wait-timeout`: overall deadline for the agent waiting locally
- Therefore “default after N seconds” needs:
  - new input/policy fields (so server knows the default)
  - new completion info fields (so UI can display “default was used”)
  - new server scheduler (so the state transition actually happens)

### What was tricky to build
- N/A (research only)

### What warrants a second pair of eyes
- Clarify whether “default after N seconds” should:
  - complete the request (status=completed, output=default), or
  - mark it as timeout (status=timeout) but also attach the default output for convenience.
  The former is closer to “default selection chosen”; the latter is closer to “user did not respond”.

### What should be done in the future
- Decide whether `expiresAt` should remain a “TTL / retention” marker (cleanup) while a new `autoCompleteAt` drives defaulting behavior.

### Code review instructions
- Start at `internal/client/client.go:WaitRequest` and `internal/server/server.go:handleWait` to understand long-poll semantics.
- Then read `internal/store/store.go:Create` to see how `expiresAt` is computed and stored.

## Step 5: Evaluate “long-term history” feasibility and identify the missing backend primitives

With the current behavior mapped, I focused on the question “does long-term history already exist?”. I checked both the Go backend and the legacy Node backend, because the repo contains both. In both cases, the answer is no: requests live in memory (Go: `internal/store/store.go`, Node: `agent-ui-system/server/index.ts`) and there is no disk persistence or listing endpoint for completed requests.

That means UI pagination cannot be “real pagination” until the backend grows a list endpoint and a persistence layer. Any UI-only paging is necessarily just a bounded local cache.

### What I did
- Verified Go server instantiates an in-memory store on `serve`:
  - `cmd/plz-confirm/main.go` creates `store.New()` (fresh on each run)
- Verified legacy Node server also uses in-memory `Map`:
  - `agent-ui-system/server/index.ts` uses `const requests = new Map<string, UIRequest>()`
- Searched for persistence-related dependencies and found SQLite driver is already present indirectly:
  - `go.mod` includes `github.com/mattn/go-sqlite3` as indirect
- Checked existing docs/tickets that already call out missing persistence/history:
  - `ttmp/2025/12/15/.../analysis/01-code-structure-analysis-agent-ui-system.md`

### Why
- “Pagination” and “long-term storage” are really the same problem: once you can page in the UI, you need a canonical source of truth to page from.

### What worked
- The system already centralizes request serialization via protobuf+protojson; persisting `protojson(UIRequest)` as a blob is viable even before building a normalized DB schema.

### What didn't work
- There is no existing endpoint to fetch history pages; the only request endpoints are create, get-by-id, wait, and submit response.

### What I learned
- The minimum backend additions for pagination are:
  - store-level `List` primitive (or equivalent)
  - HTTP `GET /api/requests?...` endpoint that uses it
  - a stable cursor strategy (opaque cursor recommended)

### What was tricky to build
- N/A (research only)

### What warrants a second pair of eyes
- Decide whether SQLite via CGO is acceptable for your distribution model (single-binary release, cross-compilation). If not, pick a pure-Go persistence alternative early.

### What should be done in the future
- Add a small “storage decision record” doc once you pick the persistence backend (SQLite vs JSONL vs KV store) so future contributors don’t re-litigate it.

### Code review instructions
- Start at `cmd/plz-confirm/main.go` to see how the server wires the store.
- Then read `internal/store/store.go` for current capabilities/limitations.

## Step 6: Identify the minimal frontend changes needed for bounded history and future paging

Finally, I mapped exactly how the frontend builds and renders history so we can decide what can be improved immediately (bounded history) versus what requires backend work (pagination over durable history). The key point is that history is currently a single array in Redux and the view simply maps it into a scroll container. There is no fetch layer for history.

This is good news: the bounded-history fix is a tiny, low-risk change (truncate after unshift). And the paging feature can be built cleanly by adding a small “history loader” state machine (cursor + loading + error) without disrupting the active-request widget flow.

### What I did
- Read the WS client handler:
  - `agent-ui-system/client/src/services/websocket.ts`
- Read the request slice:
  - `agent-ui-system/client/src/store/store.ts`
- Read the history UI:
  - `agent-ui-system/client/src/pages/Home.tsx` (history panel + `ScrollArea`)

### Why
- The UI is where the user experiences “history”. If we add metadata and defaulting semantics but never render them, we miss the value.

### What worked
- The history panel is already a scroll container; adding paging controls is a UI/state problem, not a layout problem.

### What didn't work
- N/A (research only)

### What I learned
- History is only updated by:
  - `completeRequest` (for the active request)
  - `addToHistory` (when another client completed it)
  Both just `unshift` into an unbounded array.

### What was tricky to build
- N/A (research only)

### What warrants a second pair of eyes
- When adding server-side paging, be careful to avoid duplicates if:
  - you loaded a history page that includes a request that later arrives via WS, or
  - you already saw a WS completion and then fetch a page that overlaps.

### What should be done in the future
- Add a lightweight dedup strategy (e.g. `Set` of seen IDs) once paging is introduced.

### Code review instructions
- Start at `agent-ui-system/client/src/pages/Home.tsx` for rendering, then trace back into Redux actions and WS message handler.

## Step 7: Fix duplicate history entries and queue multiple pending requests

This step addressed a concrete UX bug: completing a request in the browser could create duplicate entries in the history panel. The root cause was that the UI both updates history locally after submitting a response and also reacts to the server’s `request_completed` WebSocket echo; the WebSocket path appended to history again once the active request had already been cleared.

While fixing that, I also added a simple pending queue so multiple `new_request` events don’t overwrite the current active request. This makes it possible to “drain” a burst of requests in a predictable order.

**Commit (code):** 9f32913 — "🐛 fix: dedupe UI history + queue requests"

### What I did
- Made history updates idempotent by request id (upsert semantics).
- Changed WebSocket handling to always dispatch completion via a single path.
- Added a `pending` queue in the request slice and enqueued `new_request` events when another request is active.
- Added `make dev-backend`, `make dev-frontend`, and `make dev-tmux` dev targets.
- Added a ticket-local seed script to create multiple pending requests via the CLI:
  - `ttmp/2026/01/03/001-QOL-HISTORY-TUI--history-pagination-metadata-defaults/scripts/seed-requests-with-metadata.sh`

### Why
- The UI should never show the same request twice in history.
- A queue is the simplest way to avoid losing requests when multiple arrive before the user responds.

### What worked
- The completion reducer is now idempotent, so “local completion + WS echo” no longer produces duplicates.
- The queue ensures incoming requests don’t overwrite `active`.

### What didn't work
- N/A (straightforward change)

### What I learned
- Any client-side flow that can “optimistically” update state and also receive authoritative echoes needs idempotent reducers (or acks) to avoid duplicates.

### What was tricky to build
- Making completion logic safe for all cases:
  - completed request is active
  - completed request is pending
  - completed request is already in history (upsert)

### What warrants a second pair of eyes
- Confirm that the new queue behavior matches the intended UX (FIFO vs LIFO, and whether users should see “N pending” anywhere).

### What should be done in the future
- If/when server-side history paging is added, ensure the client keeps an id-set to avoid duplicates across “paged fetch” and WS events.

### Code review instructions
- Start in `agent-ui-system/client/src/store/store.ts` and review `enqueueRequest` + `completeRequest`.
- Then check `agent-ui-system/client/src/services/websocket.ts` message handling changes.
- For repro, run `make dev-tmux` and then `API_BASE_URL=http://localhost:3001 bash ttmp/2026/01/03/001-QOL-HISTORY-TUI--history-pagination-metadata-defaults/scripts/seed-requests-with-metadata.sh`.

## Step 8: Validate the history fix in the running UI

I validated the change by running the Vite frontend on `:3000` and the Go backend on `:3001`, then creating and completing requests via the CLI. The history panel no longer shows duplicate entries for a single request id.

**Commit (docs):** c924478 — "📝 docs: record history fix validation"

### What I did
- Started dev servers (`make dev-tmux`).
- Triggered a request via CLI and completed it in the browser UI.
- Confirmed the resulting history entry appears once per request id.

### What warrants a second pair of eyes
- In Redux DevTools you may still observe two “completion” *events* (local submit + WS echo). The reducer is now idempotent by request id, so it should not result in duplicated history entries.

## Step 9: Suppress duplicate completion actions (local submit vs WS echo)

Even though state was idempotent, Redux DevTools still showed two completion-related actions (local submit and WS `request_completed`), which is noisy when debugging. This step makes completion dispatch exactly once per request id by correlating completions in the WS/client layer using an in-memory bounded set keyed by request id (no timestamps).

**Commit (code):** eb41c20 — "🧹 ui: suppress duplicate completion actions"

### What I did
- Added a bounded `completedIds` set in the WS client to ignore duplicate `request_completed` messages for ids that have already been processed.
- Moved the `completeRequest` dispatch for local submissions into `submitResponse` so local submit and WS echo share the same “dispatch once” guard.
- Updated `WidgetRenderer` to stop dispatching `completeRequest` directly.

### Why
- Avoid duplicate Redux actions in DevTools while keeping the system robust to race order (WS first vs HTTP response first).

### What was tricky to build
- Getting the ordering right without time-based correlation:
  - if WS arrives first, the HTTP response path should not dispatch
  - if HTTP response arrives first, the WS echo should not dispatch

## Step 10: Enforce `expiresAt` server-side (authoritative timeouts)

This step makes timeouts real: the server now transitions pending requests to `status=timeout` when `expiresAt` passes, unblocks any long-poll waiters, and broadcasts the updated request via WebSocket. This is deliberately “authoritative”: it does not rely on the browser being open to enforce expiry.

I also refactored the scheduler goroutine ownership to be tied to `ListenAndServe` via an `errgroup`, so time-based background work stops reliably when the server context is canceled.

**Commit (code):** b7fd7b5 — "⏱️  server: enforce request expiresAt"

### What I did
- Added `internal/store/store.go:Expire(now)` to transition pending requests to `timeout`, set `completedAt` and a simple `error` string, and close the request `done` channel.
- Updated `internal/store/store.go:Wait` to return for `timeout`/`error` status (not only `completed`).
- Added a server-side ticker in `internal/server/server.go:ListenAndServe` that calls `Expire` and broadcasts a `request_completed` event for timed-out requests (scoped to the request’s `sessionId`).
- Updated `agent-ui-system/client/src/pages/Home.tsx` to render a distinct TIMEOUT label in history.

### Why
- Without server-side enforcement, `expiresAt` is just a TTL hint and “timeouts” are not observable/consistent across clients.

### What warrants a second pair of eyes
- Whether timeout should use `request_completed` or a dedicated WS event type (`request_timed_out`) once the UI grows more nuanced timeout UX.

## Step 11: Add a CLI WebSocket watcher for debugging timeouts

To make timeout behavior testable without relying on the browser UI, I added a small CLI command that connects to the server’s WebSocket endpoint and prints events. This lets us confirm that a request transitions to `status=timeout` and that the server broadcasts a completion event when `expiresAt` passes.

**Commit (code):** 29ef3b0 — "✨ cli: add ws event watcher command"

### What I did
- Added `plz-confirm ws` as a Cobra subcommand (not Glazed) under `cmd/plz-confirm/ws.go`.
- The command connects to `/ws?sessionId=...` (derived from `--base-url`) and prints each event as a JSON line (optional `--pretty`).

### How to use
- Watch events:
  - `go run ./cmd/plz-confirm ws --base-url http://localhost:3001 --session-id global --pretty`
- Create a request with a short timeout, then observe a `request_completed` event with `status=timeout`.

## Step 12: Run a tmux demo to watch timeouts end-to-end (server + WS + UI)

This step makes the timeout behavior easy to reproduce: a ticket-local script spins up a tmux session with the backend, Vite frontend, a CLI WS watcher, and a pre-canned CLI request with a short timeout. With `sessionId` scoping enabled in the server, the demo also ensures that the UI and CLI are on the same session (`global` by default).

In the run I captured, the request timed out (no UI click before `expiresAt`), and the WS watcher showed a `request_completed` event with `status=timeout`. The CLI command returned a clean error instead of panicking.

### What I did
- Added and used:
  - `ttmp/2026/01/03/001-QOL-HISTORY-TUI--history-pagination-metadata-defaults/scripts/tmux-timeout-ws-demo.sh`
- Verified the WS watcher sees:
  - `new_request` (status pending)
  - `request_completed` (status timeout, error "request timed out")

### Notes
- The demo session is `PLZ-TIMEOUT`.
- UI URL: `http://localhost:3000/?sessionId=global`

## Step 13: Change expiry semantics to “auto-complete” (default outputs) instead of `status=timeout`

The server-side expiry scheduler was working, but the desired semantics changed: when `expiresAt` is reached, requests should be treated as a normal completion with a synthetic default output (not a distinct `timeout` status). The key idea is to preserve idempotency and correlation via request id while making the server’s behavior compatible with CLI/automation that expects `status=completed`.

This step implements default output generation per widget type and marks these auto-completions via a shared `comment="AUTO_TIMEOUT"` marker so the UI can still label them as TIMEOUT even though `status=completed`.

**Commit (code):** 3afe968 — "⏱️ server: auto-complete expired requests"

### What I did
- Updated `internal/store/store.go:Expire` to set `status=completed` and synthesize a default output for the request’s widget type.
- Implemented `setDefaultOutputFor` to write the protobuf oneof output directly (avoids referencing the unexported generated oneof interface type).
- Updated CLI widget commands in `internal/cli/*.go` to no longer special-case `status=timeout`.
- Updated the history status badge in `agent-ui-system/client/src/pages/Home.tsx` to render TIMEOUT when `status=completed` and `output.comment == "AUTO_TIMEOUT"`.

### Why
- “Timeout” as a distinct status makes CLI callers treat expiry as an error path, which is not desired for this workflow.
- Keeping `status=completed` for expiry makes the request lifecycle uniform while still allowing the UI to communicate “this was auto-completed”.

### What worked
- `go run ./cmd/plz-confirm confirm ... --timeout 20` returns `approved=false` with `comment=AUTO_TIMEOUT` instead of erroring.
- The WS watcher receives `request_completed` with `status=completed` for expired requests.

### What didn't work
- First attempt returned a generated oneof interface type (`v1.IsUIRequest_Output`), but it’s not exported in Go generated code:
  - `internal/store/store.go:161:77: undefined: v1.IsUIRequest_Output`
- Also hit a structpb assignment mismatch while drafting the default form output:
  - `cannot use *st ... as *structpb.Struct`
- Pre-commit `exhaustive` linter required an explicit `widget_type_unspecified` case even though a `default:` existed.

### What I learned
- For protobuf oneofs in Go, it’s often easiest to assign concrete wrapper types directly to the oneof field instead of trying to name the oneof interface type.
- If the UI needs to distinguish “auto-completed” from “user-completed”, a stable marker in the payload (like `comment`) is a practical bridge until we add an explicit enum/field.

### What was tricky to build
- Designing “safe” defaults for each widget type without inventing new schema (e.g. select/table/image multi vs single), while keeping the output always present to avoid nil-handling edge cases.

### What warrants a second pair of eyes
- Confirm that using `comment="AUTO_TIMEOUT"` as the cross-widget marker is acceptable long-term, or whether we should introduce an explicit `completion_kind`/`auto_completed` field in `UIRequest`.
- Verify that the default outputs are aligned with how each CLI command and widget renderer interprets “empty” selections.

### What should be done in the future
- Add explicit completion kind fields to the protobuf envelope (instead of overloading `comment`).
- Add unit tests for expiry auto-completion output generation (one per widget type).

### Code review instructions
- Start with `internal/store/store.go:setDefaultOutputFor` and `internal/store/store.go:Expire`.
- Validate quickly with:
  - `go run ./cmd/plz-confirm serve --addr :3001`
  - `go run ./cmd/plz-confirm ws --base-url http://localhost:3001 --session-id global --pretty`
  - `go run ./cmd/plz-confirm confirm --base-url http://localhost:3001 --session-id global --timeout 5 --wait-timeout 30 --title TEST`
