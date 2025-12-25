# Tasks

## TODO

### 1) Types / schemas (wire contract)

- [ ] Extend **Go** widget output structs in `internal/types/types.go` to include an optional comment:
  - [ ] `ConfirmOutput.Comment *string  json:"comment,omitempty"`
  - [ ] `SelectOutput.Comment *string   json:"comment,omitempty"`
  - [ ] `FormOutput.Comment *string     json:"comment,omitempty"`
  - [ ] `TableOutput.Comment *string    json:"comment,omitempty"`
  - [ ] `UploadOutput.Comment *string   json:"comment,omitempty"`
  - [ ] `ImageOutput.Comment *string    json:"comment,omitempty"`
- [ ] Extend **TypeScript** output interfaces in `agent-ui-system/client/src/types/schemas.ts`:
  - [ ] Add `comment?: string` to each `*Output` interface above.

### 2) Frontend UI (folded textarea per widget)

- [ ] Add a folded “comment (optional)” section to each dialog component, default closed:
  - [ ] `agent-ui-system/client/src/components/widgets/ConfirmDialog.tsx`
  - [ ] `.../SelectDialog.tsx`
  - [ ] `.../FormDialog.tsx`
  - [ ] `.../TableDialog.tsx`
  - [ ] `.../UploadDialog.tsx`
  - [ ] `.../ImageDialog.tsx`
- [ ] Use existing UI primitives (`components/ui/collapsible.tsx` or `components/ui/accordion.tsx`) and a textarea component.
- [ ] Output behavior:
  - [ ] If comment is empty/whitespace, **omit** it from output payload (so JSON stays minimal).
  - [ ] If non-empty, include `comment: "<trimmed>"` in the submitted output object.

### 3) CLI outputs (return comment in answer)

- [ ] Update each CLI verb to include a `comment` column (empty if none):
  - [ ] `internal/cli/confirm.go`
  - [ ] `internal/cli/select.go`
  - [ ] `internal/cli/form.go`
  - [ ] `internal/cli/table.go`
  - [ ] `internal/cli/upload.go` (repeat comment on each file row)
  - [ ] `internal/cli/image.go`

### 4) Validation

- [ ] Run:
  - [ ] `go test ./...`
  - [ ] `pnpm -C agent-ui-system check`
- [ ] Manual spot-check in browser:
  - [ ] confirm: approve + comment → CLI prints comment
  - [ ] image: select + comment → CLI prints comment
  - [ ] upload: “uploadedFiles” + comment → CLI prints comment per row

### 5) Ticket hygiene

- [ ] Update ticket changelog + diary/reference as needed
- [ ] Mark all completed checkboxes in this `tasks.md`

