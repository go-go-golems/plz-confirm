---
Title: Protobuf unification + codegen proposal
Ticket: 006-USE-PROTOBUF
Status: active
Topics:
    - backend
    - frontend
    - api
    - protobuf
    - codegen
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-25T22:56:37.943155888-05:00
WhatFor: ""
WhenToUse: ""
---

# Protobuf unification + codegen proposal

## Executive Summary

Replace manual type duplication between Go backend and TypeScript frontend with Protocol Buffers (protobuf) definitions and code generation. This will provide:
- **Single source of truth** for all shared data structures
- **Compile-time type safety** across backend and frontend
- **Elimination of double marshal/unmarshal** patterns
- **Automatic code generation** for Go and TypeScript from `.proto` files

The migration will be done **in one go** — protobuf types replace manual types immediately. JSON remains the wire format (REST API continues to use JSON, serialized via `protojson`).

## Problem Statement

### Current Pain Points

1. **Manual Type Duplication**
   - Go types in `plz-confirm/internal/types/types.go`
   - TypeScript types in `plz-confirm/agent-ui-system/client/src/types/schemas.ts`
   - Explicit comment acknowledges duplication: `internal/types/types.go:3-6`
   - Adding a field requires updating two files (error-prone)

2. **Type Safety Issues**
   - `UIRequest.Input` and `UIRequest.Output` are `any` (JSON)
   - No compile-time guarantee that Go and TypeScript types match
   - Runtime type assertions required everywhere

3. **Performance Overhead**
   - CLI performs double marshal/unmarshal:
     ```go
     // From internal/cli/confirm.go:118-128
     var out agenttypes.ConfirmOutput
     if completed.Output != nil {
         b, err := json.Marshal(completed.Output)
         if err != nil { return errors.Wrap(err, "marshal output") }
         if err := json.Unmarshal(b, &out); err != nil {
             return errors.Wrap(err, "unmarshal output")
         }
     }
     ```

4. **Union Types**
   - `SelectOutput.Selected`: `string | []string`
   - `ImageOutput.Selected`: `number | []number | boolean | string | []string`
   - `TableOutput.Selected`: `any | []any`
   - Handled as `any` in Go, requires type assertions

5. **Dynamic Types**
   - `FormInput.Schema`: JSON Schema (inherently dynamic)
   - `FormOutput.Data`: Dynamic form results
   - `TableInput.Data`: Dynamic table rows

## Proposed Solution

### Architecture Overview

```
proto/
  ├── plz_confirm/
  │   ├── v1/
  │   │   ├── request.proto      # UIRequest, enums
  │   │   ├── widgets.proto      # All widget Input/Output types
  │   │   └── image.proto         # Image upload response
  │   └── ...
  │
codegen/
  ├── go/                        # Generated Go code
  │   └── plz_confirm/v1/
  ├── ts/                        # Generated TypeScript code
  │   └── plz_confirm/v1/
  └── ...
```

### Protobuf Message Hierarchy

#### Core Types (`request.proto`)

```protobuf
syntax = "proto3";

package plz_confirm.v1;

// RequestStatus enum
enum RequestStatus {
  REQUEST_STATUS_UNSPECIFIED = 0;
  REQUEST_STATUS_PENDING = 1;
  REQUEST_STATUS_COMPLETED = 2;
  REQUEST_STATUS_TIMEOUT = 3;
  REQUEST_STATUS_ERROR = 4;
}

// WidgetType enum
enum WidgetType {
  WIDGET_TYPE_UNSPECIFIED = 0;
  WIDGET_TYPE_CONFIRM = 1;
  WIDGET_TYPE_SELECT = 2;
  WIDGET_TYPE_FORM = 3;
  WIDGET_TYPE_UPLOAD = 4;
  WIDGET_TYPE_TABLE = 5;
  WIDGET_TYPE_IMAGE = 6;
}

// UIRequest - main request/response envelope
message UIRequest {
  string id = 1;
  WidgetType type = 2;
  string session_id = 3;
  
  // Widget-specific input (oneof for type safety)
  oneof input {
    ConfirmInput confirm_input = 4;
    SelectInput select_input = 5;
    FormInput form_input = 6;
    UploadInput upload_input = 7;
    TableInput table_input = 8;
    ImageInput image_input = 9;
  }
  
  // Widget-specific output (oneof for type safety)
  oneof output {
    ConfirmOutput confirm_output = 10;
    SelectOutput select_output = 11;
    FormOutput form_output = 12;
    UploadOutput upload_output = 13;
    TableOutput table_output = 14;
    ImageOutput image_output = 15;
  }
  
  RequestStatus status = 16;
  string created_at = 17;        // RFC3339Nano timestamp
  optional string completed_at = 18;
  string expires_at = 19;       // RFC3339Nano timestamp
  optional string error = 20;
}
```

#### Widget Types (`widgets.proto`)

```protobuf
syntax = "proto3";

package plz_confirm.v1;

import "google/protobuf/struct.proto";
import "google/protobuf/wrappers.proto";

// Confirm Widget
message ConfirmInput {
  string title = 1;
  optional string message = 2;
  optional string approve_text = 3;
  optional string reject_text = 4;
}

message ConfirmOutput {
  bool approved = 1;
  string timestamp = 2;  // ISO 8601
  optional string comment = 3;
}

// Select Widget
message SelectInput {
  string title = 1;
  repeated string options = 2;
  optional bool multi = 3;
  optional bool searchable = 4;
}

message SelectOutput {
  // Union type: string | []string
  oneof selected {
    string selected_single = 1;
    SelectOutputMulti selected_multi = 2;
  }
  optional string comment = 3;
}

message SelectOutputMulti {
  repeated string values = 1;
}

// Form Widget
message FormInput {
  string title = 1;
  // JSON Schema as Struct (dynamic)
  google.protobuf.Struct schema = 2;
}

message FormOutput {
  // Dynamic form results as Struct
  google.protobuf.Struct data = 1;
  optional string comment = 2;
}

// Upload Widget
message UploadInput {
  string title = 1;
  repeated string accept = 2;
  optional bool multiple = 3;
  optional int64 max_size = 4;
  optional string callback_url = 5;
}

message UploadOutput {
  repeated UploadedFile files = 1;
  optional string comment = 2;
}

message UploadedFile {
  string name = 1;
  int64 size = 2;
  string path = 3;
  string mime_type = 4;
}

// Table Widget
message TableInput {
  string title = 1;
  // Dynamic rows as repeated Struct
  repeated google.protobuf.Struct data = 2;
  repeated string columns = 3;
  optional bool multi_select = 4;
  optional bool searchable = 5;
}

message TableOutput {
  // Union type: any | []any
  oneof selected {
    google.protobuf.Struct selected_single = 1;
    TableOutputMulti selected_multi = 2;
  }
  optional string comment = 3;
}

message TableOutputMulti {
  repeated google.protobuf.Struct values = 1;
}

// Image Widget
message ImageItem {
  string src = 1;  // URL or data URI
  optional string alt = 2;
  optional string label = 3;
  optional string caption = 4;
}

message ImageInput {
  string title = 1;
  optional string message = 2;
  repeated ImageItem images = 3;
  string mode = 4;  // "select" | "confirm"
  repeated string options = 5;
  optional bool multi = 6;
}

message ImageOutput {
  // Complex union: number | []number | boolean | string | []string
  oneof selected {
    int64 selected_number = 1;
    ImageOutputNumbers selected_numbers = 2;
    bool selected_bool = 3;
    string selected_string = 4;
    ImageOutputStrings selected_strings = 5;
  }
  string timestamp = 6;  // ISO 8601
  optional string comment = 7;
}

message ImageOutputNumbers {
  repeated int64 values = 1;
}

message ImageOutputStrings {
  repeated string values = 1;
}
```

#### Image Upload (`image.proto`)

```protobuf
syntax = "proto3";

package plz_confirm.v1;

message UploadImageResponse {
  string id = 1;
  string url = 2;  // "/api/images/{id}"
  string mime_type = 3;
  int64 size = 4;
}
```

### Code Generation Pipeline

#### Go Code Generation

**Tool:** `protoc` with `--go_out` (JSON wire format, no gRPC)

**Generated Code Location:**
```
plz-confirm/
  └── proto/
      └── generated/
          └── go/
              └── plz_confirm/
                  └── v1/
                      ├── request.pb.go
                      ├── widgets.pb.go
                      └── image.pb.go
```

**Usage in Go:**
```go
import (
    "github.com/go-go-golems/plz-confirm/proto/generated/go/plz_confirm/v1"
)

// Create request
req := &v1.UIRequest{
    Type: v1.WidgetType_WIDGET_TYPE_CONFIRM,
    ConfirmInput: &v1.ConfirmInput{
        Title: "Approve deployment?",
        Message: proto.String("This will deploy to production"),
    },
    Status: v1.RequestStatus_REQUEST_STATUS_PENDING,
}

// Serialize to JSON (REST API uses JSON wire format)
jsonBytes, err := protojson.Marshal(req)
```

#### TypeScript Code Generation

**Tool:** `protoc` with `@bufbuild/protoc-gen-es` or `ts-proto`

**Generated Code Location:**
```
plz-confirm/agent-ui-system/client/src/
  └── proto/
      └── generated/
          └── plz_confirm/
              └── v1/
                  ├── request.ts
                  ├── widgets.ts
                  └── image.ts
```

**Usage in TypeScript:**
```typescript
import { UIRequest, WidgetType, ConfirmInput } from '@/proto/generated/plz_confirm/v1/request';

// Create request
const req: UIRequest = {
  type: WidgetType.WIDGET_TYPE_CONFIRM,
  confirmInput: {
    title: "Approve deployment?",
    message: "This will deploy to production",
  },
  status: RequestStatus.REQUEST_STATUS_PENDING,
};

// Serialize to JSON (for REST API)
const jsonBytes = JSON.stringify(req);
```

### Migration Strategy

**Single-step transition:** Replace manual types with protobuf-generated types immediately. JSON remains the wire format (REST API continues to use JSON, serialized via `protojson`).

#### Implementation Steps

1. **Create protobuf definitions**
   - Create `proto/plz_confirm/v1/*.proto` files
   - Set up `buf` or `protoc` build pipeline
   - Generate Go and TypeScript code

2. **Update server to use protobuf types**
   - Replace `internal/types/types.go` imports with generated protobuf types
   - Use `protojson` for JSON serialization/deserialization
   - Update all handlers to use protobuf messages

3. **Update CLI to use protobuf types**
   - Replace manual type imports with generated protobuf types
   - Remove double marshal/unmarshal pattern
   - Use `protojson` for JSON conversion at wire boundary

4. **Update frontend to use protobuf types**
   - Replace `types/schemas.ts` imports with generated TypeScript types
   - Update all components to use protobuf types
   - Use `protojson` for JSON conversion at API boundaries

5. **Remove manual type definitions**
   - Delete `plz-confirm/internal/types/types.go`
   - Delete `plz-confirm/agent-ui-system/client/src/types/schemas.ts`
   - Update all imports to use generated code

**Wire Format:**
- REST API: JSON (via `protojson.Marshal` / `protojson.Unmarshal`)
- WebSocket: JSON (via `protojson.Marshal` / `protojson.Unmarshal`)
- No binary protobuf wire format
- No gRPC endpoints

## Design Decisions

### Decision 1: Use `oneof` for Union Types

**Rationale:**
- Protobuf `oneof` provides type-safe union types
- Eliminates `any` types and runtime assertions
- Generated code has proper type guards

**Alternatives Considered:**
- Keep `any` with `google.protobuf.Value` (loses type safety)
- Use separate messages for each variant (more verbose)

**Example:**
```protobuf
message SelectOutput {
  oneof selected {
    string selected_single = 1;
    SelectOutputMulti selected_multi = 2;
  }
}
```

### Decision 2: Use `google.protobuf.Struct` for Dynamic Types

**Rationale:**
- JSON Schema and dynamic form/table data are inherently dynamic
- `Struct` provides type-safe representation of JSON objects
- Can be converted to/from JSON easily

**Alternatives Considered:**
- `bytes` with JSON (loses structure, harder to work with)
- Separate schema registry (adds complexity)

**Example:**
```protobuf
message FormInput {
  string title = 1;
  google.protobuf.Struct schema = 2;  // JSON Schema
}
```

### Decision 3: Single-Step Migration (Not Gradual)

**Rationale:**
- Cleaner implementation (no dual code paths)
- Faster to complete (no intermediate states)
- No maintenance burden of supporting both old and new types

**Alternatives Considered:**
- Gradual migration (adds complexity, dual code paths)
- Dual support forever (maintenance burden)

### Decision 4: Keep JSON Wire Format (No Binary Protobuf, No gRPC)

**Rationale:**
- REST APIs typically use JSON
- Easier to debug (human-readable)
- Browser DevTools work with JSON
- No need for HTTP/2 or gRPC complexity
- `protojson` provides seamless JSON ↔ Protobuf conversion

**Alternatives Considered:**
- Binary protobuf wire format (smaller, faster, but harder to debug)
- gRPC (adds complexity, requires HTTP/2)

**Decision:** JSON wire format via `protojson` — get type safety benefits without wire format complexity.

### Decision 5: Separate `.proto` Files by Domain

**Rationale:**
- `request.proto`: Core types (UIRequest, enums)
- `widgets.proto`: Widget Input/Output types
- `image.proto`: Image upload types
- Easier to maintain and navigate

**Alternatives Considered:**
- Single `plz_confirm.proto` file (becomes large, hard to navigate)

## Alternatives Considered

### Alternative 1: JSON Schema + Code Generation

**Approach:** Use JSON Schema as source of truth, generate Go and TypeScript types.

**Rejected Because:**
- JSON Schema doesn't handle union types well (`oneof` in protobuf is cleaner)
- Less mature tooling for multi-language codegen
- Protobuf has better performance characteristics

### Alternative 2: OpenAPI/Swagger

**Approach:** Use OpenAPI spec, generate types from it.

**Rejected Because:**
- OpenAPI is API-focused, not data-structure-focused
- Union types require complex `discriminator` patterns
- Less type-safe than protobuf

### Alternative 3: TypeScript-First with `tsc --declaration`

**Approach:** Write types in TypeScript, generate Go types from `.d.ts`.

**Rejected Because:**
- Go type generation from TypeScript is not well-supported
- Protobuf is language-agnostic and well-tooled

### Alternative 4: Keep Manual Duplication

**Approach:** Continue with current approach, add validation/linting.

**Rejected Because:**
- Doesn't solve type safety issues
- Doesn't eliminate double marshal/unmarshal
- Maintenance burden remains

## Implementation Plan

### Step 1: Protobuf Definitions & Build Pipeline

- [ ] Create `proto/plz_confirm/v1/` directory structure
- [ ] Write `request.proto` (UIRequest, enums)
- [ ] Write `widgets.proto` (all widget Input/Output types)
- [ ] Write `image.proto` (UploadImageResponse)
- [ ] Validate with `buf lint`
- [ ] Set up `buf` or `Makefile` for code generation
- [ ] Add `make proto` target
- [ ] Generate Go code (`protoc --go_out`)
- [ ] Generate TypeScript code (`protoc --ts_out` or `@bufbuild/protoc-gen-es`)
- [ ] Integrate into CI/CD
- [ ] Document code generation process

**Deliverables:**
- `.proto` files
- Generated Go code (`proto/generated/go/plz_confirm/v1/`)
- Generated TypeScript code (`agent-ui-system/client/src/proto/generated/plz_confirm/v1/`)
- `Makefile` targets
- CI/CD integration

### Step 2: Server Migration

- [ ] Replace `internal/types/types.go` imports with generated protobuf types
- [ ] Update `internal/server/server.go` to use protobuf messages
- [ ] Use `protojson.Marshal` / `protojson.Unmarshal` for JSON serialization
- [ ] Update all HTTP handlers to use protobuf types
- [ ] Update WebSocket broadcaster to use protobuf types (JSON wire format)
- [ ] Add tests for all endpoints

**Deliverables:**
- Updated `internal/server/server.go`
- Updated `internal/server/ws.go`
- Updated `internal/store/store.go` (if needed)
- Tests

### Step 3: CLI Migration

- [ ] Replace `internal/types/types.go` imports with generated protobuf types
- [ ] Update all CLI commands (`internal/cli/*.go`) to use protobuf types
- [ ] Remove double marshal/unmarshal pattern
- [ ] Use `protojson` for JSON conversion at wire boundary (HTTP client)
- [ ] Update `internal/client/client.go` to use protobuf types
- [ ] Add tests

**Deliverables:**
- Updated `internal/cli/*.go` (6 files)
- Updated `internal/client/client.go`
- Tests

### Step 4: Frontend Migration

- [ ] Replace `types/schemas.ts` imports with generated TypeScript types
- [ ] Update all components (`agent-ui-system/client/src/components/widgets/*.tsx`) to use protobuf types
- [ ] Update Redux store (`agent-ui-system/client/src/store/store.ts`) to use protobuf types
- [ ] Update WebSocket client (`agent-ui-system/client/src/services/websocket.ts`) to use protobuf types
- [ ] Use `protojson` for JSON conversion at API boundaries
- [ ] Delete `agent-ui-system/client/src/types/schemas.ts`
- [ ] Add tests

**Deliverables:**
- Updated frontend components
- Updated Redux store
- Updated WebSocket client
- Deleted `types/schemas.ts`
- Tests

### Step 5: Cleanup & Documentation

- [ ] Delete `plz-confirm/internal/types/types.go`
- [ ] Update all documentation references
- [ ] Update README
- [ ] Verify all tests pass
- [ ] Code review

**Deliverables:**
- Deleted manual type files
- Updated documentation
- All tests passing

## Open Questions

1. **WebSocket Messages:** JSON wire format (via `protojson`) — matches REST API approach
   - **Decision:** JSON for consistency and easier debugging

2. **Versioning:** How to handle protobuf message versioning?
   - **Recommendation:** Use `plz_confirm.v1`, `plz_confirm.v2` packages (protobuf supports this)

3. **Form Schema:** Should `FormInput.Schema` remain `google.protobuf.Struct` or use a schema registry?
   - **Recommendation:** Keep `Struct` for now, can add schema registry later

## References

- [Protocol Buffers Language Guide](https://protobuf.dev/programming-guides/proto3/)
- [buf.build](https://buf.build/) - Modern protobuf tooling
- [protoc-gen-go](https://github.com/protocolbuffers/protobuf-go) - Go code generation
- [@bufbuild/protoc-gen-es](https://github.com/bufbuild/protobuf-es) - TypeScript code generation
- [google.protobuf.Struct](https://protobuf.dev/reference/protobuf/google.protobuf/#struct) - Dynamic JSON representation

## Pseudocode Examples

### Go: Creating a Request

```go
import (
    "github.com/go-go-golems/plz-confirm/proto/generated/go/plz_confirm/v1"
    "google.golang.org/protobuf/encoding/protojson"
)

func createConfirmRequest(title string) (*v1.UIRequest, error) {
    req := &v1.UIRequest{
        Type: v1.WidgetType_WIDGET_TYPE_CONFIRM,
        ConfirmInput: &v1.ConfirmInput{
            Title: title,
            Message: proto.String("Approve this action?"),
        },
        Status: v1.RequestStatus_REQUEST_STATUS_PENDING,
    }
    
    // Convert to JSON for REST API
    jsonBytes, err := protojson.Marshal(req)
    if err != nil {
        return nil, err
    }
    
    // Send via HTTP client (existing code)
    // ...
    
    return req, nil
}
```

### TypeScript: Handling a Request

```typescript
import { UIRequest, WidgetType, ConfirmInput } from '@/proto/generated/plz_confirm/v1/request';

function handleConfirmRequest(req: UIRequest): void {
  if (req.type !== WidgetType.WIDGET_TYPE_CONFIRM) {
    throw new Error('Expected confirm request');
  }
  
  const input: ConfirmInput = req.confirmInput!;
  console.log(`Title: ${input.title}`);
  console.log(`Message: ${input.message ?? 'No message'}`);
  
  // Type-safe access, no runtime assertions needed
}
```

### Go: Handling Union Types

```go
func handleSelectOutput(out *v1.SelectOutput) {
    switch sel := out.Selected.(type) {
    case *v1.SelectOutput_SelectedSingle:
        // Single selection: string
        fmt.Printf("Selected: %s\n", sel.SelectedSingle)
    case *v1.SelectOutput_SelectedMulti:
        // Multi selection: []string
        fmt.Printf("Selected: %v\n", sel.SelectedMulti.Values)
    default:
        fmt.Println("No selection")
    }
}
```

### TypeScript: Handling Union Types

```typescript
function handleSelectOutput(out: SelectOutput): void {
  if (out.selectedSingle !== undefined) {
    // Single selection: string
    console.log(`Selected: ${out.selectedSingle}`);
  } else if (out.selectedMulti !== undefined) {
    // Multi selection: string[]
    console.log(`Selected: ${out.selectedMulti.values.join(', ')}`);
  }
}
```
