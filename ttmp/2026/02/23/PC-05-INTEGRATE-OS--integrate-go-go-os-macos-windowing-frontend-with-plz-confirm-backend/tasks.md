# Tasks

## Completed Foundation

- [x] Create ticket `PC-05-INTEGRATE-OS` with design-doc and diary documents
- [x] Analyze go-go-os desktop windowing + HyperCard runtime plumbing
- [x] Analyze plz-confirm backend/router/store/script engine/proto/frontend semantics
- [x] Author intern-ready architecture blueprint and upload to reMarkable
- [x] Update plan to package-first frontend (`packages/confirm-runtime`) and core-first engine widgets

## Implementation Backlog (Package-First)

### A. Core Engine Widgets (generic reuse)

- [x] A1. Add `SelectableList` to `@hypercard/engine` (single/multi/search/rich rows/keyboard)
- [x] A2. Add `SelectableDataTable` to `@hypercard/engine` (single/multi/filter/select helpers)
- [x] A3. Add `SchemaFormRenderer` to `@hypercard/engine` (schema -> `FieldConfig[]`)
- [x] A4. Add `FilePickerDropzone` to `@hypercard/engine` (picker + drag/drop + constraints)
- [x] A5. Add `ImageChoiceGrid` to `@hypercard/engine` (select/confirm/multi modes)
- [x] A6. Add `RequestActionBar` to `@hypercard/engine` (actions + optional comment + busy state)
- [x] A7. Export all new widgets from `packages/engine/src/components/widgets/index.ts` and package barrel
- [x] A8. Add/adjust unit coverage for core logic where practical
- [x] A9. Commit core widget tranche (`git commit` checkpoint)

### B. New `packages/confirm-runtime` (plz-confirm specific)

- [x] B1. Create package scaffold (`package.json`, `tsconfig.json`, `src/index.ts`)
- [x] B2. Add runtime types for request/event/session/api contracts
- [x] B3. Add store slice + selectors for request lifecycle (`new_request`, `request_updated`, `request_completed`)
- [x] B4. Add host adapter interfaces (base URL/session/window open/telemetry)
- [x] B5. Add API client + WS manager skeletons using injected adapters
- [x] B6. Add widget host skeleton that composes engine widgets (without app-specific code)
- [x] B7. Add package exports + root workspace wiring (`tsconfig` refs, build scripts as needed)
- [x] B8. Commit confirm-runtime tranche (`git commit` checkpoint)

### C. Inventory Host Integration (thin adapter)

- [x] C1. Wire confirm-runtime reducer/services into `apps/inventory` store
- [x] C2. Wire `renderAppWindow` delegation for `confirm-request:<id>`
- [x] C3. Add desktop command/menu hooks for confirm queue (minimal)
- [x] C4. Validate manual flow for window open/close lifecycle
- [x] C5. Commit inventory integration tranche (`git commit` checkpoint)

### D. Backend Integration (existing planned phases)

- [x] D1. Extract embeddable public plz-confirm package(s) from `internal/*`
- [x] D2. Mount `/confirm/*` routes in `go-inventory-chat`
- [x] D3. Add route coexistence and ws prefix tests
- [x] D4. Commit backend integration tranche (`git commit` checkpoint)

### E. Verification + Documentation

- [ ] E1. Run typecheck/tests for touched packages and fix breakages (backend go tests now passing; frontend/workspace TS verification still pending)
- [x] E2. Update design doc with implemented file references and behavior notes
- [x] E3. Keep diary updated per tranche with exact commands/errors/outcomes
- [x] E4. Update changelog and task statuses after each commit checkpoint
