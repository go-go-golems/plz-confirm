---
Title: JS Script API Reference
Slug: js-script-api
Short: API reference for plz-confirm's script widget — contract, endpoints, widget mapping, and error codes.
Topics:
- developer
- api
- javascript
- scripts
Commands:
- serve
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Reference
---

The script API lets you define a multi-step interaction as a JavaScript state machine. Instead of sending a single static widget input (e.g. `confirmInput`), you send `scriptInput.script` and the server drives the flow through runtime callbacks.

## Quick Start

Write a script file:

```javascript
// /tmp/plz-script.js
module.exports = {
  describe: function () {
    return { name: "my-flow", version: "1.0.0" };
  },
  init: function () {
    return { step: "confirm" };
  },
  view: function (state) {
    return {
      widgetType: "confirm",
      input: { title: "Continue?", approveText: "Yes", rejectText: "No" },
      stepId: "confirm"
    };
  },
  update: function (state, event) {
    return { done: true, result: { approved: !!(event.data && event.data.approved) } };
  }
};
```

Create the request:

```bash
REQ_ID=$(
  jq -n --rawfile script /tmp/plz-script.js \
    '{type:"script",sessionId:"global",scriptInput:{title:"Demo",timeoutMs:1500,script:$script}}' \
  | curl -sS -X POST http://localhost:3000/api/requests \
      -H 'Content-Type: application/json' -d @- \
  | jq -r '.id'
)
```

Submit an event:

```bash
curl -sS -X POST "http://localhost:3000/api/requests/$REQ_ID/event" \
  -H 'Content-Type: application/json' \
  -d '{"type":"submit","stepId":"confirm","data":{"approved":true}}'
```

Read the result:

```bash
curl -sS "http://localhost:3000/api/requests/$REQ_ID" | jq '.scriptOutput'
```

## Lifecycle

```text
Agent / CLI                      Server                              Browser
──────────                      ──────                              ───────
POST /api/requests         ──>  describe() -> init() -> view()
  type:"script"                 store scriptState + scriptView
  scriptInput:{script,...}      broadcast "new_request"        ──>  render widget

                                                               <──  user interacts
                           <──  POST /api/requests/{id}/event
                                update(state, event, ctx)
                                  ├─ non-terminal: patch state/view
                                  │  broadcast "request_updated"  ──>  re-render
                                  └─ terminal: store scriptOutput
                                     broadcast "request_completed" ──> show done

GET /api/requests/{id}     ──>  return final state
GET .../wait?timeout=60    ──>  long-poll until completed
```

## Script Contract

Your script must `module.exports` four functions.

### `describe(ctx) -> object`

Returns metadata identifying the script. Called once on request creation.

Required fields:
- `name` (string)
- `version` (string)

Optional fields:
- `apiVersion` (string) — for future contract versioning.
- `capabilities` (string[]) — declares what event types the script handles.

### `init(ctx) -> object`

Returns the initial state. Called once after `describe`.

- Must return a plain object (not a primitive, not an array).
- Should be deterministic for the same `ctx.props`.

### `view(state, ctx) -> object`

Projects state into a widget instruction. Called after `init` and after every non-terminal `update`.

Required fields:
- `widgetType` (string) — which widget to render (see Widget Type Reference below).
- `input` (object) — widget-specific configuration (see Widget Type Reference below).

Optional fields:
- `stepId` (string) — correlates the view to a logical step; passed back in `event.stepId`.
- `title` (string) — override title shown in UI.
- `description` (string) — supplementary text shown in UI.

### `update(state, event, ctx) -> object`

Consumes a user event and advances the flow. Return one of:

**Non-terminal** — return a state object. The server calls `view(newState, ctx)` and broadcasts `request_updated`.

```javascript
// mutating the state in place is fine
state.step = "next";
return state;
```

**Terminal** — return `{ done: true, result: {...} }`. The server stores `result` as `scriptOutput` and broadcasts `request_completed`.

```javascript
return { done: true, result: { approved: true, env: "prod" } };
```

`result` must be a plain object.

### The `ctx` Object

All four functions receive `ctx` as their last argument. It contains:

| Field | Type | Description |
|---|---|---|
| `ctx.props` | object | Values from `scriptInput.props`. Empty object if omitted. |
| `ctx.now` | string | Current server time (RFC 3339 with nanoseconds). |

Example using props:

```javascript
init: function (ctx) {
  return { step: "confirm", env: ctx.props.defaultEnv || "staging" };
}
```

### The `event` Object

The `event` passed to `update` has this shape:

| Field | Type | Description |
|---|---|---|
| `event.type` | string | Semantic event type. The UI always sends `"submit"`. |
| `event.stepId` | string? | Echoed from `view().stepId` if set. |
| `event.actionId` | string? | Action-level correlation (not commonly used). |
| `event.data` | object? | Widget output payload. Shape depends on widget type. |

## Widget Type Reference

Each `widgetType` returned by `view()` maps to an existing plz-confirm widget. The `input` object mirrors the widget's proto input, and `event.data` mirrors its proto output.

### `confirm`

Simple yes/no dialog.

`input` fields:
- `title` (string, required)
- `message` (string, optional)
- `approveText` (string, optional, default "Approve")
- `rejectText` (string, optional, default "Reject")

`event.data` on submit:
- `approved` (boolean)
- `timestamp` (string, ISO 8601)
- `comment` (string, optional)

### `select`

Single or multi-select from a list.

`input` fields:
- `title` (string, required)
- `options` (string[], required)
- `multi` (boolean, optional, default false)
- `searchable` (boolean, optional, default false)

`event.data` on submit:
- `selectedSingle` (string) — when `multi: false`
- `selectedMulti` (`{ values: string[] }`) — when `multi: true`
- `comment` (string, optional)

### `form`

Dynamic form driven by JSON Schema.

`input` fields:
- `title` (string, required)
- `schema` (object, required) — JSON Schema definition

`event.data` on submit:
- The form field values as a flat object (matches schema properties).
- `comment` (string, optional)

### `table`

Tabular data with row selection.

`input` fields:
- `title` (string, required)
- `data` (object[], required) — row objects
- `columns` (string[], required) — column keys to display
- `multiSelect` (boolean, optional, default false)
- `searchable` (boolean, optional, default false)

`event.data` on submit:
- `selectedSingle` (object) — when `multiSelect: false`
- `selectedMulti` (`{ values: object[] }`) — when `multiSelect: true`
- `comment` (string, optional)

### `upload`

File upload dialog.

`input` fields:
- `title` (string, required)
- `accept` (string[], optional) — MIME types or extensions
- `multiple` (boolean, optional, default false)
- `maxSize` (number, optional) — bytes
- `callbackUrl` (string, optional)

`event.data` on submit:
- `files` (`{ name, size, path, mimeType }[]`)
- `comment` (string, optional)

### `image`

Image display with selection or confirmation.

`input` fields:
- `title` (string, required)
- `message` (string, optional)
- `images` (`{ src, alt?, label?, caption? }[]`, required)
- `mode` (string, `"select"` or `"confirm"`)
- `options` (string[], optional) — text choices shown alongside images
- `multi` (boolean, optional, default false)

`event.data` on submit (varies by mode):
- `selectedNumber` / `selectedNumbers` — image index(es) when mode is `"select"` without `options`
- `selectedString` / `selectedStrings` — option value(s) when `options` are provided
- `selectedBool` — when mode is `"confirm"`
- `timestamp` (string, ISO 8601)
- `comment` (string, optional)

## HTTP Endpoints

### Create Script Request

```text
POST /api/requests
Content-Type: application/json
```

Body:

```json
{
  "type": "script",
  "sessionId": "global",
  "scriptInput": {
    "title": "My Flow",
    "script": "module.exports = { ... }",
    "props": { "key": "value" },
    "timeoutMs": 5000
  }
}
```

`scriptInput` fields:

| Field | Type | Required | Description |
|---|---|---|---|
| `title` | string | yes | Human-readable title |
| `script` | string | yes | JavaScript source with required exports |
| `props` | object | no | Passed to `ctx.props` in all script functions |
| `timeoutMs` | int64 | no | Per-call execution timeout in milliseconds |

Response: `UIRequest` with `scriptState`, `scriptView`, `scriptDescribe` populated.

### Submit Script Event

```text
POST /api/requests/{id}/event
Content-Type: application/json
```

Body:

```json
{
  "type": "submit",
  "stepId": "confirm",
  "data": { "approved": true }
}
```

Response: updated `UIRequest`. If terminal, includes `scriptOutput`.

### Read / Wait

```text
GET /api/requests/{id}
GET /api/requests/{id}/wait?timeout=60
```

The `/wait` endpoint long-polls until the request completes or the timeout elapses.

### WebSocket

```text
GET /ws?sessionId=<id>
```

Event envelope:

```json
{ "type": "new_request|request_updated|request_completed", "request": { ... } }
```

- `new_request` — emitted after `describe/init/view` on creation.
- `request_updated` — emitted after a non-terminal `update`.
- `request_completed` — emitted when `update` returns `{ done: true }`.

## Error Codes

| Status | Meaning | Common Cause |
|---|---|---|
| `400` | Validation failure | Missing export, non-object return from `init`/`view`, invalid `scriptInput` |
| `408` | Cancelled | Request context cancelled (e.g. client disconnect) |
| `422` | Runtime fault | Script threw an error during `update` or `view` |
| `504` | Timeout | Script execution exceeded `timeoutMs` |

## Examples

### Two-Step: Confirm then Select

```javascript
module.exports = {
  describe: function () {
    return { name: "deploy-wizard", version: "1.0.0" };
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
      if (event.data && event.data.approved) {
        return { done: true, result: { env: "prod" } };
      }
      state.step = "pick";
      return state;
    }
    return {
      done: true,
      result: { env: event.data ? event.data.selectedSingle : "staging" }
    };
  }
};
```

### Using Props for Configuration

```javascript
module.exports = {
  describe: function () {
    return { name: "configurable-confirm", version: "1.0.0" };
  },
  init: function (ctx) {
    return { action: ctx.props.action || "deploy" };
  },
  view: function (state, ctx) {
    return {
      widgetType: "confirm",
      input: {
        title: "Confirm " + state.action + "?",
        message: ctx.props.message || ""
      }
    };
  },
  update: function (state, event) {
    return { done: true, result: { action: state.action, approved: !!(event.data && event.data.approved) } };
  }
};
```

Create with props:

```bash
jq -n --rawfile script /tmp/configurable.js \
  '{type:"script",scriptInput:{title:"Deploy",script:$script,props:{action:"deploy",message:"This deploys v2.1"}}}' \
| curl -sS -X POST http://localhost:3000/api/requests -H 'Content-Type: application/json' -d @-
```

### Full API Test Sequence (Copy/Paste)

Save the two-step script to a file, then run:

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
      widgetType: "select", stepId: "select",
      input: { title: "Choose env", options: ["staging", "prod"], multi: false, searchable: false }
    };
  },
  update: function (state, event) {
    if (state.step === "confirm") { state.step = "select"; return state; }
    return { done: true, result: { env: event.data.selectedSingle } };
  }
};
JS

# Create
REQ_ID=$(
  jq -n --rawfile script /tmp/plz-script.js \
    '{type:"script",sessionId:"global",scriptInput:{title:"API Sequence",timeoutMs:1500,script:$script}}' \
  | curl -sS -X POST http://localhost:3000/api/requests \
      -H 'Content-Type: application/json' -d @- \
  | jq -r '.id'
)

# Step 1: advance past confirm
curl -sS -X POST "http://localhost:3000/api/requests/$REQ_ID/event" \
  -H 'Content-Type: application/json' \
  -d '{"type":"submit","stepId":"confirm","data":{"approved":false}}' | jq

# Step 2: complete with selection
curl -sS -X POST "http://localhost:3000/api/requests/$REQ_ID/event" \
  -H 'Content-Type: application/json' \
  -d '{"type":"submit","stepId":"select","data":{"selectedSingle":"staging"}}' | jq

# Verify
curl -sS "http://localhost:3000/api/requests/$REQ_ID" \
  | jq '{status,scriptState,scriptView,scriptOutput}'
```

## Common Mistakes

| Mistake | Symptom | Fix |
|---|---|---|
| Missing a required export (`describe`, `init`, `view`, or `update`) | `400` on create | Add all four exports to `module.exports` |
| Returning a primitive or array from `init` or `view` | `400` — "invalid return shape" | Return a plain object `{}` |
| Returning `{ done: true }` without `result` being an object | `400` — invalid terminal shape | Use `{ done: true, result: { ... } }` |
| Using an unsupported `widgetType` | Browser shows "unsupported widget" error | Use one of: `confirm`, `select`, `table`, `form`, `upload`, `image` |
| Infinite loop or heavy computation in a callback | `504` timeout | Keep callbacks simple; raise `timeoutMs` if needed |
| Accessing `event.data.x` without null-checking `event.data` | `422` runtime error | Guard: `event.data && event.data.x` |

## See Also

- `how-to-use` — end-user and agent-developer usage guide
- `adding-widgets` — adding new widget types to plz-confirm
- `js-script-development` — codebase internals, dev workflow, and troubleshooting for contributors
