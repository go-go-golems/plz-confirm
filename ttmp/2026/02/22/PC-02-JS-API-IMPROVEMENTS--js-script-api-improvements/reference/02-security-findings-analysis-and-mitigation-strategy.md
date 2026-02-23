---
Title: Security Findings Analysis and Mitigation Strategy
Ticket: PC-02-JS-API-IMPROVEMENTS
Status: active
Topics:
    - backend
    - frontend
    - api
    - javascript
    - security
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: internal/server/script.go
      Note: int->int32 conversion path for ScriptProgress mapping
    - Path: internal/scriptengine/engine.go
      Note: deterministic PRNG helper exposed to scripts via ctx.random and ctx.randomInt
    - Path: internal/client/client.go
      Note: outbound HTTP calls based on configured BaseURL
    - Path: internal/server/images.go
      Note: image file path creation/removal in image store
    - Path: internal/server/server.go
      Note: image serving path open + protojson response writer + request logging
    - Path: internal/server/ws.go
      Note: websocket session logging for user-provided sessionId
ExternalSources: []
Summary: Deep triage of static-analysis security findings with project-specific exploitability assessment and recommended mitigations.
LastUpdated: 2026-02-23T09:52:00-05:00
WhatFor: Rapidly deciding which findings require code changes versus documented suppressions.
WhenToUse: When reviewing gosec/golangci-lint security findings for server, script runtime, and client transport code.
---

# Security Findings Analysis and Mitigation Strategy

## Goal

Provide a precise, implementation-oriented triage of the reported static-analysis findings, including:

- what each finding means in this codebase,
- whether it is exploitable, contextual, or likely false-positive,
- what mitigation strategies exist,
- which mitigation strategy is the best fit for `plz-confirm` now.

## Context

The findings span mixed trust boundaries:

1. Script runtime input (`internal/server/script.go`) that can be influenced by script authors and event payloads.
2. HTTP server and websocket entry points (`internal/server/server.go`, `internal/server/ws.go`) exposed to clients.
3. Local image file storage and retrieval (`internal/server/images.go`).
4. HTTP client transport (`internal/client/client.go`) where destination is configured by `BaseURL`.
5. Script helper runtime (`internal/scriptengine/engine.go`) with deterministic pseudo-random helpers.

The same static rule class does not imply the same practical risk at every callsite. Some alerts are true positives, some are contextual, and some are scanner noise around safe-by-construction flows.

## Quick Reference

### Finding triage matrix

| Rule | File/Line(s) | CWE Theme | Practical Risk Here | Triage | Recommended Action |
|---|---|---|---|---|---|
| G115 | `internal/server/script.go:691-694` | Integer overflow/narrowing | Possible if `progress.current/total` exceed int32 range | Real bug class | Add explicit int32 bounds checks before cast; optionally parse into int64 first |
| G404 | `internal/scriptengine/engine.go:334` | Weak RNG for security use | `math/rand` used for script helper randomness, not secrets | Contextual | Keep deterministic PRNG; add explicit non-crypto comment and `#nosec` with rationale |
| G704 | `internal/client/client.go:141,272,304` | SSRF | Depends on who controls `BaseURL`; library can call arbitrary host | Contextual-to-real | Validate BaseURL scheme/host policy at client construction; add optional private-network restrictions |
| G703 | `internal/server/images.go:74,84,120`, `internal/server/server.go:637` | Path traversal | Path is generated from UUID and internal map; user path injection is not direct | Mostly false-positive but hardenable | Replace stored path trust with derived safe path from ID and containment checks |
| G705 | `internal/server/server.go:681` | XSS | Writes JSON with content-type `application/json`; no HTML context | Likely false-positive | Keep behavior; optionally add `X-Content-Type-Options: nosniff` and document suppression |
| G706 | `internal/server/ws.go:132`, `internal/server/server.go:294,450` | Log injection | `sessionId` can be user-controlled; IDs are generated server-side | Low, partially real | Sanitize/quote logged user-controlled values; prefer structured logger fields |

### Priority recommendation

1. **P1**: G115 overflow checks in script progress mapping.
2. **P1**: G704 client transport guardrails (`BaseURL` validation + policy knobs).
3. **P2**: G706 log sanitization for user-controlled `sessionId`.
4. **P2**: G703 defensive path hardening (invariant checks; reduce trust in stored path).
5. **P3**: G404/G705 suppressions with explicit rationale and tests confirming intended behavior.

## Detailed Analysis and Mitigation Options

### 1) G115 int -> int32 narrowing in `mapToScriptProgress`

#### What it means

`current` and `total` are currently parsed as `int` and cast to protobuf `int32` fields:

- `Current: int32(current)`
- `Total: int32(total)`

If attacker-controlled values exceed `math.MaxInt32`, narrowing can wrap or truncate and produce incorrect progress state.

#### Why it matters here

`view.progress` is script-provided data. Script authors are not always trusted equally, and malformed script outputs should fail closed.

#### Mitigation strategies

1. Add explicit bounds checks (`0 <= current,total <= math.MaxInt32`) before cast.
2. Parse numerics as `int64` first, then downcast only after range validation.
3. Change proto fields to `int64` (larger refactor; likely unnecessary for UI progress bars).

#### Best fit for this codebase

Use strategy 1 + 2 together:

- introduce `numberAsInt64` helper,
- validate range and semantic constraints (`total > 0`, `current >= 0`, `current <= total`, `<= MaxInt32`),
- cast only after checks.

This is low-risk, easy to test, and preserves the existing protobuf contract.

### 2) G404 weak RNG in script context helpers

#### What it means

`rand.New(rand.NewSource(seed))` is flagged because `math/rand` is not cryptographically secure.

#### Why this is mostly contextual here

The helper is used for script ergonomics (`ctx.random`, `ctx.randomInt`) and deterministic seeded behavior. It is not used for token generation, session identifiers, password resets, or cryptographic material.

#### Mitigation strategies

1. Replace with `crypto/rand`-backed helper APIs.
2. Keep `math/rand` and clearly declare non-crypto intent.
3. Expose two helpers: deterministic pseudo-random and secure random bytes.

#### Best fit for this codebase

Use strategy 2 now:

- keep deterministic `math/rand` behavior (needed for reproducible script runs),
- add code comment and linter suppression (`#nosec G404`) at the callsite with clear rationale,
- add docs warning: random helpers are not security primitives.

If secure randomness is later needed, add a separate explicit API (strategy 3), not a silent behavior change.

### 3) G704 SSRF warnings in `internal/client/client.go`

#### What it means

The scanner sees `HTTPClient.Do(req)` and tainted URL flow from configuration (`BaseURL`).

#### Why it can matter here

If untrusted input can set `BaseURL`, the client can be used as a network pivot to internal services.

#### Current context

In normal CLI usage, operator controls `BaseURL`. This reduces risk, but library consumers may pass user-derived values. Static tools cannot distinguish that trust model.

#### Mitigation strategies

1. Validate `BaseURL` scheme (`http`/`https` only) and host presence at client creation.
2. Add policy options:
- allow/deny private networks,
- allowlist hostnames,
- optional strict loopback-only mode.
3. Enforce safe transport dialer that rejects private/link-local/metadata CIDRs.
4. Document trust assumptions and suppress warnings only.

#### Best fit for this codebase

Use 1 + 2 now, 3 later if needed:

- immediate validation in `NewClient` (or equivalent constructor path),
- default safe profile for CLI (for example: allow loopback/local dev, require explicit opt-in for external/private targets),
- explicit `ClientOptions` knobs to avoid breaking existing automation.

Suppressions alone are too weak if this package is reused in broader contexts.

### 4) G703 path traversal warnings in image store/server

#### What it means

The scanner flags filesystem calls (`OpenFile`, `Remove`, `Open`) as potential path traversal sinks.

#### Current code reality

- paths are generated as `filepath.Join(storeDir, uuid)` in `Put`,
- IDs are generated server-side (`uuid.NewString()`),
- retrieval/deletion is via in-memory image map, not direct filesystem path from request.

That means direct traversal from request path is currently unlikely.

#### Residual risk

If internal state becomes corrupted or if future code starts accepting external IDs/path overrides, current trust-in-stored-path model could become fragile.

#### Mitigation strategies

1. Keep storing full `img.Path` and suppress scanner findings.
2. Store only `ID`; reconstruct path from `(dir, id)` at use time with strict ID validation and root containment checks.
3. Use hardened file APIs rooted at directory handles (`openat`-style pattern / root-scoped FS).

#### Best fit for this codebase

Use strategy 2 now:

- treat `id` as canonical identity,
- validate `id` format before path construction,
- compute `resolved := filepath.Clean(filepath.Join(dir, id))` and ensure it remains under `dir`,
- avoid trusting mutable/stored path strings.

This both hardens behavior and typically quiets taint-based path alerts.

### 5) G705 XSS warning on `w.Write(b)` in JSON writer

#### What it means

Taint analysis flags writing potentially user-influenced bytes to response.

#### Why this is usually a false positive here

The response is explicitly JSON (`Content-Type: application/json`) and bytes come from protobuf JSON marshalling, not template rendering into HTML/JS context.

#### Mitigation strategies

1. Keep current behavior, suppress finding with rationale.
2. Add `X-Content-Type-Options: nosniff` and maintain strict JSON-only endpoints.
3. HTML-escape JSON output (usually unnecessary and can break expectations).

#### Best fit for this codebase

Use 1 + 2:

- keep protojson output unchanged,
- set `nosniff` header globally or in JSON helpers,
- annotate suppression rationale where scanner remains noisy.

### 6) G706 log injection warnings

#### What it means

User-controlled strings in logs can inject line breaks/control chars and forge log records.

#### Practical risk here

- `req.Id` is generated UUID (safe).
- `req.Type` is enum (safe).
- `sessionId` from query param is user-controlled and should be sanitized.

#### Mitigation strategies

1. Quote values with `%q` so control characters are escaped.
2. Strip non-printable/control chars before logging.
3. Move to structured logger with field encoding (zerolog/slog).

#### Best fit for this codebase

Use 1 immediately, optionally 2 for stricter policy:

- update session logging to `log.Printf("... sessionId=%q", sessionID)`.
- keep existing request-id logs as-is or standardize all with `%q` for consistency.

## Recommended Implementation Plan (Project-Specific)

### Phase 1: High-signal code fixes

1. Add progress numeric range checks and safe conversion in `internal/server/script.go`.
2. Add client `BaseURL` validation and documented policy options in `internal/client/client.go`.
3. Add regression tests:
- progress values above `MaxInt32` rejected,
- invalid `BaseURL` rejected,
- optional policy test for blocked private/metadata targets (if enabled).

### Phase 2: Defensive hardening

1. Refactor image store to derive filesystem paths from validated IDs, not stored path strings.
2. Add path containment tests for image open/delete paths.
3. Sanitize/quote user-controlled log fields (`sessionId`).

### Phase 3: Documented suppressions and policy posture

1. Add `#nosec` comments with rationale for intentional non-crypto PRNG usage.
2. Add `#nosec`/rule exclusions where XSS scanner flags JSON-write sinks incorrectly, with justification in code comment and security doc.
3. Document trust boundaries in `pkg/doc/js-script-development.md`.

## Usage Examples

### Example: overflow-safe progress conversion policy

```go
if current < 0 || total <= 0 || current > total {
    return nil, fmt.Errorf("invalid progress bounds")
}
if current > math.MaxInt32 || total > math.MaxInt32 {
    return nil, fmt.Errorf("progress exceeds supported range")
}
progress := &v1.ScriptProgress{Current: int32(current), Total: int32(total)}
```

### Example: safe logging for user-controlled value

```go
log.Printf("[WS] client connected (sessionId=%q)", sessionID)
```

### Example: client BaseURL guardrails

```go
u, err := url.Parse(baseURL)
if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
    return nil, errors.New("invalid base URL")
}
```

## Decision Summary

- Treat **G115** as a real bug class and fix in code.
- Treat **G704** as contextual but important; add transport guardrails rather than pure suppression.
- Treat **G706** as low-severity but real for `sessionId`; sanitize/quote now.
- Treat **G703/G705** as primarily taint-analysis noise in current design, but still harden path invariants and headers for defense-in-depth.
- Treat **G404** as intentional non-crypto PRNG usage; document/suppress with rationale unless security randomness is introduced later.

## Related

- `ttmp/2026/02/22/PC-02-JS-API-IMPROVEMENTS--js-script-api-improvements/design-doc/01-js-script-api-improvement-proposals.md`
- `ttmp/2026/02/22/PC-02-JS-API-IMPROVEMENTS--js-script-api-improvements/reference/01-diary.md`
- `internal/server/script.go`
- `internal/scriptengine/engine.go`
- `internal/client/client.go`
- `internal/server/images.go`
- `internal/server/server.go`
- `internal/server/ws.go`
