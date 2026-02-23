# Tasks

## TODO

- [ ] Finalize `action_id` UX/API contract for script views: define action object shape (`id`, `label`, optional variant/confirm), ordering rules, and backward-compatibility guarantees.
- [ ] Extend protobuf schema to represent script-declared actions in `ScriptView`, regenerate Go/TypeScript codegen outputs, and verify wire compatibility.
- [ ] Implement server-side mapping/validation in `internal/server/script.go` for action arrays (required fields, duplicate IDs, unsupported variants).
- [ ] Add renderer support in `agent-ui-system/client/src/components/WidgetRenderer.tsx` to display declared actions for script views in both single-widget and section-based modes.
- [ ] Wire action button clicks to `submitScriptEvent` with payload `{ type: "action", stepId, actionId }` and preserve existing submit/back behavior.
- [ ] Add frontend tests for action rendering, disabled/loading behavior, and emitted event payloads.
- [ ] Add server integration tests to verify action events are accepted, persisted through request updates, and complete correctly when scripts return terminal results.
- [ ] Add runtime/engine tests showing `ctx.branch(...)` routes by `actionId` before `approved/rejected` and `event.type` fallbacks.
- [ ] Create ticket-local manual scripts under `scripts/` demonstrating multi-action flows (e.g., retry/skip/escalate) and validate end-to-end in browser.
- [ ] Update docs (`pkg/doc/js-script-api.md`, `pkg/doc/js-script-development.md`) with action contract, examples, and troubleshooting guidance.
- [ ] Add changelog notes and migration notes clarifying that existing scripts remain valid and action buttons are additive.
