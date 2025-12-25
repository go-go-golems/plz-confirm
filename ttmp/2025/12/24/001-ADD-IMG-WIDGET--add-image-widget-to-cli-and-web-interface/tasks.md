# Tasks

## TODO

### 0) Ticket setup + research (done)

- [x] Create docmgr ticket `001-ADD-IMG-WIDGET`
- [x] Create analysis document (widget architecture + touchpoints)
- [x] Create design document (decisions + rationale)
- [x] Relate key files in ticket index and docs; ensure `docmgr doctor` passes

### 1) Sanity-check assumptions (do this before coding)

- [ ] Confirm current “upload widget” is simulated in the React UI and there is **no** Go backend upload endpoint yet (so we are implementing `/api/images` from scratch)
- [ ] Confirm desired API ergonomics: **two-step** CLI flow (upload files → create request with URLs) is acceptable

### 2) Backend: image upload + serving (Go server)

- [ ] Add an image store abstraction (new file) to manage uploaded images on disk + metadata in memory:
  - [ ] Temp directory selection (`os.TempDir()/plz-confirm-images/` or similar)
  - [ ] Generate stable IDs, store `(id -> path, mime, size, createdAt, expiresAt)`
  - [ ] Enforce per-file size limit (e.g. 50MB)
  - [ ] Validate MIME type (`image/*`) and/or sniff content
  - [ ] Cleanup strategy (periodic GC + best-effort delete on expiry)
- [ ] Add server routes:
  - [ ] `POST /api/images` (multipart/form-data upload; returns JSON `{id, url, mimeType, size}`; `url` should be `/api/images/{id}`)
  - [ ] `GET /api/images/{id}` (serve file with correct `Content-Type`, `Cache-Control`, and safe path handling)
- [ ] Ensure routing doesn’t conflict with SPA/static handler (never serve static for `/api/*`)
- [ ] Add minimal logging (do not remove existing debug logging)

### 3) Shared types: Go + TypeScript

- [ ] Add new widget type + schemas:
  - [ ] `internal/types/types.go`: add `WidgetImage`, plus `ImageItem`, `ImageInput`, `ImageOutput`
  - [ ] `agent-ui-system/client/src/types/schemas.ts`: add `'image'` to `UIRequest.type` union, plus TS equivalents

### 4) CLI client: image upload helper

- [ ] Extend Go HTTP client (`internal/client/`) with an `UploadImage(...)` helper:
  - [ ] Accept a local file path
  - [ ] Build multipart request to `POST /api/images`
  - [ ] Decode server response and return the served URL (and metadata if needed)

### 5) CLI: new `plz-confirm image` command

- [ ] Implement `internal/cli/image.go`:
  - [ ] Flags: `--title`, `--message`, `--mode select|confirm`, `--image` (repeatable), optional metadata flags (`--image-label`, `--image-alt`, `--image-caption`), selection flags (`--multi`, `--option`), plus common flags (`--base-url`, `--timeout`, `--wait-timeout`, `--output`)
  - [ ] For each `--image`:
    - [ ] If it’s a local path: call `UploadImage` and use returned URL as `src`
    - [ ] If it’s an URL or `data:` URI: pass through as `src`
  - [ ] Create request (`POST /api/requests`), wait (`GET /api/requests/{id}/wait`)
  - [ ] Output: `request_id`, `selected_json` (or typed columns if we decide), and `timestamp`
- [ ] Register command in `cmd/plz-confirm/main.go`

### 6) Frontend: new widget UI

- [ ] Create `agent-ui-system/client/src/components/widgets/ImageDialog.tsx`:
  - [ ] Render title + optional message
  - [ ] Render responsive image grid (1 large, 2 side-by-side, 3+ grid)
  - [ ] Select mode:
    - [ ] **Variant A (image-pick)**: selecting images directly (click image tiles)
      - [ ] Single select (click to select, submit button)
      - [ ] Multi select (toggle selection, show count, submit button)
    - [ ] **Variant B (images-as-context + multi-select question)**: show images on top, then render a checkbox list below (from `input.options[]`)
      - [ ] Multi select (checkbox list, show count, submit button)
      - [ ] Decide output shape for this variant (indices vs strings) and keep it consistent with CLI printing
  - [ ] Confirm mode:
    - [ ] Approve/Reject buttons (match `ConfirmDialog` UX conventions)
  - [ ] Handle per-image loading + error state
  - [ ] Accessibility: `alt` text, focus states, keyboard navigation (at least basic)
- [ ] Update `agent-ui-system/client/src/components/WidgetRenderer.tsx` to handle `'image'`

### 7) Documentation + examples

- [ ] Update `plz-confirm/pkg/doc/how-to-use.md` with `plz-confirm image` section + examples
- [ ] Update `plz-confirm/README.md` “Widget Commands” list to include `image`
- [ ] Add a ticket-local smoke script under `ttmp/.../scripts/` showing typical calls:
  - [ ] select (Variant A: image-pick)
  - [ ] select (Variant B: images-as-context + multi-select question)
  - [ ] confirm (similarity)

### 8) Tests / validation

- [ ] Add backend tests for image store + endpoints (at least happy path + size/mime rejection)
- [ ] Add a minimal e2e smoke test (script is OK): upload 2 images, create request, submit response via `/api/requests/{id}/response`, verify CLI output
- [ ] Add a minimal UI validation checklist for Variant B (checkbox list below images)
- [ ] Add an API-driven CLI smoke script (auto-submit responses) under ticket `scripts/` to validate plumbing without manual browser clicks

### 9) Optional follow-ups (explicitly optional)

- [ ] Make the existing `upload` widget “real” by wiring it to a backend upload endpoint (currently simulated in UI)
- [ ] Add server-side expiry enforcement for requests + uploaded images (if needed beyond GC)

