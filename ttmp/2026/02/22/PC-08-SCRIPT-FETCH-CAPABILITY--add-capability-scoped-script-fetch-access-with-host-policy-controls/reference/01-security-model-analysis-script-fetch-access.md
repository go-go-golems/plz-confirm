---
Title: Security Model Analysis - Script fetch access
Ticket: PC-08-SCRIPT-FETCH-CAPABILITY
Status: active
Topics:
    - backend
    - security
    - javascript
    - api
    - ux
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: internal/scriptengine/engine.go
      Note: Runtime bridge where JS fetch calls can be intercepted and policy-enforced
    - Path: internal/server/script.go
      Note: Script lifecycle and approval gating path for confirmation-required network calls
    - Path: proto/plz_confirm/v1/widgets.proto
      Note: ScriptInput schema extension point for fetch capabilities and policy defaults
    - Path: pkg/doc/js-script-api.md
      Note: Public script API documentation location for fetch contract and limits
ExternalSources: []
Summary: Threat model and comparative analysis of host-side network security approaches for scripts.
LastUpdated: 2026-02-23T00:20:00Z
WhatFor: ""
WhenToUse: ""
---

# Security Model Analysis - Script fetch access

## Goal

Provide a decision framework for enabling script-initiated HTTP requests while preventing SSRF, data exfiltration, and resource abuse.

## Context

Scripts are dynamic code and must be treated as untrusted at runtime, even when authored by internal agents. Exposing raw `fetch()` without host controls allows network pivoting into internal systems and cloud metadata services. The security boundary must therefore stay on the host side, not in script logic.

## Quick Reference

### Threat categories

| Threat | Example | Impact | Baseline mitigation |
|---|---|---|---|
| SSRF to private infra | `http://10.0.0.12/admin` | Internal service compromise | Default block RFC1918, loopback, link-local |
| Cloud metadata access | `http://169.254.169.254/latest/meta-data` | Credential theft | Explicit metadata denylist + IP policy |
| DNS rebinding | Host resolves safe then flips to private IP | Policy bypass | Resolve/connect-time IP checks + redirect revalidation |
| Redirect escape | Allowed domain 302 -> disallowed host | Egress bypass | Re-check each redirect target under same policy |
| Token/header abuse | Script sets arbitrary auth headers | Secret leakage | Header allowlist + host-managed credential refs |
| Resource exhaustion | Huge responses, many calls | CPU/memory/network pressure | Byte/time/call budgets and hard caps |
| Prompt spam | Script triggers repeated confirmations | Approval fatigue | Prompt dedupe, throttling, scoped TTL grants |

### Security model comparison

| Model | Description | Security | UX | Complexity | Recommendation |
|---|---|---|---|---|---|
| Static allowlist only | Host/path/method constraints, no confirmation | Medium | High | Low | Acceptable base, weak for mixed sensitivity APIs |
| Always confirm | Every request needs operator approval | High | Low | Medium | Secure but impractical at scale |
| Hybrid policy + selective confirmation | Policy returns allow/deny/confirm with scoped grants | High | Medium/High | High | Recommended |
| External proxy-only | All calls must transit policy gateway service | Very high | Medium | Very high | Future hardening option |

### Recommended baseline defaults

1. Deny by default when `fetch_policy` is absent.
2. HTTPS only by default.
3. GET only in v1 unless policy explicitly allows additional methods.
4. No arbitrary headers; only a tight allowlist plus host-injected credentials.
5. Hard caps for timeout, response size, and request count.

## Usage Examples

### Example policy outcomes

| Request | Result | Reason |
|---|---|---|
| `GET https://status.internal/api/health` | `ALLOW` | Host/path/method allowlisted, low risk |
| `GET https://billing.internal/api/export` | `REQUIRE_CONFIRMATION` | Sensitive endpoint class |
| `GET http://169.254.169.254/latest/meta-data` | `DENY` | Metadata endpoint blocked |
| `POST https://catalog.internal/api/update` | `DENY` | Method not allowed in current policy |

### Example confirmation payload shape

```json
{
  "requestId": "req-123",
  "script": { "name": "release-checker", "version": "1.0.3" },
  "target": {
    "method": "GET",
    "url": "https://billing.internal/api/export"
  },
  "risk": "sensitive-endpoint",
  "grantOptions": ["once", "step", "request-ttl-5m"]
}
```

### Example audit event shape

```json
{
  "requestId": "req-123",
  "scriptName": "release-checker",
  "action": "fetch",
  "targetHost": "billing.internal",
  "method": "GET",
  "policyDecision": "REQUIRE_CONFIRMATION",
  "grantSource": "user-confirmed",
  "statusCode": 200,
  "responseBytes": 1459,
  "durationMs": 83
}
```

## Decision Notes

- Host-side enforcement is non-negotiable; script-side prompts are informational only.
- Credential refs should be resolved by host so scripts never receive raw secret values.
- Confirmation grants should default to request scope with short TTLs.
- Redirect behavior should default to same-host only unless explicitly broadened.

## Related

- `ttmp/2026/02/22/PC-08-SCRIPT-FETCH-CAPABILITY--add-capability-scoped-script-fetch-access-with-host-policy-controls/design-doc/01-implementation-plan-capability-scoped-script-fetch-access.md`
- `ttmp/2026/02/22/PC-07-SCRIPT-FS-READ-CAPABILITY--add-capability-scoped-script-fs-read-access-with-host-confirmation/design-doc/01-implementation-plan-capability-scoped-script-fs-read-access.md`
- `ttmp/2026/02/22/PC-01-ADD-JS-API--add-js-api-describe-extension/sources/local/plz-confirm-js.md`
