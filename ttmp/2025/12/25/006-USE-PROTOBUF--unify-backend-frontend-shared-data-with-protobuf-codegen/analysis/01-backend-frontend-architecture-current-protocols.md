---
Title: Backend↔Frontend architecture + current protocols
Ticket: 006-USE-PROTOBUF
Status: active
Topics:
    - backend
    - frontend
    - api
    - protobuf
    - codegen
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-25T22:56:37.872994249-05:00
WhatFor: ""
WhenToUse: ""
---

# Backend↔Frontend architecture + current protocols

## Executive Summary

plz-confirm uses a Go backend (`internal/server`) serving a React frontend (`agent-ui-system/client`) via REST APIs and WebSocket. **All shared data structures are manually duplicated** between Go (`internal/types/types.go`) and TypeScript (`agent-ui-system/client/src/types/schemas.ts`), with `UIRequest.Input` and `UIRequest.Output` stored as `any` (JSON). This analysis documents the current architecture, API contracts, and data flow to inform a protobuf-based unification strategy.

## Current Architecture

### Components

1. **Go Backend** (`plz-confirm/internal/server/`)
   - HTTP server with REST API + WebSocket
   - In-memory request store (`internal/store/store.go`)
   - Image storage (`internal/server/images.go`)
   - Serves static files in production (embedded filesystem)

2. **Go CLI** (`plz-confirm/internal/cli/`)
   - Commands for each widget type (confirm, select, form, upload, table, image)
   - Uses HTTP client (`internal/client/client.go`) to communicate with server
   - Performs double marshal/unmarshal to convert `any` → typed structs

3. **React Frontend** (`plz-confirm/agent-ui-system/client/`)
   - TypeScript/React UI with Redux store
   - WebSocket client for real-time updates
   - Widget components for each request type

### Communication Flow

```
CLI → HTTP POST /api/requests → Server → Store
                                    ↓
                              WebSocket broadcast
                                    ↓
Frontend ← WebSocket message ← Server
    ↓
User interaction
    ↓
Frontend → HTTP POST /api/requests/{id}/response → Server → Store
                                                          ↓
                                                    WebSocket broadcast
                                                          ↓
CLI ← HTTP GET /api/requests/{id}/wait ← Server ← Store (completion)
```

## API Endpoints

### REST API

**Base URL:** `http://localhost:3000` (default)

#### Create Request
```
POST /api/requests
Content-Type: application/json

Request Body:
{
  "type": "confirm" | "select" | "form" | "upload" | "table" | "image",
  "sessionId": string,  // Ignored by server (kept for compatibility)
  "input": any,         // Widget-specific input (JSON object)
  "timeout": number     // Expiration in seconds (default: 300)
}

Response: 201 Created
{
  "id": string,
  "type": string,
  "sessionId": string,
  "input": any,
  "status": "pending",
  "createdAt": string,    // RFC3339Nano timestamp
  "expiresAt": string,    // RFC3339Nano timestamp
  "output": null
}
```

**Implementation:** `plz-confirm/internal/server/server.go:183-213`

#### Get Request
```
GET /api/requests/{id}

Response: 200 OK
{
  "id": string,
  "type": string,
  "sessionId": string,
  "input": any,
  "output": any | null,
  "status": "pending" | "completed" | "timeout" | "error",
  "createdAt": string,
  "completedAt": string | null,
  "expiresAt": string,
  "error": string | null
}
```

**Implementation:** `plz-confirm/internal/server/server.go:238-248`

#### Submit Response
```
POST /api/requests/{id}/response
Content-Type: application/json

Request Body:
{
  "output": any  // Widget-specific output (JSON object)
}

Response: 200 OK
{
  // Same as GET /api/requests/{id}, but with status="completed"
  // and output populated
}
```

**Implementation:** `plz-confirm/internal/server/server.go:276-305`

#### Wait for Completion (Long-poll)
```
GET /api/requests/{id}/wait?timeout=60

Response: 200 OK (when completed)
{
  // Same as GET /api/requests/{id} with status="completed"
}

Response: 408 Request Timeout (if timeout exceeded)
```

**Implementation:** `plz-confirm/internal/server/server.go:307-333`

#### Upload Image
```
POST /api/images
Content-Type: multipart/form-data

Form Fields:
- file: File (image/*)
- ttlSeconds: number (optional, default: 3600)

Response: 201 Created
{
  "id": string,
  "url": string,        // "/api/images/{id}"
  "mimeType": string,
  "size": number
}
```

**Implementation:** `plz-confirm/internal/server/server.go:344-413`

#### Get Image
```
GET /api/images/{id}

Response: 200 OK
Content-Type: image/*
Cache-Control: private, max-age=60

[Image binary data]
```

**Implementation:** `plz-confirm/internal/server/server.go:415-469`

### WebSocket API

**Endpoint:** `WS /ws?sessionId={id}`

**Connection:** `plz-confirm/internal/server/ws.go:66-100`

**Messages (server → client):**

1. **New Request**
```json
{
  "type": "new_request",
  "request": {
    // UIRequest object
  }
}
```

**Broadcast:** `plz-confirm/internal/server/server.go:206-209`

2. **Request Completed**
```json
{
  "type": "request_completed",
  "request": {
    // UIRequest object (with status="completed")
  }
}
```

**Broadcast:** `plz-confirm/internal/server/server.go:298-301`

**Client Implementation:** `plz-confirm/agent-ui-system/client/src/services/websocket.ts:32-56`

## Data Structures

### Core Types

#### UIRequest

**Go:** `plz-confirm/internal/types/types.go:32-43`
**TypeScript:** `plz-confirm/agent-ui-system/client/src/types/schemas.ts:1-12`

```go
type UIRequest struct {
    ID          string        `json:"id"`
    Type        WidgetType    `json:"type"`
    SessionID   string        `json:"sessionId"`
    Input       any           `json:"input"`              // Widget-specific input
    Output      any           `json:"output,omitempty"`   // Widget-specific output
    Status      RequestStatus `json:"status"`
    CreatedAt   string        `json:"createdAt"`         // RFC3339Nano
    CompletedAt *string       `json:"completedAt,omitempty"`
    ExpiresAt   string        `json:"expiresAt"`         // RFC3339Nano
    Error       *string       `json:"error,omitempty"`
}
```

**Key Issue:** `Input` and `Output` are `any` (JSON), requiring type assertions/marshaling on both sides.

#### RequestStatus Enum

**Go:** `plz-confirm/internal/types/types.go:8-15`
**TypeScript:** `plz-confirm/agent-ui-system/client/src/types/schemas.ts:7` (union type)

```go
type RequestStatus string

const (
    StatusPending   RequestStatus = "pending"
    StatusCompleted RequestStatus = "completed"
    StatusTimeout   RequestStatus = "timeout"
    StatusError     RequestStatus = "error"
)
```

#### WidgetType Enum

**Go:** `plz-confirm/internal/types/types.go:17-26`
**TypeScript:** `plz-confirm/agent-ui-system/client/src/types/schemas.ts:3` (union type)

```go
type WidgetType string

const (
    WidgetConfirm WidgetType = "confirm"
    WidgetSelect  WidgetType = "select"
    WidgetForm    WidgetType = "form"
    WidgetUpload  WidgetType = "upload"
    WidgetTable   WidgetType = "table"
    WidgetImage   WidgetType = "image"
)
```

### Widget Input/Output Types

#### Confirm Widget

**Go:** `plz-confirm/internal/types/types.go:45-56`
**TypeScript:** `plz-confirm/agent-ui-system/client/src/types/schemas.ts:14-25`

```go
type ConfirmInput struct {
    Title       string  `json:"title"`
    Message     *string `json:"message,omitempty"`
    ApproveText *string `json:"approveText,omitempty"`
    RejectText  *string `json:"rejectText,omitempty"`
}

type ConfirmOutput struct {
    Approved  bool    `json:"approved"`
    Timestamp string  `json:"timestamp"`  // ISO 8601
    Comment   *string `json:"comment,omitempty"`
}
```

#### Select Widget

**Go:** `plz-confirm/internal/types/types.go:58-68`
**TypeScript:** `plz-confirm/agent-ui-system/client/src/types/schemas.ts:27-37`

```go
type SelectInput struct {
    Title      string   `json:"title"`
    Options    []string `json:"options"`
    Multi      *bool    `json:"multi,omitempty"`
    Searchable *bool    `json:"searchable,omitempty"`
}

type SelectOutput struct {
    Selected any     `json:"selected"`  // string | []string
    Comment  *string `json:"comment,omitempty"`
}
```

**Issue:** `Selected` is a union type (`string | []string`), handled as `any` in Go.

#### Form Widget

**Go:** `plz-confirm/internal/types/types.go:70-78`
**TypeScript:** `plz-confirm/agent-ui-system/client/src/types/schemas.ts:39-47`

```go
type FormInput struct {
    Title  string `json:"title"`
    Schema any    `json:"schema"`  // JSON Schema (dynamic)
}

type FormOutput struct {
    Data    any     `json:"data"`  // Dynamic form results
    Comment *string `json:"comment,omitempty"`
}
```

**Issue:** `Schema` and `Data` are `any` (JSON Schema is inherently dynamic).

#### Upload Widget

**Go:** `plz-confirm/internal/types/types.go:80-98`
**TypeScript:** `plz-confirm/agent-ui-system/client/src/types/schemas.ts:49-65`

```go
type UploadInput struct {
    Title       string   `json:"title"`
    Accept      []string `json:"accept,omitempty"`
    Multiple    *bool    `json:"multiple,omitempty"`
    MaxSize     *int64   `json:"maxSize,omitempty"`
    CallbackURL *string  `json:"callbackUrl,omitempty"`
}

type UploadOutput struct {
    Files   []UploadedFile `json:"files"`
    Comment *string        `json:"comment,omitempty"`
}

type UploadedFile struct {
    Name     string `json:"name"`
    Size     int64  `json:"size"`
    Path     string `json:"path"`
    MimeType string `json:"mimeType"`
}
```

#### Table Widget

**Go:** `plz-confirm/internal/types/types.go:100-111`
**TypeScript:** `plz-confirm/agent-ui-system/client/src/types/schemas.ts:67-78`

```go
type TableInput struct {
    Title       string   `json:"title"`
    Data        []any    `json:"data"`  // Dynamic rows
    Columns     []string `json:"columns,omitempty"`
    MultiSelect *bool    `json:"multiSelect,omitempty"`
    Searchable  *bool    `json:"searchable,omitempty"`
}

type TableOutput struct {
    Selected any     `json:"selected"`  // any | []any
    Comment  *string `json:"comment,omitempty"`
}
```

**Issue:** `Data` and `Selected` are `any` (dynamic table structure).

#### Image Widget

**Go:** `plz-confirm/internal/types/types.go:113-145`
**TypeScript:** `plz-confirm/agent-ui-system/client/src/types/schemas.ts:80-100`

```go
type ImageItem struct {
    Src     string  `json:"src"`  // URL or data URI
    Alt     *string `json:"alt,omitempty"`
    Label   *string `json:"label,omitempty"`
    Caption *string `json:"caption,omitempty"`
}

type ImageInput struct {
    Title   string      `json:"title"`
    Message *string     `json:"message,omitempty"`
    Images  []ImageItem `json:"images"`
    Mode    string      `json:"mode"`  // "select" | "confirm"
    Options []string     `json:"options,omitempty"`
    Multi   *bool        `json:"multi,omitempty"`
}

type ImageOutput struct {
    Selected  any     `json:"selected"`  // number | []number | boolean | string | []string
    Timestamp string  `json:"timestamp"`
    Comment   *string `json:"comment,omitempty"`
}
```

**Issue:** `Selected` is a complex union type.

### Image Upload Response

**Go Client:** `plz-confirm/internal/client/client.go:137-142`
**Go Server:** `plz-confirm/internal/server/server.go:337-342`

```go
type UploadImageResponse struct {
    ID       string `json:"id"`
    URL      string `json:"url"`       // "/api/images/{id}"
    MimeType string `json:"mimeType"`
    Size     int64  `json:"size"`
}
```

**Note:** Frontend doesn't directly use this structure (uploads via form, receives URL in `ImageItem.src`).

## Serialization Patterns

### Current Approach

1. **Server → Client (REST):**
   - Go structs → JSON via `encoding/json`
   - `Input`/`Output` stored as `any` → serialized as JSON objects
   - Frontend receives `any` → type assertions in components

2. **Client → Server (REST):**
   - TypeScript objects → JSON via `JSON.stringify`
   - Server receives `any` → stores as `any`

3. **CLI → Server:**
   - Go typed structs → JSON → `any` in `UIRequest`
   - CLI creates typed `*Input` structs, marshals to JSON, sends as `any`

4. **CLI ← Server:**
   - Server returns `UIRequest` with `Output: any`
   - CLI performs double marshal/unmarshal:
     ```go
     var out agenttypes.ConfirmOutput
     if completed.Output != nil {
         b, err := json.Marshal(completed.Output)
         if err != nil {
             return errors.Wrap(err, "marshal output")
         }
         if err := json.Unmarshal(b, &out); err != nil {
             return errors.Wrap(err, "unmarshal output")
         }
     }
     ```
   - **Implementation:** `plz-confirm/internal/cli/confirm.go:118-128`

5. **WebSocket:**
   - Server broadcasts `map[string]any` → JSON → client
   - Client parses JSON → type assertions

### Pain Points

1. **Type Safety:**
   - No compile-time guarantee that Go and TypeScript types match
   - Manual duplication (`internal/types/types.go` vs `agent-ui-system/client/src/types/schemas.ts`)
   - Explicit comment acknowledges duplication: `plz-confirm/internal/types/types.go:3-6`

2. **Performance:**
   - Double marshal/unmarshal in CLI (inefficient)
   - `any` requires runtime type assertions

3. **Maintainability:**
   - Adding a field requires updating two files (Go + TS)
   - Easy to introduce bugs (field name typos, missing fields)

4. **Union Types:**
   - `SelectOutput.Selected`: `string | []string`
   - `ImageOutput.Selected`: `number | []number | boolean | string | []string`
   - `TableOutput.Selected`: `any | []any`
   - Handled as `any` in Go, requires type assertions

5. **Dynamic Types:**
   - `FormInput.Schema`: JSON Schema (inherently dynamic)
   - `FormOutput.Data`: Dynamic form results
   - `TableInput.Data`: Dynamic table rows
   - Cannot be strongly typed without schema registry

## File References

### Backend (Go)

- **Server:** `plz-confirm/internal/server/server.go`
- **WebSocket:** `plz-confirm/internal/server/ws.go`
- **Image Storage:** `plz-confirm/internal/server/images.go`
- **Types:** `plz-confirm/internal/types/types.go`
- **Store:** `plz-confirm/internal/store/store.go`
- **Client:** `plz-confirm/internal/client/client.go`
- **CLI Commands:** `plz-confirm/internal/cli/*.go` (6 files)

### Frontend (TypeScript)

- **Types:** `plz-confirm/agent-ui-system/client/src/types/schemas.ts`
- **WebSocket Client:** `plz-confirm/agent-ui-system/client/src/services/websocket.ts`
- **Redux Store:** `plz-confirm/agent-ui-system/client/src/store/store.ts`
- **Widget Components:** `plz-confirm/agent-ui-system/client/src/components/widgets/*.tsx`

## Summary

**Current State:**
- REST API + WebSocket for real-time updates
- 11 shared data structures (UIRequest + 6 widgets × 2 + ImageItem + UploadImageResponse)
- Manual type duplication between Go and TypeScript
- `any` types for Input/Output requiring runtime assertions
- Double marshal/unmarshal pattern in CLI

**Key Challenges for Protobuf Migration:**
1. Union types (`Selected` fields)
2. Dynamic types (JSON Schema, dynamic form/table data)
3. Backward compatibility during migration
4. WebSocket message envelopes (currently ad-hoc JSON)

**Next Steps:**
- Design protobuf message hierarchy
- Propose code generation pipeline (Go + TypeScript)
- Migration strategy (gradual vs big-bang)
