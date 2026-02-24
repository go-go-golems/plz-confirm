---
Title: 'Deep Dive Q&A: Write Pump, Duplication, 409, and confirmProtoAdapter'
Ticket: PC-05-INTEGRATE-OS
Status: active
Topics:
    - architecture
    - frontend
    - backend
    - go
    - javascript
    - ux
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../plz-confirm/internal/server/ws.go
      Note: Server-side websocket broadcaster/write locking model and write-pump context
    - Path: ../../../../../../../go-go-os/packages/confirm-runtime/src/ws/confirmWsManager.ts
      Note: Client-side websocket lifecycle and reconnect-policy decision context
    - Path: ../../../../../../../go-go-os/apps/inventory/src/App.tsx
      Note: Host-level 409 reconciliation logic and duplication points
    - Path: ../../../../../../../go-go-os/packages/confirm-runtime/src/api/confirmApiClient.ts
      Note: API error typing used by host 409 reconciliation
    - Path: ../../../../../../../go-go-os/packages/confirm-runtime/src/proto/confirmProtoAdapter.ts
      Note: Protocol boundary adapter from protojson to runtime model and back
    - Path: ../../../../../../../go-go-os/packages/confirm-runtime/src/types.ts
      Note: Runtime contract types consumed by adapter and host
    - Path: ../../../../../../../go-go-os/packages/confirm-runtime/src/state/confirmRuntimeSlice.ts
      Note: Realtime event application and completion output retention behavior
    - Path: ../../../../../../../go-go-os/packages/engine/src/chat/debug/EventViewerWindow.tsx
      Note: Reference model used for UI ergonomics comparison and upcoming redesign
Summary: Long-form intern-ready Q&A answering four architecture and integration questions raised after inspector review, with concrete file/symbol references and implementation guidance.
LastUpdated: 2026-02-24T21:05:00-05:00
WhatFor: Provide a durable knowledge artifact that explains key concepts and code paths behind the current confirm integration, including risks and recommended refactors.
WhenToUse: Use for onboarding, architecture review, and planning stabilization work across runtime, host adapters, and websocket handling.
---

# Deep Dive Q&A: Write Pump, Duplication, 409, and confirmProtoAdapter

## Executive Summary

This document answers four specific questions in depth:

1. what a **write pump** is (and why it matters for websocket correctness/performance);
2. where we currently have **legacy/deprecated/duplicated** behavior in the integrated stack;
3. how the **409 reconciliation** behavior works today and where host-level duplication exists;
4. what `confirmProtoAdapter` does, why it exists, and why it is both valuable and risky.

The goal is to make these topics concrete for someone new to the codebase. Instead of theory-only explanations, each section traces real code paths in `plz-confirm` and `go-go-os`, shows failure modes, and proposes practical next steps.

## Problem Statement

After the initial integration, we now have working end-to-end behavior, but several implementation details are easy to misunderstand without deep context:

- websocket write synchronization strategy can look "fine" in low load but degrade under fan-out;
- compatibility and hardening iterations created multiple layers that can appear duplicative;
- 409 handling fixed real UX bugs quickly, but the logic currently lives at host level in two near-identical branches;
- protocol adaptation moved complexity from app-level code into a central adapter, which is good, but centralization increases blast radius if behavior drifts.

If these are not documented clearly, future contributors can either over-refactor too early or accidentally reintroduce past regressions.

## Q1. What Is a Write Pump?

### Short answer

A write pump is a dedicated, serialized writer loop per websocket connection. It accepts outbound messages through a queue/channel and is the **only** code path that writes to that connection.

### Why this concept exists

Websocket libraries (including Gorilla websocket used by `plz-confirm`) require careful concurrency discipline: concurrent writes on the same socket are unsafe unless you serialize them. Teams usually solve this one of two ways:

1. lock around every write call;
2. run one write goroutine per connection (the write pump) and send it messages.

Both serialize writes, but they behave very differently under load.

### Current implementation in this repo

In [`ws.go`](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/internal/server/ws.go), the broadcaster has:

- global `writeMu sync.Mutex`;
- `BroadcastJSON`/`BroadcastRawJSON` loops over snapshot connections;
- each `writeJSON`/`writeText` acquires `writeMu` before writing.

That means writes are serialized **across all clients**, not just per-connection.

Conceptually today:

```text
for each conn in session:
  lock(globalWriteMu)
  conn.Write(...)
  unlock(globalWriteMu)
```

This is correct for safety, but it introduces coupling between unrelated clients.

### What a write pump would change

With per-connection write pumps:

- each connection has `outbound chan []byte`;
- one goroutine reads from that channel and writes to that connection;
- no global write lock across all sockets.

Conceptual model:

```text
on connect:
  create connState{conn, outboundChan}
  start writePump(connState)

broadcast(msg):
  for each connState in session:
    try enqueue msg into connState.outboundChan

writePump(connState):
  for msg := range outboundChan:
    set deadline
    conn.WriteMessage(msg)
```

### Why this matters for us

In the integrated system, WS fan-out can spike (multiple request events, script updates, completions). A global lock means:

- one slow client can delay everyone else;
- unrelated sessions still compete on one lock;
- latency profile becomes harder to reason about.

Even if absolute throughput is currently fine, the architecture scales poorly with more clients or slower networks.

### Practical tradeoffs

Write pump benefits:

- better isolation (client A slowness does not globally block client B writes);
- cleaner backpressure policy per connection (drop oldest, disconnect, or block);
- easier observability (queue depth per connection).

Write pump costs:

- more moving pieces (per-connection goroutine + queue lifecycle);
- backpressure policy decisions become explicit (which is good, but requires design);
- slightly more memory per active client.

### Recommended shape for `plz-confirm`

Given your stack, a pragmatic path is:

1. keep existing broadcaster interface stable;
2. internally replace global write lock with `connState` map;
3. add small bounded queue per connection;
4. if queue full, drop client (with structured log) to protect server health.

Pseudo-implementation sketch:

```go
// wsConnState owns write serialization for one client.
type wsConnState struct {
  conn *websocket.Conn
  out  chan []byte // bounded
}

func (s *wsConnState) writePump(onFailure func(error)) {
  for msg := range s.out {
    _ = s.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
    if err := s.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
      onFailure(err)
      return
    }
  }
}
```

### Key takeaway

A write pump is not just a performance optimization. It is a concurrency architecture that gives you correct serialization **and** better isolation/failure control than a global write mutex.

### Failure analysis examples (concrete)

#### Example A: one slow tablet client slows everyone

Imagine session `global` has five active websocket clients, four on local desktop and one on a poor Wi-Fi connection. A burst of `request_updated` and `request_completed` events arrives (for script progression plus normal request completions). With global `writeMu`, the slow connection can occupy write time repeatedly, and every other client waits for the mutex.

Symptoms in practice:

- healthy clients receive events with noticeable jitter;
- UI appears to “batch update” instead of streaming smoothly;
- server logs show repeated write deadline warnings correlated across multiple clients.

With per-connection pumps, this degradation is localized: only the slow client queue grows. Fast clients remain responsive.

#### Example B: head-of-line blocking across sessions

Even if clients are grouped by session, global lock means session A traffic can interfere with session B traffic. This is unintuitive because code snapshots connections by session before broadcasting, but lock contention happens in shared write path.

Operationally, this causes cross-session bleed where unrelated users can influence each other’s perceived real-time quality.

### Write pump instrumentation checklist

If/when we implement write pumps in `plz-confirm`, add these counters from day one:

1. `ws_conn_out_queue_depth` (gauge, per connection/session);
2. `ws_conn_write_errors_total` (counter);
3. `ws_conn_dropped_due_backpressure_total` (counter);
4. `ws_conn_write_duration_ms` (histogram).

These metrics make it possible to answer “is the issue server fan-out, network path, or client slowness?” quickly without guessing.

### Client-side relation to reconnect policy

Write pump (server concern) and reconnect policy (client/runtime concern) are often discussed together but should be separated:

- write pump ensures outbound server concurrency and fairness;
- reconnect policy ensures client recovers from disconnects.

Your decision to inject reconnect policy via host adapters complements write pumps: host-specific retry behavior can evolve without forcing one global retry style in runtime.

---

## Q2. Do We Have Legacy/Deprecated/Duplicated Code Now?

### Short answer

Yes, we have some duplication and compatibility layers by design. Not all duplication is bad. The important distinction is:

- **intentional boundary duplication** (acceptable for now), vs
- **accidental behavioral duplication** (should be reduced).

### Where legacy pressure comes from

The current integration had to preserve behavior from the older `agent-ui-system` frontend while moving to:

- shared runtime package (`@hypercard/confirm-runtime`),
- host adapters in `apps/inventory`,
- protojson contract mapping.

Any migration like this naturally carries transitional overlap.

### Concrete duplication/legacy pockets

#### 1. Legacy compatibility normalization in adapter

`confirmProtoAdapter` now normalizes values like status aliases and payload shape variants to keep runtime stable even if incoming payloads vary.

This is **intentional compatibility logic**, not accidental duplication. But it should remain explicit and tested, because it can become a hidden policy layer.

#### 2. Host-level 409 handling duplicated in two submit paths

In [`App.tsx`](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/apps/inventory/src/App.tsx), both:

- `onSubmitResponse` catch handler;
- `onSubmitScriptEvent` catch handler;

implement nearly the same 409 reconciliation branch.

This is real duplication. It is functional and understandable, but should likely move into runtime/client helper utilities.

#### 3. UI model overlap between queue list and request detail host

`ConfirmQueueWindow` has a bespoke render list in app code; request detail rendering is in `ConfirmRequestWindowHost`. This is not exactly duplicate logic, but there is duplicated display concern around request summaries and interaction affordances.

Given your new request to redesign queue/view using Event Viewer style, this is the right moment to unify presentation patterns.

#### 4. Reconnect policy location ambiguity

Inspector review initially flagged reconnect as missing and posed whether policy should be default runtime or host-injected. You made a decision: injected via host adapters. Until implemented, code still reflects single-connect behavior.

This is not duplication yet, but it is **decision/code mismatch** until execution.

### Deprecated code?

Inside the newly integrated path, there is no obvious dead deprecated module actively causing risk. The main legacy reference is the old frontend under `plz-confirm/agent-ui-system/client` used as behavioral baseline. That legacy frontend is still useful as a reference corpus during parity checks.

### How to classify what to keep vs remove

Use this rubric:

1. If duplicated logic exists solely to bridge runtime-to-host boundary until API solidifies, keep temporarily but mark with clear TODO and issue link.
2. If duplicated logic executes the same policy in two code paths (409 logic), extract now.
3. If compatibility normalization exists for legacy payload tolerance, keep but make tests explicit and add comments clarifying why.
4. If older implementation is only parity reference, keep read-only and avoid partial cherry-pick drift.

### Suggested cleanup roadmap

Near-term:

- extract 409 reconciliation into helper in confirm-runtime (`reconcileConflictAndClose` style);
- centralize request list item rendering component for queue and future redesigned request list.

Medium-term:

- formalize adapter compatibility matrix (accepted input variants, normalized output);
- add deprecation table in docs for legacy behavior that will be removed on schedule.

Long-term:

- once host-adapter reconnect policy lands, document and remove any implicit reconnect assumptions from runtime internals.

### Key takeaway

Yes, there is duplication. Most of it is transitional and manageable. The risky part is not “having duplication,” but having duplication without clear ownership and tests. The current state is salvageable and already headed in the right direction.

### Detailed duplication inventory (by category)

#### A. Intentional duplication (acceptable)

1. Legacy parity references in `agent-ui-system` compared against `confirm-runtime`.
2. Storybook scenarios that intentionally mirror runtime interactions for design/dev handoff.
3. Compatibility branches inside adapter that normalize incoming variants.

These are useful and should remain until migration policy says otherwise.

#### B. Structural duplication (needs consolidation)

1. 409 reconciliation branch duplicated in two host submit handlers.
2. Queue/list rendering in inventory host not yet aligned with richer diagnostic UI patterns already available in Event Viewer.
3. Potential future duplication risk if reconnect logic gets copy-pasted into each host app rather than injected through a shared adapter contract.

#### C. Documentation duplication (currently tolerable)

Multiple long-form docs in `PC-05` intentionally overlap on architecture context (blueprint, postmortem, review, playbook). This is acceptable because each serves a distinct audience/use-case, but indexing and “start here” pointers must stay current.

### How interns should reason about “duplicate code”

A common beginner error is to refactor too aggressively at the first sign of similarity. For this codebase, better heuristic:

1. Ask whether two similar blocks encode identical policy.
2. Ask whether both blocks change together in real tickets.
3. Ask whether extracting now simplifies tests or only moves complexity around.

If answers are `yes/yes/yes`, extract. If not, postpone with explicit notes.

### Proposed deprecation ledger format

To manage future cleanup predictably, track each compatibility branch in a simple ledger:

```text
compat_item: adapter accepts legacy status 'expired'
introduced_in: PC-10
reason: backward tolerance for older payload emitters
remove_when: all emitters verified on timeout/error status contract
owner: confirm-runtime
test_guard: confirmProtoAdapter.test.ts :: maps timeout/error statuses ...
```

Keeping this ledger in docs avoids accidental permanent retention of short-lived compatibility behavior.

---

## Q3. What Is the 409 Reconciliation Logic and What Host Duplication Exists?

### Short answer

409 reconciliation is the conflict-recovery path used when a submit races with already-completed request state. On 409, the host refetches the latest request; if completed, it updates local state and closes the request window instead of showing a hard error.

### Why this exists

In distributed UI flows, “open request in UI” and “submit request outcome” are asynchronous. A request can be completed elsewhere (another tab/client, timeout path, concurrent action) before this window submits.

Without reconciliation:

- submit returns 409;
- UI can keep stale request visible;
- user sees confusing repeated errors;
- queue/request windows drift from true backend state.

### Where it lives now

In [`App.tsx`](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/apps/inventory/src/App.tsx), there are two near-identical catch branches:

1. response submit path (`apiClient.submitResponse(...)`)
2. script-event submit path (`apiClient.submitScriptEvent(...)`)

Both do:

```text
if error is ConfirmApiError && status == 409:
  try get latest request by id
  if latest.status != pending:
    dispatch completeRequestById(...)
    close confirm window
    return
  if refetch fails:
    remove local request
    close window
    return
else:
  log error
```

### Step-by-step behavior

#### Path A: Ideal reconcile

1. submit gets 409;
2. fetch latest request succeeds;
3. latest status is completed/timeout/error (non-pending);
4. host dispatches completion, closes window.

Result: UI converges cleanly.

#### Path B: Refetch fails

1. submit gets 409;
2. `getRequest` fails (network/transient issue);
3. host falls back to `removeRequest + closeWindow`.

Result: UI still cleans stale window, but completion metadata may be less rich than ideal path.

### What this solved

This logic addressed real observed behavior where completed requests remained visible in queue and window until manual refresh/reload.

### What is duplicated

The duplication is primarily policy code, not just boilerplate:

- fetch latest + non-pending check;
- completion dispatch + window close;
- fallback remove + close.

Duplicating policy in two handlers increases maintenance risk:

- a future tweak might update one path and forget the other;
- behavior drifts between normal submits and script-event submits;
- testing burden doubles.

### Best extraction target

There are two strong options:

1. **Runtime utility function** in confirm-runtime package:
   - takes `(dispatch, apiClient, requestId, windowCloser)`;
   - executes reconcile policy;
   - returns enum outcome.

2. **Api client helper**:
   - wrap submit methods with built-in `on409` reconciliation callback.

Option 1 is usually clearer because reconciliation is UI-state plus transport behavior, not pure API transport.

Pseudo API:

```ts
type ConflictResolution = 'completed_reconciled' | 'removed_locally' | 'not_conflict';

async function reconcileConflict(
  error: unknown,
  requestId: string,
  apiClient: ConfirmApiClient,
  dispatch: Dispatch,
  closeWindow: (id: string) => void,
): Promise<ConflictResolution>;
```

### Interaction with runtime slice

`completeRequestById` in [`confirmRuntimeSlice.ts`](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/state/confirmRuntimeSlice.ts) is already set up to move active requests into completion map and remove from active queue. Reconciliation logic is therefore mostly orchestration around that existing reducer behavior.

### Testing recommendations for this logic

1. submit response 409 + latest completed -> completion inserted, window closed.
2. submit script event 409 + latest completed -> same behavior.
3. submit 409 + refetch error -> request removed and window closed.
4. non-409 error -> no forced local removal.

### Key takeaway

The 409 reconciliation model is correct and practical. The problem is location and duplication, not policy. Extract the shared policy once, keep host wiring thin.

### Race timeline walkthrough (realistic scenario)

Below is a representative race that produced stale-window behavior before reconciliation improvements:

```text
t0  UI A opens confirm request window (status pending)
t1  UI B submits response first, backend marks completed
t2  UI A user clicks Approve
t3  UI A submit -> backend 409 request already completed
t4  without reconcile: UI A shows error, window stays open
t5  with reconcile: UI A refetches latest, marks complete locally, closes window
```

This pattern is normal in multi-client systems; it is not an edge anomaly.

### Why host-level implementation was initially reasonable

At the time, implementing in `apps/inventory` was pragmatic:

- fastest place to apply UX fix without redesigning runtime API;
- tight feedback loop during manual testing;
- avoided broad package changes while P1 issues were still being stabilized.

So duplication here is technical debt with good rationale, not careless architecture.

### What a consolidated API should return

When extracted, a shared reconciler should report outcome so hosts can apply host-specific telemetry/UX:

```ts
type ReconcileOutcome =
  | { kind: 'not_conflict' }
  | { kind: 'reconciled'; requestId: string; completedAt: string }
  | { kind: 'removed_locally'; requestId: string; reason: 'refetch_failed' | 'missing_request' };
```

This prevents opaque helper behavior and gives hosts visibility for analytics/debug logging.

### Host duplication beyond inventory (forward risk)

Today only inventory host is wired, but the package architecture is intended for reuse. If a second app integrates confirm-runtime and copies the same catch blocks, duplication doubles immediately and divergence begins.

Extracting now is cheaper than waiting for second-host divergence.

---

## Q4. Explain `confirmProtoAdapter`

### Short answer

`confirmProtoAdapter` is the protocol boundary translator between backend protojson payloads and frontend runtime models. It does two-way mapping:

1. backend `UIRequest`/WS event JSON -> runtime `ConfirmRequest`/`ConfirmRealtimeEvent`;
2. runtime submit payload -> backend oneof output JSON shape.

### Why it exists

Without an adapter, frontend code would have raw protojson details scattered across host components and reducers:

- snake/camel expectations;
- oneof field placement (`confirmOutput`, `selectOutput`, etc.);
- numeric-string normalization (`int64` fields);
- legacy compatibility edge cases.

Centralizing this into one module gives:

- one source of truth for mapping rules;
- tighter tests;
- cleaner host UI logic.

### Main responsibilities

In [`confirmProtoAdapter.ts`](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/proto/confirmProtoAdapter.ts):

#### 1. Request mapping (`mapUIRequestFromProto`)

- validates minimum fields (`id`, `sessionId`, widget type);
- picks input field by widget type (`confirmInput`, `tableInput`, etc.);
- normalizes payload where required (e.g., upload `maxSize`);
- maps script view metadata and sections.

#### 2. Event mapping (`mapRealtimeEventFromProto`)

- validates event type;
- maps embedded request;
- derives `requestId` and `completedAt`;
- now preserves completion output payloads when present.

#### 3. Submit mapping (`mapSubmitResponseToProto`)

- dispatches by widget type;
- emits correct oneof envelope:
  - `confirmOutput`
  - `selectOutput`
  - `formOutput`
  - `tableOutput`
  - `uploadOutput`
  - `imageOutput`
  - `scriptOutput`
- enforces mode-aware output variants where required.

### Why it can feel “magical”

Adapters tend to look like utility glue, but they encode business policy:

- which status strings are accepted/normalized;
- when to emit single vs multi variants;
- default timestamp behavior;
- tolerance for legacy/partial payload shapes.

So this file is effectively a policy engine at protocol boundary.

### Benefits

1. **Boundary hygiene**: host components stay focused on UI, not wire format.
2. **Consistency**: all submit paths share one output-shape policy.
3. **Testability**: many regressions can be isolated into adapter tests.
4. **Migration safety**: legacy behavior can be normalized in one place.

### Risks

1. **Single point of semantic drift**: one bad mapper change can break multiple widgets.
2. **Hidden compatibility assumptions**: if not documented, normalizations become tribal knowledge.
3. **Overgrowth**: adapter can become a dumping ground for unrelated behavior unless guarded by clear scope.

### How to keep adapter healthy

#### 1. Keep it boundary-only

Do not place UI orchestration/state mutation logic here. Keep it pure mapping and normalization.

#### 2. Add explicit mapping matrix in docs/tests

For each widget type, define:

- accepted runtime input forms;
- emitted proto oneof variants;
- fallback behavior.

#### 3. Add “dangerous-change” test checklist

Before modifying adapter:

- run targeted adapter tests;
- run host flow smoke tests for affected widget types;
- verify legacy parity scenarios.

### Simplified mental model

Think of adapter as an airport customs desk:

- inbound payloads are checked/normalized before entering runtime;
- outbound payloads are packed into the exact document format expected by backend.

It should not decide “what user should do,” only “how data crosses border correctly.”

### Practical code-path map

```text
HTTP GET /api/requests/{id}
  -> confirmApiClient.getRequest()
  -> mapUIRequestFromProto()
  -> runtime reducer/state

WS new_request/request_updated/request_completed
  -> ConfirmWsManager onmessage
  -> mapRealtimeEventFromProto()
  -> applyRealtimeEvent reducer action

UI submit action
  -> ConfirmRequestWindowHost output object
  -> mapSubmitResponseToProto(request, payload)
  -> POST /api/requests/{id}/response or /event
```

### Key takeaway

`confirmProtoAdapter` is the right abstraction for this integration. Treat it as critical boundary infrastructure: strongly tested, narrowly scoped, and explicitly documented whenever policy changes.

### Adapter anti-patterns to avoid

1. **Business branching by app mode** inside adapter (should stay in host/runtime orchestration).
2. **Silent lossy transforms** without tests/changelog notes.
3. **Ad hoc normalization** scattered outside adapter once adapter already owns that concern.

If you find yourself writing `as any` proto-shape handling in host components, it is usually a sign that logic belongs back in adapter.

### Recommended internal structure for long-term maintainability

As adapter complexity grows, split by concern:

```text
confirmProtoAdapter/
  parse/
    request.ts
    event.ts
  encode/
    confirm.ts
    select.ts
    table.ts
    image.ts
    upload.ts
  normalize/
    numbers.ts
    status.ts
  index.ts
```

This keeps each widget mapping small and makes code review safer.

### Contract traceability table

Add a “traceability” block in docs/tests for every policy-sensitive mapping:

| Runtime input | Adapter output | Proto contract field | Test |
|---|---|---|---|
| `selectedIds=['a']` + `multi=true` | `selectedMulti.values=['a']` | `SelectOutput.selected_multi` | `confirmProtoAdapter.test.ts` |
| `uploadInput.maxSize='10485760'` | `payload.maxSize=10485760` | `UploadInput.max_size` | `confirmProtoAdapter.test.ts` |
| `status='expired'` | `status='timeout'` | `RequestStatus.timeout` | `confirmProtoAdapter.test.ts` |

This table format helps new contributors verify policy quickly without reverse-engineering mapper code.

### Debugging playbook for adapter issues

When a runtime widget behaves unexpectedly:

1. capture raw backend payload (HTTP or WS frame);
2. run it through `mapUIRequestFromProto` in a focused test;
3. assert mapped runtime shape;
4. run output through `mapSubmitResponseToProto` for same request;
5. compare generated payload to expected oneof form.

Doing this in tests is faster and more deterministic than manual UI clicking.

## Proposed Solution and Next Steps

1. Keep write-pump migration as a scoped server hardening item (`P3` in inspector plan), but align with your new reconnect-policy decision (host-injected on client side).
2. Extract duplicated 409 reconciliation into confirm-runtime helper API to keep hosts thin.
3. Continue using adapter as boundary authority, but pair each policy change with explicit regression tests and changelog notes.
4. Use the new PC-11 request-view redesign to reduce app-level presentation duplication and make request triage/debugging more Event Viewer-like.

## Design Decisions

1. Reconnect policy direction is now explicit: injected via host adapters.
2. 409 policy stays user-friendly (reconcile instead of hard fail) but should be centralized.
3. Adapter remains the canonical protocol policy boundary.
4. Legacy compatibility support is acceptable when explicit and tested.

## Alternatives Considered

1. Remove adapter and decode protojson directly in host components.
   - Rejected: scattering wire semantics across UI files increases drift and regression risk.

2. Keep global websocket write lock indefinitely.
   - Rejected for long-term scale/reliability; acceptable only as interim simplification.

3. Treat 409 as fatal and force user refresh.
   - Rejected: poor UX and inconsistent with current recovery-oriented runtime design.

## Implementation Plan

1. Create `confirm-runtime` conflict reconciliation utility and migrate inventory host to use it.
2. Add host-injected reconnect policy surface to runtime WS manager interface.
3. Add dedicated tests for reconciliation helper and reconnect-injection behavior.
4. Evaluate/implement per-connection write pump in `plz-confirm/internal/server/ws.go`.

## Open Questions

1. For write pump queues, should overflow policy be drop-client or drop-message?
2. Should 409 reconciliation utility live in runtime package root API or a host submodule?
3. Should adapter normalize legacy aliases indefinitely or behind versioned compatibility flags?

## References

- [`ws.go`](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/internal/server/ws.go)
- [`confirmWsManager.ts`](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/ws/confirmWsManager.ts)
- [`App.tsx`](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/apps/inventory/src/App.tsx)
- [`confirmApiClient.ts`](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/api/confirmApiClient.ts)
- [`confirmProtoAdapter.ts`](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/proto/confirmProtoAdapter.ts)
- [`confirmRuntimeSlice.ts`](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/state/confirmRuntimeSlice.ts)
- [`EventViewerWindow.tsx`](/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/chat/debug/EventViewerWindow.tsx)
