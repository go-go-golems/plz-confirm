---
Title: 'Designer Handoff: Confirm Widget Inventory, Stories, and Consistency Scenarios'
Ticket: PC-06-UI-CONSISTENCY-HANDOFF
Status: active
Topics:
    - frontend
    - ux
    - architecture
    - javascript
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-os/apps/inventory/src/App.tsx
      Note: Current in-context queue/request window UI host surfaces
    - Path: go-go-os/apps/inventory/src/app/store.ts
      Note: confirmRuntime reducer integration point
    - Path: go-go-os/apps/inventory/src/features/confirm/stories/ConfirmRuntimeComposite.stories.tsx
      Note: Composite confirm-runtime script-section stories for full-window design review
    - Path: go-go-os/packages/confirm-runtime/src/components/ConfirmRequestWindowHost.tsx
      Note: |-
        Current package-level composition of widgets into confirm request UI
        Script composition now uses rating/grid widgets and section-aware rendering
    - Path: go-go-os/packages/engine/src/components/widgets/FilePickerDropzone.stories.tsx
      Note: Story matrix for file picker/dropzone states
    - Path: go-go-os/packages/engine/src/components/widgets/FilePickerDropzone.tsx
      Note: Upload/dropzone visual baseline for upload widgets
    - Path: go-go-os/packages/engine/src/components/widgets/GridBoard.stories.tsx
      Note: Story inventory for grid visual variants
    - Path: go-go-os/packages/engine/src/components/widgets/GridBoard.tsx
      Note: New core grid board widget for script flows
    - Path: go-go-os/packages/engine/src/components/widgets/ImageChoiceGrid.stories.tsx
      Note: Story matrix for image mode variants and states
    - Path: go-go-os/packages/engine/src/components/widgets/ImageChoiceGrid.tsx
      Note: Image selection primitive for image request flows
    - Path: go-go-os/packages/engine/src/components/widgets/RatingPicker.stories.tsx
      Note: Story inventory for rating visual variants
    - Path: go-go-os/packages/engine/src/components/widgets/RatingPicker.tsx
      Note: New core rating widget for script flows
    - Path: go-go-os/packages/engine/src/components/widgets/RequestActionBar.stories.tsx
      Note: Story matrix for action bar behaviors
    - Path: go-go-os/packages/engine/src/components/widgets/RequestActionBar.tsx
      Note: Shared footer action area with optional comment input
    - Path: go-go-os/packages/engine/src/components/widgets/SchemaFormRenderer.stories.tsx
      Note: Story matrix for schema form behavior and field states
    - Path: go-go-os/packages/engine/src/components/widgets/SchemaFormRenderer.tsx
      Note: Schema-based form renderer for confirm/form flows
    - Path: go-go-os/packages/engine/src/components/widgets/SelectableDataTable.stories.tsx
      Note: Story matrix for selectable data table visuals and interactions
    - Path: go-go-os/packages/engine/src/components/widgets/SelectableDataTable.tsx
      Note: New selectable table primitive used for table approval flows
    - Path: go-go-os/packages/engine/src/components/widgets/SelectableList.stories.tsx
      Note: Story matrix for selectable list visuals and interactions
    - Path: go-go-os/packages/engine/src/components/widgets/SelectableList.tsx
      Note: New selectable list primitive used for confirm/select flows
    - Path: go-go-os/packages/engine/src/components/widgets/index.ts
      Note: Widget export surface now includes rating/grid
    - Path: go-go-os/tooling/vite/createHypercardViteConfig.ts
      Note: Dev proxy and alias behavior affecting UI integration
ExternalSources: []
Summary: Designer-facing handoff documenting all new confirm-related widgets, their stories, usage patterns, and the remaining scenario/style work needed to reach full visual consistency.
LastUpdated: 2026-02-23T18:35:00-05:00
WhatFor: Enable a design colleague to immediately align the new confirm widgets with the established application visual language.
WhenToUse: Use before any UI polish work on confirm widgets and as the canonical checklist for visual consistency scope.
---



# Designer Handoff: Confirm Widget Inventory, Stories, and Consistency Scenarios

## Executive Summary

This document is the handoff pack for visual consistency work on the new confirm-related widget layer.

It provides:

1. Complete inventory of widgets that were newly introduced.
2. Complete inventory of Storybook stories created for those widgets.
3. Practical explanation of how each widget behaves and where it is used.
4. Scenario matrix for what still must be styled/polished to match the rest of the application.
5. A prioritized UI backlog for the design pass.

Scope note:

1. The widgets are implemented as reusable engine primitives.
2. Confirm protocol-specific composition currently lives in `@hypercard/confirm-runtime`.
3. This handoff focuses on visual behavior and consistency, not backend protocol logic.

## Problem Statement

New confirm widgets exist and are functionally testable in Storybook, but they are not yet fully aligned with the rest of the application’s visual language.

Consistency gaps to close:

1. Spacing/density alignment with existing desktop windows and card surfaces.
2. Typography hierarchy consistency for labels, helper text, and status text.
3. State styling consistency (`selected`, `active`, `warning`, `error`, `busy`, `disabled`).
4. Cross-widget interaction consistency (selection affordances, action placement, focus behavior).
5. Confirm-flow scenario coherence across widgets when embedded in request windows.

## Delivered Widget Inventory

### 1) SelectableList

Path: `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/SelectableList.tsx`

Purpose:

1. Reusable list selector for single/multi choice request flows.

Core behaviors:

1. Accepts simple string items or rich objects.
2. Supports single or multiple selection modes.
3. Supports optional search filtering.
4. Supports keyboard navigation (`ArrowUp`, `ArrowDown`, `Enter`, `Space`).
5. Supports optional submit callback from keyboard (`Enter`).
6. Supports disabled rows.

Typical usage:

```tsx
<SelectableList
  items={options}
  selectedIds={selectedIds}
  onSelectionChange={setSelectedIds}
  mode="multiple"
  searchable
/>
```

### 2) SelectableDataTable

Path: `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/SelectableDataTable.tsx`

Purpose:

1. Table view with built-in row selection and optional search.

Core behaviors:

1. Single/multi row selection.
2. Search over configured fields.
3. Configurable row key resolution.
4. Works with existing `ColumnConfig` rendering rules.

Typical usage:

```tsx
<SelectableDataTable
  items={rows}
  columns={columns}
  rowKey="id"
  selectedRowKeys={selectedRows}
  onSelectionChange={setSelectedRows}
  mode="multiple"
  searchable
/>
```

### 3) SchemaFormRenderer

Path: `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/SchemaFormRenderer.tsx`

Purpose:

1. Maps JSON-schema-ish input into `FormView` fields for confirm/form workflows.

Core behaviors:

1. Converts schema properties to `FieldConfig[]`.
2. Supports required fields.
3. Supports enum -> select mapping.
4. Coerces number/boolean values on submit.
5. Supports controlled and uncontrolled value patterns.

Typical usage:

```tsx
<SchemaFormRenderer
  schema={schema}
  value={formState}
  onValueChange={setFormState}
  onSubmit={handleSubmit}
/>
```

### 4) FilePickerDropzone

Path: `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/FilePickerDropzone.tsx`

Purpose:

1. Upload-surface primitive for file pick + drag/drop flows.

Core behaviors:

1. Drag/drop region.
2. Hidden native file input trigger.
3. Accept filtering by extension or MIME patterns.
4. Optional max file size constraint.
5. Emits accepted/rejected files and reasons.

Typical usage:

```tsx
<FilePickerDropzone
  accept={['image/*', '.png']}
  multiple
  maxSizeBytes={2 * 1024 * 1024}
  onFilesChange={handleFiles}
/>
```

### 5) ImageChoiceGrid

Path: `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/ImageChoiceGrid.tsx`

Purpose:

1. Select/confirm/multi image choice component.

Core behaviors:

1. Grid layout with configurable column count.
2. Modes: `select`, `confirm`, `multi`.
3. Handles loading and error messaging states.
4. Supports disabled images and badges.

Typical usage:

```tsx
<ImageChoiceGrid
  items={imageItems}
  selectedIds={selectedImageIds}
  onSelectionChange={setSelectedImageIds}
  mode="multi"
/>
```

### 6) RequestActionBar

Path: `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/RequestActionBar.tsx`

Purpose:

1. Shared footer action area for confirm widgets.

Core behaviors:

1. Primary action + optional secondary action.
2. Optional comment textarea.
3. Busy/disabled handling.
4. Controlled/uncontrolled comment support.

Typical usage:

```tsx
<RequestActionBar
  primaryLabel="Approve"
  secondaryLabel="Reject"
  onPrimary={handleApprove}
  onSecondary={handleReject}
  commentEnabled
/>
```

### 7) RatingPicker

Path: `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/RatingPicker.tsx`

Purpose:

1. Reusable Likert/rating control for script steps (`numbers`, `stars`, `emoji`, `slider`).

Core behaviors:

1. Scale clamped to `2..10`.
2. Supports display styles `numbers`, `stars`, `emoji`, and `slider`.
3. Supports low/high labels.
4. Controlled `value` with `onChange`.

Typical usage:

```tsx
<RatingPicker
  scale={5}
  style="stars"
  value={rating}
  onChange={setRating}
  lowLabel="Low"
  highLabel="High"
/>
```

### 8) GridBoard

Path: `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/GridBoard.tsx`

Purpose:

1. Reusable row/column board selector for script `grid` steps.

Core behaviors:

1. Renders fixed row/column board with per-cell labels/colors/disabled state.
2. Supports cell sizes (`small`, `medium`, `large`).
3. Emits normalized selection payload (`row`, `col`, `cellIndex`).
4. Supports controlled selection highlight.

Typical usage:

```tsx
<GridBoard
  rows={3}
  cols={3}
  cells={cells}
  selectedIndex={selectedCell}
  onSelect={setSelection}
/>
```

## Storybook Inventory (All New Stories)

### SelectableList stories

Path: `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/SelectableList.stories.tsx`

1. `SingleSelect`
2. `MultiSelect`
3. `Searchable`
4. `Empty`
5. `LongListScrollable`
6. `InteractiveSingle`
7. `InteractiveMultipleWithSubmit`
8. `ControlledSearchText`

### SelectableDataTable stories

Path: `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/SelectableDataTable.stories.tsx`

1. `SingleSelect`
2. `MultiSelect`
3. `Searchable`
4. `Empty`
5. `LargeDataset`
6. `InteractiveSingle`
7. `InteractiveMultipleSearch`

### SchemaFormRenderer stories

Path: `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/SchemaFormRenderer.stories.tsx`

1. `Basic`
2. `WithInitialValue`
3. `ReadOnlyFields`
4. `EdgeFallbackFields`
5. `InteractiveControlled`
6. `NumberBooleanCoercion`

### FilePickerDropzone stories

Path: `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/FilePickerDropzone.stories.tsx`

1. `Default`
2. `ImagesOnly`
3. `SingleFileOnly`
4. `MaxSizeConstrained`
5. `InteractiveResultPanel`
6. `WideSurface`

### ImageChoiceGrid stories

Path: `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/ImageChoiceGrid.stories.tsx`

1. `SelectMode`
2. `ConfirmMode`
3. `MultiMode`
4. `FourColumns`
5. `LoadingState`
6. `ErrorState`
7. `Empty`
8. `InteractiveMulti`

### RequestActionBar stories

Path: `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/RequestActionBar.stories.tsx`

1. `PrimaryOnly`
2. `PrimarySecondary`
3. `WithCommentField`
4. `BusyState`
5. `DisabledActions`
6. `ControlledComment`
7. `Interactive`

### RatingPicker stories

Path: `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/RatingPicker.stories.tsx`

1. `Numbers`
2. `Stars`
3. `Emoji`
4. `Slider`
5. `Interactive`

### GridBoard stories

Path: `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/GridBoard.stories.tsx`

1. `Medium`
2. `Small`
3. `Large`
4. `InteractiveSelection`

### Composite confirm-runtime stories

Path: `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/apps/inventory/src/features/confirm/stories/ConfirmRuntimeComposite.stories.tsx`

1. `DisplayAndConfirmSection`
2. `DisplayAndSelectSection`
3. `DisplayAndFormSection`
4. `DisplayAndTableSection`
5. `DisplayAndUploadSection`
6. `DisplayAndImageSection`
7. `BackAndProgressRating`
8. `TwoStepConfirmThenRating`
9. `InvalidSectionsContract`

## Where Widgets Are Used Right Now

Current package-level usage reference:

1. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/components/ConfirmRequestWindowHost.tsx`

Current mapping:

1. `confirm` -> `RequestActionBar`
2. `select` -> `SelectableList` + `RequestActionBar`
3. `form` -> `SchemaFormRenderer`
4. `table` -> `SelectableDataTable` + `RequestActionBar`
5. `image` -> `ImageChoiceGrid` + `RequestActionBar`
6. `upload` -> `FilePickerDropzone` + `RequestActionBar`
7. `script` -> section-aware composition with:
   - `display` context sections,
   - one interactive section validation,
   - back action support,
   - progress/title/description header support,
   - interactive widgets including `rating` and `grid`.

### Inventory host surfaces now present

As of commit `af1a085`, the inventory app now exposes two UI integration surfaces:

1. **Confirm Queue window** (`appKey = confirm-queue`):
   - shows active requests list,
   - allows opening each request into its own window.
2. **Confirm Request window** (`appKey = confirm-request:<id>`):
   - renders `ConfirmRequestWindowHost`,
   - hosts whichever widget type is active for that request.

These surfaces are where final visual consistency should be validated in-context.

## Design Consistency Scenarios To Address

The following scenario groups are still needed for full consistency pass.

### A. Shared visual foundation

1. Harmonize paddings/margins with existing window/card spacing scale.
2. Align font sizes/weights for labels, values, captions, and inline status text.
3. Align border radius/border weight/shadow behavior with current app shell.
4. Ensure state colors (`selected`, `active`, `warning`, `error`) use shared tokens.

### B. Interaction consistency

1. Define one consistent focus ring behavior for keyboard navigation.
2. Define selected-row/card background and border treatment rules.
3. Define disabled style language shared by all six widgets.
4. Define busy/loading affordance language shared by all six widgets.

### C. Per-widget polish scenarios

1. `SelectableList`:
   - selected vs active contrast,
   - long-label truncation treatment,
   - icon/meta alignment.
2. `SelectableDataTable`:
   - row height and density,
   - selected row affordance,
   - empty-state layout style.
3. `SchemaFormRenderer`:
   - label/input spacing rhythm,
   - read-only field style language,
   - error/help text style conventions.
4. `FilePickerDropzone`:
   - drag-over visual state,
   - rejection messaging style,
   - selected-file list typography.
5. `ImageChoiceGrid`:
   - selected frame treatment,
   - placeholder style for loading/error/empty,
   - consistent image aspect ratio strategy.
6. `RequestActionBar`:
   - button hierarchy (primary vs secondary),
   - textarea style alignment,
   - busy-state affordance consistency.
7. `RatingPicker`:
   - selected-state hierarchy for each style variant,
   - emoji/star sizing consistency vs surrounding typography,
   - slider track/thumb styling alignment with app controls.
8. `GridBoard`:
   - per-cell selected/disabled style language,
   - grid density and board spacing rhythm,
   - cell label truncation/readability rules.

### D. Confirm-runtime composition scenarios

1. Multi-widget request windows should share one content rhythm.
2. Footer actions should anchor consistently across widget types.
3. Title/message/comment surfaces should feel unified.
4. Script flow (future) should inherit same action/footer language.
5. Back/progress framing should stay visually stable across step transitions.
6. Display sections should have clear hierarchy separation from interactive sections.

## Future UI Work Planned (Graphical)

Priority order for remaining visual work:

1. Integrate confirm windows into inventory app shell and verify real in-context styling.
2. Complete script sections polish pass (display rendering fidelity, back/progress affordances, toast styling integration).
3. Upgrade upload flow visuals from placeholder to full file lifecycle states.
4. Add final design token adjustments across all eight widgets after designer pass.
5. Add visual polish variants for composite stories (dense, spacious, warning/error heavy, long-content stress).

## Proposed Designer Workflow

1. Start in Storybook and review each new widget story file in order listed above.
2. Capture decisions in a small style matrix:
   - spacing,
   - typography,
   - state colors,
   - interaction affordances.
3. Apply visual rules in shared CSS/token layer first, then widget-local adjustments.
4. Run composite flow review in this order:
   - `DisplayAndConfirmSection`
   - `BackAndProgressRating`
   - `TwoStepConfirmThenRating`
   - `DisplayAndTableSection`
   - `DisplayAndUploadSection`
5. Validate no regression in legacy widget stories.
6. Hand back a checklist of accepted/rejected scenario states.

## Open Questions

1. Should confirm widgets target strict parity with current windowing aesthetics, or introduce a slightly distinct "workflow panel" accent style?
2. Should `selected` states be color-driven only, or include iconography/checkmarks for accessibility clarity?
3. What is the preferred density target for tables/lists in operator workflows?
4. Should comment fields always be visible in action bars or toggled per flow?
5. How should script toasts/progress visually map into existing status/toast components?

## References

1. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/index.ts`
2. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/confirm-runtime/src/components/ConfirmRequestWindowHost.tsx`
3. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/SelectableList.stories.tsx`
4. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/SelectableDataTable.stories.tsx`
5. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/SchemaFormRenderer.stories.tsx`
6. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/FilePickerDropzone.stories.tsx`
7. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/ImageChoiceGrid.stories.tsx`
8. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/RequestActionBar.stories.tsx`
9. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/RatingPicker.stories.tsx`
10. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/widgets/GridBoard.stories.tsx`
11. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/apps/inventory/src/features/confirm/stories/ConfirmRuntimeComposite.stories.tsx`
