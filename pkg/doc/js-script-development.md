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

This page is for contributors working on the script engine itself — the runtime, server lifecycle, store, and frontend wiring. For the user-facing API contract and widget reference, see `js-script-api`.

## Codebase Map

### Runtime

| File | Role |
|---|---|
| `internal/scriptengine/engine.go` | Goja VM wrapper. Loads script, calls `describe/init/view/update`, enforces timeouts. |
| `internal/scriptengine/engine_test.go` | Contract tests, timeout/cancel tests, sandbox exposure tests. |

### Server Lifecycle

| File | Role |
|---|---|
| `internal/server/server.go` | HTTP router. `handleCreateRequest` dispatches script creation. |
| `internal/server/script.go` | Script-specific handlers: `handleScriptEvent`, error-to-HTTP-status mapping, `eventToMap` conversion. |
| `internal/server/script_test.go` | Integration tests for create -> event -> complete lifecycle, error mapping, persistence checks. |
| `internal/server/ws.go` | WebSocket broadcast. Serialized writes (shared mutex). Emits `new_request`, `request_updated`, `request_completed`. |
| `internal/server/ws_test.go` | Ordering tests for script lifecycle events. |

### Persistence

| File | Role |
|---|---|
| `internal/store/store.go` | In-memory store. `Create` persists script fields. `PatchScript` updates `scriptState`/`scriptView` on non-terminal updates. `CreatedAt` sort for deterministic pending replay. |

### Frontend

| File | Role |
|---|---|
| `agent-ui-system/client/src/services/websocket.ts` | WS client. Handles `request_updated`. `submitScriptEvent()` posts to `/api/requests/{id}/event`. Guards against stale updates for completed requests. |
| `agent-ui-system/client/src/components/WidgetRenderer.tsx` | Maps `scriptView.widgetType` to existing widget components. Passes `input` and wires `onSubmit` to `submitScriptEvent`. |
| `agent-ui-system/client/src/store/store.ts` | Zustand store. `patchRequest` reducer for in-place request updates. `createAppStore()` factory for test isolation. |

### Proto Schema

| File | Role |
|---|---|
| `proto/plz_confirm/v1/request.proto` | `WidgetType.script`, `UIRequest` oneofs for `scriptInput`/`scriptOutput`, `script_state`/`script_view`/`script_describe` fields. |
| `proto/plz_confirm/v1/widgets.proto` | `ScriptInput`, `ScriptOutput`, `ScriptEvent`, `ScriptView`, `ScriptDescribe` messages. |

Generated outputs:
- Go: `proto/generated/go/plz_confirm/v1/*.pb.go`
- TS: `agent-ui-system/client/src/proto/generated/plz_confirm/v1/*.ts`

Regenerate with `make codegen`.

## Runtime Internals

### Goja VM

The script engine uses [goja](https://github.com/nicholasgasior/goja) (a pure-Go ES5.1 runtime) directly, without the `go-go-goja` engine wrapper. This keeps the dependency surface small.

Each `InitAndView` or `UpdateAndView` call creates a fresh `goja.Runtime`, loads the script source, and calls the relevant exports. There is no VM reuse across calls — this is intentional for isolation.

### Sandbox Constraints

The VM exposes no host bridge:
- `require` is `undefined`
- `process` is `undefined`
- No filesystem, network, or OS access

These constraints are validated by `TestSandboxHasNoHostBridge` in `engine_test.go`.

### Timeout and Cancellation

`runWithTimeout` derives a `context.WithTimeout` and starts a watcher goroutine that calls `vm.Interrupt()` when the context completes. After the script function returns, a stop-channel terminates the watcher.

Error classification:
- `*goja.InterruptedError` → check if context deadline exceeded → `timeout` or `cancelled`
- Other errors → `runtime fault`

The server maps these to HTTP status codes via `scriptErrorStatus()` in `script.go`.

### Export Validation

The engine enforces:
1. All four exports (`describe`, `init`, `view`, `update`) must be callable functions.
2. `describe` must return an object with string `name` and `version`.
3. `init` and `view` must return objects (not primitives or arrays).
4. Terminal `update` returns must have `result` as an object.

Validation failures surface as `400 Bad Request`.

## Server Request Flow

### Creation (`POST /api/requests`)

1. Validate `type == "script"` and `scriptInput` present.
2. Call `engine.InitAndView(ctx, scriptInput)`:
   - Runs `describe(ctx)` → validates name/version.
   - Runs `init(ctx)` → produces initial state.
   - Runs `view(state, ctx)` → produces initial view.
3. Build `UIRequest` with `scriptState`, `scriptView`, `scriptDescribe`.
4. `store.Create(req)`.
5. Broadcast `new_request` over WS.

### Event (`POST /api/requests/{id}/event`)

1. Load existing request from store. Reject if not pending.
2. Parse `ScriptEvent` from body → convert to `map[string]any` via `eventToMap`.
3. Call `engine.UpdateAndView(ctx, scriptInput, state, event)`.
4. If non-terminal:
   - `store.PatchScript(id, newState, newView)`.
   - Broadcast `request_updated`.
5. If terminal:
   - Complete request with `scriptOutput.result`.
   - Broadcast `request_completed`.

### ctx Construction

`defaultScriptContext()` in `engine.go` builds:

```go
map[string]any{
    "props": in.GetProps().AsMap(),   // or empty map
    "now":   time.Now().UTC().Format(time.RFC3339Nano),
}
```

### Error Mapping

`scriptErrorStatus()` in `script.go`:

| Error Pattern | HTTP Status |
|---|---|
| Contains "timeout" | `504` |
| Contains "cancel" | `408` |
| Contains "must export", "required", "must return", "invalid" | `400` |
| Everything else | `422` |

## Frontend Wiring

### WebSocket Event Handling

The WS client in `websocket.ts` handles three event types:

- `new_request` → enqueue in store.
- `request_updated` → patch existing request's `scriptState`/`scriptView`. Ignores stale updates for already-completed IDs. If the request ID is unknown, enqueues it.
- `request_completed` → move request to history.

### Widget Rendering

`WidgetRenderer.tsx` detects script requests by checking `active.scriptView`. It:

1. Reads `scriptView.widgetType` (lowercased, trimmed).
2. Reads `scriptView.input` as the widget props.
3. Renders the matching widget component (`ConfirmDialog`, `SelectDialog`, etc.).
4. On submit, calls `submitScriptEvent(requestId, { type: "submit", stepId, data: output })`.

## Local Development

```bash
# Terminal 1: backend
go run ./cmd/plz-confirm serve --addr :3001

# Terminal 2: frontend dev server (proxies /api and /ws to :3001)
pnpm -C agent-ui-system dev --host --port 3000
```

Open `http://localhost:3000?sessionId=global` in a browser.

Test backend:

```bash
go test ./internal/scriptengine ./internal/server ./internal/store -count=1
```

Test frontend:

```bash
pnpm -C agent-ui-system run check
pnpm -C agent-ui-system exec vitest run
```

Regenerate proto:

```bash
make codegen
```

## Troubleshooting

| Problem | Likely Cause | Where to Look |
|---|---|---|
| `400` on create: "must export" | Script missing `describe`, `init`, `view`, or `update` | `engine.go` export validation |
| `400` on create: "invalid return shape" | `init` or `view` returned non-object | `engine.go` shape checks |
| `422` on event | Script threw during `update` or subsequent `view` | Check script logic; add `event.data` guards |
| `504` on create or event | Script exceeded `timeoutMs` | Simplify callbacks or raise timeout |
| Browser: "unsupported widget" | `widgetType` not in renderer switch | `WidgetRenderer.tsx` switch statement |
| Request stuck pending | `update` never returns `{ done: true }` | Verify terminal condition logic |
| WS events arrive out of order | Concurrent writes or client reconnect | `ws.go` write mutex; `websocket.ts` stale-update guard |
| `request_updated` overwrites completed state | Stale update arrived after completion | `websocket.ts` ignores updates for completed IDs |

## Adding New Capabilities

### Adding a new `ctx` field

1. Update `defaultScriptContext()` in `engine.go`.
2. Update the `ctx` table in `js-script-api.md`.
3. Add a test in `engine_test.go` that verifies the field is accessible from script functions.

### Supporting a new `widgetType`

1. Add the widget component under `agent-ui-system/client/src/components/widgets/`.
2. Add a `case` in `WidgetRenderer.tsx`.
3. Document the `input` and `event.data` shapes in `js-script-api.md`.

### Changing error classification

Update `scriptErrorStatus()` in `script.go` and the corresponding test in `script_test.go`.

## See Also

- `js-script-api` — user-facing API contract and widget reference
- `adding-widgets` — full widget implementation guide (proto through frontend)
