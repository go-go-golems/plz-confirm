---
Title: Image Widget Design Specification
Ticket: 001-ADD-IMG-WIDGET
Status: active
Topics:
    - cli
    - backend
    - agent-ui-system
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Complete design specification for the image widget feature, including all design decisions, rationale, and implementation details.
LastUpdated: 2025-12-25T00:00:00.000000000-05:00
WhatFor: Reference during implementation to understand design decisions and rationale
WhenToUse: Use when implementing the image widget to ensure consistency with design decisions
---

# Image Widget Design Specification

## Executive Summary

The image widget adds a new interaction pattern to plz-confirm: an agent can present a prompt *plus images* and collect a structured answer. This is the missing piece for workflows like “pick the best screenshot”, “select all images that contain X”, or “are these two images similar?” where text-only prompts are insufficient.

The design keeps plz-confirm’s existing request/response architecture intact: the CLI creates a request (JSON), the Go server stores/broadcasts it, and the React UI renders a widget and submits the user’s response. The only new “infrastructure” is an explicit image upload + serving API so local file paths can become URLs the browser can fetch.

**Key Design Decisions:**
- **Image sources**: Accept image **URLs**, **data URIs** (`data:image/...`), and **local file paths**.
- **Local files**: The CLI uploads file paths to `POST /api/images` and then uses returned `/api/images/{id}` URLs in the request payload.
- **Two UI modes**: `select` (single or multi-select) and `confirm` (approve/reject).
- **Responsive layout**: Render 1 image large, 2 images side-by-side, 3+ as a grid.
- **Failure-tolerant UI**: Per-image loading/error states so one broken image doesn’t brick the whole request.

## Goals / Non-Goals

This section clarifies scope so implementation stays focused. It also makes later “why didn’t we do X?” discussions faster.

**Goals**
- Let agents show a **title + optional message** plus one or more images.
- Support **select** (single/multi) and **confirm** (approve/reject) responses.
- Support **local files** (agent has a path) without forcing base64 payloads.
- Keep the UX consistent with existing widgets (`ConfirmDialog`, `SelectDialog`).

**Non-goals (for this ticket)**
- Turning the existing `upload` widget into a fully-real upload pipeline (it is currently simulated in the UI).
- Long-term/persistent media storage (we store images in a temp directory and clean them up).
- Image editing/annotation tooling (crop, draw, compare overlays).

## Problem Statement

AI agents need a way to:
- Display images alongside text prompts to users.
- Ask users to select images from a set (e.g., “Which image matches the description?”).
- Ask users to confirm similarity or other image-based questions (e.g., “Are these images similar?”).
- Receive structured feedback that can be consumed in scripts (JSON/YAML/table output).

Current plz-confirm widgets don’t support image display. Also, despite the name, the current React `UploadDialog` **simulates uploads** (it generates fake `/tmp/uploads/...` paths). That means there is no server-side binary upload API we can reuse yet; the image widget must introduce explicit upload/serve endpoints.

## Proposed Solution

We add a new widget type: `image`.

At a high level, it takes an input payload containing a prompt plus a list of images, renders them in the browser, and returns either a selection (indices) or a confirmation boolean. Local file paths work by uploading them first and using returned URLs in the request payload.

**Capabilities**
- **Inputs**: title, optional message, list of images (`src` + optional metadata), mode (`select`/`confirm`).
- **Outputs**:
  - select mode: index or list of indices
  - confirm mode: boolean

## Architecture Overview (data flow + lifecycle)

This widget intentionally follows the same lifecycle as the existing ones; the only new component is an image upload “side-channel” used by the CLI when it receives local file paths.

**Data flow (happy path)**

```text
CLI (agent)
  ├─ for each local image path: POST /api/images  ───────────────┐
  │                                                             │
  ├─ POST /api/requests { type:"image", input:{... urls ...} } ──┼─> Go server (store)
  │                                                             │
  ├─ GET /api/requests/{id}/wait  <──────────────────────────────┤
  │                                                             │
  └─ (prints structured output)                                 │
                                                                │
React UI (browser) <─ WS new_request ───────────────────────────┘
  └─ POST /api/requests/{id}/response { output:{...} }
```

**CLI pseudocode**

```pseudo
images = []
for raw in flags.image:
  if raw is local-file-path:
    uploadResp = POST /api/images (multipart file=raw)
    images.append({ src: uploadResp.url, label: maybeLabel, alt: maybeAlt, caption: maybeCaption })
  else:
    images.append({ src: raw, label: maybeLabel, alt: maybeAlt, caption: maybeCaption })

req = POST /api/requests { type:"image", input:{ title, message, mode, images, multi } }
done = GET /api/requests/{req.id}/wait
print(done.output)
```

**Backend pseudocode (upload + serve)**

```pseudo
POST /api/images:
  parse multipart form
  validate size <= MaxUploadBytes
  sniff/validate content-type is image/*
  write to temp dir under generated id
  return { id, url: "/api/images/{id}", mimeType, size }

GET /api/images/{id}:
  look up id -> file path + mime
  set Content-Type
  stream file
```

## ASCII UI Sketches (concrete “screenshots”)

These are intentionally low-fidelity. The goal is to lock down *layout and interaction*, not colors/typography.

### Select mode (single-select)

```text
┌───────────────────────────────────────────────────────────────┐
│ IMAGE REQUEST                                                  │
├───────────────────────────────────────────────────────────────┤
│ Title: Select the best screenshot                              │
│ Message: Pick the image that matches the final UI.             │
├───────────────────────────────────────────────────────────────┤
│  [ ] (1)  ┌──────────────┐    [ ] (2)  ┌──────────────┐        │
│           │   IMG_1      │             │   IMG_2      │        │
│           │  (loaded)    │             │  (loaded)    │        │
│           └──────────────┘             └──────────────┘        │
│                                                               │
│  Hint: click to select, Enter to submit                        │
├───────────────────────────────────────────────────────────────┤
│ Selected: 1                                                    │
│                                         [ CONFIRM_SELECTION ]  │
└───────────────────────────────────────────────────────────────┘
```

### Select mode (multi-select)

```text
┌───────────────────────────────────────────────────────────────┐
│ IMAGE REQUEST                                                  │
├───────────────────────────────────────────────────────────────┤
│ Title: Select all images that contain a cat                    │
│ Message: Choose every image that matches.                      │
├───────────────────────────────────────────────────────────────┤
│  [x] (1) ┌──────────┐  [ ] (2) ┌──────────┐  [x] (3) ┌──────┐ │
│          │  IMG_1   │          │  IMG_2   │          │IMG_3 │ │
│          └──────────┘          └──────────┘          └──────┘ │
│                                                               │
│  Tip: Space toggles selection, Tab moves focus                 │
├───────────────────────────────────────────────────────────────┤
│ Selected: 2                                                    │
│                                         [ CONFIRM_SELECTION ]  │
└───────────────────────────────────────────────────────────────┘
```

### Select mode (images as context + multi-select question below)

This is the “survey-style” variant: the images are shown *for context*, but the actual answer is a multi-select list of **text options** below the images. This is useful when the model wants to ask a categorical question about the images (e.g., “which problems do you see?”) rather than asking the user to pick images by index.

```text
┌───────────────────────────────────────────────────────────────┐
│ IMAGE REQUEST                                                  │
├───────────────────────────────────────────────────────────────┤
│ Title: Review these images                                     │
│ Message: Look at the screenshots first, then answer below.     │
├───────────────────────────────────────────────────────────────┤
│   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│   │    IMG_1     │  │    IMG_2     │  │    IMG_3     │         │
│   │   (loaded)   │  │   (loaded)   │  │   (loaded)   │         │
│   └──────────────┘  └──────────────┘  └──────────────┘         │
├───────────────────────────────────────────────────────────────┤
│ Question (multi-select): Which issues are present?             │
│                                                               │
│   [x] Text is too small                                        │
│   [ ] Button alignment is off                                  │
│   [x] Wrong color theme                                         │
│   [ ] Missing icon                                              │
│                                                               │
│ Tip: Space toggles, Enter submits                               │
├───────────────────────────────────────────────────────────────┤
│ Selected: 2                                                    │
│                                         [ SUBMIT_ANSWER ]       │
└───────────────────────────────────────────────────────────────┘
```

### Confirm mode (similarity check)

```text
┌───────────────────────────────────────────────────────────────┐
│ IMAGE REQUEST                                                  │
├───────────────────────────────────────────────────────────────┤
│ Title: Are these images similar?                               │
│ Message: Compare the two images and answer yes/no.             │
├───────────────────────────────────────────────────────────────┤
│   ┌──────────────────────┐      ┌──────────────────────┐      │
│   │        IMG_A         │      │        IMG_B         │      │
│   │      (loaded)        │      │      (loaded)        │      │
│   └──────────────────────┘      └──────────────────────┘      │
├───────────────────────────────────────────────────────────────┤
│ [ REJECT ]                                        [ APPROVE ] │
└───────────────────────────────────────────────────────────────┘
```

### Per-image error state (still allow interaction)

```text
┌───────────────────────────────────────────────────────────────┐
│ Title: Select the correct image                                │
├───────────────────────────────────────────────────────────────┤
│  [ ] (1) ┌──────────────┐   [ ] (2) ┌──────────────────────┐   │
│          │   IMG_1      │           │   ERROR_LOADING      │   │
│          │  (loaded)    │           │  (broken URL / 404)  │   │
│          └──────────────┘           └──────────────────────┘   │
├───────────────────────────────────────────────────────────────┤
│ Note: one image failed to load; selection still works.         │
└───────────────────────────────────────────────────────────────┘
```

## Design Decisions

### Decision 1: Image Serving Strategy

**Decision:** Hybrid approach supporting multiple image sources: URLs, base64 data URIs, and file upload + serving.

**Rationale:**
- **URLs**: Best for external/public images, no storage needed, efficient, no size limit
- **Base64 data URIs**: Useful as a pass-through when an agent already has a data URI (e.g. generated images). In practice, keep them reasonably small because they bloat JSON payloads.
- **File paths**: Uploaded to server and served via `/api/images/{id}` endpoint, supports large images efficiently

**Implementation:**
- CLI accepts `--image` flag with file paths, URLs, or base64 data URIs
- **File paths**: Uploaded to server via POST `/api/images`, returns image URL, then included in request
- **URLs**: Passed through as-is (no upload needed)
- **Base64 data URIs**: Passed through as-is (for convenience, e.g., generated images)

**Why upload all files instead of base64?**
- **Simpler**: Consistent handling for all files, no encoding/decoding
- **No size limits**: Supports any image size
- **Better performance**: No base64 overhead (~33% size increase)
- **Cleaner code**: One code path instead of two (base64 vs upload)
- **Two requests is fine**: Upload files first, then create request (sequential, not parallel)

**File Upload Infrastructure:**
- **Upload endpoint**: POST `/api/images` - accepts multipart/form-data, stores file, returns `{id, url}`
- **Storage**: Temporary directory (`os.TempDir()/plz-confirm-images/`) with unique filenames
- **Serving endpoint**: GET `/api/images/{id}` - serves file with proper Content-Type headers
- **Cleanup**: Files cleaned up when request expires (tied to request expiration time)
- **Security**: File type validation (image/* MIME types), size limits, path sanitization

**Why file upload + serving?**
- **Complexity is manageable**: ~180 lines of code (upload endpoint, storage, serving, cleanup)
- **Enables large images**: No 1MB limit, supports high-resolution images
- **Better performance**: No base64 encoding overhead (~33% size increase)
- **Simpler CLI usage**: Just pass file paths, server handles the rest
- **Consistent with upload widget pattern**: Similar infrastructure can be reused

**Size Limits:**
- File upload: Max 50MB per image (configurable, reasonable for high-res images)
- Base64 data URIs: No hard limit (user-provided, but recommend < 1MB for performance)
- URLs: No size limit (browser handles)
- Request JSON payload: Only contains URLs (small), no image data

### Decision 2: Image Input Format

**Decision:** Unified `ImageItem` structure with `src` (URL or data URI) and optional metadata.

**Structure:**
```go
type ImageItem struct {
    Src      string  `json:"src"`                // URL or data URI
    Alt      *string `json:"alt,omitempty"`      // Accessibility text
    Label    *string `json:"label,omitempty"`    // Display label (for select mode)
    Caption  *string `json:"caption,omitempty"`  // Optional caption
}
```

**Rationale:**
- Single `src` field simplifies frontend rendering
- Optional metadata supports accessibility and UX
- Label field enables select mode with labeled images
- Caption field allows additional context

**CLI Interface:**
```bash
# File path (always uploaded; request will contain an /api/images/{id} URL)
--image /path/to/image.jpg

# URL
--image https://example.com/image.png

# Base64 data URI
--image "data:image/png;base64,iVBORw0KGgo..."

# With label (for select mode)
--image /path/to/image.jpg --image-label "Option 1"
```

**Image Upload Flow:**
1. CLI processes all `--image` flags:
   - **File paths**: POST to `/api/images`, get back `{id, url}`, store URL
   - **URLs**: Pass through as-is
   - **Base64 data URIs**: Pass through as-is
2. CLI creates request with all image URLs in `ImageInput.Images[]`
3. Frontend receives image URLs (external URLs, `/api/images/{id}`, or data URIs)
4. Frontend loads images from URLs
5. Server cleans up uploaded files when request expires

**Why two requests?**
- Current API is JSON-only (`POST /api/requests` with JSON body)
- Uploading files first, then creating request is simpler than multipart/form-data
- Sequential requests are fine (upload → create request)
- Keeps API consistent with existing widgets

### Decision 3: Widget Modes

**Decision:** Two distinct modes: `select` and `confirm`.

**Select Mode:**
- Displays images with labels/options
- User selects one or more images
- Returns selected image indices or labels
- Similar to existing `select` widget but with images

**Confirm Mode:**
- Displays images with approve/reject buttons
- User confirms or rejects based on images
- Returns boolean confirmation
- Similar to existing `confirm` widget but with images

**Rationale:**
- Covers the two main use cases identified
- Consistent with existing widget patterns
- Simple to implement and understand
- Can be extended later with additional modes if needed

**Mode Selection:**
- Default: `select` mode (most common use case)
- Explicit via `--mode select` or `--mode confirm`
- Mode determines available options and output format

### Decision 4: Image Layout

**Decision:** Responsive grid layout that adapts to number of images.

**Layout Rules:**
- **1 image**: Large centered display (max-width: 800px)
- **2 images**: Side-by-side comparison (50% each, responsive)
- **3-4 images**: 2x2 grid
- **5+ images**: Responsive grid (3 columns on desktop, 2 on tablet, 1 on mobile)

**Rationale:**
- Single image: Large display for clarity
- Two images: Side-by-side for comparison (common similarity check use case)
- Multiple images: Grid for efficient space usage
- Responsive: Works on all screen sizes

**Alternative Considered:** Carousel/slider
- **Rejected**: More complex, less accessible, harder to compare multiple images

### Decision 5: Type Definitions

**Decision:** Add `WidgetImage` type and corresponding Input/Output structs.

**Go Types (`internal/types/types.go`):**
```go
const WidgetImage WidgetType = "image"

type ImageInput struct {
    Title    string      `json:"title"`
    Message  *string     `json:"message,omitempty"`
    Images   []ImageItem `json:"images"`
    Mode     string      `json:"mode"` // "select" or "confirm"
    Options  []string    `json:"options,omitempty"` // For select mode
    Multi    *bool       `json:"multi,omitempty"`   // For select mode
}

type ImageOutput struct {
    Selected   any    `json:"selected"`   // int | []int | bool
    Timestamp  string `json:"timestamp"`
}
```

**TypeScript Types (`schemas.ts`):**
```typescript
export interface ImageInput {
  title: string;
  message?: string;
  images: ImageItem[];
  mode: 'select' | 'confirm';
  options?: string[];
  multi?: boolean;
}

export interface ImageItem {
  src: string;        // URL or data URI
  alt?: string;
  label?: string;
  caption?: string;
}

export interface ImageOutput {
  selected: number | number[] | boolean;
  timestamp: string;
}
```

**Rationale:**
- Consistent with existing widget type patterns
- `selected` field is `any` in Go to support multiple types (int, []int, bool)
- TypeScript uses union types for type safety
- Timestamp included for consistency with other widgets

### Decision 6: CLI Command Interface

**Decision:** Single `image` command with mode selection and flexible image input.

**Command Structure:**
```bash
plz-confirm image \
  --title "Dialog Title" \
  --message "Optional message" \
  --image <path|url|data-uri> \
  [--image-label "Label"] \
  [--mode select|confirm] \
  [--option "Option 1"] \
  [--multi] \
  [--base-url URL] \
  [--timeout SECONDS] \
  [--wait-timeout SECONDS]
```

**Flags:**
- `--title` (required): Dialog title
- `--message` (optional): Prompt message/description
- `--image` (repeatable, required): Image source (path, URL, or data URI)
- `--image-label` (repeatable, optional): Label for corresponding image (for select mode)
- `--mode` (optional, default: "select"): Interaction mode
- `--option` (repeatable, optional): Option labels (for select mode, if not using image labels)
- `--multi` (optional): Allow multiple selections (select mode only)
- Common flags: `--base-url`, `--timeout`, `--wait-timeout`, `--output`

**Rationale:**
- Single command simplifies usage
- Mode selection via flag is explicit and clear
- Repeatable `--image` flag matches existing patterns (`--option` in select widget)
- Optional labels support both labeled and unlabeled image selection

**Example Usage:**
```bash
# Select mode: Which image matches?
plz-confirm image \
  --title "Select Matching Image" \
  --message "Which image best matches the description?" \
  --image /path/to/img1.jpg --image-label "Option 1" \
  --image /path/to/img2.jpg --image-label "Option 2" \
  --mode select

# Confirm mode: Are images similar?
plz-confirm image \
  --title "Image Similarity Check" \
  --message "Are these images similar?" \
  --image https://example.com/img1.jpg \
  --image https://example.com/img2.jpg \
  --mode confirm

# Multi-select: Select all relevant images
plz-confirm image \
  --title "Select Relevant Images" \
  --image /path/to/img1.jpg \
  --image /path/to/img2.jpg \
  --image /path/to/img3.jpg \
  --mode select \
  --multi
```

### Decision 7: Frontend Component Structure

**Decision:** Single `ImageDialog` component with mode-based rendering.

**Component Props:**
```typescript
interface ImageDialogProps {
  requestId: string;
  input: ImageInput;
  onSubmit: (output: ImageOutput) => Promise<void>;
  loading?: boolean;
}
```

**Component Structure:**
- Header: Title and message
- Image Grid: Responsive image display
- Interaction Area: Selection or confirmation buttons
- Footer: Submit button and status

**State Management:**
- Selected indices (for select mode)
- Confirmation state (for confirm mode)
- Image loading states
- Error states

**Rationale:**
- Single component reduces code duplication
- Mode-based rendering keeps logic clear
- Consistent with other widget components
- State management handles user interaction

### Decision 8: Image Loading and Error Handling

**Decision:** Progressive loading with error states and fallbacks.

**Loading Strategy:**
- Show placeholder/skeleton while loading
- Load images in parallel
- Show individual error states for failed images
- Allow submission even if some images fail (with warning)

**Error Handling:**
- Invalid image format: Show error message, disable that image
- Network error (URLs): Show retry option or error message
- File read error: Show error in CLI, don't create request
- Size limit exceeded: Show error in CLI

**Rationale:**
- Progressive loading improves perceived performance
- Individual error states prevent one failure from blocking all images
- Graceful degradation maintains usability

### Decision 9: Accessibility

**Decision:** Full accessibility support with ARIA labels and keyboard navigation.

**Requirements:**
- Alt text for all images (from `alt` field or auto-generated)
- Keyboard navigation for selection
- Screen reader announcements
- Focus management
- ARIA labels on interactive elements

**Rationale:**
- Essential for inclusive design
- Required for production use
- Consistent with web accessibility standards

**Implementation:**
- Use `alt` field from `ImageItem` when provided
- Auto-generate alt text from labels or indices when missing
- Keyboard shortcuts: Arrow keys for navigation, Space/Enter for selection
- Focus visible indicators

### Decision 10: Output Format

**Decision:** Structured output with selected values and timestamp.

**Select Mode Output:**
- Single select: `{ "selected": 0, "timestamp": "..." }` (index)
- Multi select: `{ "selected": [0, 2], "timestamp": "..." }` (indices)

**Confirm Mode Output:**
- `{ "selected": true, "timestamp": "..." }` (boolean)

**CLI Output:**
- Table format: `request_id`, `selected`, `timestamp`
- JSON format: Full output object
- YAML format: Full output object

**Rationale:**
- Consistent with other widgets (timestamp included)
- Indices are more reliable than labels (labels might change)
- Boolean for confirm mode is clear and simple
- Multiple output formats support different use cases

## Alternatives Considered

### Alternative 1: Separate Commands (`image-select`, `image-confirm`)

**Rejected Because:**
- More commands to maintain
- Duplication of common logic
- Less flexible (can't easily add new modes)

**Chosen Approach:** Single command with mode flag is more flexible and maintainable.

### Alternative 2: Base64 for Small Files, Upload for Large Files

**Rejected Because:**
- Adds complexity (two code paths: base64 vs upload)
- Arbitrary threshold (500KB) is confusing
- Base64 encoding overhead (~33% size increase) even for small files
- Two HTTP requests (upload → create request) is fine and simpler

**Chosen Approach:** Upload all file paths, keep base64 data URIs for convenience (e.g., generated images). Single code path, no size limits, better performance.

### Alternative 3: Carousel/Slider Layout

**Rejected Because:**
- Less accessible
- Harder to compare multiple images
- More complex implementation

**Chosen Approach:** Responsive grid layout is simpler and more accessible.

### Alternative 4: Separate Image Widget Types (`image-select`, `image-confirm`)

**Rejected Because:**
- Inconsistent with existing widget pattern (single type per widget)
- More type definitions to maintain
- Less flexible

**Chosen Approach:** Single `image` widget type with mode field is consistent and flexible.

## Implementation Plan

### Phase 1: Backend Infrastructure

1. **Add image storage** (`internal/server/images.go` - NEW)
   - Create `ImageStore` struct with temp directory management
   - Implement `Upload()` method (store file, return ID)
   - Implement `Get()` method (retrieve file by ID)
   - Implement `Cleanup()` method (remove expired files)
   - Add file validation (MIME type, size limits)

2. **Add image endpoints** (`internal/server/server.go`)
   - Add POST `/api/images` handler (multipart upload)
   - Add GET `/api/images/{id}` handler (serve file)
   - Integrate `ImageStore` into `Server` struct
   - Add cleanup goroutine (runs periodically, cleans expired images)

3. **Add Go types** (`internal/types/types.go`)
   - Add `WidgetImage` constant
   - Add `ImageItem` struct
   - Add `ImageInput` struct
   - Add `ImageOutput` struct

### Phase 2: CLI Command

4. **Create CLI command** (`internal/cli/image.go` - NEW)
   - Implement `ImageCommand` struct
   - Implement `ImageSettings` struct
   - Implement `NewImageCommand()` with parameter definitions
   - Implement `RunIntoGlazeProcessor()` with:
     - File path handling (upload all files via POST `/api/images`)
     - URL validation
     - Base64 data URI validation
     - Image upload via HTTP client (multipart/form-data)
     - Request creation with image URLs
     - Request waiting and output

5. **Register command** (`cmd/plz-confirm/main.go`)
   - Add `imageCmd` registration

### Phase 3: Frontend Types and Component

4. **Add TypeScript types** (`agent-ui-system/client/src/types/schemas.ts`)
   - Add `'image'` to `UIRequest.type` union
   - Add `ImageInput` interface
   - Add `ImageItem` interface
   - Add `ImageOutput` interface

5. **Create ImageDialog component** (`agent-ui-system/client/src/components/widgets/ImageDialog.tsx`)
   - Implement image grid layout
   - Implement select mode (single and multi)
   - Implement confirm mode
   - Add image loading states
   - Add error handling
   - Add accessibility support

6. **Update WidgetRenderer** (`agent-ui-system/client/src/components/WidgetRenderer.tsx`)
   - Add `'image'` case to switch statement
   - Import `ImageDialog`

### Phase 4: Testing and Documentation

7. **CLI Testing**
   - Test file path handling
   - Test URL handling
   - Test base64 data URI handling
   - Test size limits
   - Test select mode (single/multi)
   - Test confirm mode
   - Test error cases

8. **Frontend Testing**
   - Test image loading
   - Test selection interaction
   - Test confirmation interaction
   - Test responsive layout
   - Test error states
   - Test accessibility

9. **Integration Testing**
   - End-to-end flow: CLI → Backend → Frontend → Response
   - Test WebSocket broadcasting
   - Test with various image sources

10. **Documentation**
    - Update `pkg/doc/how-to-use.md`
    - Add examples to README
    - Document image size limits and formats

## Security Considerations

1. **File Path Validation**
   - Validate file paths to prevent directory traversal
   - Check file size before reading
   - Validate file extensions (optional, for safety)

2. **URL Validation**
   - Validate URL format
   - Consider CORS restrictions (browser handles)
   - Consider rate limiting for external URLs

3. **Base64 Validation**
   - Validate base64 format
   - Check data URI format (`data:image/...;base64,...`)
   - Enforce size limits

4. **Payload Size Limits**
   - Max 1MB per image (base64)
   - Max 10MB total payload
   - Reject oversized requests

## Performance Considerations

1. **Image Optimization**
   - Recommend image compression for large images
   - Consider WebP format for better compression
   - Lazy loading for many images (future enhancement)

2. **Payload Size**
   - Base64 encoding increases size by ~33%
   - 1MB image becomes ~1.33MB in base64
   - Total payload limit prevents memory issues

3. **Network Performance**
   - URLs load asynchronously (browser handles)
   - Base64 embedded images load immediately
   - Consider CDN for external URLs

## Future Enhancements

1. **Image Upload + Serving**
   - Add `/api/images/{id}` endpoint
   - Implement file storage and cleanup
   - Support larger images via upload

2. **Advanced Layouts**
   - Carousel/slider for many images
   - Side-by-side comparison mode
   - Thumbnail grid with lightbox

3. **Image Annotations**
   - Drawing/annotation tools
   - Crop/rotate tools
   - Text overlays

4. **Image Comparison**
   - Side-by-side diff view
   - Overlay comparison
   - Similarity score display

## Related Documents

- [Analysis Document](../analysis/01-image-widget-implementation-analysis.md) - Architecture analysis and implementation requirements
- [Diary](../reference/01-diary.md) - Research and decision tracking

## Open Questions (Resolved)

1. ✅ **Image Storage:** Upload all file paths, URLs and base64 data URIs pass through
2. ✅ **Image Size Limits:** 50MB per image (upload), no limit for URLs/base64 (browser handles)
3. ✅ **Image Formats:** All browser-supported formats (JPEG, PNG, WebP, SVG, GIF)
4. ✅ **Layout Options:** Responsive grid with mode-specific layouts
5. ✅ **Accessibility:** Full support with alt text, keyboard navigation, ARIA labels
6. ✅ **Performance:** Upload all files (no base64 overhead), sequential requests (upload → create request)
