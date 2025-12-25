---
Title: Image Widget Implementation Analysis
Ticket: 001-ADD-IMG-WIDGET
Status: active
Topics:
    - cli
    - backend
    - agent-ui-system
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Analysis of the plz-confirm codebase to understand how to add a new image widget that allows models to display text prompts with one or more images, and receive user feedback via select/multi-select or confirm buttons.
LastUpdated: 2025-12-24T19:10:54.676145628-05:00
WhatFor: Understanding the architecture and implementation requirements for adding an image widget
WhenToUse: Reference when implementing the image widget feature
---

# Image Widget Implementation Analysis

## Overview

This document analyzes the plz-confirm codebase to understand how to add a new **image widget** that allows AI models to:
- Display a text prompt with one or more images
- Present selection options (single-select, multi-select) or confirmation buttons
- Receive user feedback about image similarity, selection, or confirmation

## Current Widget Architecture

### Widget Types

plz-confirm currently supports five widget types:

1. **confirm** - Yes/no confirmation dialogs
2. **select** - Single or multi-select menus
3. **form** - JSON Schema-based forms
4. **upload** - File upload dialogs
5. **table** - Data table with selection

### Architecture Pattern

All widgets follow a consistent three-layer architecture:

1. **CLI Layer** (`internal/cli/`) - Command-line interface for agents
2. **Backend Layer** (`internal/server/`) - HTTP API + WebSocket server
3. **Frontend Layer** (`agent-ui-system/client/`) - React-based web UI

### Communication Flow

```
Agent (CLI) → HTTP POST /api/requests → Backend Store → WebSocket Broadcast → Frontend UI
                                                              ↓
Agent (CLI) ← HTTP GET /api/requests/{id}/wait ← Backend Store ← WebSocket POST ← User Response
```

## Implementation Components

### 1. Type Definitions

**Location:** `internal/types/types.go`

Widget types are defined as constants:
```go
type WidgetType string

const (
    WidgetConfirm WidgetType = "confirm"
    WidgetSelect  WidgetType = "select"
    WidgetForm    WidgetType = "form"
    WidgetUpload  WidgetType = "upload"
    WidgetTable   WidgetType = "table"
)
```

Each widget has corresponding Input/Output structs:
- `ConfirmInput` / `ConfirmOutput`
- `SelectInput` / `SelectOutput`
- `FormInput` / `FormOutput`
- `UploadInput` / `UploadOutput`
- `TableInput` / `TableOutput`

**For Image Widget:** Need to add:
- `WidgetImage` constant
- `ImageInput` struct (title, message, images[], mode: "select"|"confirm", options[]?)
- `ImageOutput` struct (selected: string|string[]|boolean, timestamp)

### 2. CLI Command Implementation

**Pattern:** Each widget has a command file in `internal/cli/`:
- `confirm.go`
- `select.go`
- `form.go`
- `upload.go`
- `table.go`

**Structure:**
1. Command struct embedding `*cmds.CommandDescription`
2. Settings struct with `glazed.parameter` tags
3. `New*Command()` constructor that defines parameters
4. `RunIntoGlazeProcessor()` method that:
   - Parses settings from layers
   - Creates client and builds input struct
   - Calls `client.CreateRequest()`
   - Waits for response via `client.WaitRequest()`
   - Outputs result as rows via `gp.AddRow()`

**Example (SelectCommand):**
```go
type SelectCommand struct {
    *cmds.CommandDescription
}

type SelectSettings struct {
    BaseURL     string `glazed.parameter:"base-url"`
    TimeoutS    int    `glazed.parameter:"timeout"`
    WaitTimeout int    `glazed.parameter:"wait-timeout"`
    Title       string `glazed.parameter:"title"`
    Options     []string `glazed.parameter:"option"`
    Multi       bool   `glazed.parameter:"multi"`
    Searchable  bool   `glazed.parameter:"searchable"`
}
```

**For Image Widget:** Need to create `internal/cli/image.go` with:
- `ImageCommand` struct
- `ImageSettings` struct with flags for:
  - `--title` (required)
  - `--message` (optional)
  - `--image` (repeatable, file paths or URLs)
  - `--mode` (select|confirm, default: select)
  - `--option` (repeatable, for select mode)
  - `--multi` (for select mode)
  - Common flags: `--base-url`, `--timeout`, `--wait-timeout`

### 3. Command Registration

**Location:** `cmd/plz-confirm/main.go`

Commands are registered in `main()`:
```go
confirmCmd, err := agentcli.NewConfirmCommand(layersList...)
cobraConfirmCmd, err := glazed_cli.BuildCobraCommand(confirmCmd, ...)
rootCmd.AddCommand(cobraConfirmCmd)
```

**For Image Widget:** Add similar registration for `imageCmd`.

### 4. Backend Server

**Location:** `internal/server/server.go`

The server handles:
- HTTP POST `/api/requests` - Create new request
- HTTP GET `/api/requests/{id}` - Get request status
- HTTP POST `/api/requests/{id}/response` - Submit response
- HTTP GET `/api/requests/{id}/wait` - Long-poll for completion
- WebSocket `/ws` - Real-time updates

**Key Methods:**
- `handleCreateRequest()` - Creates request in store, broadcasts via WebSocket
- `handleSubmitResponse()` - Completes request, broadcasts completion
- `handleWait()` - Long-polling wait mechanism

**For Image Widget:** No backend changes needed - the server is widget-agnostic. It stores `Input` as `any` and forwards it via WebSocket.

### 5. Store Implementation

**Location:** `internal/store/store.go`

In-memory store with:
- `Create()` - Create new request
- `Get()` - Get request by ID
- `Complete()` - Mark request as completed with output
- `Wait()` - Wait for request completion (event-driven via channels)
- `Pending()` - Get all pending requests

**For Image Widget:** No changes needed - store is type-agnostic.

### 6. Frontend Widget Renderer

**Location:** `agent-ui-system/client/src/components/WidgetRenderer.tsx`

Central widget dispatcher:
```tsx
const renderWidget = () => {
  switch (active.type) {
    case 'confirm':
      return <ConfirmDialog {...commonProps} />;
    case 'select':
      return <SelectDialog {...commonProps} />;
    // ... other widgets
    default:
      return <div>ERROR: UNKNOWN_WIDGET_TYPE</div>;
  }
};
```

**For Image Widget:** Need to:
1. Add `'image'` case to switch statement
2. Create `ImageDialog` component

### 7. Frontend Widget Component

**Location:** `agent-ui-system/client/src/components/widgets/`

Each widget has its own component:
- `ConfirmDialog.tsx`
- `SelectDialog.tsx`
- `FormDialog.tsx`
- `UploadDialog.tsx`
- `TableDialog.tsx`

**Common Pattern:**
- Props: `requestId`, `input`, `onSubmit`, `loading`
- State management for user interaction
- Submit handler calls `onSubmit(output)`
- Styled with Tailwind CSS and cyberpunk theme

**For Image Widget:** Need to create `ImageDialog.tsx` with:
- Display text prompt (title + message)
- Display one or more images (from URLs or base64)
- Mode: "select" - show options with images, allow single/multi-select
- Mode: "confirm" - show images with approve/reject buttons
- Submit selected options or confirmation

### 8. TypeScript Type Definitions

**Location:** `agent-ui-system/client/src/types/schemas.ts`

Defines TypeScript interfaces matching Go types:
```typescript
export interface UIRequest {
  type: 'confirm' | 'select' | 'form' | 'upload' | 'table';
  // ...
}

export interface SelectInput {
  title: string;
  options: string[];
  multi?: boolean;
  searchable?: boolean;
}
```

**For Image Widget:** Need to add:
- `'image'` to `UIRequest.type` union
- `ImageInput` interface
- `ImageOutput` interface

## Image Handling Considerations

### Image Sources

Images can come from:
1. **File paths** - CLI passes local file paths, need to serve via HTTP
2. **URLs** - Direct URLs to images (external or internal)
3. **Base64 data URIs** - Embedded image data

### Image Serving Strategy

**Option 1: Base64 Embedding**
- Pros: No additional server endpoints needed, works offline
- Cons: Large payloads, inefficient for large images

**Option 2: File Upload + URL Serving**
- Pros: Efficient, supports large images
- Cons: Requires file storage and serving infrastructure

**Option 3: External URLs**
- Pros: Simple, no storage needed
- Cons: Requires images to be publicly accessible

**Recommendation:** Support multiple approaches:
- Accept `--image` flag with file paths (upload to temp storage, serve via `/api/images/{id}`)
- Accept `--image-url` flag for external URLs
- Accept base64 data URIs in input JSON

### Image Storage

For file-based images:
- Upload files to temporary storage on server
- Generate unique IDs for each image
- Serve via `/api/images/{id}` endpoint
- Clean up expired images (based on request expiration)

## Implementation Requirements

### CLI Command

**Command:** `plz-confirm image`

**Flags:**
- `--title` (required) - Dialog title
- `--message` (optional) - Prompt message
- `--image` (repeatable) - Image file path or URL
- `--mode` (select|confirm) - Interaction mode (default: select)
- `--option` (repeatable, for select mode) - Option labels
- `--multi` (for select mode) - Allow multiple selections
- Common: `--base-url`, `--timeout`, `--wait-timeout`, `--output`

**Example Usage:**
```bash
# Select mode with images
plz-confirm image \
  --title "Which image matches?" \
  --message "Select the image that best matches the description" \
  --image /path/to/image1.jpg \
  --image /path/to/image2.jpg \
  --mode select \
  --option "Image 1" \
  --option "Image 2" \
  --multi

# Confirm mode with images
plz-confirm image \
  --title "Are these images similar?" \
  --message "Review the images and confirm if they are similar" \
  --image https://example.com/img1.jpg \
  --image https://example.com/img2.jpg \
  --mode confirm
```

### Frontend Component

**Component:** `ImageDialog.tsx`

**Features:**
- Display title and message
- Render images in grid or carousel layout
- Select mode: Show images with labels, allow selection
- Confirm mode: Show images with approve/reject buttons
- Handle image loading states and errors
- Submit selected indices or confirmation boolean

**Layout Considerations:**
- Single image: Large display
- Multiple images: Grid layout (responsive)
- With options: Image + label pairs
- Similarity check: Side-by-side comparison

## Key Files to Modify/Create

### Go Backend

1. **`internal/types/types.go`**
   - Add `WidgetImage` constant
   - Add `ImageInput` struct
   - Add `ImageOutput` struct

2. **`internal/cli/image.go`** (NEW)
   - Create `ImageCommand` struct
   - Create `ImageSettings` struct
   - Implement `NewImageCommand()`
   - Implement `RunIntoGlazeProcessor()`

3. **`cmd/plz-confirm/main.go`**
   - Register `imageCmd` in `main()`

4. **`internal/server/server.go`** (OPTIONAL)
   - Add `/api/images/{id}` endpoint for serving uploaded images
   - Add image upload handler if supporting file paths

### Frontend

1. **`agent-ui-system/client/src/types/schemas.ts`**
   - Add `'image'` to `UIRequest.type`
   - Add `ImageInput` interface
   - Add `ImageOutput` interface

2. **`agent-ui-system/client/src/components/WidgetRenderer.tsx`**
   - Add `'image'` case to switch statement
   - Import `ImageDialog`

3. **`agent-ui-system/client/src/components/widgets/ImageDialog.tsx`** (NEW)
   - Create component with image display logic
   - Implement select and confirm modes
   - Handle image loading and errors

## Testing Considerations

1. **CLI Testing**
   - Test with file paths
   - Test with URLs
   - Test with base64 data
   - Test select mode (single/multi)
   - Test confirm mode
   - Test error handling (invalid images, network errors)

2. **Frontend Testing**
   - Test image loading
   - Test selection interaction
   - Test confirmation interaction
   - Test responsive layout
   - Test error states

3. **Integration Testing**
   - End-to-end flow: CLI → Backend → Frontend → Response
   - Test WebSocket broadcasting
   - Test image serving (if implemented)

## Related Widgets for Reference

### Select Widget
- Similar selection logic
- Reference: `SelectDialog.tsx` for selection UI patterns

### Confirm Widget
- Similar confirmation logic
- Reference: `ConfirmDialog.tsx` for button layout

### Upload Widget
- Similar file handling
- Reference: `UploadDialog.tsx` for file/image display patterns

## Open Questions

1. **Image Storage:** Should we implement file upload/storage, or rely on external URLs?
2. **Image Size Limits:** What are reasonable limits for embedded vs. served images?
3. **Image Formats:** Which formats should we support? (JPEG, PNG, WebP, SVG?)
4. **Layout Options:** Should we support different layouts (grid, carousel, side-by-side)?
5. **Accessibility:** How to handle alt text and screen reader support?
6. **Performance:** How to handle many large images efficiently?

## Next Steps

1. Design the `ImageInput` and `ImageOutput` structs in detail
2. Decide on image serving strategy (base64 vs. file upload vs. URLs)
3. Implement CLI command (`internal/cli/image.go`)
4. Register command in `main.go`
5. Update TypeScript types
6. Create `ImageDialog` component
7. Update `WidgetRenderer` to include image widget
8. Test end-to-end flow
9. Add documentation
