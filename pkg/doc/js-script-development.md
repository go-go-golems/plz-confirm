---
Title: JS Script Engine — Development Guide
Slug: js-script-development
Short: Codebase map, runtime internals, dev workflow, and troubleshooting for contributors working on the script engine.
Topics:
- developer
- architecture
- javascript
- scripts
- backend
- frontend
Commands:
- serve
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

This guide is for contributors who need to work on the script engine itself — fixing bugs, adding features, or understanding how the pieces fit together. If you're writing scripts (rather than working on the engine that runs them), see `js-script-api` instead.

The script engine is the machinery that takes a user's JavaScript source, executes it in a sandbox, and wires the results into plz-confirm's request lifecycle. It spans four layers: a Go-side JavaScript runtime, server HTTP handlers, an in-memory store, and frontend rendering logic. This guide walks through each one.

## Finding Your Way Around the Codebase

The script engine touches about a dozen files across the repository. Here's where each piece lives and what it does.

### The Runtime (`internal/scriptengine/`)

This is the core — a thin wrapper around the Goja JavaScript VM that knows how to call the four script contract functions.

- **`engine.go`** — Creates a fresh Goja VM, loads the user's script, and exposes two methods: `InitAndView` (runs `describe` + `init` + `view` for request creation) and `UpdateAndView` (runs `update` and optionally `view` for event handling). Also handles timeout enforcement and export validation.
- **`engine_test.go`** — Tests the contract end-to-end: valid scripts, missing exports, timeout behavior, cancellation, and sandbox exposure checks.

### The Server (`internal/server/`)

These files handle HTTP routing, event dispatch, and WebSocket broadcasting.

- **`server.go`** — The main HTTP router. `handleCreateRequest` detects `type: "script"` and dispatches to the script creation path. This is where script requests diverge from regular widget requests.
- **`script.go`** — All script-specific logic lives here: `handleScriptEvent` processes incoming events, `eventToMap` converts proto events to plain Go maps for the runtime, and `scriptErrorStatus` maps runtime errors to HTTP status codes.
- **`script_test.go`** — Integration tests that exercise the full create-event-complete lifecycle, verify error status mapping, and check that patched state persists correctly.
- **`ws.go`** — WebSocket broadcast logic. Uses a shared write mutex to serialize messages and prevent concurrent write panics. Emits `new_request`, `request_updated`, and `request_completed` events.
- **`ws_test.go`** — Tests that verify event ordering: a script lifecycle should always produce events in the correct sequence, and initial pending replay should be sorted by creation time.

### The Store (`internal/store/`)

- **`store.go`** — An in-memory request store. For scripts, the key addition is `PatchScript`, which updates `scriptState` and `scriptView` on a pending request without completing it. The store also sorts pending requests by `CreatedAt` for deterministic replay when a new WebSocket client connects.

### The Frontend (`agent-ui-system/client/src/`)

The browser side handles rendering script widgets and sending events back to the server.

- **`services/websocket.ts`** — The WebSocket client. Handles `request_updated` events (new for scripts — regular widgets only have `new_request` and `request_completed`). Includes a guard against stale updates: if a `request_updated` arrives for a request that's already completed, it's ignored. Also provides `submitScriptEvent()`, which posts to `/api/requests/{id}/event`.
- **`components/WidgetRenderer.tsx`** — The component that decides what to render. For script requests, it reads `scriptView.widgetType` and `scriptView.input`, then renders the matching widget component (`ConfirmDialog`, `SelectDialog`, `GridDialog`, etc.). On submit, it calls `submitScriptEvent` instead of the regular response endpoint.
- **`store/store.ts`** — Zustand store with a `patchRequest` reducer for in-place updates. Also exports `createAppStore()` as a factory function for test isolation (so tests don't share state).

### Proto Schema (`proto/plz_confirm/v1/`)

The protobuf definitions are the single source of truth for the wire format.

- **`request.proto`** — Defines the `script` value in the `WidgetType` enum, adds `scriptInput`/`scriptOutput` to the `UIRequest` oneofs, and adds `script_state`, `script_view`, `script_describe`, and `script_logs` fields.
- **`widgets.proto`** — Defines `ScriptInput`, `ScriptOutput`, `ScriptEvent`, `ScriptView`, and `ScriptDescribe` messages.

Generated code lives in:
- Go: `proto/generated/go/plz_confirm/v1/*.pb.go`
- TypeScript: `agent-ui-system/client/src/proto/generated/plz_confirm/v1/*.ts`

After changing `.proto` files, regenerate with `make codegen`.

## How the Runtime Works

### Fresh VM Per Call

The engine creates a brand-new runtime for every `InitAndView` or `UpdateAndView` call. It now uses a `go-go-goja` factory to build owned runtimes per invocation (still one fresh VM per call, no VM reuse, no cached script state between calls).

This is intentional. It provides strong isolation (a misbehaving `update` can't corrupt the runtime for the next call) at the cost of some overhead. For the script sizes we expect (small state machines, not heavy computation), this tradeoff is well worth it.

Under the hood this still executes on [Goja](https://github.com/nicholasgasior/goja), but runtime lifecycle is now owned through `go-go-goja` factory/runtime APIs so runtime setup and teardown are explicit and centralized.

### The Sandbox

The script runtime intentionally exposes a constrained Node-style surface:

- `require` is available.
- `console` is available (`log/info/warn/error`) and output is captured by the server.
- `process` is `undefined` — no access to environment variables or OS primitives.

The runtime still primarily exposes standard ES5.1 built-ins (Object, Array, JSON, Math, etc.) plus server-injected context (`__pc_ctx`, `__pc_state`, `__pc_event`), and now includes `require` and capture-aware `console`.

Context helpers now include deterministic random utilities (`ctx.seed`, `ctx.random()`, `ctx.randomInt(min,max)`) and a declarative branch helper (`ctx.branch(state, event, spec)`).

These constraints are enforced by sandbox/runtime tests in `engine_test.go`, including `require` availability, `process` absence, and console capture behavior.

### Timeout and Cancellation

Scripts run under a time limit to prevent infinite loops from locking up the server. Here's how it works:

1. `runWithTimeout` derives a child context with `context.WithTimeout` based on `timeoutMs`.
2. A watcher goroutine starts and blocks on `<-ctx.Done()`. When the context expires (or is cancelled), it calls `vm.Interrupt()` to halt the JavaScript execution.
3. After the script function returns (or is interrupted), a stop-channel signal terminates the watcher goroutine.

When an interruption happens, the engine checks why:

- If the context deadline was exceeded → classified as **timeout** → maps to HTTP `504`.
- If the context was cancelled (e.g. client disconnected) → classified as **cancelled** → maps to HTTP `408`.
- Any other error (script threw an exception, returned wrong shape, etc.) → classified as **runtime fault** → maps to HTTP `422`.

### Export Validation

Before running the lifecycle, the engine validates that the script exports what it should:

1. All four names (`describe`, `init`, `view`, `update`) must exist and be callable functions.
2. `describe()` must return an object with string-typed `name` and `version` fields.
3. `init()` and `view()` must return plain objects (not primitives, not arrays).
4. When `update()` returns `{ done: true }`, the `result` field must be an object.

Any validation failure surfaces as a `400 Bad Request` with a descriptive error message.

## Server Request Flow in Detail

### What Happens on Create (`POST /api/requests`)

1. The server validates that `type == "script"` and `scriptInput` is present with a `script` field.
2. It calls `engine.InitAndView(ctx, scriptInput)`, which runs three functions in sequence:
   - `describe(ctx)` — validates the returned name/version.
   - `init(ctx)` — produces the initial state object.
   - `view(state, ctx)` — produces the first widget to show.
3. The server builds a `UIRequest` proto with `scriptState` (from init), `scriptView` (from view), `scriptDescribe` (from describe), and `scriptLogs` (captured console output from that run).
4. `store.Create(req)` persists the request.
5. A `new_request` event is broadcast to all connected WebSocket clients in the session.

### What Happens on Event (`POST /api/requests/{id}/event`)

1. The server loads the existing request from the store. If it's not in `pending` status, the event is rejected.
2. The incoming `ScriptEvent` proto is parsed from the request body and converted to a `map[string]any` via `eventToMap()`.
3. The server calls `engine.UpdateAndView(ctx, scriptInput, currentState, event)`.
4. If the result is **non-terminal** (no `done: true`):
   - `store.PatchScript(id, newState, newView, logs)` updates the request in place.
   - A `request_updated` event is broadcast over WebSocket.
5. If the result is **terminal** (`done: true`):
   - The request is completed with `scriptOutput.result` and `scriptOutput.logs`.
   - Top-level `scriptLogs` is also updated with the latest run logs.
   - A `request_completed` event is broadcast over WebSocket.

### How ctx Gets Built

The `defaultScriptContext()` function in `engine.go` constructs the context object that all script functions receive:

```go
map[string]any{
    "props": in.GetProps().AsMap(),  // from scriptInput.props, or empty map
    "now":   time.Now().UTC().Format(time.RFC3339Nano),
    "seed":  <per-request deterministic seed>,
    "random":    <seeded float generator>,
    "randomInt": <seeded int-range generator>,
}
```

If you need to add new fields to `ctx` (like a request ID or session info), this is the function to modify.

### How Errors Map to HTTP Status Codes

The `statusForScriptError()` function in `script.go` first classifies using typed script-engine errors (`errors.Is`) and then uses conservative string fallbacks:

| If the error message contains... | HTTP status |
|---|---|
| `"timeout"` | `504 Gateway Timeout` |
| `"cancel"` | `408 Request Timeout` |
| `"must export"`, `"required"`, `"must return"`, `"invalid"` | `400 Bad Request` |
| Anything else | `422 Unprocessable Entity` |

This is deliberately conservative — known validation patterns get `400`, known infrastructure patterns get `504`/`408`, and everything else (script bugs, unexpected exceptions) gets `422`.

## How the Frontend Handles Scripts

### WebSocket Events

The WebSocket client in `websocket.ts` handles three event types for scripts:

- **`new_request`** — adds the request to the pending queue in the Zustand store.
- **`request_updated`** — patches the existing request's `scriptState` and `scriptView` in place. There are two safety guards here:
  - If the request ID is already in the completed set, the update is ignored (prevents stale updates from overwriting final state).
  - If the request ID is unknown (not in pending or completed), the request is enqueued as new (handles the case where the WS client missed the initial `new_request`).
- **`request_completed`** — moves the request from pending to the history/completed set.

### Widget Rendering

When `WidgetRenderer.tsx` detects that the active request has a `scriptView`, it enters the script rendering path:

1. Reads `scriptView.widgetType`, lowercased and trimmed.
2. Reads `scriptView.input` as the widget props.
3. If `scriptView.progress` is present, renders a progress indicator above the widget card.
4. If `scriptView.toast` is present, emits a transient toast notification keyed by request/step/content.
5. If `scriptView.sections` is present, renders composite sections in order (`DisplayWidget` plus exactly one interactive widget). Otherwise, renders the single widget from `scriptView.widgetType`.
6. Renders the matching interactive widget component (`ConfirmDialog`, `SelectDialog`, `GridDialog`, `RatingDialog`, `TableDialog`, `FormDialog`, `UploadDialog`, or `ImageDialog`).
7. When the user submits, it calls `submitScriptEvent(requestId, { type: "submit", stepId, data: output })` instead of the regular `/response` endpoint. If `allowBack` is enabled and the user clicks back, it sends `{ type: "back", stepId }`.

This means script widgets look and behave exactly like regular widgets from the user's perspective — the only difference is what happens when they submit.

Defaults support: script widgets can read `input.defaults` (select/form/table/rating). The renderer keys script widgets by `stepId` so defaults apply on step transitions while preserving in-step user edits during rerenders.

Select widgets support both legacy string options and rich object options with `value/label/description/badge/icon/disabled` fields.

## Local Development

The standard two-terminal setup works for script development:

```bash
# Terminal 1: Go backend on port 3001
go run ./cmd/plz-confirm serve --addr :3001

# Terminal 2: Vite dev server on port 3000 (proxies /api and /ws to :3001)
pnpm -C agent-ui-system dev --host --port 3000
```

Open `http://localhost:3000?sessionId=global` in your browser, then create script requests against port 3000. The Vite proxy forwards API calls and WebSocket connections to the Go backend.

### Running Tests

Backend (the most important ones for script work):

```bash
# Script-specific packages
go test ./internal/scriptengine ./internal/server ./internal/store -count=1

# Full repo
go test ./... -count=1
```

Frontend:

```bash
# Type checking
pnpm -C agent-ui-system run check

# Unit tests (includes script reducer and renderer tests)
pnpm -C agent-ui-system exec vitest run
```

### Regenerating Proto Code

After changing `.proto` files:

```bash
make codegen
```

This runs `protoc` for Go and the TS proto plugin for the frontend. Always run this before committing proto changes.

## Troubleshooting

When something goes wrong in the script engine, the symptoms usually point to a specific layer. Here's a guide to narrowing down where to look.

| What you're seeing | Most likely cause | Where to investigate |
|---|---|---|
| `400` on create with "must export" | Script is missing `describe`, `init`, `view`, or `update` | Check the script source. Look at export validation in `engine.go`. |
| `400` on create with "invalid return shape" | `init` or `view` returned a non-object (string, number, array) | Check what your script functions return. Look at shape validation in `engine.go`. |
| `422` on event submission | Script threw an unhandled exception during `update` or the subsequent `view` call | Check script logic — most commonly, `event.data` is undefined and the script accesses a property on it. |
| `504` on create or event | A script function took longer than `timeoutMs` | Simplify the callback logic or increase the timeout. Check `runWithTimeout` in `engine.go`. |
| Browser shows "unsupported widget" | `view()` returned a `widgetType` that's not in the renderer's switch statement | Check `WidgetRenderer.tsx` — supported types are `confirm`, `select`, `grid`, `rating`, `table`, `form`, `upload`, `image`. |
| Request is stuck in pending forever | `update` keeps returning non-terminal states and never returns `{ done: true }` | Walk through the script's update logic and verify the terminal condition. |
| WebSocket events arrive in wrong order | Concurrent writes to the same WS connection, or client reconnected mid-flow | Check `ws.go` write mutex. Check `websocket.ts` stale-update guard. |
| A `request_updated` event overwrites completed state in the UI | A stale update arrived after the request was already completed | The `websocket.ts` client should be ignoring these — check the completion guard logic. |

## How to Add New Capabilities

### Adding a new field to `ctx`

1. Add the field to `defaultScriptContext()` in `engine.go`.
2. Add a test in `engine_test.go` that runs a script accessing the new field and verifies it gets the right value.
3. Document the new field in the `ctx` table in `js-script-api.md`.

### Supporting a new widget type

1. Create the widget component under `agent-ui-system/client/src/components/widgets/`.
2. Add a `case` branch in the `switch (widgetType)` block in `WidgetRenderer.tsx`.
3. Document the `input` fields and `event.data` shape in the Widget Type Reference section of `js-script-api.md`.

### Changing how errors are classified

Update the string-matching logic in `scriptErrorStatus()` in `script.go`, and add or update the corresponding test case in `script_test.go`.

## See Also

- `js-script-api` — the user-facing API reference, contract, and widget type documentation
- `adding-widgets` — full guide for implementing new widget types across the entire stack (proto, server, CLI, frontend)
