# Tasks

## Execution Checklist (Current Turn)

- [x] Expand `tasks.md` into a detailed phased checklist for the JS describe extension
- [x] Add a new diary step documenting the task-management workflow and rationale
- [x] Refresh ticket file relationships with `docmgr doc relate` (index + focused docs)
- [x] Run ticket hygiene checks (`docmgr doctor` + targeted frontmatter validation) and record outcomes
- [x] Update changelog with this turn's tasking/checkoff progress
- [x] Commit each completed work block incrementally

## Implementation Backlog (Intern Start Checklist)

- [x] Phase 0: Confirm scope and naming decisions
- [x] Decide whether canonical widget type name is `script` or `flow`
- [x] Decide whether script runtime state is persisted in `UIRequest` or server-only session memory
- [x] Finalize `describe` contract versioning and compatibility behavior

- [x] Phase 1: Define JS extension contract
- [x] Specify required `describe()` return shape (`name`, `version`, supported handlers)
- [x] Specify `init(input, context)` return envelope and error behavior
- [x] Specify `view(state, context)` return schema for renderable widget payloads
- [x] Specify `update(state, event, context)` transition semantics and idempotency expectations
- [x] Specify cancellation/timeout contract exposed to JS runtime

- [x] Phase 2: Protobuf and wire protocol updates
- [x] Add/adjust widget enums and oneofs in `proto/plz_confirm/v1/request.proto`
- [x] Add script-specific input/output/view/state message types in `proto/plz_confirm/v1/widgets.proto`
- [x] Add websocket event(s) for incremental request updates in `proto/plz_confirm/v1/ws.proto` (or active event proto)
- [ ] Run code generation and verify regenerated Go and TS outputs

- [x] Phase 3: Server lifecycle integration
- [x] Add request preflight validation for script requests in `internal/server/server.go`
- [x] Implement runtime session initialization path (`describe` then `init`)
- [x] Implement event loop path for `update` and projection path for `view`
- [x] Implement request-update broadcast path over websocket
- [x] Ensure final submit path remains compatible with existing `wait`/`response` semantics
- [ ] Add robust error mapping (validation error vs runtime fault vs timeout)

- [x] Phase 4: Store and persistence behavior
- [x] Decide persisted fields for script progression (`state`, `view`, metadata pointers)
- [x] Update store create/update clone behavior in `internal/store/store.go`
- [ ] Ensure history queries remain stable with partially progressed script requests
- [x] Ensure backward compatibility for non-script request types

- [x] Phase 5: Frontend runtime and rendering
- [x] Add script widget rendering branch in `agent-ui-system/client/src/components/WidgetRenderer.tsx`
- [x] Add reducer handlers for request update events in `agent-ui-system/client/src/store/store.ts`
- [x] Add websocket client handling for new event type in `agent-ui-system/client/src/services/websocket.ts`
- [x] Normalize incoming proto payloads for script view/state in `agent-ui-system/client/src/proto/normalize.ts`
- [x] Add UI affordances for runtime errors and recoverable retry states

- [x] Phase 6: CLI/client integration
- [x] Extend CLI request creation path for script input in `internal/client/client.go`
- [x] Add/adjust command wiring in `cmd/plz-confirm/main.go` and relevant `internal/cli/*.go`
- [x] Ensure session scoping and timeout behavior are preserved for script flows
- [x] Ensure non-script commands remain unchanged

- [ ] Phase 7: Runtime ownership and go-go-goja alignment
- [x] Choose integration strategy: direct `go-go-goja` engine usage vs thin local wrapper
- [x] Implement bounded execution/interrupt handling consistent with go-go-goja patterns
- [ ] Validate module exposure and host bridge constraints for script API surface
- [ ] Add runtime lifecycle cleanup hooks to avoid leaked goroutines/resources

- [ ] Phase 8: Testing and validation
- [x] Add unit tests for contract validation and runtime envelope decoding
- [x] Add server tests for create/update/finalize lifecycle for script requests
- [x] Add websocket tests for incremental update events and ordering guarantees
- [ ] Add frontend tests for reducer + renderer behavior for script views
- [x] Add smoke/e2e script that exercises end-to-end script progression
- [x] Run full repo checks (`go test`, frontend checks) and capture known environment limitations

- [ ] Phase 9: Docs and rollout
- [ ] Update user/developer docs with JS extension contract and examples
- [ ] Add troubleshooting notes for runtime errors, timeouts, and interrupt handling
- [ ] Define rollout strategy (feature flag or guarded command path)
- [ ] Define post-rollout observability checks and regression watchpoints
