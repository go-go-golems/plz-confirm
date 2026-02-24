---
Title: 'Implementation Plan: Host-Injected WS Reconnect Policy'
Ticket: PC-14-WS-RECONNECT-POLICY
Status: active
Topics:
    - architecture
    - frontend
    - javascript
    - ux
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/host/types.ts
    - /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/ws/confirmWsManager.ts
    - /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/runtime/createConfirmRuntime.ts
    - /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/apps/inventory/src/App.tsx
ExternalSources: []
Summary: Add host-injected reconnect policy for websocket runtime resiliency without hardwired retry behavior.
LastUpdated: 2026-02-24T09:44:00-05:00
WhatFor: Make reconnect strategy explicit and host-configurable.
WhenToUse: Use when tuning reconnect behavior for kiosk/desktop/web hosts.
---

# Implementation Plan: Host-Injected WS Reconnect Policy

## Executive Summary

`ConfirmWsManager` currently opens one websocket and never retries after disconnect. The runtime remains offline until app reload.

This ticket adds a reconnect policy hook that is injected from host adapters, consistent with the design decision recorded in the review doc.

## Problem Statement

### Current behavior

- `connect()` opens websocket once.
- `onclose` only dispatches disconnected state.
- no reconnect/backoff path.

### Operational impact

Transient network interruptions require manual page reload/restart.

## Proposed Solution

### Contract

Add a host-facing reconnect policy callback:

```ts
type ConfirmWsReconnectPolicy = (ctx) => ({ reconnect: boolean; delayMs?: number } | null)
```

Context includes attempt count and close metadata (code/reason/clean close).

### Manager behavior

- track `disconnectRequested` and retry timer;
- on unexpected close, evaluate policy;
- if policy says reconnect, schedule delayed reconnect;
- clear pending retry on explicit `disconnect()`.

### Runtime wiring

`createConfirmRuntime` reads host policy and passes it to ws manager.

### Host usage

Inventory app provides initial exponential backoff strategy (bounded max delay).

## Design Decisions

1. Host injection over hard-coded defaults.
2. Reconnect disabled by default when no policy is supplied.
3. `disconnect()` is terminal until explicit `connect()` call.

## Alternatives Considered

1. Hardwired runtime backoff values.
Rejected: conflicts with host-specific reliability needs and explicit architecture decision.

2. Reconnect loop in app layer only.
Rejected: duplicates socket lifecycle logic per app and weakens package cohesion.

## Implementation Plan

1. Add reconnect policy types to `host/types.ts`.
2. Implement retry scheduling/canceling in `confirmWsManager.ts`.
3. Thread policy through `createConfirmRuntime.ts`.
4. Add policy in inventory host adapter.
5. Add manager tests for retry/no-retry/disconnect-cancel.

## Pseudocode

```ts
onclose(event) {
  if (disconnectRequested) return
  const decision = reconnectPolicy?.({ attempt: attempt + 1, ...event })
  if (!decision?.reconnect) return
  scheduleReconnect(decision.delayMs ?? 0)
}
```

## Open Questions

1. Should we add jitter utility in runtime package or keep per-host?
2. Should max-attempt enforcement remain policy-level only?

## References

- Inspector finding `P3 #8` in `PC-05` review.
- Decision update: reconnect policy injected via host adapters.
