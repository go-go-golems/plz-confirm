# Tasks

## TODO

- [ ] Finalize FS capability contract in `ScriptInput` (mount declarations, limits, policy hints) and document deny-by-default semantics.
- [ ] Extend protobuf schema for mount/policy/limit messages and regenerate Go and TypeScript artifacts.
- [ ] Implement traversal-resistant mount resolver using canonical path checks and root containment guarantees; block absolute and malformed paths.
- [ ] Implement read-only fs adapter (`readText`, `readBytes`, `stat`, `glob`, `exists`) with strict per-call and per-request byte/read budgets.
- [ ] Add sensitive-path classifier rules (default deny/confirm set for credential-like files) with configurable policy hooks.
- [ ] Implement policy engine that returns `ALLOW`, `DENY`, or `REQUIRE_CONFIRMATION` for each fs operation.
- [ ] Design and implement request-scoped grant store (scope, operation, TTL, byte cap, source) with deterministic evaluation order.
- [ ] Implement host-side confirmation flow for `REQUIRE_CONFIRMATION` decisions (including resume semantics for interrupted script execution).
- [ ] Add UI path for fs approval prompts that clearly shows script identity, requested path/op, risk reason, and grant options.
- [ ] Implement audit logging for all fs decisions and grant events with request/script identifiers and policy rationale.
- [ ] Add runtime and server tests for traversal attacks, symlink escapes, sensitive file gating, policy cache behavior, and timeout/cancel interactions.
- [ ] Add red-team style negative tests for prompt spam, repeated high-volume reads, and policy bypass attempts.
- [ ] Add ticket-local scripts under `scripts/` for realistic file-reading UI flows (diff preview, log triage, config summary).
- [ ] Update docs with security model guidance, operational defaults, and examples for confirmation-required paths.
- [ ] Define rollout and incident controls: kill-switches, per-session disable, and safe fallback when policy/grant state is unavailable.
