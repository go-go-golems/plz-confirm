# Tasks

## TODO

### 1) Types / schemas (wire contract)

- [x] Extend **Go** widget output structs in `internal/types/types.go` to include an optional comment:
  - [x] `ConfirmOutput.Comment *string  json:"comment,omitempty"`
  - [x] `SelectOutput.Comment *string   json:"comment,omitempty"`
  - [x] `FormOutput.Comment *string     json:"comment,omitempty"`
  - [x] `TableOutput.Comment *string    json:"comment,omitempty"`
  - [x] `UploadOutput.Comment *string   json:"comment,omitempty"`
  - [x] `ImageOutput.Comment *string    json:"comment,omitempty"`
- [x] Extend **TypeScript** output interfaces in `agent-ui-system/client/src/types/schemas.ts`:
  - [x] Add `comment?: string` to each `*Output` interface above.

### 2) Frontend UI (folded textarea per widget)

- [x] Add a folded “comment (optional)” section to each dialog component, default closed:
  - [x] `agent-ui-system/client/src/components/widgets/ConfirmDialog.tsx`
  - [x] `.../SelectDialog.tsx`
  - [x] `.../FormDialog.tsx`
  - [x] `.../TableDialog.tsx`
  - [x] `.../UploadDialog.tsx`
  - [x] `.../ImageDialog.tsx`
- [x] Use existing UI primitives (`components/ui/collapsible.tsx` or `components/ui/accordion.tsx`) and a textarea component.
- [x] Output behavior:
  - [x] If comment is empty/whitespace, **omit** it from output payload (so JSON stays minimal).
  - [x] If non-empty, include `comment: "<trimmed>"` in the submitted output object.

### 3) CLI outputs (return comment in answer)

- [x] Update each CLI verb to include a `comment` column (empty if none):
  - [x] `internal/cli/confirm.go`
  - [x] `internal/cli/select.go`
  - [x] `internal/cli/form.go`
  - [x] `internal/cli/table.go`
  - [x] `internal/cli/upload.go` (repeat comment on each file row)
  - [x] `internal/cli/image.go`

### 4) Validation

- [x] Run:
  - [x] `go test ./...`
  - [x] `pnpm -C agent-ui-system check`
- [ ] Manual spot-check in browser:
  - [ ] confirm: approve + comment → CLI prints comment
  - [ ] image: select + comment → CLI prints comment
  - [ ] upload: “uploadedFiles” + comment → CLI prints comment per row

### 5) Ticket hygiene

- [ ] Update ticket changelog + diary/reference as needed
- [x] Mark all completed checkboxes in this `tasks.md`

