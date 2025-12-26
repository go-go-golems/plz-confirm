---
Title: Analysis: Non-Closing Hint Prompt Events + Request Updates
Ticket: 005-HINT-PROMPT
Slug: hint-prompt-events-and-request-updates-analysis
Short: Analyze current CLI↔server↔UI surface and propose designs for hint prompts that emit intermediate events and allow updating the active widget
Topics:
- analysis
- cli
- backend
- frontend
- protocol
- yaml
---

## What we want (requirements in plain language)

We want every widget to optionally include a **hint prompt** that the user can trigger to ask for more context/help **without closing the widget**.

Concretely:

- Each widget can optionally expose a folded “Hint / Ask for help” area.
- The hint UI is configurable per request:
  - **textarea** (“ask a question”)
  - **select** (choose from options)
  - **buttons** (pre-canned choices)
- When the user triggers a hint:
  - the **request stays pending**
  - the system emits a **hint event** to the CLI/agent (not a final answer)
  - the CLI returns “user asked for hint” (including payload)
- The agent can then **update the existing widget** with its response (e.g., add more info, change options, show additional images) and the user can continue.

This implies two new capabilities:

1) **Intermediate events** on a request (hint events)
2) **Mutating an existing request** (updating `input` of a pending request)

## Current surface area (how plz-confirm works today)

This section maps the current contracts with filenames and symbols, so it’s clear what has to change.

### Server (Go): request lifecycle is “create → complete → wait”

**REST endpoints (Go server)**

- Create request:
  - `POST /api/requests`
  - `internal/server/server.go`: `handleCreateRequest`
  - Stores request via: `internal/store/store.go`: `(*Store).Create`
  - Broadcasts WS: `type: "new_request"`

- Submit response:
  - `POST /api/requests/{id}/response`
  - `internal/server/server.go`: `handleSubmitResponse`
  - Completes via: `internal/store/store.go`: `(*Store).Complete`
  - Broadcasts WS: `type: "request_completed"`

- Wait for completion (long poll):
  - `GET /api/requests/{id}/wait?timeout=...`
  - `internal/server/server.go`: `handleWait`
  - Waits via: `internal/store/store.go`: `(*Store).Wait`

**Store internals**

`internal/store/store.go` is a minimal in-memory store:

- `requestEntry` holds:
  - `req types.UIRequest`
  - `done chan struct{}` (closed once on completion)
- `(*Store).Wait(ctx,id)` blocks on `done` channel, then returns the request

**Key implication**

There is only one “wake-up” moment for waiters: **completion**.
There is no concept of:

- intermediate events (hint)
- request updates that unblock waiters
- an event stream / cursor

### Client (Go): CLI waits only for completion

The shared HTTP client is `internal/client/client.go`.

- `(*Client).CreateRequest(...)` posts to `/api/requests`
- `(*Client).WaitRequest(ctx, id, waitTimeoutS)` long-polls `/api/requests/{id}/wait`
  - retries on HTTP 408
  - returns only when server returns 200 (which currently only happens when request is completed)

Each widget command (`internal/cli/*.go`) is basically:

```text
CreateRequest -> WaitRequest -> decode output -> print row(s)
```

**Key implication**

Today the CLI cannot “return on hint” because there is no hint event to wait for.

### Frontend (React): submitting always completes

The UI receives new requests via WebSocket and shows exactly one `active` request.

**Receiving WS**

- `agent-ui-system/client/src/services/websocket.ts`: `connectWebSocket()`
  - handles `data.type === "new_request"` → `setActiveRequest(request)`
  - handles `data.type === "request_completed"` → `completeRequest(...)` or `addToHistory(...)`

**Submitting response**

- `agent-ui-system/client/src/components/WidgetRenderer.tsx`
  - `handleSubmit(output)` calls:
    - `submitResponse(active.id, output)` (REST)
    - then `dispatch(completeRequest(...))` (clears `active`)

**Redux store shape**

- `agent-ui-system/client/src/store/store.ts`
  - `completeRequest` always moves `active` to history and sets `active=null`

**Key implication**

There is no UI concept of “send a hint event but keep the widget open”.
We need a separate submit path that does **not** call `completeRequest`.

## What has to change (high-level)

To support non-closing hint prompts + request updates, we need:

- **New server endpoints** (or new semantics on existing ones) for:
  - “submit hint event”
  - “update request input”
  - “wait for next event” (hint or completion)
- **New store model**:
  - completion-only `done` channel is insufficient
  - we need a per-request event stream / versioned state changes
- **Frontend UX + state**:
  - optional hint UI per widget, folded by default
  - when user triggers hint → send hint event → keep active request on screen
  - when agent updates request input → UI updates active widget in place
- **CLI API + verbs**:
  - ability to wait for “hint event” (and return early)
  - ability to “update request” (push new input to server)

## Proposed wire model: “events” + “updates”

This section proposes a clean way to express hint prompts and request updates.

### Data types (conceptual)

#### 1) Add `hint` spec to widget input

Each widget input can optionally carry:

```yaml
hint:
  kind: textarea|select|buttons
  title: "Need more context?"
  message: "Ask a question. The agent will update this widget."
  placeholder: "Type your question..."
  options: ["Explain A", "Explain B"]        # for select
  buttons: ["Show more", "Why?", "Example"] # for buttons
```

This is UI-facing: it configures what the user sees and can click/type.

#### 2) Introduce an event envelope

Instead of treating “output” as only the final answer, model:

```json
{
  "event": {
    "kind": "hint" | "final",
    "type": "hint.textarea" | "hint.select" | "hint.buttons" | "final.<widget>",
    "payload": { ... },
    "createdAt": "..."
  }
}
```

The key is **kind**:

- `hint` events do NOT complete the request
- `final` events complete the request (today’s behavior)

#### 3) Add “update request input”

Agent updates an in-flight request:

- full replace: `PUT /api/requests/{id}/input`
- patch: `PATCH /api/requests/{id}/input` (JSON Merge Patch)

Server broadcasts:

```json
{ "type": "request_updated", "request": <UIRequest> }
```

UI updates `active.input` in-place.

## Design options (choose one)

This section gives multiple approaches with tradeoffs. All can work; the decision is about complexity vs clarity vs compatibility.

### Option A (recommended): Add explicit event + update endpoints

**New endpoints**

- `POST /api/requests/{id}/events` with body:
  - `{ "event": { "kind": "hint", "payload": {...} } }`
  - `{ "event": { "kind": "final", "payload": {...} } }` (optional; could keep existing `/response`)

- `PATCH /api/requests/{id}/input` (or `PUT`) to update the widget input

- `GET /api/requests/{id}/events/wait?cursor=N&timeout=25`
  - returns the next event after cursor

**Store changes**

Replace `done chan` with a versioned event stream:

```pseudo
entry:
  req: UIRequest
  events: []Event
  version: int
  changed: chan struct{} (or condvar)

appendEvent(kind=hint):
  events.append(...)
  version++
  signal changed

complete(kind=final):
  set req.status=completed
  set req.output=payload (or keep legacy output)
  appendEvent(kind=final)
  signal changed

updateInput(newInput):
  req.input = newInput
  version++
  signal changed

waitForVersion(ctx, since):
  block until version > since or ctx timeout
  return (version, new events since, latest req snapshot)
```

**UI changes**

- Add hint UI based on `input.hint`.
- When hint is submitted:
  - call `POST /api/requests/{id}/events` (kind=hint)
  - DO NOT call `completeRequest`
  - optionally show “hint sent” toast / spinner until request_updated arrives
- Handle WS message `request_updated`:
  - update `active` in redux without closing it

**CLI changes**

Add a new CLI command family (clean separation):

- `plz-confirm wait --id <id> --return-on hint|final|any`
  - long-polls `/events/wait`
  - returns when matching event arrives
- `plz-confirm update --id <id> --input @file.yaml` (or patch)
  - pushes new input

Pros:
- Clean separation of “events” vs “final response”
- Doesn’t break existing `/response` flows (can keep `/response` as “final”)
- Scales to future “progress” events

Cons:
- Most new plumbing (store + endpoints + UI WS message + new CLI verbs)

### Option B: Overload `/response` with `kind=hint|final`

Instead of new endpoints, modify existing response body:

```json
POST /api/requests/{id}/response
{
  "kind": "hint",
  "output": { ...hint payload... }
}
```

Server behavior:

- if `kind=final` → complete request (today’s behavior)
- if `kind=hint` → store as event, DO NOT complete

Pros:
- Reuses existing endpoint shape
- UI can keep calling “submitResponse”, but now it chooses `kind`

Cons:
- Requires changing server semantics and client code
- Still needs “wait for hint” and “update input”
- More risk of accidental “complete vs hint” mistakes

### Option C: Represent hints as separate linked requests

When user triggers hint, UI creates a new request “hint” linked to the parent:

```json
{ "type": "hint", "input": { "parentId": "...", ... } }
```

Agent watches for new hint requests and responds by updating parent.

Pros:
- Reuses existing request lifecycle (create/complete)
- No new event stream semantics

Cons:
- The CLI currently creates requests; the UI doesn’t.
- The agent would need to “listen” for hint requests (no WS client in CLI)
- You end up reintroducing event/polling anyway

## YAML DSL exploration (input/output instead of flags)

You asked whether we can use a YAML DSL rather than a forest of CLI flags. This fits especially well once we add “update input” and “hint events”, because those are naturally structured.

### Why YAML helps here

- Hint spec is nested (`hint.kind`, `hint.options`, `hint.buttons`, placeholders, etc.)
- Updates are naturally “replace this nested substructure”
- A file-based spec is easier to generate from an agent than assembling flags

### DSL Option 1: Generic request spec command (recommended)

Add a new CLI command:

```bash
plz-confirm request --spec @request.yaml
plz-confirm wait --id <id> --return-on hint
plz-confirm update --id <id> --input @updated-input.yaml
plz-confirm wait --id <id> --return-on final
```

Example `request.yaml`:

```yaml
type: confirm
timeout: 300
sessionId: global
input:
  title: "Deploy?"
  message: "Deploy to prod?"
  hint:
    kind: textarea
    title: "Ask for more context"
    placeholder: "What do you want to know?"
```

Implementation notes:
- Parse YAML into `map[string]any` (or a typed struct) and pass as `CreateRequestParams`.
- This can be implemented without changing Glazed flags for existing commands.

### DSL Option 2: Per-widget `--spec @file.yaml` override

Each existing command supports `--spec` which, if present, overrides flags.

Pros:
- Keeps current command names (`plz-confirm confirm`, etc.)

Cons:
- More complicated precedence rules (flags vs spec)
- Harder to keep consistent across commands

### DSL Option 3: YAML for output/events too

Events could be printed as YAML by default:

```yaml
event:
  kind: hint
  type: hint.textarea
  payload:
    text: "What does this button do?"
request_id: ...
cursor: ...
```

This is mostly an output formatting choice; it works nicely with Glazed’s output modes (and we can still offer JSON for scripts).

## Recommendations / decision points

If you want the cleanest long-term protocol, pick **Option A**:

- explicit `/events` + `/input` update endpoints
- explicit CLI verbs: `wait` + `update`

If you want minimal API churn and accept a bit more coupling, pick **Option B**.

YAML DSL is most valuable if we do Option A, because “wait/update/event” flows become common, and YAML is a natural representation for nested `input` + `hint`.

## Next questions for you (so we can implement the right thing)

- Should hints be **per widget only**, or also available as a global “ask agent” panel even when no request is active?
- When a hint is submitted, should the UI:
  - stay interactive, or
  - temporarily disable the widget until an update arrives?
- Do you prefer:
  - **full input replace** updates (`PUT input`), or
  - **patch** updates (`PATCH input`), or both?
- Do we need to persist a list of hint events in history, or is “deliver once to CLI” enough?


