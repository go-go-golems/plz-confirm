# Tasks

## TODO

- [ ] Finalize node-tree API contract and compatibility policy (legacy widget mode, node mode, mixed mode precedence, and feature-flag gating).
- [ ] Extend protobuf schema for `ScriptNode` and node-root fields on `ScriptView`; regenerate Go and TypeScript artifacts.
- [ ] Define and implement backend validator for node trees: required fields, allowed node types, path-aware error messages, and type-specific prop validation.
- [ ] Implement hard safety limits for node payloads (max depth, max node count, max serialized bytes) and return deterministic `400` errors on violations.
- [ ] Add runtime `ctx.ui` DSL helpers in the script engine prelude that emit schema-compliant node objects, while allowing raw object returns for advanced scripts.
- [ ] Refactor frontend architecture by extracting a `ScriptNodeRenderer` from `WidgetRenderer` and preserving request lifecycle orchestration in a thin wrapper.
- [ ] Implement node-type registry and adapters for layout/content nodes (`page`, `stack`, `row`, `panel`, `markdown`, `text`, `callout`, `code`, `diff`).
- [ ] Implement interactive node adapters that wrap existing dialogs (`confirm`, `select`, `form`, `table`, `upload`, `image`, `grid`, `rating`) without behavior regression.
- [ ] Implement unified event dispatch from node interactions (`submit`, `action`, `back`) with correct `stepId` and optional `actionId` propagation.
- [ ] Define keying/state-preservation strategy so rerenders do not unexpectedly reset in-progress user edits for stable node keys.
- [ ] Add frontend tests for recursive rendering, unknown node fallback, interaction dispatch, and legacy fallback behavior.
- [ ] Add server tests for node validation failures, compatibility behavior, and node-mode lifecycle transitions.
- [ ] Add manual smoke scripts under ticket `scripts/` that cover nested layouts, mixed content+input, and action-rich flows.
- [ ] Add observability counters for node-mode usage, render/validation failures, and payload-limit rejections.
- [ ] Update docs (`pkg/doc/js-script-api.md`, `pkg/doc/js-script-development.md`) with node schema reference, migration examples, and troubleshooting.
- [ ] Produce a phased rollout plan (internal-only, opt-in tenants/sessions, default-on) with explicit rollback switches.
