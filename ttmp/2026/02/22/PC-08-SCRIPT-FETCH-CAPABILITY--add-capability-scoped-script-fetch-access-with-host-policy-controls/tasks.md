# Tasks

## TODO

- [ ] Finalize the script fetch capability contract in `ScriptInput` (policy, limits, credentials) and document deny-by-default behavior when policy is absent.
- [ ] Extend protobuf schema for fetch policy/limits/credential references and regenerate Go and TypeScript artifacts.
- [ ] Implement a canonical request builder that normalizes scheme/host/port/path/query and strips unsupported option fields before policy evaluation.
- [ ] Implement a capability-aware HTTP adapter for scripts (`ctx.fetch`) with strict allowlists for methods, headers, and response modes.
- [ ] Enforce hard transport limits: timeout cap, redirect hop cap, response byte cap, and per-request call-count budget.
- [ ] Add SSRF protections: block private/link-local/loopback ranges, cloud metadata endpoints, and disallowed ports/schemes.
- [ ] Implement DNS-based safeguards against rebinding and ensure redirect targets are revalidated against policy and IP restrictions.
- [ ] Build a policy evaluator returning `ALLOW`, `DENY`, or `REQUIRE_CONFIRMATION` with reason codes for UI/audit surfacing.
- [ ] Implement host-side confirmation flow for gated fetch requests, including approval scopes (`once`, `step`, `ttl`) and script resume semantics.
- [ ] Add request-scoped grant store with deterministic matching, expiration handling, and invalidation on script/version/request boundary changes.
- [ ] Implement credential reference resolution so scripts request named creds while host injects scoped auth headers without exposing raw secrets to JS.
- [ ] Add response/header sanitization and redaction rules to prevent leaking sensitive headers/tokens into script logs or UI outputs.
- [ ] Implement detailed audit logging for each fetch attempt (normalized target, policy result, grant source, status, bytes, duration, reason).
- [ ] Add runtime and server integration tests for allowed requests, denied requests, confirmation-required flows, and grant reuse/expiration behavior.
- [ ] Add adversarial tests for SSRF bypass attempts (encoded hosts, redirects, DNS tricks, metadata aliases) and resource exhaustion patterns.
- [ ] Add ticket-local scripts under `scripts/` for realistic fetch-backed UI flows and for explicit denied/confirmation-required scenarios.
- [ ] Update JS API docs and operator/developer docs with fetch contract, risk model, defaults, and operational override controls.
- [ ] Define rollout controls and incident handling: global fetch kill-switch, per-request disable flag, and safe fallback behavior on policy/credential failures.
