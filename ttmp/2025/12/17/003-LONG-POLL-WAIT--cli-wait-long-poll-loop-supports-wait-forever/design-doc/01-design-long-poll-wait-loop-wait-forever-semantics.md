---
Title: 'Design: long-poll wait loop + wait-forever semantics'
Ticket: 003-LONG-POLL-WAIT
Status: active
Topics:
    - cli
    - backend
    - go
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Design for making plz-confirm CLI wait robust via a long-poll loop, including --wait-timeout 0 to wait forever, without requiring a CLI websocket."
LastUpdated: 2025-12-17T17:01:08.906679527-05:00
---

# Design: long-poll wait loop + wait-forever semantics

## Executive Summary

We will change the CLI’s “wait for response” behavior from a single long-poll request into a **long-poll loop**:

- Break the wait into repeated `GET /api/requests/{id}/wait?timeout=<pollS>` calls.
- Treat `--wait-timeout` as the **overall** time budget.
- Support `--wait-timeout 0` to **wait forever** (until the user cancels).

This avoids adding a CLI WebSocket implementation while still enabling “infinite” waits and working around intermediary timeouts (client/proxy/LB).

## Problem Statement

Today, the CLI uses a single `GET /wait?timeout=<waitTimeout>` request to block until the user responds. This has two issues:

1. The Go HTTP client is configured with a fixed `Timeout: 30s`, which makes waits longer than 30 seconds unreliable.
2. “Wait forever” is not possible without a long-lived connection; even if we remove the client timeout, intermediaries (proxies, LBs) can terminate idle connections.

## Proposed Solution

### API behavior (unchanged)

We keep using:
- `POST /api/requests` to create a request
- `GET /api/requests/{id}/wait?timeout=<seconds>` to wait (server blocks up to `<seconds>`)
- `POST /api/requests/{id}/response` to complete

### Client behavior (changed)

Implement a loop in `internal/client.Client.WaitRequest`:

- If `waitTimeoutS > 0`:
  - create an overall deadline: `ctx = context.WithTimeout(ctx, waitTimeoutS)`
- Else (`waitTimeoutS <= 0`):
  - no overall deadline; loop until completion or user cancellation (`ctx.Done()`).

Each iteration:
- pick `pollS = min(defaultPollS, remainingSeconds)` (if there is an overall deadline)
- call a single long-poll: `GET /wait?timeout=pollS`
- if response is `408 Request Timeout`, continue loop
- if response is `200`, return the request
- any other error terminates

### HTTP client timeout

Remove the global `http.Client.Timeout` (or set it to “no timeout”) and rely on contexts for request timeouts/deadlines. The long-poll loop ensures each HTTP request is bounded.

## Design Decisions

- **Implement looping in the shared client** (`internal/client.WaitRequest`) so all widget commands (`confirm`, `select`, `form`, `upload`, `table`) benefit without duplicating logic.
- **Use `--wait-timeout 0` for “wait forever”** so users don’t need a new flag.
- **Use long-poll loop instead of CLI WebSocket**:
  - lower complexity (no WS framing, reconnect semantics, multiplexing)
  - still robust against typical infra timeouts

## Alternatives Considered

- **CLI WebSocket**: clean push semantics, but adds complexity (reconnect, message routing, auth/session semantics).
- **Single long-lived HTTP wait**: simplest, but fragile in practice due to client/proxy timeouts.

## Implementation Plan

- Update `internal/client/client.go`:
  - remove global 30s `http.Client.Timeout`
  - add `ErrWaitTimeout` sentinel for 408 from `/wait`
  - implement long-poll loop and overall deadline semantics
- Update CLI help strings for `--wait-timeout`:
  - document `0 = wait forever`
- Add unit tests for `WaitRequest` retry/loop behavior using `httptest.Server`

## Open Questions

- Default `pollS`: use 20–30s to avoid common proxy/LB idle timeouts while keeping request rate low.
- Should we add a small backoff if the server returns 408 immediately (misconfiguration) to avoid tight loops?

## References

- Implementation anchor: `plz-confirm/internal/client/client.go`
- Server wait handler: `plz-confirm/internal/server/server.go` (`handleWait`)
