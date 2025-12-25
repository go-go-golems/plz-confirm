# Tasks

## TODO

### 0) Ticket setup + research (done)

- [x] Create docmgr ticket `001-ADD-IMG-WIDGET`
- [x] Create analysis document (widget architecture + touchpoints)
- [x] Create design document (decisions + rationale)
- [x] Relate key files in ticket index and docs; ensure `docmgr doctor` passes

### 1) Sanity-check assumptions (do this before coding)

- [x] Confirm current “upload widget” is simulated in the React UI and there is **no** Go backend upload endpoint yet (so we are implementing `/api/images` from scratch)
- [x] Confirm desired API ergonomics: **two-step** CLI flow (upload files → create request with URLs) is acceptable

### 2) Backend: image upload + serving (Go server)

- [x] Add an image store abstraction (new file) to manage uploaded images on disk + metadata in memory:
  - [x] Temp directory selection (`os.TempDir()/plz-confirm-images/` or similar)
  - [x] Generate stable IDs, store `(id -> path, mime, size, createdAt, expiresAt)`
  - [x] Enforce per-file size limit (e.g. 50MB)
  - [x] Validate MIME type (`image/*`) and/or sniff content
  - [x] Cleanup strategy (periodic GC + best-effort delete on expiry)
- [x] Add server routes:
  - [x] `POST /api/images` (multipart/form-data upload; returns JSON `{id, url, mimeType, size}`; `url` should be `/api/images/{id}`)
  - [x] `GET /api/images/{id}` (serve file with correct `Content-Type`, `Cache-Control`, and safe path handling)
- [x] Ensure routing doesn’t conflict with SPA/static handler (never serve static for `/api/*`)
- [x] Add minimal logging (do not remove existing debug logging)

### 3) Shared types: Go + TypeScript

- [x] Add new widget type + schemas:
  - [x] `internal/types/types.go`: add `WidgetImage`, plus `ImageItem`, `ImageInput`, `ImageOutput`
  - [x] `agent-ui-system/client/src/types/schemas.ts`: add `'image'` to `UIRequest.type` union, plus TS equivalents

### 4) CLI client: image upload helper

- [x] Extend Go HTTP client (`internal/client/`) with an `UploadImage(...)` helper:
  - [x] Accept a local file path
  - [x] Build multipart request to `POST /api/images`
  - [x] Decode server response and return the served URL (and metadata if needed)

### 5) CLI: new `plz-confirm image` command

- [x] Implement `internal/cli/image.go`:
  - [x] Flags: `--title`, `--message`, `--mode select|confirm`, `--image` (repeatable), optional metadata flags (`--image-label`, `--image-alt`, `--image-caption`), selection flags (`--multi`, `--option`), plus common flags (`--base-url`, `--timeout`, `--wait-timeout`, `--output`)
  - [x] For each `--image`:
    - [x] If it’s a local path: call `UploadImage` and use returned URL as `src`
    - [x] If it’s an URL or `data:` URI: pass through as `src`
  - [x] Create request (`POST /api/requests`), wait (`GET /api/requests/{id}/wait`)
  - [x] Output: `request_id`, `selected_json` (or typed columns if we decide), and `timestamp`
- [x] Register command in `cmd/plz-confirm/main.go`

### 6) Frontend: new widget UI

- [x] Create `agent-ui-system/client/src/components/widgets/ImageDialog.tsx`:
  - [x] Render title + optional message
  - [x] Render responsive image grid (1 large, 2 side-by-side, 3+ grid)
  - [x] Select mode:
    - [x] **Variant A (image-pick)**: selecting images directly (click image tiles)
      - [x] Single select (click to select, submit button)
      - [x] Multi select (toggle selection, show count, submit button)
    - [x] **Variant B (images-as-context + multi-select question)**: show images on top, then render a checkbox list below (from `input.options[]`)
      - [x] Multi select (checkbox list, show count, submit button)
      - [x] Decide output shape for this variant (indices vs strings) and keep it consistent with CLI printing
  - [x] Confirm mode:
    - [x] Approve/Reject buttons (match `ConfirmDialog` UX conventions)
  - [x] Handle per-image loading + error state
  - [x] Accessibility: `alt` text, focus states, keyboard navigation (at least basic)
- [x] Update `agent-ui-system/client/src/components/WidgetRenderer.tsx` to handle `'image'`

### 7) Documentation + examples

- [x] Update `plz-confirm/pkg/doc/how-to-use.md` with `plz-confirm image` section + examples
- [x] Update `plz-confirm/README.md` “Widget Commands” list to include `image`
- [x] Add a ticket-local smoke script under `ttmp/.../scripts/` showing typical calls:
  - [x] select (Variant A: image-pick)
  - [x] select (Variant B: images-as-context + multi-select question)
  - [x] confirm (similarity)

### 8) Tests / validation

- [x] Add backend tests for image store + endpoints (at least happy path + size/mime rejection)
- [x] Add a minimal e2e smoke test (script is OK): upload 2 images, create request, submit response via `/api/requests/{id}/response`, verify CLI output
- [x] Add a minimal UI validation checklist for Variant B (checkbox list below images)
- [x] Add an API-driven CLI smoke script (auto-submit responses) under ticket `scripts/` to validate plumbing without manual browser clicks

### 9) Optional follow-ups (explicitly optional)

- [ ] Make the existing `upload` widget “real” by wiring it to a backend upload endpoint (currently simulated in UI)
- [ ] Add server-side expiry enforcement for requests + uploaded images (if needed beyond GC)

## Status

- [x] Ticket implementation complete (all non-optional items above are done)

