---
Title: Implementation Plan - capability-scoped script FS read access
Ticket: PC-07-SCRIPT-FS-READ-CAPABILITY
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
      Note: Runtime bridge where fs capabilities and host policy checks will be enforced
    - Path: internal/server/script.go
      Note: Script lifecycle flow where approval-required interruptions can be mapped to UI updates
    - Path: internal/server/server.go
      Note: Request creation path for fs policies and per-request capability bootstrap
    - Path: proto/plz_confirm/v1/widgets.proto
      Note: Script input schema extension for read capability declarations
    - Path: proto/plz_confirm/v1/request.proto
      Note: Potential state fields for pending host approvals and resumable fs operations
    - Path: pkg/doc/js-script-api.md
      Note: Public docs for fs API and security constraints
    - Path: pkg/doc/js-script-development.md
      Note: Internal docs for policy enforcement and auditing model
ExternalSources: []
Summary: Detailed FS read capability design, policy models, and host confirmation enforcement.
LastUpdated: 2026-02-23T00:02:00Z
WhatFor: ""
WhenToUse: ""
---

# Implementation Plan - capability-scoped script FS read access

## Executive Summary

We want scripts to read files so they can build richer UIs (diff previews, config summaries, log triage), but unrestricted host filesystem access is unsafe. This plan introduces read-only, capability-scoped filesystem access with layered policy enforcement and optional host confirmation before each sensitive read.

The recommended model is hybrid:

- Static capabilities (mounts and path constraints) define upper bounds.
- Runtime policy evaluates each read attempt.
- Host-side confirmation can be required by policy before granting specific reads.
- All accesses are audited and bounded.

## Problem Statement

Current scripts cannot read files at all. This blocks key use cases:

- Show changed file previews before approval.
- Build context-driven forms from repository config.
- Present log slices or generated artifacts for human review.

At the same time, naive file access introduces major risk:

- Secret exfiltration from home directories or environment files.
- Path traversal and symlink escapes.
- Excessive reads causing performance or data exposure incidents.

We need a model that enables controlled reads while keeping security enforcement in Go host code, not in script code.

## Goals and Non-Goals

### Goals

- Read-only filesystem access from scripts.
- Per-request scoped capabilities with least privilege.
- Host-enforced policy and optional confirmation before reads.
- Strong traversal resistance and path canonicalization.
- Auditability for all allowed/denied read attempts.

### Non-Goals

- No write, delete, rename, or chmod operations.
- No arbitrary shell command execution.
- No network/file bridge that bypasses policy.

## Proposed API Surface

### ScriptInput extension

Add fields to `ScriptInput`:

- `repeated ScriptFileMount mounts`
- `optional ScriptFSLimits fs_limits`
- `optional ScriptFSPolicy fs_policy`

`ScriptFileMount` draft:

- `name` (logical handle, e.g. `repo`)
- `root` (host path configured/validated server-side)
- `mode` (`READ_ONLY` only for v1)
- `allow_globs` (optional include patterns)
- `deny_globs` (optional deny patterns)

### Runtime `ctx.fs` methods (v1)

- `readText(path, options?)`
- `readBytes(path, options?)` (bounded base64 result optional)
- `stat(path)`
- `glob(pattern)`
- `exists(path)`

Path format:

- `mountName:relative/path`

No absolute path support in script API.

## Security Model Options

### Model A: Static allowlist only

- Mounts and glob filters are fixed at request creation.
- Reads either allowed or denied automatically.

Pros: simple and predictable.
Cons: weak for sensitive-but-legitimate files requiring human gate.

### Model B: Always prompt host for every read

- Every read triggers host confirmation.

Pros: strongest human control.
Cons: unusable UX, high latency, noisy prompt fatigue.

### Model C: Hybrid policy + selective confirmation (recommended)

- Default allow for low-risk files within mount policy.
- Required confirmation for high-risk paths or policy rules.
- Grant caching with TTL and scope (single path, glob, or mount).

Pros: practical UX with strong controls.
Cons: most implementation complexity.

## Recommended Security Architecture (Model C)

### 1. Capability boundary (coarse-grained)

- Only mounted roots are visible.
- Use traversal-resistant host APIs and canonical path checks.
- Enforce read-only and deny symlink escapes.

### 2. Policy engine (fine-grained)

Evaluate each read attempt against:

- mount includes/excludes
- max file size limits
- binary/text restrictions
- sensitive path classifiers
- per-request rate/volume limits

Policy result:

- `ALLOW`
- `DENY`
- `REQUIRE_CONFIRMATION`

### 3. Host confirmation gateway

When policy returns `REQUIRE_CONFIRMATION`:

- Pause fs operation in host.
- Emit an approval request to the current user session with:
  - script name/version
  - requested path and operation
  - reason/risk category
  - proposed grant scope options
- On approval, store grant in request-scoped grant store.
- Resume operation and continue script lifecycle.

### 4. Grant model

Grant dimensions:

- scope: exact path, glob, mount
- operation: readText/readBytes/stat/glob
- TTL: short-lived by default
- max bytes

All grants are stored in server-side request state and never controlled by script code.

## Host Confirmation Flow Design

Two viable mechanics:

### Option 1: Synchronous wait inside script call

- fs bridge blocks awaiting host decision.
- Risk: timeout interactions and harder cancellation semantics.

### Option 2: Interrupt-and-resume flow (preferred)

- fs bridge raises structured interruption (`ErrFSApprovalRequired`).
- Server persists pending operation and returns a script update view that asks for approval.
- User approves/rejects through standard script event path.
- Server resumes original operation with recorded decision.

Reasoning: fits existing event-driven architecture and avoids long blocking waits in VM execution.

## Threat Model and Mitigations

### Threat: path traversal (`../../`)

Mitigation:

- mount-name + relative path API only
- canonicalization and root containment checks
- deny traversal sequences after normalization

### Threat: symlink escape

Mitigation:

- resolve and validate final target remains under mount root
- deny symlinks when policy disallows

### Threat: secret exfiltration

Mitigation:

- denylist patterns (`*.pem`, `.env`, ssh, cloud credentials)
- confirmation-required rules for sensitive classifiers
- audit logs and optional policy hard fail

### Threat: large/binary file abuse

Mitigation:

- max read bytes and max file size limits
- binary detection and optional denial
- per-request read budget (bytes and operations)

### Threat: prompt spam / confirmation fatigue

Mitigation:

- grant caching with TTL
- coalesced prompts for repeated similar accesses
- default deny after inactivity timeout

## Design Decisions

### Decision 1: Read-only v1

No write APIs in initial release.

### Decision 2: Hybrid policy model

Balance usability and security by mixing static bounds with selective confirmations.

### Decision 3: Host-enforced confirmations

JS cannot bypass confirmation checks because policy and grants live only in Go host.

### Decision 4: Audit first-class

Every access decision (allow/deny/prompt) is logged with request id, script describe info, mount, path, bytes, and decision source.

## Alternatives Considered

### Alternative A: No confirmation support, static mounts only

Rejected because sensitive reads need a user gate in many environments.

### Alternative B: User-implemented confirmation inside script

Rejected because malicious scripts could skip it; enforcement must be host-side.

### Alternative C: Separate sidecar process sandbox immediately

Deferred. Strong isolation but high operational complexity for v1.

## Implementation Plan

### Phase 0: Contract and policy baseline

1. Extend protobuf for mounts/policy/limits.
2. Add server-side schema validation and defaults.
3. Add deny-by-default policy when mounts are absent.

### Phase 1: Safe filesystem adapter

1. Build `sandbox_fs.go` with mount registry and traversal-safe path resolver.
2. Implement read-only methods with strict byte limits.
3. Add per-request read budget accounting.

### Phase 2: Policy engine

1. Implement policy evaluator (`ALLOW`, `DENY`, `REQUIRE_CONFIRMATION`).
2. Add path classifier hooks for sensitive files.
3. Add grant store abstraction.

### Phase 3: Host confirmation flow

1. Implement interruption error type for approval-required reads.
2. Add server lifecycle handling for pending fs approvals.
3. Add approval UI path and event schema wiring.
4. Resume blocked fs operation after approval decision.

### Phase 4: Observability and docs

1. Add audit logs and counters.
2. Add troubleshooting docs and examples.
3. Add smoke scripts for typical approval workflows.

## Testing Strategy

- Unit tests for path resolver and traversal prevention.
- Unit tests for policy decision matrix.
- Integration tests for approval-required read flow and resume semantics.
- Security tests for symlink/path escape attempts.
- Load tests for repeated reads and budget exhaustion.

## Documentation Plan

- API docs for `ctx.fs` methods and mount path syntax.
- Security docs with policy examples and operational guidance.
- Runbook for reviewing and approving read prompts safely.

## Open Questions

- Should grants survive process restarts for long-running requests?
- Should approval prompts include file content preview snippets, and under what limits?
- Should policy support organization-level defaults via config files?

## References

- `ttmp/2026/02/22/PC-01-ADD-JS-API--add-js-api-describe-extension/sources/local/plz-confirm-js.md`
- Go 1.24 `os.Root` traversal-resistant API guidance
