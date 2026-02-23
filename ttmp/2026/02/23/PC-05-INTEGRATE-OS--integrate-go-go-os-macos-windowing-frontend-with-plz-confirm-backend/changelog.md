# Changelog

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

