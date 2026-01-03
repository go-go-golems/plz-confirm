# Tasks

## TODO

- [x] Fix UI request history duplicate entries
  - [x] Make completion/history updates idempotent by request id
  - [x] Stop WS completion handler from double-appending already-completed items

- [x] Add a pending queue for multiple incoming requests
  - [x] Enqueue `new_request` events when another request is already active
  - [x] Pop next pending request when the active one completes

- [x] Add dev helpers for reproducing history/queue behavior
  - [x] Add Makefile dev targets for Go backend + Vite frontend
  - [x] Add ticket-local seed script to create multiple pending requests via CLI

- [x] Remove Node legacy server (agent-ui-system/server)
  - [x] Remove `agent-ui-system/server/index.ts`
  - [x] Update `agent-ui-system/package.json` scripts to drop Node server build/start
  - [x] Remove Node-server-only dependencies from `agent-ui-system/package.json` (express/ws/cors/body-parser + @types)
  - [x] Ensure Go embedding pipeline still works (`go generate ./...` uses `pnpm -C agent-ui-system run build`)

- [x] Make history panel fixed-height and prevent page scroll
  - [x] Make `agent-ui-system/client/src/components/Layout.tsx` main area fixed-height (`h-screen`) and `overflow-hidden`
  - [x] Make `agent-ui-system/client/src/pages/Home.tsx` grid fill available height (`flex-1 min-h-0`) so internal ScrollAreas scroll
  - [x] Ensure history panel height is bounded to viewport and doesn’t push left column out of view

- [ ] Make `expiresAt` enforceable “UI timeout” with countdown, pausable on interaction
  - [ ] Define semantics precisely:
    - [ ] When `now >= expiresAt`, server transitions request to `timeout` (or completes with default output; decide)
    - [ ] “Interaction stops timeout”: is it a permanent disable, or does it extend/refresh `expiresAt`?
    - [ ] What counts as interaction (any click/keydown anywhere in widget vs only input changes)?
  - [x] Add server-side expiry scheduler (authoritative, not browser-only)
    - [x] Periodically scan pending requests and expire those past `expiresAt`
    - [x] On expire: broadcast WS event (`request_completed` with `status=timeout`)
  - [ ] Add an “activity/touch” API so the UI can pause/disable expiry
    - [ ] `POST /api/requests/{id}/touch` (or similar) marks request as “touched/active” and disables expiry enforcement
    - [ ] Decide idempotency and rate-limiting (to avoid spam from keypress handlers)
  - [ ] Extend protobuf envelope to record timeout/interaction state
    - [ ] Add fields like `touched_at`, `expiry_paused_at`, `expiry_disabled` (final shape TBD)
    - [ ] Ensure protojson output exposes these fields for UI countdown display
  - [ ] Implement UI countdown display
    - [ ] Render countdown badge in `agent-ui-system/client/src/components/WidgetRenderer.tsx` (or per-widget)
    - [ ] Stop/hide countdown once server confirms timeout paused/disabled
  - [ ] Implement UI interaction detection + debounced touch calls
    - [ ] Hook into widget containers to capture click/keydown/input events
    - [ ] Debounce touch calls (e.g. once per N seconds)
  - [ ] Update CLI behavior on timeout
    - [ ] Decide how CLI commands exit when `status=timeout` (non-zero exit? structured output? error message?)
  - [ ] Add tests
    - [ ] Go unit tests for expiry scheduler + touch semantics
    - [ ] Frontend: at least typecheck + minimal behavioral coverage if test framework exists

- [ ] Add optional default “timeout response” string returned when timeout expires
  - [ ] Decide where the value lives:
    - [ ] In `UIRequest` envelope (generic), or per-widget input (widget-specific)?
  - [ ] Decide how it maps to outputs:
    - [ ] Does the request end as `status=timeout` with `error` set to this string?
    - [ ] Or does it end as `status=completed` with a synthetic output derived from this string?
  - [ ] Extend create-request JSON contract to accept this value (e.g. `"timeoutDefault": "..."`)
  - [ ] Implement server-side timeout completion behavior using the configured string
  - [ ] Update UI history to show the timeout default result clearly
  - [ ] Update CLI commands to surface the timeout default result in their outputs
  - [ ] Add tests (server timeout path + CLI surface)

- [ ] Document current “request lifecycle” (CLI → server → WS → UI → response → wait)
- [ ] Identify current history mechanisms and limits (UI + server)
- [ ] Choose UX: bounded-only vs backend paging vs both
- [ ] Choose persistence backend (SQLite vs JSONL vs KV) and document decision
- [ ] Specify server-side history API: `GET /api/requests?...` (cursor, limit, filters)
- [ ] Specify request metadata schema (proto) and capture points:
  - CLI-provided (cwd/pid/ppid/parent pids)
  - server-enriched (remote addr / user-agent)
- [ ] Specify auto-default schema and semantics:
  - `autoCompleteAt` vs reuse `expiresAt`
  - completion kind (`user_submitted` vs `auto_default` vs `expired`)
  - how UI labels and summarizes defaulted results
- [ ] Identify required migrations:
  - proto changes + `make codegen`
  - store refactor (introduce interface; persistent impl)
  - new endpoints + UI fetch layer
