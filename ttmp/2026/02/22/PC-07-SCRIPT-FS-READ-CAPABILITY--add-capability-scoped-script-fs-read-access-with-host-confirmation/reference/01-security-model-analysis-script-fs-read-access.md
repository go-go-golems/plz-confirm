---
Title: Security Model Analysis - Script FS read access
Ticket: PC-07-SCRIPT-FS-READ-CAPABILITY
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
      Note: Current runtime isolation baseline with no fs bridge
    - Path: internal/server/script.go
      Note: Event lifecycle hooks where approval interruption/resume can be integrated
    - Path: proto/plz_confirm/v1/widgets.proto
      Note: ScriptInput schema extension point for fs capabilities
ExternalSources: []
Summary: Threat model and comparative analysis of host-side file-read security approaches for scripts.
LastUpdated: 2026-02-23T00:02:00Z
WhatFor: ""
WhenToUse: ""
---

# Security Model Analysis - Script FS read access

## Goal

Provide a decision framework for enabling script file reads without compromising host security, while preserving enough flexibility for rich review UIs.

## Context

Scripts are dynamic code provided via request payloads. Even when scripts are authored by trusted agents, runtime safety must assume malformed, buggy, or adversarial behavior. Read access therefore needs host-side enforcement with strict policy boundaries.

## Quick Reference

### Threat categories

| Threat | Example | Impact | Baseline mitigation |
|---|---|---|---|
| Traversal escape | `repo:../../.ssh/id_rsa` | Secret exposure | Canonicalization + root containment |
| Symlink escape | In-repo symlink points outside root | Secret exposure | Resolve target and enforce root scope |
| Sensitive read abuse | Reading `.env`, cloud creds | Credential leakage | Denylist + confirmation-required policy |
| Resource abuse | Repeated large reads | CPU/memory pressure | Per-read limits + total read budgets |
| Prompt abuse | Forcing many confirmations | UX denial and bad approvals | Prompt coalescing + rate limits + TTL grants |

### Security model comparison

| Model | Description | Security | UX | Complexity | Recommendation |
|---|---|---|---|---|---|
| Static allowlist | Mounts and path rules only | Medium | High | Low | Good baseline but insufficient for sensitive paths |
| Always-confirm | Every read asks user | High | Low | Medium | Too noisy for practical workflows |
| Hybrid policy + selective confirmation | Policy decides allow/deny/prompt, with cached grants | High | Medium/High | High | Recommended |
| Out-of-process sandbox only | Isolated worker process handles reads | Very high | Medium | Very high | Future hardening option |

### Recommended policy stack

1. Capability bounds (mount roots, read-only methods).
2. Policy evaluation (allow/deny/confirm).
3. Host approval gateway for `REQUIRE_CONFIRMATION`.
4. Scoped grants with TTL and byte caps.
5. Audit logs for all decisions.

## Usage Examples

### Example policy outcomes

| Requested operation | Policy result | Reason |
|---|---|---|
| `readText(repo:README.md)` | `ALLOW` | In-bounds, non-sensitive, under size cap |
| `readText(repo:.env)` | `REQUIRE_CONFIRMATION` | Sensitive classifier match |
| `readText(repo:../../etc/passwd)` | `DENY` | Traversal/root violation |
| `readBytes(repo:large.bin)` | `DENY` | Exceeds max bytes policy |

### Example host approval prompt payload

```json
{
  "requestId": "...",
  "script": { "name": "deploy-review", "version": "1.2.0" },
  "operation": "readText",
  "path": "repo:.env",
  "risk": "sensitive-file",
  "grantOptions": ["once", "this-step", "request-ttl-5m"],
  "maxBytes": 8192
}
```

### Example grant record

```json
{
  "scope": "repo:.env",
  "op": "readText",
  "expiresAt": "2026-02-23T05:20:00Z",
  "maxBytes": 8192,
  "source": "user-confirmed"
}
```

## Decision Notes

- Host-side confirmation is mandatory for enforceability. Script-side prompts are advisory only.
- Approval state should be request-scoped by default to minimize long-lived permissions.
- Deny-by-default remains the safest startup mode when no mounts are declared.

## See Also

- `ttmp/2026/02/22/PC-07-SCRIPT-FS-READ-CAPABILITY--add-capability-scoped-script-fs-read-access-with-host-confirmation/design-doc/01-implementation-plan-capability-scoped-script-fs-read-access.md`
- `ttmp/2026/02/22/PC-01-ADD-JS-API--add-js-api-describe-extension/sources/local/plz-confirm-js.md`
