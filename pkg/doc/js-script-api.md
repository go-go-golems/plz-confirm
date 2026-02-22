---
Title: JS Script API Reference
Slug: js-script-api
Short: Complete developer guide for plz-confirm's script widget API, contract, lifecycle, and troubleshooting.
Topics:
- developer
- api
- javascript
- scripts
- backend
- frontend
Commands:
- serve
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This page is the end-to-end reference for the JS script functionality in `plz-confirm`. It is written for developers who are new to the codebase and need both a conceptual model and exact API details.

The script API lets you define a multi-step interaction as JavaScript and execute it through the same request lifecycle as other widgets. Instead of sending a static widget input (for example `confirmInput`), you send `scriptInput.script` and let runtime callbacks drive the flow.

## What You Are Building

A script flow is a state machine with four required exports:

- `describe(ctx)` returns metadata (`name`, `version`, optional fields).
- `init(ctx)` returns initial state.
- `view(state, ctx)` returns the UI projection (`widgetType` + `input`).
- `update(state, event, ctx)` returns either next state or terminal `{ done: true, result: ... }`.

The server executes this contract and stores progression in request fields:

- `scriptState`
- `scriptView`
- `scriptDescribe`
- `scriptOutput` (on completion)

## Where This Lives In The Codebase

Start here when reading implementation:

- Runtime:
  - `internal/scriptengine/engine.go`
  - `internal/scriptengine/engine_test.go`
- Server lifecycle:
  - `internal/server/server.go`
  - `internal/server/script.go`
  - `internal/server/script_test.go`
  - `internal/server/ws.go`
  - `internal/server/ws_test.go`
- Persistence:
  - `internal/store/store.go`
- Frontend integration:
  - `agent-ui-system/client/src/services/websocket.ts`
  - `agent-ui-system/client/src/components/WidgetRenderer.tsx`
  - `agent-ui-system/client/src/store/store.ts`

## Lifecycle Overview

The script flow extends the default request lifecycle with one extra endpoint: `/event`.

1. Client creates request:
   - `POST /api/requests` with `type: "script"` + `scriptInput`.
2. Server executes:
   - `describe -> init -> view`.
3. Server stores pending request with `scriptState` and `scriptView`.
4. Browser receives `new_request` over WS.
5. Browser submits events:
   - `POST /api/requests/{id}/event` with `ScriptEvent`.
6. Server executes:
   - `update(state, event, ctx)`.
7. If non-terminal:
   - patch request (`scriptState`/`scriptView`), broadcast `request_updated`.
8. If terminal:
   - complete request with `scriptOutput`, broadcast `request_completed`.
9. CLI/API consumer can observe completion via:
   - `GET /api/requests/{id}`
   - `GET /api/requests/{id}/wait?timeout=<seconds>`

## API Reference

### Create Script Request

Endpoint:

```text
POST /api/requests
Content-Type: application/json
```

Required top-level fields:

- `type`: must be `"script"`.
- `sessionId`: optional, defaults to `"global"` if omitted.
- `scriptInput`: object with `title` and `script`.

`scriptInput` fields:

- `title` (string): human-readable title.
- `script` (string): JavaScript source with required exports.
- `props` (object, optional): values passed to `ctx.props`.
- `timeoutMs` (int64, optional): per-call timeout for runtime invocations.

### Submit Script Event

Endpoint:

```text
POST /api/requests/{id}/event
Content-Type: application/json
```

Payload (`ScriptEvent`):

- `type` (string, required): semantic event type (`submit`, `next`, etc.).
- `stepId` (string, optional): step correlation.
- `actionId` (string, optional): action-level correlation.
- `data` (object, optional): event payload.

### Read Request

Endpoints:

```text
GET /api/requests/{id}
GET /api/requests/{id}/wait?timeout=60
```

Use `GET` to inspect current pending state/view or final output.

### WebSocket Event Types

Endpoint:

```text
GET /ws?sessionId=<id>
```

Event envelope:

```json
{
  "type": "new_request|request_updated|request_completed",
  "request": { "...UIRequest protojson..." }
}
```

For script flows:

- `new_request`: initial `describe/init/view` snapshot.
- `request_updated`: intermediate progression after `update`.
- `request_completed`: terminal state with `scriptOutput`.

## Script Contract In Detail

### `describe(ctx)`

Purpose:

- identify script behavior and version.

Must return object with:

- `name` (string, required)
- `version` (string, required)

Optional:

- `apiVersion` (string)
- `capabilities` (string[])

### `init(ctx)`

Purpose:

- produce initial state object.

Requirements:

- must return an object.
- should be deterministic for same `ctx.props` unless intentional randomness is required.

### `view(state, ctx)`

Purpose:

- project state to a renderable widget instruction.

Must return object with:

- `widgetType` (string, required), one of currently supported renderer mappings:
  - `confirm`
  - `select`
  - `table`
  - `form`
  - `upload`
  - `image`
- `input` (object, optional but usually present)

Optional fields used by UI:

- `stepId`
- `title`
- `description`

### `update(state, event, ctx)`

Purpose:

- consume input event and advance flow.

Return either:

- next state object, or
- terminal object:

```json
{ "done": true, "result": { "...any object..." } }
```

Notes:

- If `done: true`, `result` must be an object.
- If non-terminal, returned object becomes new `scriptState`, then `view` is called again.

## Runtime Behavior And Constraints

Execution model:

- Runtime is Goja (`internal/scriptengine`).
- Server enforces required exports.
- Each invocation is timeout-bounded (`timeoutMs` or default).

Current sandbox constraints:

- `require` is unavailable.
- `process` is unavailable.
- No host module bridge is exposed.

Cancellation and timeout:

- Timeout/cancel interrupts runtime execution.
- Cancellation and timeout are surfaced as mapped HTTP statuses (below).

## Error Mapping

Script errors map to response status classes:

- `400 Bad Request`:
  - contract/payload validation failures
  - missing required exports/fields
  - invalid shape (non-object where object required)
- `422 Unprocessable Entity`:
  - runtime script fault during execution (for example thrown error in `update`)
- `504 Gateway Timeout`:
  - script execution exceeded `timeoutMs`
- `408 Request Timeout`:
  - execution cancelled by request context cancellation

## Example 1: Minimal One-Step Confirm Script

```javascript
module.exports = {
  describe: function () {
    return { name: "minimal-confirm", version: "1.0.0" };
  },
  init: function () {
    return { step: "confirm" };
  },
  view: function () {
    return {
      widgetType: "confirm",
      input: {
        title: "Continue?",
        approveText: "Yes",
        rejectText: "No"
      },
      stepId: "confirm"
    };
  },
  update: function (state, event) {
    return {
      done: true,
      result: { approved: !!(event.data && event.data.approved) }
    };
  }
};
```

## Example 2: Two-Step Confirm -> Select Script

```javascript
module.exports = {
  describe: function () {
    return {
      name: "deploy-wizard",
      version: "1.0.0",
      apiVersion: "v1",
      capabilities: ["submit"]
    };
  },
  init: function () {
    return { step: "confirm" };
  },
  view: function (state) {
    if (state.step === "confirm") {
      return {
        widgetType: "confirm",
        stepId: "confirm",
        input: {
          title: "Ship to production?",
          message: "Reject to pick a target env manually",
          approveText: "Ship",
          rejectText: "Choose env"
        }
      };
    }
    return {
      widgetType: "select",
      stepId: "pick-env",
      input: {
        title: "Pick environment",
        options: ["staging", "prod"],
        multi: false,
        searchable: false
      }
    };
  },
  update: function (state, event) {
    if (state.step === "confirm") {
      if (event.type === "submit" && event.data && event.data.approved === true) {
        return { done: true, result: { approved: true, env: "prod" } };
      }
      state.step = "pick";
      return state;
    }

    return {
      done: true,
      result: {
        approved: false,
        env: event.data ? event.data.selectedSingle : "staging"
      }
    };
  }
};
```

## Example 3: Full API Test Sequence (Copy/Paste)

Create script source:

```bash
cat >/tmp/plz-script.js <<'JS'
module.exports = {
  describe: function () { return { name: "api-seq", version: "1.0.0" }; },
  init: function () { return { step: "confirm" }; },
  view: function (state) {
    if (state.step === "confirm") {
      return { widgetType: "confirm", stepId: "confirm", input: { title: "Proceed?" } };
    }
    return {
      widgetType: "select",
      stepId: "select",
      input: { title: "Choose env", options: ["staging", "prod"], multi: false, searchable: false }
    };
  },
  update: function (state, event) {
    if (state.step === "confirm") {
      state.step = "select";
      return state;
    }
    return { done: true, result: { env: event.data.selectedSingle } };
  }
};
JS
```

Create request:

```bash
REQ_ID=$(
  jq -n --rawfile script /tmp/plz-script.js \
    '{type:"script",sessionId:"global",scriptInput:{title:"API Sequence",timeoutMs:1500,script:$script}}' \
  | curl -sS -X POST http://localhost:3000/api/requests \
      -H 'Content-Type: application/json' \
      -d @- \
  | jq -r '.id'
)
echo "$REQ_ID"
```

Advance first step:

```bash
curl -sS -X POST "http://localhost:3000/api/requests/$REQ_ID/event" \
  -H 'Content-Type: application/json' \
  -d '{"type":"submit","stepId":"confirm","data":{"approved":false}}' | jq
```

Complete second step:

```bash
curl -sS -X POST "http://localhost:3000/api/requests/$REQ_ID/event" \
  -H 'Content-Type: application/json' \
  -d '{"type":"submit","stepId":"select","data":{"selectedSingle":"staging"}}' | jq
```

Verify persisted/final state:

```bash
curl -sS "http://localhost:3000/api/requests/$REQ_ID" | jq '{status,scriptState,scriptView,scriptOutput}'
```

## Local Development Workflow

Recommended local setup:

```bash
# terminal 1 (backend)
go run ./cmd/plz-confirm serve --addr :3001

# terminal 2 (frontend dev server)
pnpm -C agent-ui-system dev --host --port 3000
```

Open browser:

- `http://localhost:3000?sessionId=global`

In this mode:

- Browser connects WS to `/ws` on port `3000` (proxied to backend `:3001`).
- API calls to `/api/*` on `3000` are proxied to backend `:3001`.

## Common Mistakes

- Missing required export:
  - script compiles, but create fails because `describe/init/view/update` is incomplete.
- Returning primitive from `init` or `view`:
  - server expects object and returns validation error.
- Returning `done: true` without object result:
  - server treats result shape as invalid.
- Unsupported `widgetType` in `view`:
  - frontend renders explicit unsupported-widget error.
- Ignoring timeout:
  - long loops in script callbacks can trigger `504`.

## Troubleshooting

| Problem | Likely Cause | Solution |
|---|---|---|
| `400` on create (`script init failed` / `must export`) | Missing required export or invalid return shape | Validate contract: all four exports must exist and return objects where required |
| `422` on event (`script update failed`) | Script threw runtime error in `update` or `view` | Add guards in script logic and log intermediate state/event values |
| `504` on create/event | Callback exceeded `timeoutMs` | Simplify callback work, avoid loops, or raise `timeoutMs` conservatively |
| Browser shows unsupported script widget | `view.widgetType` not mapped in renderer | Use a supported widget type or add renderer support in `WidgetRenderer.tsx` |
| Request appears stuck pending | Non-terminal `update` result keeps cycling or no completion event submitted | Verify event payload and that terminal condition returns `{done:true,result:{...}}` |

## Rollout Guidance

- Gate script flow usage (config or allowlist) before broad enablement.
- Start with internal sessions.
- Track and alert on script status mix (`400/408/422/504`), timeout spikes, and runtime-fault spikes.
- Keep non-script command paths unchanged while rolling out script behavior.

## See Also

- `how-to-use`
- `adding-widgets`
- `internal/scriptengine/engine.go` (runtime contract implementation)
- `internal/server/script.go` (event lifecycle and mapping)
