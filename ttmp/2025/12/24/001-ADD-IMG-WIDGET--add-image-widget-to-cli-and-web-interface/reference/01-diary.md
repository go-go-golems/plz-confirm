---
Title: Diary
Ticket: 001-ADD-IMG-WIDGET
Status: active
Topics:
    - cli
    - backend
    - agent-ui-system
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Step-by-step diary of research and implementation work for adding image widget
LastUpdated: 2025-12-24T19:10:57.920234442-05:00
WhatFor: Tracking research, decisions, and implementation progress
WhenToUse: Reference during implementation to understand context and decisions
---

# Diary

## Goal

Track the research, analysis, and implementation of adding an image widget to plz-confirm that allows AI models to display text prompts with images and receive user feedback via selection or confirmation.

## Step 1: Initial Codebase Exploration

This step involved understanding the overall architecture of plz-confirm by exploring key files and understanding how widgets are implemented across CLI, backend, and frontend layers.

### What I did
- Explored workspace structure to understand project layout
- Read `README.md` to understand project purpose and features
- Searched for widget implementation patterns
- Read `internal/types/types.go` to understand type definitions
- Read `internal/cli/confirm.go` and `internal/cli/select.go` to understand CLI command patterns
- Read `cmd/plz-confirm/main.go` to understand command registration
- Read `internal/server/server.go` to understand backend architecture
- Read `internal/client/client.go` to understand client communication
- Read `agent-ui-system/client/src/components/WidgetRenderer.tsx` to understand frontend widget routing
- Read `agent-ui-system/client/src/types/schemas.ts` to understand TypeScript type definitions
- Read widget component examples (`ConfirmDialog.tsx`, `SelectDialog.tsx`, `UploadDialog.tsx`)

### Why
- Needed to understand the three-layer architecture (CLI, backend, frontend)
- Required to identify the pattern for adding new widgets
- Needed to understand how images/files are currently handled (via upload widget)

### What worked
- Found clear widget implementation pattern across all layers
- Identified that backend is widget-agnostic (stores `Input` as `any`)
- Confirmed WebSocket-based real-time communication pattern
- Understood that each widget needs: type definition, CLI command, frontend component

### What I learned
- All widgets follow consistent pattern: CLI command → HTTP API → WebSocket broadcast → Frontend component
- Backend server doesn't need widget-specific logic - it's type-agnostic
- Frontend uses a switch statement in `WidgetRenderer` to route to appropriate component
- Type definitions exist in both Go (`internal/types/types.go`) and TypeScript (`schemas.ts`)
- Upload widget handles files but doesn't display images - it's for file uploads, not image display

### What was tricky to build
- Understanding the relationship between Go types and TypeScript types (manually duplicated, not codegen)
- Identifying where image serving would happen (currently no image serving endpoint exists)
- Understanding the difference between upload widget (for file uploads) and what we need (for displaying images)

### What warrants a second pair of eyes
- Image serving strategy: Should we use base64 embedding, file upload + serving, or external URLs?
- Whether to add a new `/api/images/{id}` endpoint or use existing patterns
- Performance implications of embedding large images vs. serving them

### What should be done in the future
- Consider code generation for Go ↔ TypeScript type synchronization
- Document image serving strategy decision
- Consider image optimization/compression for embedded images
- Add image format validation (JPEG, PNG, WebP, SVG support)

### Code review instructions
- Review `internal/types/types.go` to see current widget type definitions
- Review `internal/cli/select.go` as reference for CLI command pattern
- Review `agent-ui-system/client/src/components/widgets/SelectDialog.tsx` for selection UI patterns

### Technical details

**Widget Type Constants:**
```go
const (
    WidgetConfirm WidgetType = "confirm"
    WidgetSelect  WidgetType = "select"
    WidgetForm    WidgetType = "form"
    WidgetUpload  WidgetType = "upload"
    WidgetTable   WidgetType = "table"
)
```

**CLI Command Pattern:**
1. Command struct with `*cmds.CommandDescription`
2. Settings struct with `glazed.parameter` tags
3. `New*Command()` constructor defining parameters
4. `RunIntoGlazeProcessor()` method handling request/response

**Frontend Widget Pattern:**
- Switch statement in `WidgetRenderer.tsx`
- Component receives: `requestId`, `input`, `onSubmit`, `loading`
- Component calls `onSubmit(output)` when user responds

## Step 2: Analysis Document Creation

This step created a comprehensive analysis document that captures all findings from the codebase exploration, identifies implementation requirements, and outlines the approach for adding the image widget.

### What I did
- Created analysis document via `docmgr doc add`
- Documented current widget architecture
- Identified all components that need modification
- Outlined image handling considerations
- Listed key files to modify/create
- Documented implementation requirements
- Identified open questions and next steps

### Why
- Need structured documentation of findings
- Required for future reference during implementation
- Helps identify all touchpoints before starting implementation

### What worked
- Comprehensive coverage of all architectural layers
- Clear identification of files to modify
- Good separation of concerns (CLI, backend, frontend)

### What I learned
- The analysis document structure helps organize complex multi-layer changes
- Identifying open questions early prevents implementation blockers
- Documenting patterns helps ensure consistency with existing widgets

### What was tricky to build
- Balancing detail vs. brevity in the analysis
- Identifying all potential edge cases (image formats, sizes, layouts)
- Deciding on image serving strategy without full requirements

### What warrants a second pair of eyes
- Image serving strategy decision (base64 vs. file upload vs. URLs)
- Whether to support multiple image sources or pick one approach
- Performance and security considerations for image handling

### What should be done in the future
- Make decision on image serving strategy
- Design `ImageInput` and `ImageOutput` structs in detail
- Create mockups/wireframes for image widget UI
- Define image size/format limits

### Code review instructions
- Review analysis document: `analysis/01-image-widget-implementation-analysis.md`
- Focus on "Implementation Requirements" and "Key Files to Modify/Create" sections
- Verify all architectural layers are covered

### Technical details

**Files to Create:**
- `internal/cli/image.go` - CLI command
- `agent-ui-system/client/src/components/widgets/ImageDialog.tsx` - Frontend component

**Files to Modify:**
- `internal/types/types.go` - Add widget type and structs
- `cmd/plz-confirm/main.go` - Register command
- `agent-ui-system/client/src/types/schemas.ts` - Add TypeScript types
- `agent-ui-system/client/src/components/WidgetRenderer.tsx` - Add widget case

**Open Questions:**
1. Image storage strategy (base64 vs. file upload vs. URLs)
2. Image size limits
3. Supported image formats
4. Layout options
5. Accessibility considerations

## Step 3: Design Document Creation

This step created a comprehensive design document that makes concrete decisions on all open questions from the analysis phase, providing detailed specifications and rationale for each design choice.

### What I did
- Created design document via `docmgr doc add`
- Made decisions on all open questions from analysis
- Documented image serving strategy (hybrid: URLs + base64, no file upload initially)
- Designed type definitions (ImageInput, ImageOutput, ImageItem)
- Specified CLI command interface with examples
- Designed frontend component structure
- Documented accessibility requirements
- Created implementation plan with phases
- Documented security and performance considerations

### Why
- Need concrete decisions before implementation
- Rationale documentation helps future maintainers
- Resolves all open questions from analysis phase
- Provides clear specification for implementation

### What worked
- Comprehensive coverage of all design aspects
- Clear rationale for each decision
- Concrete examples and code snippets
- Implementation plan broken into phases

### What I learned
- Hybrid approach (URLs + base64) covers most use cases without complexity
- Single command with mode flag is more flexible than separate commands
- Responsive grid layout is simpler and more accessible than carousel
- Size limits (1MB per image, 10MB total) balance functionality and performance

### What was tricky to build
- Balancing flexibility with simplicity (chose hybrid approach over file upload)
- Deciding on default mode (chose "select" as most common use case)
- Layout decisions (chose responsive grid over carousel for accessibility)

### What warrants a second pair of eyes
- Image serving strategy decision (hybrid vs. file upload)
- Size limits (1MB per image, 10MB total) - may need adjustment based on real-world usage
- Layout choices (responsive grid vs. carousel)
- Accessibility implementation details

### What should be done in the future
- Monitor real-world usage to validate size limits
- Consider adding file upload + serving endpoint if needed
- Consider advanced layouts (carousel, side-by-side diff) as enhancements
- Add image annotation tools if needed

### Code review instructions
- Review design document: `design-doc/01-image-widget-design-specification.md`
- Focus on "Design Decisions" section for rationale
- Verify type definitions match implementation plan
- Check that CLI interface is consistent with existing widgets

### Technical details

**Key Design Decisions:**
1. **Image Serving**: Hybrid (URLs + base64), no file upload initially
2. **Widget Modes**: Two modes (select, confirm) via `--mode` flag
3. **Image Input**: Unified `ImageItem` struct with `src`, `alt`, `label`, `caption`
4. **Layout**: Responsive grid (1 image: large, 2: side-by-side, 3+: grid)
5. **Size Limits**: 1MB per image (base64), 10MB total payload
6. **Output Format**: Indices for select mode, boolean for confirm mode

**Type Definitions:**
```go
type ImageItem struct {
    Src     string  `json:"src"`
    Alt     *string `json:"alt,omitempty"`
    Label   *string `json:"label,omitempty"`
    Caption *string `json:"caption,omitempty"`
}

type ImageInput struct {
    Title   string      `json:"title"`
    Message *string     `json:"message,omitempty"`
    Images  []ImageItem `json:"images"`
    Mode    string      `json:"mode"` // "select" or "confirm"
    Options []string    `json:"options,omitempty"`
    Multi   *bool       `json:"multi,omitempty"`
}
```

**CLI Command:**
```bash
plz-confirm image \
  --title "Title" \
  --message "Message" \
  --image <path|url|data-uri> \
  [--image-label "Label"] \
  [--mode select|confirm] \
  [--option "Option"] \
  [--multi]
```

## Step 4: Design Revision - File Upload + Serving

This step revised the design decision to include file upload + serving infrastructure after reconsidering the complexity.

### What I did
- Re-evaluated file upload complexity (~180 lines of code)
- Updated design document to include file upload + serving
- Changed size limits: 500KB threshold (base64 vs upload)
- Updated implementation plan to include image storage infrastructure
- Revised "Alternatives Considered" section

### Why
- User questioned whether file upload complexity was really that significant
- Upon review, complexity is manageable (~180 lines)
- Benefits outweigh costs: large images, better performance, simpler CLI usage

### What worked
- File upload infrastructure is straightforward:
  - Upload endpoint: POST `/api/images` (~50 lines)
  - Storage: Temp directory + in-memory map (~50 lines)
  - Serving endpoint: GET `/api/images/{id}` (~30 lines)
  - Cleanup: Tied to request expiration (~30 lines)
  - Security: File validation (~20 lines)

### What I learned
- Initial complexity assessment was overly conservative
- File upload enables better user experience (large images, no size limits)
- Infrastructure can be reused for other features
- Server already has static file serving, so image serving is similar

### What was tricky to build
- Balancing when to use base64 vs upload (chose 500KB threshold)
- Deciding on cleanup strategy (tied to request expiration)

### What warrants a second pair of eyes
- File upload security (MIME type validation, size limits, path sanitization)
- Cleanup strategy (request expiration vs periodic cleanup)
- Temp directory location and permissions

### What should be done in the future
- Monitor file upload usage patterns
- Consider persistent storage option if needed
- Add image optimization/compression if needed

### Code review instructions
- Review updated Decision 1 in design document
- Focus on file upload infrastructure implementation
- Verify security considerations are addressed

### Technical details

**Revised Image Serving Strategy:**
- **File paths**: Uploaded if >= 500KB, base64 if < 500KB
- **URLs**: Passed through as-is
- **Base64**: Passed through as-is

**File Upload Infrastructure:**
- POST `/api/images` - multipart upload, returns `{id, url}`
- GET `/api/images/{id}` - serves file with Content-Type
- Storage: `os.TempDir()/plz-confirm-images/`
- Cleanup: When request expires
- Security: MIME type validation, size limits (50MB), path sanitization

**Size Limits:**
- Base64: 500KB per image (prevents large JSON payloads)
- Upload: 50MB per image (reasonable for high-res images)
- Total payload: 5MB (base64 only)

## Step 5: Corrections after re-reading the code (grain of salt pass)

This step is a deliberate “sanity sweep” over earlier assumptions. The core finding is that the existing `upload` widget in the React UI **does not actually upload** to the Go server right now—it simulates an upload client-side and returns fake `/tmp/uploads/...` paths. That means any “we can reuse existing upload infrastructure” assumption was wrong: we’re implementing real upload/serve endpoints for images from scratch.

### What I did
- Re-checked the React `UploadDialog` implementation and confirmed it simulates uploads (progress loop + synthetic file paths).
- Re-checked the Go server routes and confirmed there is currently no `/api/images` or file upload endpoint.
- Reconciled the design doc with the actual API shape: `POST /api/requests` expects JSON (not multipart), so “upload files then create request with URLs” is the cleanest path.

### Why
- The earlier research was useful, but it mixed in an assumption that “upload exists” because the widget is named upload and the CLI prints file_path.
- If we don’t correct that now, implementation will get stuck when the UI/CLI expects endpoints that don’t exist.

### What I learned
- The current “upload” feature is effectively a UI mock; the Go backend is widget-agnostic and stores arbitrary JSON, but does not handle binary payloads.
- Implementing `/api/images` is not hard, but it is foundational work (storage, serving, cleanup, security).

### What warrants a second pair of eyes
- Security posture of `POST /api/images` (MIME sniffing, size limits, safe path handling).
- Cleanup semantics (how we decide expiry for uploaded images; periodic GC is likely enough initially).

## Step 6: Update tasks.md to match the corrected design

This step rewrote the ticket task list into a concrete implementation checklist that reflects the corrected realities: real image upload endpoints need to be implemented, and the CLI should perform a two-step flow (upload local files → create request with URLs).

### What I did
- Replaced the placeholder `tasks.md` with a structured checklist:
  - Done: ticket + analysis + design docs.
  - Next: backend `/api/images` + storage, types, CLI command, frontend widget, docs, tests.
- Explicitly added a “sanity-check assumptions” section so we don’t regress into the earlier mistaken belief that upload endpoints already exist.

### Why
- A correct task list prevents us from starting implementation in the wrong place.
- The list now encodes the “source of truth” architecture: **upload → request → websocket → response**.

### What should be done next
- Start with backend image upload + serving (everything else depends on it).

## Step 7: Improve design doc readability + add ASCII UI sketches

This step revised the design document to be more readable and more concrete. The earlier version contained the right ideas, but it read like a dense spec: lots of lists, not enough narrative “glue”, and no visual anchor for what the UI is supposed to look like. Following the technical writing guidelines we’re using elsewhere, I rewrote the top of the doc to introduce sections with short prose paragraphs, then added structured content (bullets, diagrams, pseudocode) only after the reader has context.

### What I did
- Read the writing guidelines doc and applied its main pattern: narrative context first, then structured content.
- Reworked the design doc’s **Executive Summary** into multiple paragraphs + clearer bullets.
- Added a **Goals / Non-Goals** section so scope stays stable during implementation.
- Added an **Architecture Overview** section with:
  - a simple data-flow diagram (CLI ↔ server ↔ UI), and
  - pseudocode for the CLI flow (upload → create request → wait), and server flow (upload + serve).
- Added **ASCII UI sketches** for:
  - select mode (single-select),
  - select mode (multi-select),
  - confirm mode (two-image similarity),
  - per-image error state.
- Fixed a stale line that still implied “auto base64 conversion for small files” — the design now treats local files as “always upload”, with data URIs supported only as pass-through.

### Why
- Concrete UI sketches prevent “implementation by interpretation” (especially around grid layout, submit buttons, and error states).
- The narrative-first structure makes the doc easier to read linearly, while still allowing quick scanning for specifics.

### What warrants a second pair of eyes
- The UI sketches: confirm they reflect the desired interaction model and that we’re not missing a critical state (e.g., loading, disabled submit).
- The cleanup story: we described “cleanup when request expires”, but implementation will likely be “periodic GC + best-effort expiry tracking” — reviewers should check we don’t over-promise.

## Step 8: Implement backend image upload + serving endpoints

This step introduced the first piece of “real implementation” for the image widget: a minimal server-side image store plus HTTP endpoints to upload local files and serve them back to the browser. Without this, the CLI can’t safely accept local image paths, because the web UI has no way to fetch bytes from the agent’s filesystem.

I also fixed a build-time sharp edge: the Go `embed` directive previously caused `go test ./...` to fail when generated frontend assets weren’t present. The solution was to gate embedding behind a build tag so dev/test builds compile cleanly without running asset generation first.

**Commit (code):** 41a79bcd5635c8f1d497d48aae3b6704e60a5545 — "Server: add image upload+serve endpoints"

### What I did
- Added `internal/server/images.go` implementing an `ImageStore`:
  - stores uploaded images on disk (temp directory) and indexes metadata in memory
  - supports expiry via `ExpiresAt` + periodic cleanup
- Added two new endpoints in `internal/server/server.go`:
  - `POST /api/images` (multipart upload, MIME sniffing + `image/*` validation, returns `{id,url,mimeType,size}`)
  - `GET /api/images/{id}` (serves the stored image with `Content-Type` + conservative caching)
- Added an expiry cleanup goroutine in `Server.ListenAndServe` (runs every 30s).
- Made embedded frontend assets optional for dev/test builds:
  - `internal/server/embed.go` now builds only with `-tags embed`
  - new `internal/server/embed_none.go` defines `embeddedPublicFS = nil` for default builds

### Why
- The image widget needs a stable URL (`/api/images/{id}`) for each local file path so the browser can load images.
- Keeping embedding behind a build tag prevents “fresh clone” compilation failures and keeps dev workflows smooth.

### What worked
- `go test ./... -count=1` now succeeds in default builds without requiring generated `internal/server/embed/public`.
- Upload + serve is end-to-end “plumbed” at the API level (handlers exist and compile, ready for CLI integration next).

### What was tricky to build
- Correctly validating “this is an image” without overcomplicating it (we currently sniff the first 512 bytes via `http.DetectContentType`).
- Avoiding another “fake” implementation: the endpoints actually persist bytes to disk and serve them back.

### What warrants a second pair of eyes
- Security posture of `POST /api/images` (size limits, MIME sniffing edge cases, and any potential resource exhaustion).
- Cleanup semantics (ticker interval, expiry defaults, and whether we need tighter coupling to request expiry later).

### Code review instructions
- Start with `internal/server/server.go` and search for `handleImagesCollection` / `handleImagesItem`.
- Then review `internal/server/images.go` for storage and cleanup logic.
- Validate with: `go test ./... -count=1`

## Step 9: Add CLI-side UploadImage helper (streaming multipart)

This step added the client-side building block the `plz-confirm image` command will rely on: a helper in the Go HTTP client that can upload a local image file to the backend and receive back a stable URL (`/api/images/{id}`). The main design constraint here is avoiding buffering large files in memory, since high-res screenshots can be many megabytes.

**Commit (code):** 56d958f381ca39afe0b6fe720e99b4b7eaf59496 — "Client: add UploadImage helper"

### What I did
- Added `Client.UploadImage(ctx, filePath, ttlSeconds)` in `internal/client/client.go`.
- Implemented it using an `io.Pipe` + `multipart.Writer` goroutine so the request body streams instead of buffering the entire file into a `bytes.Buffer`.
- The helper posts to `POST /api/images` and decodes `{id,url,mimeType,size}`.

### Why
- The upcoming `plz-confirm image` CLI command needs to accept local file paths and convert them into browser-fetchable URLs.
- Streaming multipart keeps memory use predictable and avoids surprises for large files.

### What warrants a second pair of eyes
- Error propagation from the goroutine writing the multipart body (we use `CloseWithError`; reviewers should confirm we don’t leak goroutines on early HTTP failures).
- Whether we want to enforce client-side “file exists / size” checks before attempting the upload (currently we just try to open and stream).

### Code review instructions
- Start in `internal/client/client.go`, search for `UploadImage`.
- Validate with: `go test ./... -count=1`

## Step 10: Add image widget schemas (Go + TypeScript)

This step introduced the shared wire-level schemas for the new widget type. In plz-confirm, the server treats `input`/`output` as `any`, but the CLI and frontend still benefit from typed structs/interfaces so we keep request payloads predictable and avoid “shape drift”.

**Commit (code):** 41e469e2e314f2388b32fc9eac61ef41fd9cf086 — "Types: add image widget schemas"

### What I did
- Added `WidgetImage` and `ImageItem` / `ImageInput` / `ImageOutput` to `internal/types/types.go`.
- Updated `agent-ui-system/client/src/types/schemas.ts`:
  - extended `UIRequest.type` union with `'image'`
  - added `ImageItem`, `ImageInput`, and `ImageOutput` interfaces.

### Why
- The CLI needs a stable schema to marshal `input` for `POST /api/requests`.
- The frontend needs types so `WidgetRenderer` and `ImageDialog` can be implemented without guessing.

### What warrants a second pair of eyes
- Output shape decisions:
  - image-pick select mode: do we want `selected` to be indices (`number|number[]`) only?
  - “images as context + checkbox question” variant: do we return labels (`string[]`) or indices into `options[]`?
  - confirm mode: `boolean`
  - We currently allow both indices and labels in `ImageOutput.selected` to stay flexible, but we should lock this down once UI behavior is finalized.

## Step 11: Implement `plz-confirm image` CLI command

This step added the actual CLI entrypoint agents will call: `plz-confirm image`. The command handles local file paths by uploading them to the backend first (`POST /api/images`) and then creates a standard widget request (`POST /api/requests`) containing browser-fetchable image URLs.

**Commit (code):** 462ab1ac7ee8f44b1baea9527ff6c67114977c1f — "CLI: add image widget command"

### What I did
- Added `internal/cli/image.go` implementing the Glazed command:
  - Parses `--image` (repeatable) plus optional per-image metadata (`--image-label`, `--image-alt`, `--image-caption`).
  - Uploads local paths via `Client.UploadImage` and replaces them with `/api/images/{id}` URLs.
  - Creates the request as `type=image` and waits for completion, outputting `selected_json` + `timestamp`.
- Registered the command in `cmd/plz-confirm/main.go`.
- Fixed a `.gitignore` footgun: the pattern `plz-confirm` was unintentionally ignoring the directory `cmd/plz-confirm/`. Changed it to `/plz-confirm` so it only ignores the top-level binary.

### Why
- Without this CLI command, the feature isn’t usable by agents.
- The `.gitignore` fix is required so the command registration code can actually be tracked and reviewed.

### What warrants a second pair of eyes
- The “local path vs URL vs data:” heuristic (we currently treat non-`http(s)://` and non-`data:` as a local path).
- Whether we want stricter validation on `--mode` values (currently passed through as a string).

### Code review instructions
- Start with `internal/cli/image.go` (flag parsing + upload loop + request creation).
- Then review `cmd/plz-confirm/main.go` registration.
- Validate with: `go test ./... -count=1`

## Step 12: Implement ImageDialog in the web UI (variants A + B + confirm)

This step implemented the React side of the feature: rendering the image widget and emitting a structured output payload. The UI supports three interaction styles in one component: (1) select images directly (Variant A), (2) show images as context and answer a checkbox question below (Variant B), and (3) confirm mode with approve/reject buttons.

**Commit (code):** bae8d811bb4bcca56987ef9fe0679258f22a83f9 — "UI: add ImageDialog widget"

### What I did
- Added `agent-ui-system/client/src/components/widgets/ImageDialog.tsx`.
  - Renders images in a responsive grid (1 large, 2 side-by-side, 3+ grid).
  - Select mode:
    - Variant A: click-to-select image tiles (single or multi via `input.multi`)
    - Variant B: if `input.options[]` is present, render a checkbox list below the images and select text options
  - Confirm mode: approve/reject buttons, returns boolean.
  - Per-image error state (broken URL shows ERROR_LOADING but doesn’t block submission).
- Updated `WidgetRenderer.tsx` to route `active.type === 'image'` to `ImageDialog`.
- Ran `pnpm -C agent-ui-system check` (installed deps via `pnpm install --frozen-lockfile` first since this repo didn’t have node_modules).

### Why
- This is the user-visible part of the feature; without it the server/CLI work can’t be exercised.
- Keeping both Variant A and Variant B in one component ensures we can support both “pick an image” and “answer a question about images” flows.

### What warrants a second pair of eyes
- Output shape consistency: Variant A returns indices, Variant B returns strings (option labels). We should confirm this is what we want long-term.
- UI affordances: ensure the “click tile” selection UX feels consistent with `SelectDialog` styling and keyboard navigation expectations.

## Step 13: Document the image command + add a smoke script

This step turned the implementation into something other developers can actually use without reading code. I updated the user-facing docs (`README.md` and `pkg/doc/how-to-use.md`) to mention the new `plz-confirm image` command, and added a ticket-local smoke script that exercises the three primary flows (Variant A, Variant B, confirm).

**Commit (docs):** 375c846d30b2266ba659b6cb97cf6caaa34365e5 — "Docs: add image command docs and smoke script"

### What I did
- Updated `pkg/doc/how-to-use.md` to include an **Image Command** section with:
  - flag list
  - example calls for Variant A / Variant B / confirm
  - notes on output shape (`selected_json`)
- Updated `README.md` to:
  - include “image prompts” in the feature list
  - add `plz-confirm image` to the available commands
  - add a simple “Image Prompt” example snippet
- Added `ttmp/.../scripts/smoke-image-widget.sh` with runnable examples (expects a running server + UI).

### Why
- Without docs + examples, it’s easy to “have the feature” but still be unsure how to call it correctly.
- The smoke script is the fastest path for manual parity testing during UI development.

## Step 14: Add backend tests for /api/images

This step added basic tests for the new image upload/serve endpoints. The goal is not exhaustive coverage, but to lock down the main contract so future refactors don’t break the feature silently.

**Commit (code):** 8483bbac53310eea1ad37457e2648307654de894 — "Server: test /api/images upload+serve"

### What I did
- Added `internal/server/images_test.go`:
  - happy path: upload a tiny PNG header and then GET it back
  - rejection path: uploading non-image content returns 400
- Verified with `go test ./... -count=1`.

### What warrants a second pair of eyes
- The PNG “minimal bytes” assumption: we rely on `http.DetectContentType` recognizing the header.
- Whether we should also test expiry behavior (we currently set `ttlSeconds` and use cleanup on a ticker).

## Step 15: Add API-driven CLI smoke script (auto-submit responses)

This step captured the “run all CLI verbs, but auto-answer them” workflow into a real script file under the ticket’s `scripts/` folder. The earlier approach of pasting long one-off shell commands into chat is hard to tweak and doesn’t leave a clean trail for future debugging; putting the logic into a versioned script makes iteration and review much easier.

### What I did
- Added `scripts/auto-e2e-cli-via-api.sh` to the ticket folder.
- The script:
  - assumes the Go server is running and writes to a parseable logfile (default: `/tmp/plz-confirm-server.log`)
  - runs each CLI verb in the background (`go run ./cmd/plz-confirm <verb> ...`)
  - scrapes the created request id from the server log (`Created request <id> (<type>)`)
  - submits a response via `POST /api/requests/{id}/response`
  - waits for the CLI to print output
  - includes image-widget cases (Variant A, Variant B, confirm), plus a `/api/images` sanity upload

### Why
- Enables fast validation of request/response plumbing without needing browser clicks every time.
- Leaves a versioned trail for future debugging (“what exactly did we run?”).

### What warrants a second pair of eyes
- Log parsing robustness: if server log format changes, the script will need to be updated.
- Dependency assumptions: script expects `jq`, `curl`, `go`, and `base64`.

## Step 16: Run the tmux dev stack + execute the API-driven smoke script

This step validated that the system actually runs end-to-end in the intended dev topology: Go server on `:3001`, Vite UI on `:3000` proxying `/api` and `/ws`, and the CLI talking to the UI base URL (`http://localhost:3000`). The smoke script then exercised all CLI verbs and confirmed that the CLI unblocks and prints output once the server receives `/response`.

### What I did
- Started a tmux session with two windows (server + Vite), logging to:
  - `/tmp/plz-confirm-server.log`
  - `/tmp/plz-confirm-vite.log`
- Ran `scripts/auto-e2e-cli-via-api.sh`.

### What worked
- All CLI verbs completed successfully via auto-submitted responses:
  - `confirm`, `select`, `form`, `table`, `upload`, and `image` (Variant A / Variant B / confirm).
- The server log showed the expected lifecycle:
  - `[API] Created request <id> (<type>)`
  - `[API] Request <id> completed`

### What didn’t work (small)
- The script uses `curl -I` (HEAD) as a quick sanity check for `/api/images/{id}`, and the endpoint returned `405 Method Not Allowed` even though `GET` worked.

## Step 17: Allow HEAD requests for /api/images/{id}

This is a small HTTP compatibility fix: allowing `HEAD` makes it easier to validate endpoints with `curl -I` and matches common behavior for “static-ish” resources.

**Commit (code):** 13b380c57c827891049519a28865cc3f10907a81 — "Server: allow HEAD on /api/images/{id}"

### What I did
- Updated the image serving handler to accept `HEAD` in addition to `GET`.

### What worked
- After restarting the running server, `curl -I http://localhost:3001/api/images/{id}` returns `200 OK` with the expected headers.

## Step 18: Update help docs (how-to-use) + add a developer guide for new widgets

This step wrapped up the work with two documentation improvements aimed at onboarding. First, it corrected the main user-facing “how to use plz-confirm” page so it accurately reflects the new image widget and its local-file upload behavior. Second, it added a didactic developer guide that explains the full workflow for adding a complex widget end-to-end (types → server → CLI → UI → docs → tests).

**Commit (docs):** 150fc49480800fb0fccf445e3427126cb55c0169 — "Docs: add widget developer guide and update how-to-use"

### What I did
- Updated `pkg/doc/how-to-use.md`:
  - Corrected “five widget types” → “six widget types”
  - Added important context on dev topology (`:3001` Go backend + `:3000` Vite UI proxy)
  - Added an explicit explanation of local file upload for `plz-confirm image` via `/api/images`
- Added a new embedded help page: `pkg/doc/adding-widgets.md` (slug: `adding-widgets`)
  - Includes background, diagrams, pseudocode, and concrete file/symbol pointers
  - Uses the image widget as a worked example
- Verified the doc shows up in the help system using:
  - `go run ./cmd/plz-confirm help adding-widgets`

### What was tricky to build
- It’s easy to test help docs against a stale installed `plz-confirm` binary. The reliable approach is using `go run ./cmd/plz-confirm ...` so the embedded docs are always current.


