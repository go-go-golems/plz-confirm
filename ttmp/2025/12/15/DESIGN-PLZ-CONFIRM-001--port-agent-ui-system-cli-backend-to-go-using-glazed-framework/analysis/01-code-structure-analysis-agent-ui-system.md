---
Title: Code Structure Analysis - agent-ui-system
Ticket: DESIGN-PLZ-CONFIRM-001
Status: active
Topics:
    - go
    - glazed
    - cli
    - backend
    - porting
    - agent-ui-system
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Comprehensive analysis of agent-ui-system architecture, API contracts, widget types, and CLI patterns for porting to Go/Glazed
LastUpdated: 2025-12-15T15:35:24.290360572-05:00
---

# Code Structure Analysis - agent-ui-system

## Executive Summary

The `agent-ui-system` is a TypeScript/Node.js application that enables CLI tools to request user input via a web-based UI. The system consists of:

1. **Backend Server** (Express + WebSocket): REST API for request creation and WebSocket for real-time updates
2. **Frontend** (React + Redux): Web UI that renders interactive widgets based on request types
3. **CLI Integration** (Python): Demonstration of how CLI tools interact with the system

**Key Architecture Points:**
- Session-based WebSocket connections
- In-memory request storage (Map-based)
- Five widget types: confirm, select, form, table, upload
- Dual communication: REST for creation, WebSocket for real-time updates
- Long-polling fallback for CLI tools

## System Architecture

### High-Level Overview

```
┌─────────────┐         REST API          ┌──────────────┐
│   CLI Tool  │──────────────────────────>│              │
│  (Python)   │                           │   Backend    │
│             │<───────────────────────────│   Server     │
│             │    Long-poll / WebSocket   │  (Express)   │
└─────────────┘                           └──────┬───────┘
                                                  │
                                                  │ WebSocket
                                                  │
┌─────────────┐                                 │
│   Web UI    │<────────────────────────────────┘
│  (React)    │
│             │
│  WebSocket  │
└─────────────┘
```

### Component Breakdown

#### Backend Server (`server/index.ts`)

**Technology Stack:**
- Express.js for HTTP server
- `ws` library for WebSocket server
- In-memory storage using JavaScript Maps
- CORS enabled for cross-origin requests

**Key Data Structures:**
```typescript
// Request storage
const requests = new Map<string, UIRequest>();

// Session -> WebSocket clients mapping
const sessions = new Map<string, WebSocket[]>();
```

**Request Lifecycle:**
1. CLI creates request via `POST /api/requests`
2. Request stored with `pending` status
3. WebSocket clients notified via `new_request` message
4. Frontend displays widget
5. User submits response via `POST /api/requests/:id/response`
6. Request updated to `completed`
7. WebSocket clients notified via `request_completed` message
8. CLI receives response via long-poll or WebSocket

#### Frontend (`client/`)

**Technology Stack:**
- React 19 with TypeScript
- Redux Toolkit for state management
- WebSocket client for real-time updates
- Radix UI components for widgets
- Tailwind CSS for styling

**State Management:**
- `session`: Connection state, session ID
- `request`: Active request, history
- `notifications`: Notification items

**Key Components:**
- `WidgetRenderer`: Routes requests to appropriate widget component
- `Home`: Main page with request display and history
- Widget components: `ConfirmDialog`, `SelectDialog`, `FormDialog`, `TableDialog`, `UploadDialog`

#### CLI Integration (`demo_cli.py`)

**Pattern:**
1. Create request with type and input data
2. Wait for response using long-polling endpoint
3. Process response and continue workflow

**Session Management:**
- Fixed session ID for demo: `550e8400-e29b-41d4-a716-446655440000`
- All requests use same session ID

## API Contract

### REST Endpoints

#### POST `/api/requests`

Create a new UI request.

**Request Body:**
```typescript
{
  type: 'confirm' | 'select' | 'form' | 'upload' | 'table',
  sessionId: string,
  input: any,  // Widget-specific input schema
  timeout?: number  // Default: 300 seconds
}
```

**Response:** `201 Created`
```typescript
{
  id: string,
  type: string,
  sessionId: string,
  input: any,
  status: 'pending',
  createdAt: string,
  expiresAt: string
}
```

#### GET `/api/requests/:id`

Get request status (polling fallback).

**Response:** `200 OK`
```typescript
UIRequest  // Full request object with current status
```

#### POST `/api/requests/:id/response`

Submit response to a request.

**Request Body:**
```typescript
{
  output: any  // Widget-specific output schema
}
```

**Response:** `200 OK`
```typescript
UIRequest  // Updated request object with output
```

**Error Responses:**
- `404`: Request not found
- `409`: Request already completed

#### GET `/api/requests/:id/wait`

Long-poll endpoint for CLI to wait for completion.

**Query Parameters:**
- `timeout`: Maximum wait time in seconds (default: 60)

**Response:** `200 OK` (when completed)
```typescript
UIRequest  // Completed request object
```

**Error Responses:**
- `404`: Request not found
- `408`: Timeout waiting for response

**Behavior:**
- Polls every 500ms until request is completed or timeout
- Returns immediately if request is already completed

### WebSocket Protocol

#### Connection

**Endpoint:** `/ws?sessionId=<session-id>`

**Connection Flow:**
1. Client connects with `sessionId` query parameter
2. Server validates session ID (currently no validation)
3. Server adds WebSocket to session's client list
4. Server sends all pending requests for that session

**Error Handling:**
- Missing `sessionId`: Connection closed with code 1008

#### Message Types

**Server → Client: `new_request`**

Notifies client of a new request.

```typescript
{
  type: "new_request",
  request: UIRequest
}
```

**Server → Client: `request_completed`**

Notifies client that a request was completed.

```typescript
{
  type: "request_completed",
  request: UIRequest
}
```

**Client → Server:**

No messages sent from client to server (unidirectional).

## Widget Types and Schemas

### 1. Confirm Widget

**Type:** `confirm`

**Input Schema:**
```typescript
{
  title: string,
  message?: string,
  approveText?: string,  // Default: "APPROVE"
  rejectText?: string    // Default: "REJECT"
}
```

**Output Schema:**
```typescript
{
  approved: boolean,
  timestamp: string  // ISO 8601 format
}
```

**Use Case:** Yes/No confirmation dialogs

**Example:**
```json
{
  "type": "confirm",
  "input": {
    "title": "System Update Required",
    "message": "A critical security patch is available. Install now?",
    "approveText": "Install & Restart",
    "rejectText": "Remind Me Later"
  }
}
```

### 2. Select Widget

**Type:** `select`

**Input Schema:**
```typescript
{
  title: string,
  options: string[],
  multi?: boolean,      // Default: false
  searchable?: boolean   // Default: false
}
```

**Output Schema:**
```typescript
{
  selected: string | string[]  // Single string if multi=false, array if multi=true
}
```

**Use Case:** Single or multi-select from a list of options

**Example:**
```json
{
  "type": "select",
  "input": {
    "title": "Select Region",
    "options": ["us-east-1", "us-west-2", "eu-central-1"],
    "multi": false,
    "searchable": true
  }
}
```

### 3. Form Widget

**Type:** `form`

**Input Schema:**
```typescript
{
  title: string,
  schema: JSONSchema  // JSON Schema object
}
```

**Output Schema:**
```typescript
{
  data: Record<string, any>  // Form field values
}
```

**Use Case:** Dynamic forms based on JSON Schema

**Supported JSON Schema Features:**
- Field types: `string`, `number`, `boolean`
- Validation: `minLength`, `maxLength`, `minimum`, `maximum`, `pattern`
- Formats: `email`, `password`
- Required fields: `required` array

**Example:**
```json
{
  "type": "form",
  "input": {
    "title": "Administrator Details",
    "schema": {
      "properties": {
        "username": {"type": "string", "minLength": 3},
        "email": {"type": "string", "format": "email"},
        "accessLevel": {"type": "number", "minimum": 1, "maximum": 5}
      },
      "required": ["username", "email"]
    }
  }
}
```

### 4. Table Widget

**Type:** `table`

**Input Schema:**
```typescript
{
  title: string,
  data: any[],           // Array of objects (rows)
  columns?: string[],    // Optional column names (auto-derived if omitted)
  multiSelect?: boolean,  // Default: false
  searchable?: boolean    // Default: false
}
```

**Output Schema:**
```typescript
{
  selected: any | any[]  // Single object if multiSelect=false, array if multiSelect=true
}
```

**Use Case:** Tabular data with selection capabilities

**Row Identification:**
- Uses `id` field if present
- Falls back to `JSON.stringify(row)` if no `id` field

**Features:**
- Column sorting (click header)
- Search/filter (if `searchable=true`)
- Single or multi-select

**Example:**
```json
{
  "type": "table",
  "input": {
    "title": "Select Server",
    "data": [
      {"id": 1, "name": "server-1", "status": "running"},
      {"id": 2, "name": "server-2", "status": "stopped"}
    ],
    "columns": ["name", "status"],
    "multiSelect": false,
    "searchable": true
  }
}
```

### 5. Upload Widget

**Type:** `upload`

**Input Schema:**
```typescript
{
  title: string,
  accept?: string[],     // File extensions or MIME types (e.g., [".log", ".txt"])
  multiple?: boolean,    // Default: false
  maxSize?: number,      // Maximum file size in bytes
  callbackUrl?: string   // Optional callback URL (not implemented)
}
```

**Output Schema:**
```typescript
{
  files: Array<{
    name: string,
    size: number,
    path: string,        // Server-side file path
    mimeType: string
  }>
}
```

**Use Case:** File upload with validation

**Note:** Current implementation simulates upload (no actual file handling visible in code)

**Example:**
```json
{
  "type": "upload",
  "input": {
    "title": "Upload Logs",
    "accept": [".log", ".txt"],
    "multiple": true,
    "maxSize": 5242880  // 5MB
  }
}
```

## Request Object Structure

### UIRequest Interface

```typescript
interface UIRequest {
  id: string;                    // Generated using nanoid()
  type: 'confirm' | 'select' | 'form' | 'upload' | 'table';
  sessionId: string;
  input: any;                    // Widget-specific input schema
  output?: any;                  // Widget-specific output schema (set on completion)
  status: 'pending' | 'completed' | 'timeout' | 'error';
  createdAt: string;             // ISO 8601 timestamp
  completedAt?: string;          // ISO 8601 timestamp (set on completion)
  expiresAt: string;             // ISO 8601 timestamp (createdAt + timeout)
  error?: string;                // Error message (if status is 'error')
}
```

### Request Lifecycle States

1. **pending**: Request created, waiting for user response
2. **completed**: User submitted response, request fulfilled
3. **timeout**: Request expired before completion (not implemented)
4. **error**: Request failed (not implemented)

## Session Management

### Session Structure

- **Session ID**: UUID string (e.g., `550e8400-e29b-41d4-a716-446655440000`)
- **WebSocket Clients**: Array of WebSocket connections per session
- **Request Association**: Requests are associated with sessions via `sessionId` field

### Session Lifecycle

1. **Creation**: Implicit when first WebSocket client connects
2. **Expansion**: Additional clients can connect with same session ID
3. **Cleanup**: Session deleted when last WebSocket client disconnects

### Multi-Client Behavior

- Multiple WebSocket clients can connect to the same session
- All clients receive notifications for requests in their session
- When a request is completed, all clients are notified
- Request history is maintained per session (in frontend, not backend)

## CLI Integration Patterns

### Python CLI (`demo_cli.py`)

**Workflow:**
1. Create request via `POST /api/requests`
2. Wait for response via `GET /api/requests/:id/wait`
3. Process response and continue

**Key Functions:**
- `create_request(type, input_data)`: Creates request and returns request ID
- `wait_for_response(req_id)`: Long-polls until response received

**Example Usage:**
```python
# Create confirm request
req_id = create_request("confirm", {
    "title": "System Update Required",
    "message": "Install update?",
    "approveText": "Install",
    "rejectText": "Cancel"
})

# Wait for response
result = wait_for_response(req_id)
if result.get("approved"):
    # Continue workflow
```

### Long-Polling Pattern

The CLI uses long-polling as a fallback mechanism:

1. CLI calls `GET /api/requests/:id/wait?timeout=60`
2. Server polls request status every 500ms
3. Returns immediately when request is completed
4. Returns 408 timeout if max wait time exceeded

**Advantages:**
- Works without WebSocket support
- Simple HTTP-based approach
- Timeout handling built-in

**Disadvantages:**
- Less efficient than WebSocket
- Polling overhead
- Higher latency

## Frontend Architecture Details

### Redux Store Structure

```typescript
{
  session: {
    id: string | null,
    connected: boolean,
    reconnecting: boolean,
    error: string | null
  },
  request: {
    active: UIRequest | null,
    history: UIRequest[],
    loading: boolean
  },
  notifications: {
    items: Notification[]
  }
}
```

### WebSocket Client Behavior

**Connection:**
- Connects on app mount
- Uses session ID from Redux store
- Auto-reconnects on disconnect (3s delay)

**Message Handling:**
- `new_request`: Sets active request, triggers widget display
- `request_completed`: Moves active to history, clears active

**Error Handling:**
- Connection errors set error state
- Reconnection attempts automatically
- Failed messages logged to console

### Widget Rendering Flow

1. `WidgetRenderer` receives active request from Redux
2. Routes to appropriate widget component based on `request.type`
3. Widget displays with `request.input` data
4. User interaction triggers `onSubmit` callback
5. `onSubmit` calls `submitResponse()` API function
6. Response updates request via `POST /api/requests/:id/response`
7. Redux store updated, request moved to history

## Key Implementation Details

### Request ID Generation

- Uses `nanoid()` library
- Generates URL-safe unique IDs
- Example: `YveuI0VPKQ9KnPPMIg-0l`

### Timestamp Handling

- All timestamps in ISO 8601 format
- Generated using `new Date().toISOString()`
- Used for: `createdAt`, `completedAt`, `expiresAt`

### Request Expiration

- Expiration time calculated: `createdAt + timeout * 1000`
- Expiration logic not fully implemented (no cleanup visible)
- `expiresAt` field exists but not actively used

### Error Handling

**Backend:**
- 400: Missing required fields
- 404: Request not found
- 409: Request already completed
- 408: Timeout waiting for response

**Frontend:**
- WebSocket errors logged to console
- API errors shown via error state
- No user-facing error messages visible

### CORS Configuration

- CORS enabled for all origins
- Allows cross-origin requests
- No authentication required

## Testing and E2E Verification

### E2E Test (`verify_e2e.py`)

**Approach:**
- Spawns CLI demo process
- Monitors stdout for request creation
- Automatically submits responses based on request type
- Validates complete workflow

**Test Flow:**
1. Start CLI demo
2. Detect request creation from stdout
3. Wait 2 seconds (simulate user reading)
4. Submit appropriate response based on request type
5. Verify CLI receives response

**Supported Request Types:**
- `confirm`: Always approves
- `select`: Selects `us-west-2`
- `form`: Fills with test data

## Porting Considerations

### Go Port Requirements

1. **HTTP Server**: Replace Express with `net/http` or framework (Gin, Echo, etc.)
2. **WebSocket**: Use `gorilla/websocket` or `nhooyr.io/websocket`
3. **Storage**: Replace Map with Go map or persistent storage
4. **CLI**: Use Glazed framework for command structure
5. **Types**: Convert TypeScript interfaces to Go structs

### Glazed Integration Points

1. **CLI Commands**: Each widget type becomes a Glazed command
2. **Output Formatting**: Use Glazed's structured output for responses
3. **Parameter Parsing**: Use Glazed's parameter layer system
4. **Help System**: Leverage Glazed's help generation

### Key Challenges

1. **WebSocket Handling**: Go WebSocket API differs from Node.js
2. **Concurrency**: Go's goroutines vs Node.js event loop
3. **Type Safety**: Go's type system vs TypeScript
4. **JSON Schema**: Need Go library for form validation
5. **File Upload**: Implement actual file handling

### Recommended Go Libraries

- **HTTP Server**: `net/http` (standard library) or `gin-gonic/gin`
- **WebSocket**: `gorilla/websocket` or `nhooyr.io/websocket`
- **JSON Schema**: `github.com/xeipuuv/gojsonschema`
- **UUID**: `github.com/google/uuid`
- **CLI**: Glazed framework (`github.com/go-go-golems/glazed`)

## Missing Features / Incomplete Implementation

1. **Request Expiration**: `expiresAt` field exists but cleanup not implemented
2. **Error Handling**: `error` status exists but not used
3. **Timeout Handling**: `timeout` status exists but not implemented
4. **File Upload**: Upload widget simulates upload, no actual file handling
5. **Session Validation**: No validation of session IDs
6. **Authentication**: No authentication or authorization
7. **Persistence**: All data in-memory, no persistence layer
8. **Request History**: History maintained in frontend only, not backend

## Related Files

Key files analyzed for this document:

- `/vibes/2025-12-15/agent-ui-system/server/index.ts` - Backend server implementation
- `/vibes/2025-12-15/agent-ui-system/client/src/types/schemas.ts` - Type definitions
- `/vibes/2025-12-15/agent-ui-system/client/src/components/WidgetRenderer.tsx` - Widget routing
- `/vibes/2025-12-15/agent-ui-system/client/src/components/widgets/*.tsx` - Widget implementations
- `/vibes/2025-12-15/agent-ui-system/client/src/store/store.ts` - Redux store
- `/vibes/2025-12-15/agent-ui-system/client/src/services/websocket.ts` - WebSocket client
- `/vibes/2025-12-15/agent-ui-system/demo_cli.py` - CLI demonstration
- `/vibes/2025-12-15/agent-ui-system/verify_e2e.py` - E2E test script

## Next Steps

1. **Design Document**: Create detailed design for Go port
2. **Type Mapping**: Map TypeScript types to Go structs
3. **Command Design**: Design Glazed command structure
4. **WebSocket Design**: Design WebSocket server implementation
5. **Storage Design**: Design request/session storage strategy
6. **Testing Strategy**: Design testing approach for Go port
