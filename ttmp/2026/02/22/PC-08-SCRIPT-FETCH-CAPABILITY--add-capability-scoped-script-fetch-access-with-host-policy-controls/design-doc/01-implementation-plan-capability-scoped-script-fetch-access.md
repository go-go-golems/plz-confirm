---
Title: Implementation Plan - capability-scoped script fetch access
Ticket: PC-08-SCRIPT-FETCH-CAPABILITY
Status: active
Topics:
    - backend
    - security
    - javascript
    - api
    - ux
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: internal/scriptengine/engine.go
      Note: Runtime bridge where fetch capability and host policy checks will be enforced
    - Path: internal/server/script.go
      Note: Script lifecycle path where approval-required network operations can be modeled
    - Path: internal/server/server.go
      Note: Request creation path for network policy bootstrap and defaults
    - Path: proto/plz_confirm/v1/widgets.proto
      Note: Script input schema extension for fetch capability declarations
    - Path: pkg/doc/js-script-api.md
      Note: Public docs for fetch API and safety constraints
    - Path: pkg/doc/js-script-development.md
      Note: Internal docs for policy enforcement, transport limits, and auditing
ExternalSources: []
Summary: Detailed fetch capability design with policy and confirmation controls.
LastUpdated: 2026-02-23T00:10:00Z
WhatFor: ""
WhenToUse: ""
---

# Implementation Plan - capability-scoped script fetch access

## Executive Summary

Script-side `fetch()` would unlock high-value workflows (querying internal metadata APIs, validating deploy status, loading catalog choices), but unrestricted network access from scripts is a major SSRF and data-exfiltration risk. This plan introduces host-enforced, capability-scoped fetch access with strict egress policy, bounded response handling, and optional human confirmation for sensitive requests.

Recommended architecture mirrors the FS approach:

- Static capability bounds (allowed hosts/routes/methods) define hard limits.
- Runtime policy decides allow/deny/require-confirmation per request.
- Host-side confirmation can gate high-risk calls before execution.
- Auditing and budgets are mandatory.

## Problem Statement

Without `fetch()`, scripts cannot build dynamic network-backed UI flows. With naive `fetch()`, scripts could:

- Access cloud metadata endpoints (`169.254.169.254`) or internal admin APIs.
- Probe private network ranges via SSRF patterns.
- Exfiltrate sensitive tokens or large data to external endpoints.
- Abuse redirects to bypass hostname allowlists.

We need network flexibility for safe use cases while preserving strong host control and least privilege.

## Goals and Non-Goals

### Goals

- Provide read-oriented HTTP capability (`GET` first, optional controlled POST in later phase).
- Enforce host-side egress policy independent of script logic.
- Prevent SSRF into private/internal targets by default.
- Support selective host confirmation before risky fetch operations.
- Bound latency, response size, and request volume.

### Non-Goals

- No arbitrary socket/TCP/UDP access.
- No unrestricted method/header/body forwarding in v1.
- No script control over transport-level trust (custom CAs, insecure TLS, proxy bypass) by default.

## Proposed API Surface

### ScriptInput extension

Add fetch capability fields:

- `optional ScriptFetchPolicy fetch_policy`
- `optional ScriptFetchLimits fetch_limits`
- `repeated ScriptFetchCredentialRef fetch_creds` (optional host-resolved credentials)

`ScriptFetchPolicy` draft elements:

- allowed schemes (`https` only default)
- allowed host patterns
- allowed ports
- allowed methods
- allowed path regex/glob per host
- redirect policy
- private-network policy
- confirmation rules

### Runtime `ctx.fetch`

Two safe options; choose one for v1:

1. `ctx.fetch(url, options)` custom API with tightly defined options.
2. Provide global `fetch` with host-side wrapper that enforces same constraints.

Recommended for v1: `ctx.fetch` to avoid browser/API shape expectations that imply unsupported behaviors.

Minimal options:

- method (default GET)
- headers (allowlisted only)
- query
- timeoutMs (clamped)
- responseType (`text`/`json`)

Response shape:

- status
- headers (filtered)
- body (`text`) or `json` object
- truncated flag

## Security Model Options

### Model A: Host allowlist only (no confirmation)

- Hard allowlist on hosts/methods/paths.

Pros: simple and deterministic.
Cons: insufficient for mixed-trust endpoints.

### Model B: Always confirm before each request

Pros: strong human control.
Cons: unusable for repeated/benign API reads.

### Model C: Hybrid policy + selective confirmation (recommended)

- Allow low-risk requests automatically.
- Require confirmation for elevated risk categories.
- Cache grants per scope/TTL.

Pros: practical UX with strong controls.
Cons: highest implementation complexity.

## Recommended Security Architecture (Model C)

### 1. Capability boundary

At request creation, compile immutable capability set:

- host/path/method constraints
- header allowlist
- credential reference scope
- size/time/rate limits

### 2. Request normalization and canonicalization

Before policy evaluation:

- normalize URL and punycode host
- resolve redirects under policy (or deny)
- normalize method and headers
- canonicalize default ports

### 3. Policy engine

Per request, return:

- `ALLOW`
- `DENY`
- `REQUIRE_CONFIRMATION`

Policy signals include:

- host sensitivity tier
- method risk (`GET` < `POST`)
- path pattern class
- credential attachment requested
- data volume estimate

### 4. SSRF defenses

Mandatory checks:

- block link-local, loopback, RFC1918 private ranges by default
- block cloud metadata IPs and known aliases
- DNS resolution guard with IP-range enforcement
- re-check IP on redirects
- optional DNS pinning per request to reduce rebinding risk

### 5. Host confirmation gateway

If policy says `REQUIRE_CONFIRMATION`:

- present request details to user: host/path/method/headers/risk reason
- offer grant scopes: once, step, request TTL
- on approval, store scoped grant and continue

### 6. Credential model

Scripts should not handle raw secrets directly. Use host-managed credential refs:

- script requests `credRef: "service-x-ro"`
- host resolves and injects allowed auth header if policy permits
- script never sees secret value unless explicitly configured

### 7. Auditing

Log every decision and request outcome:

- request id, script name/version
- normalized target
- policy result and reason
- approval id/grant id
- response status, bytes, duration

## Design Decisions

### Decision 1: HTTPS-only default

Reject non-HTTPS by default; allow HTTP only via explicit policy for known local dev targets.

### Decision 2: GET-only initial rollout

Start with read-style GET to reduce blast radius. Extend methods later by policy tier.

### Decision 3: Host-managed credential refs

Keep secrets out of script space and enforce per-endpoint credential policy.

### Decision 4: Response/body budget caps

Hard cap response bytes and total network budget per request lifecycle.

### Decision 5: Redirects restricted by policy

Default deny cross-host redirects. Allow same-host redirects with bounded hop count.

## Implications and Tradeoffs

### Security

- Strong reduction of SSRF risk with network/IP controls and host confirmation.
- Complexity increases in policy engine and transport wrappers.

### UX

- Selective confirmation keeps routine flows smooth.
- High-risk calls can still be user-approved when needed.

### Performance

- DNS/IP checks and policy evaluation add overhead.
- Bounded retries/timeouts reduce tail latency risk.

### Operational

- Requires policy config lifecycle (defaults, overrides, environment-specific rules).
- Needs observability dashboards for denied/confirmed requests.

## Alternatives Considered

### Alternative A: Full unrestricted fetch in scripts

Rejected due to unacceptable SSRF and exfiltration risk.

### Alternative B: Proxy all requests through external gateway only

Deferred. Strong central control but operational dependency and rollout overhead.

### Alternative C: No credentials at all in v1

Possible but limits practical internal API use. Prefer credential refs with strict scope.

## Implementation Plan

### Phase 0: Contract and policy baseline

1. Extend protobuf with fetch policy/limits/credential refs.
2. Add server validation and defaults.
3. Add deny-by-default behavior when fetch policy missing.

### Phase 1: Safe transport adapter

1. Implement `sandbox_http.go` with normalized request builder.
2. Enforce scheme/host/method/path/header constraints.
3. Enforce request timeout, max redirects, and response-size cap.

### Phase 2: SSRF guardrail layer

1. Implement DNS resolution checks and blocked-range enforcement.
2. Add redirect target revalidation.
3. Add metadata endpoint blocklist.

### Phase 3: Policy + confirmation

1. Implement policy evaluator returning allow/deny/confirm.
2. Implement grant store with scoped TTL grants.
3. Integrate host confirmation flow and script resume behavior.

### Phase 4: Credentials + observability

1. Add credential ref resolver and policy checks.
2. Add audit logs and metrics.
3. Document operational controls and runbooks.

## Testing Strategy

- Unit tests for URL normalization and policy matching.
- Unit tests for SSRF blockers (private/link-local/metadata/redirect edge cases).
- Integration tests for allow/deny/confirmation lifecycle.
- Adversarial tests for header abuse, oversized responses, redirect chains, DNS rebinding simulation.
- Load tests for repeated network calls under budget limits.

## Documentation Plan

- `pkg/doc/js-script-api.md`: fetch contract, limits, and examples.
- `pkg/doc/js-script-development.md`: transport internals and debugging guidance.
- Ticket-local smoke scripts for safe and denied fetch scenarios.

## Open Questions

- Should we support streaming responses, or only buffered with strict caps?
- How should we expose filtered response headers to scripts?
- Should policy support per-session trust levels (internal automation vs external agent)?
- Should confirmation prompt include sanitized response preview for user validation?

## References

- `ttmp/2026/02/22/PC-07-SCRIPT-FS-READ-CAPABILITY--add-capability-scoped-script-fs-read-access-with-host-confirmation/design-doc/01-implementation-plan-capability-scoped-script-fs-read-access.md`
- `ttmp/2026/02/22/PC-01-ADD-JS-API--add-js-api-describe-extension/sources/local/plz-confirm-js.md`
