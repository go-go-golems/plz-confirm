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

## Executive summary

The current plz-confirm request lifecycle is deliberately simple: **create a request**, **wait**, then **complete it once**. That simplicity is exactly why it works well for “one-shot” interactions, but it’s also why we can’t support “ask for more context” mid-flight: there’s no concept of an intermediate event, and there’s no supported way to update a pending request.

This ticket proposes a new interaction loop where a user can trigger a **hint prompt** (a small embedded “ask for help” UI) that emits an **intermediate event** back to the agent while keeping the widget open. The agent can then respond by **updating the current request’s input**, allowing the user to continue in the same dialog without restarting the flow.

There are a few viable ways to implement this. The cleanest long-term approach is to introduce explicit **event** and **update** endpoints (Option A) and a small event stream concept in the store. A lower-churn alternative is to overload the existing `/response` endpoint with a `kind=hint|final` field (Option B). We also consider a “linked request” approach (Option C), but it tends to reintroduce event semantics indirectly.

Finally, because hint specs and request updates are inherently nested/structured, we also explore a **YAML DSL** approach (in addition to CLI flags) that could make both agent generation and human debugging easier.

## What we want (requirements in plain language)

We want every widget to optionally include a **hint prompt** that the user can trigger to ask for more context/help **without closing the widget**.

The key idea is that “hint” is *not* the final answer—it’s an interruption that asks the agent to provide more information, after which the *same* widget continues. This makes the user experience feel more conversational and reduces the pressure to “get it right” on the first screen.

Concretely, we want:

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

From an implementation perspective, this implies two capabilities we do not have today:

1) **Intermediate events** on a request (hint events)
2) **Mutating an existing request** (updating `input` of a pending request)

## Current surface area (how plz-confirm works today)

This section maps the current contracts with filenames and symbols, so it’s clear what has to change.

### Server (Go): request lifecycle is “create → complete → wait”

At the server layer, the contract is intentionally widget-agnostic: `input` and `output` are `any`, and the server only cares about “pending vs completed”. That’s great for flexibility, but it also means there’s no native notion of events or partial progression.

**REST endpoints (Go server)** (today)

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

The important piece to internalize here is that `WaitRequest` is currently synonymous with “wait for completion”. It does a long-poll loop against `/wait`, and `/wait` only returns 200 when the store considers the request completed.

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

This is where “non-closing hint prompts” become visible: the current UI code path treats *every* submission as the final response. In other words, “submitting anything” today implies “closing the widget”.

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

To support non-closing hint prompts + request updates, we need changes in three places. It helps to think of this as “protocol + storage + UI ergonomics” rather than “just add a button”.

**Protocol / server**

- **New server endpoints** (or new semantics on existing ones) for:
  - “submit hint event”
  - “update request input”
  - “wait for next event” (hint or completion)

**Storage / state model**

- **New store model**:
  - completion-only `done` channel is insufficient
  - we need a per-request event stream / versioned state changes

**UX + agent control loop**

- **Frontend UX + state**:
  - optional hint UI per widget, folded by default
  - when user triggers hint → send hint event → keep active request on screen
  - when agent updates request input → UI updates active widget in place
- **CLI API + verbs**:
  - ability to wait for “hint event” (and return early)
  - ability to “update request” (push new input to server)

If we do this right, we unlock a more conversational pattern:

```text
agent creates widget -> user triggers hint -> agent updates widget -> user completes widget
```

## Proposed wire model: “events” + “updates”

This section proposes a clean way to express hint prompts and request updates.

### Data types (conceptual)

#### 1) Add `hint` spec to widget input

The hint prompt is best treated as part of the widget’s `input`—it’s UI configuration. This keeps the request self-contained: the browser can render the hint UI based on the request alone, and the agent can decide per-request whether hints should be enabled and what choices to offer.

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

Once hints exist, “output” is no longer a single thing. Sometimes it’s a final answer, and sometimes it’s “I need more context”. That’s why an explicit event envelope helps: it clearly distinguishes **intermediate events** from **completion**.

Instead of treating “output” as only the final answer, model an event:

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

After a hint is emitted, the agent needs to push new information into the active widget. That means we need a supported way to modify `req.Input` while the request remains pending.

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

If you want a quick “how do these compare?” snapshot before the details, here’s a rough comparison:

| Option | What it changes | Complexity | Back-compat risk | Long-term clarity |
|---|---|---:|---:|---:|
| **A** Events + Update endpoints | adds `/events` + `/input` + event-wait | High | Low | High |
| **B** Overload `/response` | modifies `/response` semantics | Medium | Medium | Medium |
| **C** Linked requests | treats hints as separate requests | Medium | Medium | Low |

### Option A (recommended): Add explicit event + update endpoints

Option A turns “hint” into a first-class concept: a request can accumulate events while still pending, and the agent can update `input` without completing. This is the most work up-front, but it keeps the protocol clean and extensible.

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

Option B keeps the API surface smaller by teaching the existing `/response` endpoint a new behavior. This can be attractive if we want to avoid adding multiple endpoints right away, but it increases coupling because one endpoint now serves two “modes”.

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

Option C tries to “stay within the current lifecycle” by treating a hint as its own request. On paper this seems simpler, but in practice it creates a second channel of coordination and usually drags us back into building an event stream anyway.

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

Think of YAML here as “a stable, nested wire contract that’s easy for both humans and agents to generate”. Flags remain great for quick interactive use, but YAML becomes compelling as soon as you need nested objects and iterative updates.

### Why YAML helps here

- Hint spec is nested (`hint.kind`, `hint.options`, `hint.buttons`, placeholders, etc.)
- Updates are naturally “replace this nested substructure”
- A file-based spec is easier to generate from an agent than assembling flags

### DSL Option 1: Generic request spec command (recommended)

This option introduces explicit “protocol verbs” for the new capabilities, and uses YAML as the natural input format for nested request specs and updates. It also avoids complicating existing widget commands.

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

This keeps existing command names but adds a “spec override” escape hatch. It can work, but it tends to grow precedence rules (“what if both flags and spec are provided?”), which can be stressful for both users and maintainers.

Each existing command supports `--spec` which, if present, overrides flags.

Pros:
- Keeps current command names (`plz-confirm confirm`, etc.)

Cons:
- More complicated precedence rules (flags vs spec)
- Harder to keep consistent across commands

### DSL Option 3: YAML for output/events too

This is mostly about ergonomics. If hint events become a first-class thing, rendering them as YAML by default can be pleasant for humans (“print the event envelope and payload”), while JSON remains best for automated piping.

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

At this stage, the most useful outcome is for you to pick a direction that matches your appetite for protocol changes and future-proofing. In particular, the decision hinges on whether we’re comfortable introducing an explicit “event stream” concept now (Option A), or whether we prefer to keep the API smaller and accept more coupling (Option B).

If you want the cleanest long-term protocol, pick **Option A**:

- explicit `/events` + `/input` update endpoints
- explicit CLI verbs: `wait` + `update`

If you want minimal API churn and accept a bit more coupling, pick **Option B**.

YAML DSL is most valuable if we do Option A, because “wait/update/event” flows become common, and YAML is a natural representation for nested `input` + `hint`.

## Next questions for you (so we can implement the right thing)

These questions are about locking down the interaction semantics so we don’t build a protocol that feels wrong in practice.

- Should hints be **per widget only**, or also available as a global “ask agent” panel even when no request is active?
- When a hint is submitted, should the UI:
  - stay interactive, or
  - temporarily disable the widget until an update arrives?
- Do you prefer:
  - **full input replace** updates (`PUT input`), or
  - **patch** updates (`PATCH input`), or both?
- Do we need to persist a list of hint events in history, or is “deliver once to CLI” enough?


