---
Title: Diary
Ticket: PC-14-WS-RECONNECT-POLICY
Status: active
Topics:
    - architecture
    - frontend
    - javascript
    - ux
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/ws/confirmWsManager.ts
    - /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/runtime/createConfirmRuntime.ts
ExternalSources: []
Summary: Diary for host-injected websocket reconnect policy implementation.
LastUpdated: 2026-02-24T09:44:00-05:00
WhatFor: Preserve lifecycle/retry behavior details and testing notes.
WhenToUse: Use when extending reconnect behavior.
---

# Diary

## Goal

Track implementation of host-injected websocket reconnect behavior in confirm-runtime.

## Step 1: Baseline and contract direction

I confirmed the websocket manager has no reconnect lifecycle and validated the earlier decision that reconnect policy belongs in host adapters rather than hardcoded runtime defaults. This step scoped the API surface and manager state transitions needed for deterministic behavior.

I then documented implementation plan, retry semantics, and test cases before changing runtime code.

### Prompt Context

**User prompt (verbatim):** (see PC-12 Step 1)

**Assistant interpretation:** Address P3 reconnect hardening with explicit ticketing and implementation details.

**Inferred user intent:** Keep runtime reusable while giving hosts explicit control over operational retry strategy.

**Commit (code):** N/A (planning step)

### What I did

- Created ticket `PC-14-WS-RECONNECT-POLICY`.
- Confirmed missing reconnect path in `confirmWsManager.ts`.
- Wrote host-injected policy contract + lifecycle plan.

### Why

- Avoid hidden retry behavior while eliminating manual recovery from transient disconnects.

### What worked

- Existing manager API is small, so policy injection can be added without breaking broad surface area.

### What didn't work

- N/A in this step.

### What I learned

- Reconnect behavior needs explicit cancellation semantics on `disconnect()` to prevent zombie retries.

### What was tricky to build

- N/A yet; code changes pending.

### What warrants a second pair of eyes

- Potential edge cases with stale socket close events during reconnect transitions.

### What should be done in the future

- Add metrics hooks for retry attempts/failures if operations require observability.

### Code review instructions

- Start with `ConfirmWsManager.connect/disconnect` lifecycle.
- Validate timer cancellation and stale close event handling in tests.

### Technical details

- Current manager is single-shot connect/disconnect with no retry timer state.

## Step 2: Implement host-injected reconnect policy in websocket manager

I extended `ConfirmWsManager` with retry state and timer handling, then threaded a host-supplied reconnect policy function from runtime creation down into websocket lifecycle decisions. Reconnect remains opt-in: if no policy is provided, behavior stays single-shot.

I also configured the inventory host with a bounded backoff policy and added manager tests to confirm reconnect scheduling and timer cancellation on explicit disconnect.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Land the reconnect policy decision as concrete runtime behavior and tests.

**Inferred user intent:** Improve resilience without embedding one hard-coded retry strategy into shared runtime package.

**Commit (code):** `5b373ac` — "confirm-runtime: add reconnect policy and 409 conflict helper"

### What I did

- Added reconnect contract types in `host/types.ts`.
- Implemented `reconnectPolicy` support in `ws/confirmWsManager.ts`:
  - tracks attempts,
  - schedules delayed reconnects,
  - cancels retries on disconnect,
  - ignores stale socket close events.
- Passed policy through `runtime/createConfirmRuntime.ts`.
- Added inventory host policy via `resolveWsReconnectPolicy` in `App.tsx`.
- Added `ws/confirmWsManager.test.ts` with reconnect/no-reconnect/disconnect-cancel cases.

### Why

- Remove manual recovery requirement after transient websocket drops.
- Keep operational policy host-owned per architectural decision.

### What worked

- Reconnect tests passed with fake timers.
- Existing ws manager payload parsing path remained intact.

### What didn't work

- Full repo typecheck still fails from unrelated pre-existing baseline issues (React typings and story typing errors in engine/apps).

### What I learned

- Explicit `disconnectRequested` + stale-socket identity checks are necessary to avoid accidental reconnect loops.

### What was tricky to build

- Avoiding reconnect races where old socket close events fire after a new socket has been created.

### What warrants a second pair of eyes

- Backoff constants in inventory host (`500ms` exponential factor with cap) may need tuning under real production conditions.

### What should be done in the future

- Optional jitter helper and telemetry hooks for reconnect outcomes.

### Code review instructions

- Review `ConfirmWsManager` reconnect lifecycle first.
- Then review host policy injection in `createConfirmRuntime` and inventory host adapter wiring.
- Run:
  - `npx vitest run packages/confirm-runtime/src/ws/confirmWsManager.test.ts`

### Technical details

- Policy shape: `(context) => { reconnect: boolean; delayMs?: number }`.
- Context includes attempt count and close metadata (`code/reason/wasClean`).
