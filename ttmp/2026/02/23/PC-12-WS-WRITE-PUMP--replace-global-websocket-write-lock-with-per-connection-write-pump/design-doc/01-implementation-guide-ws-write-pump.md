---
Title: 'Implementation Guide: WS Write Pump'
Ticket: PC-12-WS-WRITE-PUMP
Status: active
Topics:
    - architecture
    - backend
    - go
    - ux
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/internal/server/ws.go
    - /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/internal/server/server.go
    - /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/internal/server/ws_test.go
ExternalSources: []
Summary: Replace global websocket write serialization with per-connection write pumps to avoid cross-client head-of-line blocking.
LastUpdated: 2026-02-24T09:40:00-05:00
WhatFor: Improve websocket fanout reliability and scalability while preserving current event contracts.
WhenToUse: Use when changing websocket broadcast behavior or debugging dropped/stalled websocket clients.
---

# Implementation Guide: WS Write Pump

## Executive Summary

The current websocket broadcaster serializes **all writes for all clients** behind a single mutex. This guarantees no concurrent writes to a `*websocket.Conn`, but it also introduces global head-of-line blocking: one slow client can delay every other client.

This ticket replaces the global write lock with a **per-connection write pump**:

1. each connection gets its own queue;
2. one dedicated goroutine drains that queue and performs socket writes;
3. broadcasters enqueue payloads without globally blocking unrelated clients.

## Problem Statement

### Current behavior

`internal/server/ws.go` currently keeps a `writeMu` at broadcaster level. Both `BroadcastJSON` and `BroadcastRawJSON` loop over session clients and call write helpers that lock `writeMu`.

### Failure mode

If one client is slow, congested, or stalled, writes to that client hold `writeMu`, delaying writes to all other clients across the same process. This is a scale and responsiveness issue and makes broadcast latency sensitive to worst-case client behavior.

## Proposed Solution

### Data model changes

Introduce a `wsClient` wrapper for each socket:

- `conn *websocket.Conn`
- `sessionID string`
- `send chan []byte` (bounded)
- `done chan struct{}` (shutdown signal)
- `closeOnce` + `closed flag` (idempotent stop)

`wsBroadcaster` stores clients by session and by conn for lookup/removal.

### Write path changes

- `BroadcastJSON`: marshal once to raw JSON bytes, delegate to `BroadcastRawJSON`.
- `BroadcastRawJSON`: snapshot session clients and enqueue payload to each `wsClient`.
- On enqueue failure (queue full/closed), drop that client only.

### Connection lifecycle

- On `add`, create/start write pump goroutine for the connection.
- On `remove`, remove from maps and stop client (close conn + done channel).
- On write pump error, remove/drop only that client.

### Connect-time pending replay

`handleWS` initial pending replay should enqueue through the same per-client queue to preserve the single write path invariant.

## Design Decisions

1. **Bounded queue per client**
Why: avoids unbounded memory growth under slow consumers.

2. **Drop slow clients when queue pressure persists**
Why: protects process-level responsiveness and keeps healthy clients flowing.

3. **Single writer goroutine per connection**
Why: gorilla websocket requires write serialization per connection.

4. **Session-scoped broadcast unchanged**
Why: preserve external behavior and API contract.

## Alternatives Considered

1. Keep global `writeMu`
Rejected: preserves current blocking bottleneck.

2. Per-connection mutex only (no queue)
Rejected: still does synchronous writes in broadcaster call path and keeps slow-client backpressure in hot path.

3. Unbounded queue
Rejected: unacceptable memory risk under stalled consumers.

## Implementation Plan

1. Implement `wsClient` and pump lifecycle in `ws.go`.
2. Update broadcaster maps/snapshot/removal semantics.
3. Route initial pending events through enqueue path.
4. Add tests and run `go test ./internal/server -count=1`.
5. Record outcomes in diary/changelog and commit.

## Pseudocode

```go
for client in sessionSnapshot {
  if err := client.enqueue(msg); err != nil {
    broadcaster.remove(client.conn) // isolate failure
  }
}

func (c *wsClient) writePump(onErr func(error)) {
  for {
    select {
    case <-c.done:
      return
    case msg := <-c.send:
      if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
        onErr(err)
        return
      }
    }
  }
}
```

## Open Questions

1. Queue size defaults: keep constant in code for now, or expose as server config later?
2. Do we want queue overflow metrics counters before broad rollout?

## References

- `internal/server/ws.go`
- `internal/server/server.go`
- Inspector finding `P3 #9` in `PC-05` review doc
