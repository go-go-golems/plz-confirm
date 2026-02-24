# Changelog

## 2026-02-23

Completed an exhaustive inspector-style code review of the full integration surface (architecture, APIs, runtime/state, widgets, tests, and backward compatibility), published as a new long-form report, and uploaded it to reMarkable.

### Related Files

- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/design-doc/04-inspector-review-plz-confirm-integration-quality-audit.md — New comprehensive quality audit report
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/reference/01-diary.md — Steps 22/23 record review process and publication workflow
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/index.md — Added inspector report to ticket key links

## 2026-02-23

Added a new long-form reusable integration playbook for future external software onboarding into go-go-os, then uploaded it to reMarkable as `PC-05 go-go-os Integration Playbook v1.pdf` and verified remote placement.

### Related Files

- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/design-doc/03-playbook-integrating-external-software-into-go-go-os.md — New reusable integration playbook deliverable
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/index.md — Added playbook to ticket entry-point links
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/reference/01-diary.md — Added Steps 20/21 covering authoring + reMarkable upload

## 2026-02-23

Uploaded the new postmortem bundle to reMarkable as `PC-05 Integration Postmortem v1.pdf` and verified it in `/ai/2026/02/23/PC-05-INTEGRATE-OS` alongside prior blueprint uploads.

### Related Files

- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/design-doc/02-postmortem-plz-confirm-integration-into-go-go-os.md — Source postmortem uploaded to reMarkable
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/reference/01-diary.md — Step 19 upload workflow and verification record

## 2026-02-23

Added a full-length technical postmortem document for the complete plz-confirm -> go-go-os integration, expanding the diary/changelog/commit evidence into an intern-ready retrospective with architecture diagrams, pseudocode, API references, incident root-cause analysis, and a reusable future integration playbook.

### Related Files

- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/design-doc/02-postmortem-plz-confirm-integration-into-go-go-os.md — New comprehensive postmortem deliverable
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/reference/01-diary.md — Added Step 18 documenting postmortem production process
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/index.md — Added postmortem link for ticket entry-point discoverability

## 2026-02-23

- Initial workspace created.
- Added design document `design-doc/01-integration-blueprint-plz-confirm-on-go-go-os-macos-windowing.md` with deep analysis of:
  - go-go-os desktop/windowing architecture and HyperCard runtime setup
  - plz-confirm backend/router/proto/script runtime and frontend semantics
  - package extraction requirements for embeddable plz-confirm server
  - `/confirm/*` route namespace strategy inside `go-inventory-chat`
  - websocket/event state machine and request lifecycle synchronization
  - widget-by-widget macOS UI mapping and component gap analysis
  - phased implementation roadmap, testing strategy, risks, and intern onboarding plan
- Added detailed diary `reference/01-diary.md` with chronological command-level investigation notes.
- Updated ticket index metadata and links for onboarding readability.
- Prepared and executed reMarkable publication workflow for the final bundle.

## 2026-02-23

Completed PC-05 documentation package: finalized 1000+ line integration blueprint, filled detailed chronological diary, related key source files, and uploaded verified reMarkable bundle to /ai/2026/02/23/PC-05-INTEGRATE-OS.

### Related Files

- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/design-doc/01-integration-blueprint-plz-confirm-on-go-go-os-macos-windowing.md — Primary deliverable completed
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/reference/01-diary.md — Detailed diary completed with upload evidence

## 2026-02-23

Applied package-first plan update and executed first two implementation tranches: engine core widgets (commit 48c2724) and new @hypercard/confirm-runtime package scaffold (commit 6e38a7d). Updated tasks/blueprint/diary accordingly.

### Related Files

- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/components/ConfirmRequestWindowHost.tsx — Request window host skeleton using engine widgets
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/runtime/createConfirmRuntime.ts — Package-first runtime wiring and host adapters
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/SchemaFormRenderer.tsx — Schema-driven form rendering helper
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/SelectableDataTable.tsx — New reusable selectable table widget
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/SelectableList.tsx — New reusable selectable list widget


## 2026-02-23

Uploaded refreshed reMarkable bundle after package-first implementation commits: PC-05-INTEGRATE-OS Integration Blueprint v2.pdf (same remote folder, versioned filename).

### Related Files

- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/reference/01-diary.md — Step 9 captures v2 publication details


## 2026-02-23

Added copious Storybook coverage for the six new engine widgets (commit 203181b) and validated taxonomy check success.

### Related Files

- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/FilePickerDropzone.stories.tsx — New upload/dropzone stories
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/ImageChoiceGrid.stories.tsx — New select/confirm/multi image stories
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/RequestActionBar.stories.tsx — New action bar behavior stories
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/SchemaFormRenderer.stories.tsx — New schema variant stories
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/SelectableDataTable.stories.tsx — New searchable/single/multi/table stories
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/SelectableList.stories.tsx — New interactive and edge-case stories
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/reference/01-diary.md — Step 10 documents story tranche and validation


## 2026-02-23

Completed inventory host integration tranche (commit af1a085): wired confirm-runtime reducer, runtime connection lifecycle, confirm request window delegation, confirm queue command/menu wiring, and /confirm dev proxy aliasing.

### Related Files

- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/apps/inventory/src/App.tsx — Confirm queue and request window host integration
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/apps/inventory/src/app/store.ts — confirmRuntime reducer wiring
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/tooling/vite/createHypercardViteConfig.ts — /confirm proxy and alias support

## 2026-02-23

Completed backend integration tranche:

1. Commit `56e40ec` in `plz-confirm`:
   - Added public embeddable backend package `pkg/backend` (server wrapper + prefix mounting).
   - Switched CLI `serve` command to consume the new public package.
   - Added package tests for direct and `/confirm`-prefixed request creation.
2. Commit `3e79c2a` in `go-go-os`:
   - Mounted plz-confirm backend under `/confirm/*` in `go-inventory-chat`.
   - Added integration tests for route coexistence (`/chat` + `/api/timeline` + `/confirm/api/requests`) and prefixed confirm websocket (`/confirm/ws` pending replay).

Validation executed:

- `go test ./pkg/backend ./cmd/plz-confirm -count=1` (pass)
- `go test ./... -count=1` in `plz-confirm` via pre-commit hook (pass)
- `go test ./cmd/hypercard-inventory-server -count=1` in `go-inventory-chat` with workspace resolution (pass)
- `go test ./... -count=1` in `go-inventory-chat` with workspace resolution (pass)

Known follow-up:

- `go-inventory-chat` currently resolves `github.com/go-go-golems/plz-confirm/pkg/backend` through local workspace composition (new package is not present in published `plz-confirm` v0.0.3 yet).

## 2026-02-23

Fixed confirm-runtime protocol mismatch that caused `Unsupported widget type: undefined` in inventory confirm windows (commit `2ffac96` in `go-go-os`).

What changed:

1. Added proto adapter layer to normalize backend protojson `UIRequest`/WS events into frontend runtime shape:
   - `packages/confirm-runtime/src/proto/confirmProtoAdapter.ts`
2. Wired API + WS paths to use adapter normalization:
   - `packages/confirm-runtime/src/api/confirmApiClient.ts`
   - `packages/confirm-runtime/src/ws/confirmWsManager.ts`
3. Switched response submission to emit proper proto oneof payloads (`confirmOutput`, `selectOutput`, etc.) instead of generic `{output: ...}`.
4. Updated inventory host integration call-site to pass request context to submit encoder:
   - `apps/inventory/src/App.tsx`
5. Added adapter unit tests:
   - `packages/confirm-runtime/src/proto/confirmProtoAdapter.test.ts`

Validation executed:

- `npm exec vitest run packages/confirm-runtime/src/proto/confirmProtoAdapter.test.ts` (pass)

Known environment note:

- Workspace `tsc -b` remains blocked by pre-existing `packages/engine` typing/tooling issues unrelated to this fix.

## 2026-02-23

Continued integration execution with three outcomes:

1. Repaired `plz-confirm` CLI compatibility with embedded `/confirm` backend:
   - Fixed Glazed tag decode regression in all request commands (`glazed.parameter` -> `glazed`).
   - Fixed `plz-confirm ws` path-prefix handling so `--base-url http://host/confirm` resolves to `/confirm/ws`.
2. Archived all ad-hoc validation scripts retroactively into ticket-owned scripts directory:
   - `scripts/repro_cli_base_url_decode_regression.sh`
   - `scripts/ws_prefix_connect_smoke.sh`
   - `scripts/e2e_cli_confirm_roundtrip.sh`
   - `scripts/debug_ws_confirm_dual.sh`
3. Resumed C4/C3 runtime parity work in `@hypercard/confirm-runtime`:
   - Script metadata mapping (`stepId`, `title`, `description`) + unknown script widget preservation.
   - Script submit/back event semantics in host (`type: submit/back`, with `stepId`).
   - Script sections rendering with display blocks and one-interactive-section validation.
   - Upload widget host support via `FilePickerDropzone`.

Validation snapshots:

- `go test ./internal/cli ./internal/client ./cmd/plz-confirm` (pass)
- `npm exec vitest run packages/confirm-runtime/src/proto/confirmProtoAdapter.test.ts` (pass)
- Ticket script E2E roundtrip (`scripts/e2e_cli_confirm_roundtrip.sh`) (pass)

### Related Files

- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/cmd/plz-confirm/ws.go — Embedded prefix websocket URL fix
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/internal/cli/confirm.go — Glazed decode tag fix
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/internal/cli/select.go — Glazed decode tag fix
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/internal/cli/form.go — Glazed decode tag fix
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/internal/cli/table.go — Glazed decode tag fix
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/internal/cli/upload.go — Glazed decode tag fix
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/internal/cli/image.go — Glazed decode tag fix
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/scripts/e2e_cli_confirm_roundtrip.sh — Reproducible CLI/backend E2E harness
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/components/ConfirmRequestWindowHost.tsx — Script-mode parity and upload host support
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/proto/confirmProtoAdapter.ts — Script metadata + image bool mapping
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/proto/confirmProtoAdapter.test.ts — Coverage for new adapter behavior
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/types.ts — Script view type extensions

## 2026-02-23

Advanced core-first widget plan by adding two new engine primitives and wiring script usage:

- Added `RatingPicker` and `GridBoard` to `packages/engine`.
- Added Storybook stories for both components.
- Exported both from engine widget barrel.
- Updated `ConfirmRequestWindowHost` script composition to render and submit `rating` + `grid` flows.

Validation snapshots:

- `npm run storybook:check` (pass)
- `npm run test -w packages/engine` (pass)
- `npm exec vitest run packages/confirm-runtime/src/proto/confirmProtoAdapter.test.ts` (pass)

### Related Files

- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/RatingPicker.tsx — New core rating widget
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/GridBoard.tsx — New core grid widget
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/RatingPicker.stories.tsx — Rating visual/state story matrix
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/GridBoard.stories.tsx — Grid visual/state story matrix
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/components/ConfirmRequestWindowHost.tsx — Script integration path for rating/grid

## 2026-02-23

Closed stale-queue + missing timestamp regression reported during manual confirm testing.

1. `go-go-os` commit `686006b`:
   - Confirm runtime now maps request `status/completedAt` and immediately removes non-pending requests from active queue state.
   - Inventory host now handles `409 request already completed` by refetching request and reconciling local state/window lifecycle.
   - Proto adapter now emits default timestamps for confirm/image outputs when omitted by UI payload.
2. `plz-confirm` commit `850b79c`:
   - Backend `handleSubmitResponse` now auto-populates missing `confirmOutput.timestamp` and `imageOutput.timestamp` before completion write.
   - Added server regression tests for missing timestamp population.

Validation snapshots:

- `npm exec vitest run packages/confirm-runtime/src/proto/confirmProtoAdapter.test.ts` (pass)
- `go test ./internal/server -count=1` (pass)
- pre-commit hooks in `plz-confirm`: `golangci-lint run -v` + `go test ./... -count=1` (pass)

### Related Files

- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/apps/inventory/src/App.tsx — 409 reconciliation on submit and stale window cleanup
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/api/confirmApiClient.ts — typed API error with status/body for 409 handling
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/proto/confirmProtoAdapter.ts — response timestamp defaults + request status mapping
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/proto/confirmProtoAdapter.test.ts — timestamp/status regression assertions
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/state/confirmRuntimeSlice.ts — completed request upsert eviction
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/state/selectors.ts — root typing compatibility widening
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/types.ts — status/completedAt shape additions
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/internal/server/server.go — backend timestamp auto-fill guardrail
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/internal/server/response_timestamp_test.go — server regression tests for timestamp fill
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/reference/01-diary.md — Step 16 implementation diary

## 2026-02-23

Added composite confirm-runtime script-section stories for handoff-complete coverage (commit `e1b2023` in `go-go-os`) and updated both PC-05 + PC-06 ticket docs accordingly.

Composite scenarios added:

1. `DisplayAndConfirmSection`
2. `DisplayAndSelectSection`
3. `DisplayAndFormSection`
4. `DisplayAndTableSection`
5. `DisplayAndUploadSection`
6. `DisplayAndImageSection`
7. `BackAndProgressRating`
8. `TwoStepConfirmThenRating`
9. `InvalidSectionsContract`

Validation snapshot:

- `npm run storybook:check` (pass, 63 stories)

### Related Files

- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/apps/inventory/src/features/confirm/stories/ConfirmRuntimeComposite.stories.tsx — New composite script-section story suite for request-window composition
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/tasks.md — Marked C4 manual lifecycle validation done
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/reference/01-diary.md — Step 17 records composite story tranche
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-06-UI-CONSISTENCY-HANDOFF--confirm-widget-visual-consistency-handoff/design-doc/01-designer-handoff-confirm-widget-inventory-stories-and-consistency-scenarios.md — Composite stories added to handoff inventory and workflow

## 2026-02-23

Added long-form deep-dive Q&A document answering write pump, duplication/deprecation status, 409 reconciliation host duplication, and confirmProtoAdapter architecture; updated inspector review with reconnect-policy decision (host adapter injection).

### Related Files

- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/design-doc/04-inspector-review-plz-confirm-integration-quality-audit.md — Decision update for reconnect policy injection via host adapters
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/design-doc/05-deep-dive-q-a-write-pump-duplication-409-and-confirmprotoadapter.md — New multi-question deep-dive response document
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/index.md — Added deep-dive Q&A to key links


## 2026-02-24

Added a dedicated click-through seeding harness that creates all core widget requests plus a multi-step JS `script` request, so manual UI verification can exercise both normal widget rendering and backend-driven script view/update transitions in one run.

### Related Files

- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/scripts/seed_clickthrough_all_widgets_with_js_script.sh — New one-command queue seeder for confirm/select/form/table/upload/image/script click-through testing
