---
Title: Diary
Ticket: PC-12-WS-WRITE-PUMP
Status: active
Topics:
    - architecture
    - backend
    - go
    - ux
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/internal/server/ws.go
    - /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/internal/server/ws_test.go
ExternalSources: []
Summary: Step-by-step implementation log for websocket write pump migration.
LastUpdated: 2026-02-24T09:40:00-05:00
WhatFor: Preserve implementation context, risks, and validation details for future maintainers.
WhenToUse: Use to reconstruct design intent and debugging context.
---

# Diary

## Goal

Track implementation of per-connection websocket write pumps, including concurrency decisions, testing, and follow-up checks.

## Step 1: Ticket initialization and technical baseline

I created a dedicated P3 ticket workspace and mapped the exact code paths that currently enforce global websocket write serialization. The main hotspot is `internal/server/ws.go`, where a broadcaster-wide `writeMu` is shared across all sessions and all clients.

This step also captured intended invariants before edits: session-scoped fanout must stay intact, connect-time pending replay ordering must remain deterministic, and client drop behavior should isolate failures to the affected socket.

### Prompt Context

**User prompt (verbatim):** "- Create a ticket to add the writer pump, with implementation guide and technical details.
- separate ticket for extracting the 409 redudancy
- 

Then continue and address the P3 issues in their respective tickets."

**Assistant interpretation:** Create/expand ticket documentation first, then implement P3 findings in code with traceable diary updates.

**Inferred user intent:** The user wants production-quality hardening work done in a ticketed, auditable way so multiple developers can continue safely.

**Commit (code):** N/A (planning step)

### What I did

- Created and inspected ticket workspace `PC-12-WS-WRITE-PUMP`.
- Read current websocket broadcaster and tests.
- Wrote implementation guide with architecture and pseudocode.
- Wrote concrete task list to drive execution.

### Why

- Prevent ad-hoc concurrency edits without explicit invariants.
- Make behavior and risks legible for handoff/review.

### What worked

- Existing tests already cover lifecycle order and pending replay order, giving a baseline guardrail.

### What didn't work

- N/A in this step.

### What I learned

- The existing lock is correctness-safe but throughput-hostile.

### What was tricky to build

- N/A yet; implementation pending.

### What warrants a second pair of eyes

- Queue overflow policy once implemented: drop vs retry trade-off.

### What should be done in the future

- Add websocket metrics counters after core migration is stable.

### Code review instructions

- Start at `internal/server/ws.go` and trace write path from broadcaster to socket.
- Validate with `go test ./internal/server -count=1` once implementation lands.

### Technical details

- Prior write path: `Broadcast*` -> `writeMu` -> socket write.
- Target write path: `Broadcast*` -> per-client queue -> per-client pump goroutine.

## Step 2: Implement per-connection writer pump and regression tests

I replaced the broadcaster-wide write mutex with a `wsClient` abstraction that owns a bounded outbound queue and a dedicated write goroutine. Broadcast calls now enqueue payloads per target client, so unrelated clients are no longer serialized behind one global writer lock.

I also changed connect-time pending replay to enqueue through the same path, preserving a single write pipeline invariant. After implementation, I ran server tests and they passed.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Finish the writer-pump code path and verify it with tests.

**Inferred user intent:** Improve runtime resilience/scale without changing external event contracts.

**Commit (code):** pending

### What I did

- Added `wsClient` with:
  - bounded `send` queue,
  - `writePump` goroutine,
  - timed/non-timed enqueue methods,
  - idempotent `stop()` lifecycle.
- Reworked `wsBroadcaster` maps from raw `*websocket.Conn` to `*wsClient` entries.
- Updated broadcast methods to enqueue and isolate-drop failing clients.
- Removed global `writeMu` serialization and direct write helpers.
- Added tests for queue-full and closed-client enqueue behavior.

### Why

- Prevent one slow socket from stalling all websocket clients.
- Keep gorilla websocket write constraints valid via single writer goroutine per connection.

### What worked

- `go test ./internal/server -count=1` passed after migration.
- Existing ordering tests remained green, indicating no regression in event lifecycle ordering.

### What didn't work

- N/A; no failing test iterations were required for this tranche.

### What I learned

- A small queue + explicit drop policy gives predictable failure isolation with low complexity.

### What was tricky to build

- Ensuring `remove()` stays safe and idempotent across both read-loop disconnects and write-pump failures.

### What warrants a second pair of eyes

- Queue size constant (`64`) may need tuning under higher fanout workloads.

### What should be done in the future

- Add instrumentation around enqueue drops and write-pump failures for operations visibility.

### Code review instructions

- Review `wsClient.enqueue*`, `writePump`, and `wsBroadcaster.remove` in `internal/server/ws.go`.
- Run `go test ./internal/server -count=1`.

### Technical details

- Non-blocking broadcast path: enqueue with immediate queue-full detection.
- Connect-time replay path: timed enqueue (`5s`) to avoid indefinite blocking during initial sync.
