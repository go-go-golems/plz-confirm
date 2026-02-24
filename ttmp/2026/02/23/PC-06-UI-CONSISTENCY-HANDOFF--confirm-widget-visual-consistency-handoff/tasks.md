# Tasks

## Documentation

- [x] Create handoff ticket and add design-doc + diary
- [x] Inventory all newly created widgets
- [x] Inventory all Storybook stories for new widgets
- [x] Document current usage mapping in confirm-runtime
- [x] Inventory composite confirm-runtime request-window stories for script sections/back/progress
- [x] Document future consistency scenarios and UI backlog

## Design Handoff Checklist

- [x] Align spacing/density tokens across all eight widgets (13 CSS custom properties in tokens.css)
- [x] Align typography hierarchy and helper/status text styles (heading 13px / body 12px / caption 10px)
- [x] Define canonical selected/active/disabled/busy/error visual language (inverted bg/fg, opacity 0.45, pointer-events:none)
- [x] Confirm keyboard focus ring and navigation affordances (2px solid fg, 1px offset on all interactive parts)
- [x] Approve image grid selection treatment and empty/loading/error visuals (confirm-image-card with inverted selected state)
- [x] Approve upload dropzone drag-over and rejection states (dashed border, highlight background on drag-over)
- [x] Approve action bar button hierarchy and comment field treatment (border-top separator, flex-end button row)
- [x] Sign off on confirm-runtime composed request-window visuals (validated 5 composite flows in Storybook)

## Implementation Checklist

- [x] Define shared CSS tokens for confirm widgets (tokens.css)
- [x] Add confirm-specific CSS rules to primitives.css
- [x] Add 17 new data-part names to parts.ts
- [x] Polish SelectableList widget
- [x] Polish SelectableDataTable widget
- [x] Polish FilePickerDropzone widget
- [x] Polish ImageChoiceGrid widget
- [x] Polish RatingPicker and GridBoard widgets
- [x] Polish RequestActionBar widget
- [x] Polish ConfirmRequestWindowHost composition host
- [x] Fix Storybook @hypercard/confirm-runtime alias
- [x] Validate composite stories visually
- [x] Run storybook:check (pass, 63 files)
- [x] Update ticket docs (diary step 5, changelog, tasks)
