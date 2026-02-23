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

Most plz-confirm widgets are one-shot: you create a request, the user responds, done. Scripts are different. A script is a small JavaScript program that drives a **multi-step conversation** — showing one widget, reacting to the user's answer, then showing another widget (or finishing). Think of it as a wizard or flow builder that lives entirely in a single JS file.

Under the hood, the server runs your script in a sandboxed JavaScript runtime. You don't need to set up a frontend, manage WebSocket connections, or write any Go code. You just write four functions and the server handles the rest.

## Quick Start

Here's the smallest possible script — a single confirmation step:

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

Send it to the server:

```bash
REQ_ID=$(
  jq -n --rawfile script /tmp/plz-script.js \
    '{type:"script",sessionId:"global",scriptInput:{title:"Demo",timeoutMs:1500,script:$script}}' \
  | curl -sS -X POST http://localhost:3000/api/requests \
      -H 'Content-Type: application/json' -d @- \
  | jq -r '.id'
)
```

At this point the browser shows a "Continue?" confirmation dialog. Once the user clicks, submit their answer:

```bash
curl -sS -X POST "http://localhost:3000/api/requests/$REQ_ID/event" \
  -H 'Content-Type: application/json' \
  -d '{"type":"submit","stepId":"confirm","data":{"approved":true}}'
```

And read back the final result:

```bash
curl -sS "http://localhost:3000/api/requests/$REQ_ID" | jq '.scriptOutput'
# => { "result": { "approved": true } }
```

That's the full cycle: create a script request, the user interacts through the browser, you collect the result.

## How the Lifecycle Works

A script flow has three participants — the agent (or CLI) that creates the request, the server that runs your JavaScript, and the browser that renders widgets to the user.

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

When you create a script request, the server immediately runs three of your functions in sequence: `describe` (to identify the script), `init` (to set up initial state), and `view` (to produce the first widget the user sees). That initial widget is broadcast to the browser over WebSocket.

Each time the user interacts (clicks a button, submits a form), the browser sends an event to the server via `POST /api/requests/{id}/event`. The server calls your `update` function with the current state and the event. Your `update` either returns a new state (and the cycle repeats — `view` is called again, a new widget appears) or returns `{ done: true, result: {...} }` to finish the flow.

## The Script Contract

Your script must export exactly four functions via `module.exports`. Each one has a specific job in the lifecycle.

### `describe(ctx)` — Identify the script

Called once when the request is created. Returns metadata so the server (and humans reading logs) know what script is running.

```javascript
describe: function (ctx) {
  return {
    name: "deploy-wizard",    // required — a short identifier
    version: "1.0.0"          // required — helps track which version ran
  };
}
```

You can also include `apiVersion` (for future contract versioning) and `capabilities` (an array of strings declaring what event types the script handles), but neither is required today.

### `init(ctx)` — Set up initial state

Called once, right after `describe`. Returns a plain object representing the starting state of your flow. This state is what gets passed to `view` and later to `update`.

```javascript
init: function (ctx) {
  return { step: "confirm", retries: 0 };
}
```

The only hard rule: you must return a plain object (not a string, number, or array). Beyond that, the shape is entirely up to you — put whatever your flow needs to track in here.

### `view(state, ctx)` — Decide what to show

Called after `init` and again after every non-terminal `update`. Given the current state, return a widget instruction that tells the browser what to render.

```javascript
view: function (state, ctx) {
  if (state.step === "confirm") {
    return {
      widgetType: "confirm",                          // required — which widget
      input: { title: "Ship it?", approveText: "Go" }, // required — widget config
      stepId: "confirm"                                // optional — echoed back in events
    };
  }
  return {
    widgetType: "select",
    input: { title: "Pick env", options: ["staging", "prod"] },
    stepId: "pick-env"
  };
}
```

The return object supports two modes:

- **Single-widget mode (backward compatible):** include `widgetType` and `input`.
- **Composite mode:** include `sections`, where each section has its own `widgetType` + `input`. In composite mode, exactly one section must be interactive and any additional sections should be `display`.

Single-widget `widgetType` values are `confirm`, `select`, `grid`, `form`, `table`, `upload`, or `image`. See the Widget Type Reference below for details.

You can also include `stepId` (a string that gets echoed back in the event, useful for correlating which step the user responded to), `title`, and `description` (both shown in the UI above the widget).

### `update(state, event, ctx)` — React to user input

Called each time the user submits a response. You receive the current state and the event from the browser. Return either:

**A new state** (non-terminal) — the flow continues. The server calls `view` with your new state and shows the next widget:

```javascript
// Mutating state in place works fine — it's just a plain object
state.step = "pick-env";
state.retries++;
return state;
```

**A terminal result** — the flow is done. The server stores your result as `scriptOutput` and completes the request:

```javascript
return { done: true, result: { approved: true, env: "prod" } };
```

The `result` must be a plain object. It becomes the final output that the CLI or API consumer reads.

### The `ctx` Object

All four functions receive a `ctx` object as their last argument. It gives you access to configuration and server context:

| Field | Type | What it is |
|---|---|---|
| `ctx.props` | object | Custom values you passed in `scriptInput.props` when creating the request. Defaults to `{}` if you didn't send any. |
| `ctx.now` | string | Current server time as an RFC 3339 timestamp with nanoseconds. Useful for generating unique IDs or recording when things happened. |

Props are the main way to make scripts configurable without changing the source code. For example, you might pass `{ defaultEnv: "staging" }` in props and use it in `init`:

```javascript
init: function (ctx) {
  return { step: "confirm", env: ctx.props.defaultEnv || "prod" };
}
```

### The `event` Object

When the user interacts with a widget, the browser sends an event to the server, which passes it to your `update` function. The event looks like this:

| Field | Type | What it is |
|---|---|---|
| `event.type` | string | Always `"submit"` for standard widget interactions. |
| `event.stepId` | string or undefined | Echoed from whatever `stepId` you set in `view()`. Useful for knowing which step the user just responded to. |
| `event.actionId` | string or undefined | Optional action-level correlation. Not commonly used. |
| `event.data` | object or undefined | The actual user response. Its shape depends on the widget type — a confirm gives you `{ approved: true }`, a select gives you `{ selectedSingle: "prod" }`, etc. |

**Always guard against `event.data` being undefined** — if the browser sends a malformed event, accessing `event.data.approved` directly will crash your script with a runtime error (`422`).

```javascript
// Safe pattern:
var approved = event.data && event.data.approved;
```

## Widget Type Reference

Each `widgetType` you return from `view()` maps to one of plz-confirm's existing widgets. This section documents what goes into `input` (the configuration you provide) and what comes back in `event.data` (the user's response).

### `confirm` — Yes/No Dialog

The simplest widget. Shows a question with two buttons.

**What you put in `input`:**

| Field | Type | Default | Description |
|---|---|---|---|
| `title` | string | (required) | The question to display |
| `message` | string | — | Additional context below the title |
| `approveText` | string | "Approve" | Label for the positive button |
| `rejectText` | string | "Reject" | Label for the negative button |

**What you get back in `event.data`:**

| Field | Type | Description |
|---|---|---|
| `approved` | boolean | `true` if the user clicked the approve button |
| `timestamp` | string | ISO 8601 timestamp of the response |
| `comment` | string? | Optional comment if the UI supports it |

### `select` — Pick from a List

Shows a dropdown or list of options. Supports single or multi-selection.

**What you put in `input`:**

| Field | Type | Default | Description |
|---|---|---|---|
| `title` | string | (required) | Prompt displayed above the list |
| `options` | string[] | (required) | The choices to display |
| `multi` | boolean | `false` | Allow selecting multiple options |
| `searchable` | boolean | `false` | Show a search/filter box |

**What you get back in `event.data`:**

For single-select (`multi: false`):
- `selectedSingle` (string) — the chosen option.

For multi-select (`multi: true`):
- `selectedMulti` (`{ values: ["a", "b"] }`) — array of chosen options wrapped in an object.

Both may include an optional `comment`.

### `grid` — Spatial Grid Selection

Renders a clickable 2D board for spatial interactions (games, seating, calendars).

**What you put in `input`:**

| Field | Type | Default | Description |
|---|---|---|---|
| `title` | string | (required) | Heading above the grid |
| `rows` | number | (required) | Number of rows |
| `cols` | number | (required) | Number of columns |
| `cells` | array | (required) | Flat array of `rows * cols` cells. Each cell can include `value`, `style`, `disabled`, `label`, and `color`. |
| `cellSize` | string | `"medium"` | One of `"small"`, `"medium"`, `"large"` |

**What you get back in `event.data`:**

| Field | Type | Description |
|---|---|---|
| `row` | number | Zero-based row index |
| `col` | number | Zero-based column index |
| `cellIndex` | number | Flat zero-based index in the `cells` array |

### `display` — Read-Only Context Section

Used in composite `sections` mode to show formatted context above or between interactive widgets.

**What you put in `input`:**

| Field | Type | Default | Description |
|---|---|---|---|
| `content` | string | (required) | Text/markup to render |
| `format` | string | `"markdown"` | One of `"markdown"`, `"text"`, `"html"` |

`display` sections are read-only and do not produce `event.data`.

### `form` — Dynamic Form

Renders a form from a JSON Schema definition. Great for collecting structured data with validation.

**What you put in `input`:**

| Field | Type | Description |
|---|---|---|
| `title` | string | Form heading |
| `schema` | object | A JSON Schema object defining the form fields, types, and validation rules |

**What you get back in `event.data`:**

A flat object with keys matching your schema's `properties`. For example, if your schema has `username` and `email` fields, you get `{ username: "alice", email: "alice@example.com" }`.

### `table` — Row Selection

Displays tabular data and lets the user pick one or more rows.

**What you put in `input`:**

| Field | Type | Default | Description |
|---|---|---|---|
| `title` | string | (required) | Table heading |
| `data` | object[] | (required) | Array of row objects |
| `columns` | string[] | (required) | Which keys to show as columns |
| `multiSelect` | boolean | `false` | Allow selecting multiple rows |
| `searchable` | boolean | `false` | Show a search/filter box |

**What you get back in `event.data`:**

For single-select: `selectedSingle` contains the full row object.
For multi-select: `selectedMulti` contains `{ values: [{...}, {...}] }`.

### `upload` — File Upload

Shows a file picker with optional type and size restrictions.

**What you put in `input`:**

| Field | Type | Default | Description |
|---|---|---|---|
| `title` | string | (required) | Upload prompt |
| `accept` | string[] | all types | MIME types or extensions (e.g. `".pdf"`, `"image/png"`) |
| `multiple` | boolean | `false` | Allow uploading more than one file |
| `maxSize` | number | — | Maximum file size in bytes |
| `callbackUrl` | string | — | Optional callback URL (not currently implemented) |

**What you get back in `event.data`:**

- `files` — an array of `{ name, size, path, mimeType }` objects, one per uploaded file.

### `image` — Image Display with Selection

Shows one or more images and lets the user respond by selecting images, picking text options, or confirming yes/no.

**What you put in `input`:**

| Field | Type | Default | Description |
|---|---|---|---|
| `title` | string | (required) | Heading above the images |
| `message` | string | — | Prompt text below the heading |
| `images` | array | (required) | Array of `{ src, alt?, label?, caption? }` objects. `src` can be a URL or data URI. |
| `mode` | string | `"select"` | Either `"select"` (pick images/options) or `"confirm"` (yes/no) |
| `options` | string[] | — | Text options shown alongside the images (when provided, the user picks from these instead of the images themselves) |
| `multi` | boolean | `false` | Allow selecting multiple items |

**What you get back in `event.data`** depends on the mode:

- **Select mode without `options`:** `selectedNumber` (int) or `selectedNumbers` (`{ values: [0, 2] }`) — zero-based image indexes.
- **Select mode with `options`:** `selectedString` or `selectedStrings` (`{ values: ["opt1", "opt2"] }`).
- **Confirm mode:** `selectedBool` (boolean).

All modes also include `timestamp` (ISO 8601) and an optional `comment`.

## HTTP Endpoints

### Create Script Request

```text
POST /api/requests
Content-Type: application/json
```

This is the same endpoint used for all widget types. For scripts, set `type` to `"script"` and provide a `scriptInput` object:

```json
{
  "type": "script",
  "sessionId": "global",
  "scriptInput": {
    "title": "My Flow",
    "script": "module.exports = { describe: ... }",
    "props": { "env": "staging" },
    "timeoutMs": 5000
  }
}
```

| Field | Type | Required | What it does |
|---|---|---|---|
| `title` | string | yes | Human-readable title shown in the UI and logs |
| `script` | string | yes | Your JavaScript source code (the full content of the script file) |
| `props` | object | no | Arbitrary values made available to your script as `ctx.props` |
| `timeoutMs` | int64 | no | Maximum execution time per function call in milliseconds. If your `init` or `update` takes longer than this, the server kills it and returns a `504`. |

The response is a full `UIRequest` object with `scriptState`, `scriptView`, and `scriptDescribe` already populated from the initial `describe/init/view` run.

### Submit Script Event

```text
POST /api/requests/{id}/event
Content-Type: application/json
```

Send the user's response to advance the flow:

```json
{
  "type": "submit",
  "stepId": "confirm",
  "data": { "approved": true }
}
```

The server calls your `update` function with the current state and this event. The response is the updated `UIRequest` — either still pending (with a new `scriptView`) or completed (with `scriptOutput`).

### Read and Wait

```text
GET /api/requests/{id}
GET /api/requests/{id}/wait?timeout=60
```

Use plain `GET` to check the current state of a request at any time. Use `/wait` to long-poll until the request completes or the timeout (in seconds) elapses — this is what the CLI uses to block until the user finishes.

### WebSocket

```text
GET /ws?sessionId=<id>
```

The browser connects here to receive real-time updates. Each message is a JSON envelope:

```json
{ "type": "new_request", "request": { "...full UIRequest..." } }
```

Three event types are relevant for scripts:

- **`new_request`** — the script was just created; initial widget is ready to render.
- **`request_updated`** — the user submitted an event and the script advanced to a new step (non-terminal `update`).
- **`request_completed`** — the script finished (`update` returned `{ done: true }`).

## Error Codes

When something goes wrong, the server returns one of four HTTP status codes. The status tells you whether the problem is in your script, your request, or the environment:

| Status | What happened | Typical cause |
|---|---|---|
| **400** | Your request or script has a structural problem | Missing one of the four required exports, `init` or `view` returned a non-object, `scriptInput` is malformed |
| **408** | The request was cancelled | The HTTP client disconnected, or the request context was cancelled server-side |
| **422** | Your script crashed at runtime | An unhandled exception in `update` or `view` — often caused by accessing `event.data.x` when `event.data` is undefined |
| **504** | Your script took too long | A function call exceeded `timeoutMs`. Usually caused by infinite loops or heavy computation |

If you're seeing `422` errors, the most common fix is adding null guards around `event.data`. If you're seeing `504`, try raising `timeoutMs` or simplifying your callback logic.

## Examples

### Two-Step Flow: Confirm then Select

This script asks "Ship to production?" first. If the user approves, it finishes immediately. If they reject, it shows a second step where they pick an environment manually.

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
      // User approved — ship directly to prod
      if (event.data && event.data.approved) {
        return { done: true, result: { env: "prod" } };
      }
      // User rejected — show the environment picker
      state.step = "pick";
      return state;
    }
    // User picked an environment — we're done
    return {
      done: true,
      result: { env: event.data ? event.data.selectedSingle : "staging" }
    };
  }
};
```

Notice how `update` uses `state.step` to know which step the user just responded to, and how returning a modified `state` (non-terminal) vs returning `{ done: true, result }` (terminal) controls the flow.

### Configurable Scripts with Props

Props let you reuse the same script source with different configurations. Here the script reads `action` and `message` from props so the caller can customize it without changing the JavaScript:

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
    return {
      done: true,
      result: { action: state.action, approved: !!(event.data && event.data.approved) }
    };
  }
};
```

Create it with custom props:

```bash
jq -n --rawfile script /tmp/configurable.js \
  '{type:"script",scriptInput:{title:"Deploy",script:$script,props:{action:"deploy",message:"This deploys v2.1 to production."}}}' \
| curl -sS -X POST http://localhost:3000/api/requests -H 'Content-Type: application/json' -d @-
```

### Full API Test Sequence (Copy-Paste Ready)

This is the complete sequence for testing a two-step flow from the command line. Save it, run it, and watch the requests flow through:

```bash
# Write the script to a temp file
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

# Create the script request
REQ_ID=$(
  jq -n --rawfile script /tmp/plz-script.js \
    '{type:"script",sessionId:"global",scriptInput:{title:"API Sequence",timeoutMs:1500,script:$script}}' \
  | curl -sS -X POST http://localhost:3000/api/requests \
      -H 'Content-Type: application/json' -d @- \
  | jq -r '.id'
)
echo "Created request: $REQ_ID"

# Step 1: Respond to the confirm dialog (reject, so we advance to select)
echo "--- Advancing past confirm step ---"
curl -sS -X POST "http://localhost:3000/api/requests/$REQ_ID/event" \
  -H 'Content-Type: application/json' \
  -d '{"type":"submit","stepId":"confirm","data":{"approved":false}}' | jq .status

# Step 2: Pick an environment to complete the flow
echo "--- Completing with selection ---"
curl -sS -X POST "http://localhost:3000/api/requests/$REQ_ID/event" \
  -H 'Content-Type: application/json' \
  -d '{"type":"submit","stepId":"select","data":{"selectedSingle":"staging"}}' | jq .status

# Check the final result
echo "--- Final state ---"
curl -sS "http://localhost:3000/api/requests/$REQ_ID" \
  | jq '{status, scriptOutput}'
```

## Common Mistakes

These are the issues that come up most often when writing scripts. Each one has a clear symptom and a quick fix.

| What went wrong | What you see | How to fix it |
|---|---|---|
| You forgot one of the four required exports (`describe`, `init`, `view`, `update`) | `400` on create | Make sure `module.exports` has all four functions |
| `init` or `view` returned a string, number, or array instead of a plain object | `400` — "invalid return shape" | Always return `{}` objects, even if they're simple like `{ step: "start" }` |
| You returned `{ done: true }` but `result` isn't an object (or is missing) | `400` — invalid terminal shape | Use `{ done: true, result: { ... } }` — result must be a `{}` |
| `view` returned a `widgetType` that doesn't exist | Browser shows "unsupported widget" error | Use one of: `confirm`, `select`, `grid`, `table`, `form`, `upload`, `image` |
| Your script has a tight loop or expensive computation | `504` timeout after `timeoutMs` | Keep callbacks lightweight. If you need more time, increase `timeoutMs` |
| You accessed `event.data.approved` without checking if `event.data` exists | `422` runtime error — script crashed | Guard with `event.data && event.data.approved` |

## See Also

- `how-to-use` — end-user and agent-developer usage guide with CLI examples
- `adding-widgets` — guide for implementing new widget types across the full stack
- `js-script-development` — codebase internals, dev workflow, and troubleshooting for contributors
