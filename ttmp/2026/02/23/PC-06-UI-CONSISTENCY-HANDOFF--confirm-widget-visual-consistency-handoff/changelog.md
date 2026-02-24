# Changelog

## 2026-02-24

Executed full visual consistency polish pass across all 8 confirm widgets and the composition host:

- Added 13 CSS custom properties (`--hc-confirm-*`) to `tokens.css` for spacing, typography, focus, selection, and disabled states.
- Added ~150 lines of attribute-scoped CSS rules to `primitives.css` covering all confirm-specific data-parts.
- Added 17 new data-part names to `parts.ts` (`confirm-section`, `confirm-heading`, `confirm-display`, etc.).
- Rewrote 7 widget components to use semantic `confirm-*` data-parts instead of generic engine parts (SelectableList, SelectableDataTable, FilePickerDropzone, ImageChoiceGrid, RatingPicker, GridBoard, RequestActionBar). SchemaFormRenderer delegates to FormView and needed no markup changes.
- Rewrote `ConfirmRequestWindowHost` composition host with proper heading hierarchy, display section treatment, and widget body layout.
- Fixed missing `@hypercard/confirm-runtime` alias in `.storybook/main.ts`.
- Validated all 5 composite flows visually (DisplayAndConfirmSection, BackAndProgressRating, TwoStepConfirmThenRating, DisplayAndTableSection, DisplayAndUploadSection).
- `npm run storybook:check` passes (63 story files).

### Related Files

- go-go-os/packages/engine/src/theme/desktop/tokens.css — New confirm widget CSS tokens
- go-go-os/packages/engine/src/theme/desktop/primitives.css — New confirm widget CSS rules
- go-go-os/packages/engine/src/parts.ts — 17 new data-part entries
- go-go-os/packages/engine/src/components/widgets/SelectableList.tsx — Widget rewrite
- go-go-os/packages/engine/src/components/widgets/SelectableDataTable.tsx — Widget rewrite
- go-go-os/packages/engine/src/components/widgets/FilePickerDropzone.tsx — Widget rewrite
- go-go-os/packages/engine/src/components/widgets/ImageChoiceGrid.tsx — Widget rewrite
- go-go-os/packages/engine/src/components/widgets/RatingPicker.tsx — Widget rewrite
- go-go-os/packages/engine/src/components/widgets/GridBoard.tsx — Widget rewrite
- go-go-os/packages/engine/src/components/widgets/RequestActionBar.tsx — Widget rewrite
- go-go-os/packages/confirm-runtime/src/components/ConfirmRequestWindowHost.tsx — Composition host rewrite
- go-go-os/.storybook/main.ts — Alias fix

## 2026-02-23

- Created dedicated UI consistency handoff ticket `PC-06-UI-CONSISTENCY-HANDOFF`.
- Added designer handoff document with:
  - complete widget inventory,
  - complete story inventory,
  - usage mapping into confirm-runtime,
  - consistency scenario matrix,
  - prioritized future UI backlog.
- Added UI-specific diary and task checklist for collaborative handoff.

## 2026-02-23

Recorded UI-facing inventory host integration tranche (commit af1a085): confirm queue window surface, request-window app-key delegation, and host runtime wiring for @hypercard/confirm-runtime.

### Related Files

- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/apps/inventory/src/App.tsx — UI host surfaces for confirm queue and request windows
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-06-UI-CONSISTENCY-HANDOFF--confirm-widget-visual-consistency-handoff/reference/01-diary.md — Step 2 records UI integration tranche


## 2026-02-23

Updated designer handoff scope for newly added engine widgets and script host composition:

- Added `RatingPicker` and `GridBoard` to widget inventory.
- Added their Storybook matrices to story inventory.
- Updated current usage mapping to reflect script section parity and upload composition now using `FilePickerDropzone` + `RequestActionBar`.
- Expanded per-widget design scenario checklist and future backlog for rating/grid polish.

### Related Files

- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-06-UI-CONSISTENCY-HANDOFF--confirm-widget-visual-consistency-handoff/design-doc/01-designer-handoff-confirm-widget-inventory-stories-and-consistency-scenarios.md — Updated inventory/story/scenario matrix
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/RatingPicker.stories.tsx — New rating story matrix
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/GridBoard.stories.tsx — New grid story matrix

## 2026-02-23

Added composite confirm-runtime Storybook suite for designer handoff coverage (commit `e1b2023` in `go-go-os`) and refreshed this ticket’s design-doc/tasks/diary to include those scenarios.

Composite stories added:

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

- `npm run storybook:check` (pass, 63 story files)

### Related Files

- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/apps/inventory/src/features/confirm/stories/ConfirmRuntimeComposite.stories.tsx — New composite scenario suite for script-section/back/progress review
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-06-UI-CONSISTENCY-HANDOFF--confirm-widget-visual-consistency-handoff/design-doc/01-designer-handoff-confirm-widget-inventory-stories-and-consistency-scenarios.md — Updated story inventory and review workflow
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-06-UI-CONSISTENCY-HANDOFF--confirm-widget-visual-consistency-handoff/tasks.md — Added checked composite-story inventory task
- /home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/23/PC-06-UI-CONSISTENCY-HANDOFF--confirm-widget-visual-consistency-handoff/reference/01-diary.md — Step 4 records implementation and handoff update

## 2026-02-24

Ticket closed

