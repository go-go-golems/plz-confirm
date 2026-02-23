---
Title: 'Integration Blueprint: plz-confirm on go-go-os macOS Windowing'
Ticket: PC-05-INTEGRATE-OS
Status: active
Topics:
    - architecture
    - frontend
    - backend
    - go
    - javascript
    - ux
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main.go
      Note: |-
        Current inventory backend router composition and mount points
        Current route ownership and mount integration point
        Implemented /confirm mount documented in router section
    - Path: ../../../../../../../go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main_integration_test.go
      Note: |-
        Route coexistence and /confirm/ws integration coverage added during backend tranche
        Integration test strategy now partially implemented
    - Path: ../../../../../../../go-go-os/packages/confirm-runtime/src/components/ConfirmRequestWindowHost.tsx
      Note: Package-first confirm widget host skeleton
    - Path: ../../../../../../../go-go-os/packages/confirm-runtime/src/runtime/createConfirmRuntime.ts
      Note: Package-first confirm runtime orchestrator
    - Path: ../../../../../../../go-go-os/packages/engine/src/chat/ws/wsManager.ts
      Note: WebSocket plus timeline hydration behavior in frontend
    - Path: ../../../../../../../go-go-os/packages/engine/src/components/shell/windowing/PluginCardSessionHost.tsx
      Note: |-
        HyperCard plugin runtime lifecycle in frontend
        HyperCard plugin runtime session hosting and lifecycle
    - Path: ../../../../../../../go-go-os/packages/engine/src/components/shell/windowing/useDesktopShellController.tsx
      Note: |-
        Desktop shell orchestration and window-content adapter chain
        Desktop shell controller and window adapter chain
    - Path: ../../../../../../../go-go-os/packages/engine/src/components/widgets/SchemaFormRenderer.tsx
      Note: Core-first schema form primitive added in implementation phase
    - Path: ../../../../../../../go-go-os/packages/engine/src/components/widgets/SelectableDataTable.tsx
      Note: Core-first generic table primitive added in implementation phase
    - Path: ../../../../../../../go-go-os/packages/engine/src/components/widgets/SelectableList.tsx
      Note: Core-first generic list primitive added in implementation phase
    - Path: ../../../../../../../go-go-os/packages/engine/src/desktop/core/state/windowingSlice.ts
      Note: |-
        Window lifecycle state model and navigation stacks
        Windowing state actions and selectors
    - Path: ../../../../../../../go-go-os/packages/engine/src/hypercard/artifacts/artifactProjectionMiddleware.ts
      Note: Runtime card extraction and injection trigger path
    - Path: ../../../../../../../go-go-os/packages/engine/src/plugin-runtime/runtimeService.ts
      Note: |-
        QuickJS plugin runtime service and safety/timeouts
        QuickJS runtime service boundaries and timeout model
    - Path: agent-ui-system/client/src/components/WidgetRenderer.tsx
      Note: Current browser widget routing including script sections
    - Path: internal/scriptengine/engine.go
      Note: |-
        Script runtime contract, sandbox, timeout, and deterministic random helpers
        Backend script runtime authoritative behavior
    - Path: internal/server/script.go
      Note: |-
        Script event/update lifecycle and server validation logic
        Script event/update and completion semantics
    - Path: internal/server/server.go
      Note: |-
        plz-confirm HTTP router and request lifecycle endpoints
        plz-confirm request API lifecycle endpoints
    - Path: internal/server/ws.go
      Note: |-
        plz-confirm websocket event broadcast model
        WebSocket event stream contract
    - Path: pkg/backend/backend.go
      Note: |-
        Public embeddable server wrapper extracted from internal plz-confirm backend
        Implemented public backend package documented in status section
    - Path: pkg/backend/backend_test.go
      Note: Public backend package tests for direct and prefixed mount flows
    - Path: proto/plz_confirm/v1/request.proto
      Note: UIRequest envelope and widget enums
    - Path: proto/plz_confirm/v1/widgets.proto
      Note: |-
        Widget input/output payload contracts including script
        Widget payload contract source
ExternalSources: []
Summary: Deep integration plan for reusing plz-confirm backend semantics with go-go-os macOS-style desktop frontend in go-inventory-chat, including router, protocol bridge, widget mapping, runtime boundaries, rollout, and onboarding details.
LastUpdated: 2026-02-23T19:15:00-05:00
WhatFor: Guide implementation of PC-05 by defining architecture and phased execution for integrating plz-confirm request/script flows into go-go-os desktop windows.
WhenToUse: Read before any code changes for PC-05; use as primary onboarding + implementation blueprint for interns and maintainers.
---




# Integration Blueprint: plz-confirm on go-go-os macOS Windowing

## Executive Summary

Goal: keep `plz-confirm` as the backend contract and script runtime, but replace the current browser widget UI with `go-go-os` desktop windows (macOS System 1 style) when running inside `go-inventory-chat`.

What we are integrating:

1. `plz-confirm` request lifecycle (`/api/requests`, `/api/requests/{id}/response`, `/api/requests/{id}/event`, `/ws`) and script engine (`describe/init/view/update`) stay authoritative on the backend.
2. `go-go-os` desktop shell (`DesktopShell`, windowing slice, contribution system, macOS theme) becomes the frontend interaction surface for human approvals and scripted multi-step widgets.
3. `go-inventory-chat` becomes the composition host that mounts both chat routes and a plz-confirm-compatible route group, plus a frontend bridge that turns plz-confirm requests into desktop windows.

Most important technical facts discovered during analysis:

1. `go-inventory-chat` currently owns its own `http.ServeMux` with `/chat`, `/ws`, `/api/timeline`, `/api/*`, and `/` static UI routes.
2. `plz-confirm` server implementation exists under `internal/server` and `internal/store`, which cannot be imported directly by `go-inventory-chat` because Go `internal` visibility forbids cross-module imports.
3. `go-go-os` already has most generic UI primitives needed (buttons, list/select-like controls, tables, forms, progress, dialog) but does not yet have plz-confirm-specific composite widgets (upload flow with actual server upload, image selection semantics, script sections renderer).
4. `go-go-os` plugin runtime (`QuickJSCardRuntimeService`) is separate from plz-confirm script runtime (`goja`), and should remain separate for this phase.

Recommended architecture:

1. Extract a public `plzconfirm` package from `plz-confirm/internal/*` for embeddable API/router/store/script-service primitives.
2. In `go-inventory-chat`, mount plz-confirm routes under a namespace (recommended `/confirm/*`) while preserving existing chat routes.
3. Add a new `go-go-os` frontend module (`confirm-runtime`) that:
   - subscribes to plz-confirm WS events,
   - stores active requests in Redux,
   - opens one desktop window per active request,
   - renders widget-specific window bodies,
   - submits outputs/events back to plz-confirm APIs.
4. Keep script logic backend-owned; frontend only renders `scriptView` + emits script events.

This document is written as an implementation manual for an intern starting from zero context.

## Reader Orientation (Start Here)

If you are new, read sections in this exact order:

1. **System Baseline** — understand current backend and frontend architecture.
2. **Constraints and Non-Negotiables** — know what cannot be broken.
3. **Target Architecture** — the integrated design.
4. **Widget Mapping** — concrete UI work to implement.
5. **Router and Messaging Flows** — API and event wiring.
6. **Implementation Plan** — step-by-step execution.
7. **Testing and Rollout** — how to validate and ship safely.

## Problem Statement

We currently have two separate interaction systems:

1. `plz-confirm`: robust request contract and script flow backend, but frontend is a dedicated web UI (`agent-ui-system/client`) and not integrated into the HyperCard desktop shell.
2. `go-go-os` + `go-inventory-chat`: rich desktop/windowing frontend and chat pipeline, but no direct consumption of plz-confirm widget/script requests.

Desired outcome:

1. Agent/automation can still issue plz-confirm requests using existing protocol semantics.
2. Human users respond inside `go-go-os` macOS desktop windows instead of plz-confirm web dialogs.
3. Script workflow semantics remain exactly in plz-confirm backend for now.

Key technical challenge:

`go-inventory-chat` cannot currently import plz-confirm server internals due to Go `internal` package boundaries. This is the first engineering gate.

## Scope and Non-Goals

### In Scope

1. Integration design for backend route mounting and frontend message bridge.
2. Detailed widget mapping from plz-confirm contracts to go-go-os window widgets.
3. Script flow rendering plan where script execution remains server-side in plz-confirm.
4. Phased implementation roadmap with explicit file-level change recommendations.
5. Risk analysis, test plan, and intern onboarding guidance.

### Out of Scope (for PC-05 phase)

1. Replacing plz-confirm script runtime with go-go-os plugin runtime.
2. Merging chat timeline protocol with plz-confirm protocol into one schema.
3. Major redesign of existing go-go-os chat semantics or hypercard artifact pipeline.
4. General-purpose auth/tenant multi-user redesign.

## Current System Baseline

### A. go-inventory-chat backend today

Source: `go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main.go`

Current routes on `appMux`:

1. `POST /chat` and `/chat/`
2. `GET /ws?conv_id=...`
3. `GET /api/timeline?conv_id=...`
4. `/api/*` delegated to `srv.APIHandler()`
5. `/` delegated to `srv.UIHandler()`

Additional details:

1. Optional root prefix wrapping exists (`--root`) via `http.StripPrefix`.
2. Runtime composer is strict about runtime key (`inventory`) and disallows runtime overrides.
3. HyperCard-related events are emitted through structured extraction middleware in `internal/pinoweb/hypercard_extractors.go` and mapped to SEM/timeline in `hypercard_events.go`.

Implication for integration:

1. Router composition is explicit and centralized, which is good for mounting another route group.
2. Path collisions must be controlled because `/api/*` is already occupied.

### B. go-go-os frontend desktop/windowing today

Sources: `packages/engine/src/components/shell/windowing/*`, `desktop/core/state/*`, `theme/desktop/*`

Architecture summary:

1. `DesktopShell` is controller/view split:
   - behavior in `useDesktopShellController.tsx`
   - presentational structure in `DesktopShellView.tsx`
2. Window state is Redux-driven (`windowingSlice.ts`) with:
   - lifecycle: `openWindow`, `focusWindow`, `closeWindow`
   - geometry: `moveWindow`, `resizeWindow`
   - per-session card nav stack: `sessionNavGo/Back/Home`
3. High-frequency drag/resize uses `dragOverlayStore` (ephemeral lane) and commits final geometry to Redux.
4. Window body rendering uses adapter chain:
   - contribution adapters first
   - default app adapter
   - default hypercard card adapter (`PluginCardSessionHost`)
   - fallback adapter
5. macOS-like look is token/theme driven (`tokens.css`, `shell.css`, `desktop-theme-macos1.ts`).

Implication for integration:

1. We already have a mature shell for hosting request UIs as windows.
2. Contribution and adapter APIs let us integrate plz-confirm without rewriting shell internals.

### C. HyperCard plugin runtime setup today

Sources: `PluginCardSessionHost.tsx`, `runtimeService.ts`, `pluginCardRuntimeSlice.ts`, `artifactProjectionMiddleware.ts`

Flow summary:

1. Stack defines `plugin.bundleCode` in domain stack (`apps/inventory/src/domain/stack.ts`).
2. `PluginCardSessionHost` registers a runtime session, loads bundle into QuickJS, renders cards, and emits intents.
3. Runtime intents split into:
   - local state (`card` / `session`) applied directly
   - `domain` and `system` intents gated by capabilities and queued/dispatched.
4. Runtime cards can be injected dynamically via artifact projection path (`registerRuntimeCard` + registry injection).

Implication for integration:

1. This runtime is independent from plz-confirm script engine and should stay that way initially.
2. We can host plz-confirm widgets as separate app windows without touching plugin runtime internals.

### D. plz-confirm backend/runtime today

Sources: `plz-confirm/internal/server/*`, `internal/store/store.go`, `internal/scriptengine/engine.go`, proto files

Contract summary:

1. `UIRequest` proto includes widget type enum and oneof inputs/outputs.
2. Supported widget types: `confirm`, `select`, `form`, `upload`, `table`, `image`, `script`.
3. Script lifecycle:
   - create: runs `describe/init/view` and stores `scriptState/scriptView/scriptDescribe/scriptLogs`
   - event: runs `update` and either patches (`request_updated`) or completes (`request_completed`)
4. WS events emitted per session channel:
   - `new_request`
   - `request_updated`
   - `request_completed`

Critical architectural constraint:

1. plz-confirm backend code is in `internal/*` packages, so currently not embeddable by another Go module.

### E. plz-confirm frontend today (browser UI)

Sources: `agent-ui-system/client/src/*`

Key behaviors relevant for replication:

1. Redux store tracks active/pending/history requests.
2. WebSocket client updates store and deduplicates stale completions/updates.
3. Widget renderer supports both direct widgets and script composite sections:
   - interactive widget + optional display sections
   - optional back button
   - optional progress indicator
   - optional toasts

Implication for integration:

1. Behavior we need already exists conceptually and can be mirrored in go-go-os app windows.
2. We should treat existing client behavior as semantic reference, not necessarily visual reference.

## Constraints and Non-Negotiables

1. **Script execution remains in plz-confirm backend** in this phase.
2. **No breaking changes to plz-confirm wire contract** (`UIRequest`, event payload shapes).
3. **No regressions to existing inventory chat routes**.
4. **Go module boundaries must be respected**; do not rely on unsupported internal imports.
5. **Desktop frontend must remain responsive**; avoid flooding Redux on high-frequency interactions.
6. **Session-scoped behavior must be preserved** so multi-session usage still works.

## Proposed Solution

### High-Level Architecture

Proposed composition inside `go-inventory-chat`:

```text
                   +-------------------------------------+
                   |       hypercard-inventory-server    |
                   |         (single HTTP process)       |
                   +------------------+------------------+
                                      |
           +--------------------------+---------------------------+
           |                                                      |
   Existing chat stack                                      plz-confirm stack
 (/chat, /ws?conv_id, /api/timeline, /api/*)         (mounted under /confirm/*)
           |                                                      |
   go-go-os chat frontend                                go-go-os confirm frontend bridge
 (ChatConversationWindow etc.)                          (Desktop windows per request)
```

Recommended routing namespace:

1. `POST /confirm/api/requests`
2. `GET /confirm/ws?sessionId=...`
3. `POST /confirm/api/requests/{id}/response`
4. `POST /confirm/api/requests/{id}/event`
5. `POST /confirm/api/requests/{id}/touch`
6. `GET /confirm/api/requests/{id}` and `/wait`

Why namespaced routes instead of root-level mounting:

1. Prevents collisions with existing `/api/*` and `/ws` endpoints.
2. Allows frontend side-by-side usage (chat and confirm simultaneously).
3. Makes traffic observability and reverse-proxy rules clearer.

### Packaging Strategy (Required Prerequisite)

#### Current blocker

`go-inventory-chat` cannot import:

1. `github.com/go-go-golems/plz-confirm/internal/server`
2. `github.com/go-go-golems/plz-confirm/internal/store`
3. `github.com/go-go-golems/plz-confirm/internal/scriptengine`

because of Go internal visibility.

#### Required extraction

Status: implemented in this ticket as `plz-confirm/pkg/backend`.

Implemented public API surface used by host applications:

```go
type Server struct { ... }

func NewServer() *Server
func (s *Server) Handler() http.Handler
func (s *Server) ListenAndServe(ctx context.Context, opts ListenOptions) error
func (s *Server) Mount(mux *http.ServeMux, prefix string)
func Mount(mux *http.ServeMux, prefix string, handler http.Handler)
```

Notes:

1. Internal store/scriptengine remain encapsulated behind the public server wrapper.
2. Prefix mounting is handled by the public package and used by `go-inventory-chat` for `/confirm/*`.
3. CLI `serve` now consumes `pkg/backend` so the public surface is exercised by default.

#### Module/workspace update

Current `go.work` includes `go-go-goja` and `plz-confirm`, but not `go-go-os/go-inventory-chat`.

For local integration development, add module to workspace:

1. `./go-go-os/go-inventory-chat`

and in `go-go-os/go-inventory-chat/go.mod` add dependency on:

1. `github.com/go-go-golems/plz-confirm`

(with local workspace resolution during development; versioned module for CI later).

### Frontend Bridge Strategy in go-go-os

Package-first strategy (updated): implement frontend confirm integration as a reusable package in `go-go-os/packages/confirm-runtime` from day 1, then consume it in `apps/inventory`.

`packages/confirm-runtime` responsibilities:

1. `confirmRuntime` state slice (active requests, pending queue, history).
2. `confirmWsManager` service (session WS connection to `/confirm/ws`).
3. `confirmApi` service (`submitResponse`, `submitScriptEvent`, `touch`, etc.).
4. `ConfirmRequestWindow` renderer family (widget-specific components).
5. Pluggable host hooks for app integration:
   - base URL resolver,
   - session ID provider,
   - `openWindow` adapter callback,
   - optional telemetry callbacks.

`apps/inventory` responsibilities:

1. Wire package reducer/middleware/services into app store bootstrap.
2. Register desktop contributions (`Confirm Queue`, optional hotkeys/menu items).
3. Delegate `renderAppWindow` for `confirm-request:<requestId>` to package renderers.
4. Keep inventory-specific window placement defaults and icon conventions.

Window model recommendation (unchanged):

1. One desktop window per request ID.
2. `dedupeKey = confirm-request:<requestId>`.
3. On completion, either:
   - close window automatically (default), or
   - show read-only completion state for a short grace period.

## Detailed Widget Mapping

Map plz-confirm input/output contracts to go-go-os widget components.

### 1. Confirm

Backend contract:

1. Input: `ConfirmInput { title, message, approveText, rejectText }`
2. Output: `ConfirmOutput { approved, timestamp, optional comment }`

go-go-os implementation:

1. Reuse `AlertDialog` style and `Btn` primitives.
2. Add optional comment field below actions.
3. Emit output with timestamp generated client-side to match existing behavior.

Gap analysis:

1. Needs dedicated `ConfirmRequestWindow` wrapper with comment support.

### 2. Select

Backend contract:

1. Input: `SelectInput { title, options[], multi, searchable }`
2. Output: single or multi selection + optional comment.

go-go-os implementation:

1. Reuse `ListBox` / possibly `DropdownMenu` depending on count.
2. Add search input when `searchable == true`.
3. Support both simple string options and rich object options (plz-confirm script extension currently accepts rich options in frontend behavior).

Gap analysis:

1. `ListBox` currently assumes simple strings and no metadata badges/icons.
2. Need `SelectRequestWindow` abstraction with richer option rows.

### 3. Form

Backend contract:

1. Input: `FormInput { title, schema (JSON Schema-ish Struct) }`
2. Output: `FormOutput { data Struct, optional comment }`

go-go-os implementation:

1. Reuse `FormView` and `FieldRow` for simple primitives.
2. Add schema-to-fields mapper covering common types:
   - string, number, boolean, enum/select
3. Preserve unknown schema shapes by rendering fallback JSON editor block (phase 2+).

Gap analysis:

1. Existing `FormView` expects pre-built `FieldConfig[]`; no schema parser yet.

### 4. Table

Backend contract:

1. Input: rows (Struct[]), columns[], multiSelect, searchable
2. Output: selected row(s) + optional comment

go-go-os implementation:

1. Reuse `DataTable` with selection layer.
2. Add client-side filtering and sort toggles (phase 1 minimal: filtering only).
3. Convert selected row(s) back to generic JSON object for API submit.

Gap analysis:

1. `DataTable` currently read-focused; selection utilities need to be added.

### 5. Upload

Backend contract:

1. Input: title, accept[], multiple, maxSize, callbackUrl
2. Output: `UploadedFile[]` metadata + optional comment

go-go-os implementation:

1. Build native file picker + drag-drop component in desktop window.
2. Use plz-confirm upload API (`POST /confirm/api/images` currently image-only; may need generalized file endpoint if not existing in plz-confirm).
3. Persist uploaded server path in output payload.

Gap analysis:

1. plz-confirm existing browser UploadDialog currently simulates upload client-side in UI code; integration should avoid simulation and use real backend endpoint.
2. Verify/extend plz-confirm server for general uploads if required beyond image path.

### 6. Image

Backend contract:

1. Input: image list + mode (`select` or `confirm`) + optional options/multi
2. Output: union type (index selection, bool confirmation, option strings, multi forms)

go-go-os implementation:

1. Add `ImageRequestWindow` with grid layout and confirm/select variants.
2. Reuse existing desktop theming and button primitives.
3. Keep exact output union semantics to avoid backend mismatch.

Gap analysis:

1. Need image loading error states and multi-select UX.

### 7. Script (critical for PC-05)

Backend contract:

1. Input includes script source and props.
2. Backend returns and updates:
   - `scriptState`
   - `scriptView`
   - `scriptDescribe`
   - `scriptLogs`
3. Frontend submits `ScriptEvent` to `/event` endpoint.

go-go-os implementation principle:

1. Frontend does **no script execution**.
2. It renders current `scriptView` only.
3. On interaction, emits `ScriptEvent` back to backend.

Script view rendering requirements:

1. Single-widget mode: render `scriptView.widgetType + input`.
2. Sections mode: render display sections + exactly one interactive section.
3. Back action support using `allowBack/backLabel`.
4. Progress bar support (`current/total/label`).
5. Toast support (`message/style/durationMs`).

Gap analysis:

1. go-go-os does not currently have a script section renderer equivalent.
2. Must implement a `ScriptRequestWindow` component orchestrating these rules.

## Router Integration Design

### Proposed Route Table (host process)

| Route Group | Purpose | Owner |
|---|---|---|
| `/chat`, `/ws`, `/api/timeline`, `/api/*` | Inventory chat and timeline | go-inventory-chat / pinocchio |
| `/confirm/api/*`, `/confirm/ws` | plz-confirm request workflow | plz-confirm public package mounted by host |
| `/` | Inventory app UI | go-inventory-chat static/UI handler |

### Mounting Pattern

Implementation shape in `main.go` after creating `appMux`:

1. Instantiate plz-confirm server object in-process.
2. Mount it under `/confirm` via the public `Mount` helper.
3. Ensure CORS/websocket behavior remains compatible under prefixed paths.

Implemented code shape:

```go
confirmSrv := plzconfirmbackend.NewServer()
confirmSrv.Mount(appMux, "/confirm")
```

### Session strategy

Recommendation for phase 1:

1. Reuse existing plz-confirm session concept.
2. Default session = `global` unless inventory app introduces explicit user/session partition.
3. WS URL from frontend uses `sessionId` query param.

### Root-prefix interaction

`go-inventory-chat` already supports `--root` and strips prefix.

Important requirement:

1. When root prefix is active, both chat routes and `/confirm/*` routes must remain reachable through the same prefix.
2. Frontend must derive base prefixes consistently (`/confirm` plus optional host root prefix).

## Backend-Frontend Messaging Design

### Message transport layers

We now have two websocket channels in the integrated app:

1. Chat channel: `ws?conv_id=...` (SEM frames for timeline/chat)
2. Confirm channel: `confirm/ws?sessionId=...` (request lifecycle events)

Keep them separate in phase 1.

Why:

1. Different payload schemas and semantics.
2. Lower coupling and lower migration risk.
3. Easier to debug with per-channel logs.

### Confirm event state machine

Expected frontend state transitions:

```text
new_request      -> active/pending queue -> window open
request_updated  -> patch existing request + rerender current window
request_completed-> mark complete + close/move to history
```

Deduplication requirements (mirror plz-confirm client behavior):

1. Ignore updates for already-completed request IDs.
2. Handle out-of-order arrivals by enqueueing unknown updates as new requests.
3. Keep bounded completed-ID cache to prevent memory growth.

### API submit semantics

For non-script widgets:

1. Submit to `/confirm/api/requests/{id}/response` with typed output oneof JSON.

For script widgets:

1. Submit to `/confirm/api/requests/{id}/event` with `{type, stepId, actionId?, data}`.
2. If response comes back completed, close request window and archive.
3. Else patch state/view and rerender.

### Touch semantics

To preserve timeout-disable behavior:

1. On first interaction in a window, call `/confirm/api/requests/{id}/touch`.
2. Avoid duplicate calls with in-flight/touched cache.

## Widget Construction in macOS Desktop UI

### Window composition model

Each request window should use a consistent shell:

1. Title bar:
   - widget type badge
   - request short ID
   - optional timer / timeout-disabled indicator
2. Body:
   - widget-specific interactive content
3. Footer:
   - optional comment field
   - submit buttons
   - script progress/toast/back affordances when type = script

### Core-first widgets to adapt/add

Updated strategy: add generic, reusable interaction primitives to `@hypercard/engine` first, then keep plz-confirm-specific orchestration in `packages/confirm-runtime`.

Add/adapt in `@hypercard/engine`:

1. `SelectableList` (from `ListBox` baseline):
   - single/multi selection,
   - optional search box,
   - rich rows (`label`, `description`, `icon`, `meta`),
   - keyboard navigation and `onSubmit` hook.
2. `SelectableDataTable` (from `DataTable` baseline):
   - single/multi row selection,
   - optional filter/search text,
   - selected row extraction helpers.
3. `SchemaFormRenderer` (on top of `FormView`/`FieldRow`):
   - JSON-schema-ish mapping to engine `FieldConfig[]`,
   - core primitive coverage: string/number/boolean/enum,
   - fallback passthrough for unsupported fields.
4. `FilePickerDropzone`:
   - drag/drop + native picker,
   - accept/type/size constraints,
   - multi-file support.
5. `ImageChoiceGrid`:
   - select/confirm/multi-select modes,
   - optional labels and option badges,
   - loading/error placeholders.
6. `RequestActionBar`:
   - primary/secondary actions,
   - optional comment input,
   - loading/disabled state handling.

Keep in `packages/confirm-runtime` (not engine):

1. `UIRequest` protocol decoding/encoding.
2. Request lifecycle reconciliation for `new_request`, `request_updated`, `request_completed`.
3. Script sections orchestration (`interactive` + `display`, back/progress/toast).
4. Endpoint/path/session plumbing specific to `/confirm/*`.

### Existing primitives to reuse

1. Buttons: `Btn`
2. Lists: `ListBox`
3. Tables: `DataTable`
4. Forms: `FormView`/`FieldRow`
5. Dialog style: `AlertDialog`
6. Progress indicator: `ProgressBar`
7. Desktop styling + data-part theming tokens

## Script Runtime Boundary and Interoperability

### Do not conflate runtimes

`go-go-os` plugin runtime:

1. QuickJS
2. card-centric UI DSL
3. runtime intents to Redux domain/system

`plz-confirm` script runtime:

1. Goja
2. describe/init/view/update contract
3. view schema maps to plz-confirm widget model

Phase-1 rule:

1. plz-confirm script runtime remains authoritative.
2. go-go-os frontend is a projection target for `scriptView` only.

### Future interoperability option (not now)

Could later map certain `scriptView` shapes into HyperCard plugin-card UINodes, but that is phase 2/3 and not part of this ticket scope.

## Security and Reliability Considerations

### Security

1. Keep script execution server-side with existing timeouts and cancellation semantics.
2. Preserve typed proto contract validation on backend.
3. Maintain origin policy awareness for mounted WS/API endpoints (currently permissive in both stacks).
4. Ensure uploaded file/image APIs enforce size/type constraints and TTL cleanup.

### Reliability

1. WS reconnect with exponential or fixed backoff.
2. On reconnect, server sends pending requests; client must reconcile with existing state.
3. All state transitions idempotent by request ID.
4. Route-level health checks recommended:
   - chat health
   - confirm health

### Observability

Add structured logs/metrics per subsystem:

1. Confirm request create/update/complete counts.
2. Script update error/timeout/cancel rates.
3. WS connect/disconnect counts for confirm channel.
4. Widget submit latency and failure counts.

## Implementation Plan (Phased)

## Phase 0: Prerequisites and Contract Freeze

1. Freeze plz-confirm request/event JSON contracts for this integration cycle.
2. Add explicit fixture set of sample requests for each widget type including script sections.
3. Confirm route namespace decision (`/confirm/*`).

Deliverables:

1. Contract fixture docs.
2. Confirmed API path map.

## Phase 1: Make plz-confirm Embeddable

1. Extract public server package from `internal/server` and supporting internals.
2. Keep `cmd/plz-confirm` behavior unchanged (regression guard).
3. Add focused integration tests in plz-confirm for embedded mode under prefix.

Deliverables:

1. Public package importable by external modules.
2. Tests proving handler works under `http.StripPrefix`.

## Phase 2: Mount confirm routes in go-inventory-chat

1. Add plz-confirm dependency to `go-inventory-chat/go.mod`.
2. Mount handler under `/confirm` in main mux.
3. Add server integration tests validating:
   - `/chat` unaffected
   - `/confirm/api/requests` works
   - `/confirm/ws` works

Deliverables:

1. Single process exposing both chat and confirm stacks.
2. Regression tests for route compatibility.

## Phase 3: Add reusable core widgets in `@hypercard/engine`

1. Implement `SelectableList` as a superset of current `ListBox`.
2. Implement `SelectableDataTable` as a superset of current `DataTable`.
3. Implement `SchemaFormRenderer` above `FormView`/`FieldRow`.
4. Add `FilePickerDropzone`, `ImageChoiceGrid`, and `RequestActionBar`.
5. Export all new primitives from engine barrel + storybook examples.

Deliverables:

1. Generic widgets available to any app/package in go-go-os.
2. No plz-confirm coupling in engine layer.

## Phase 4: Build `packages/confirm-runtime` (package-first)

1. Create new workspace package `packages/confirm-runtime`.
2. Add request store, ws manager, api client, and request-window renderer orchestration.
3. Implement plz-confirm-specific widget adapters using core engine widgets.
4. Implement script view orchestration (single + sections, back/progress/toast).

Deliverables:

1. Reusable confirm frontend runtime package with app-agnostic host adapters.

## Phase 5: Integrate package into `apps/inventory`

1. Register confirm-runtime reducer/services in app store bootstrap.
2. Wire desktop contribution commands and `renderAppWindow` delegation.
3. Auto-open/close request windows with dedupe semantics.
4. Validate full flow for core widgets and script widget.

Deliverables:

1. End-to-end confirm workflows running inside inventory desktop shell using shared package.

## Phase 6: Hardening and rollout

1. Add full e2e tests.
2. Add metrics and logging dashboards.
3. Feature-flag rollout and fallback path (old frontend optionally retained during soak).

Deliverables:

1. Production-ready integration with rollback strategy.

## Implementation Status (2026-02-23)

Completed in this ticket execution:

1. **Engine generic widgets tranche** (commit `48c2724` in `go-go-os`):
   - Added `SelectableList`, `SelectableDataTable`, `SchemaFormRenderer`, `FilePickerDropzone`, `ImageChoiceGrid`, `RequestActionBar`.
   - Added helper-focused unit tests for selection/table/schema coercion logic.
   - Exported new widgets via engine widget barrel.
2. **Package-first confirm runtime scaffold** (commit `6e38a7d` in `go-go-os`):
   - Created `packages/confirm-runtime` with api/ws/state/host/runtime/component layers.
   - Added app-agnostic host adapters and runtime wiring skeleton.
   - Added workspace wiring in root `tsconfig.json` and root build script.
3. **Inventory host integration** (commit `af1a085` in `go-go-os`):
   - Wired `confirmRuntime` reducer/services into `apps/inventory`.
   - Added request window delegation (`confirm-request:<id>`) and queue command/window.
   - Added dev proxy routes and alias wiring for `/confirm` + `/confirm/ws`.
4. **Backend embeddable extraction and mount** (commits `56e40ec` in `plz-confirm`, `3e79c2a` in `go-go-os`):
   - Added `plz-confirm/pkg/backend` public wrapper around internal server/store.
   - Switched `cmd/plz-confirm serve` to the new public package.
   - Mounted plz-confirm backend under `/confirm/*` in `go-inventory-chat`.
   - Added integration tests for route coexistence and prefixed confirm websocket replay.

Still pending:

1. Manual UI lifecycle validation for open/close/focus behavior under real confirm traffic.
2. End-to-end script sections parity and upload/image backend semantics hardening.
3. Frontend visual consistency pass (tracked in `PC-06-UI-CONSISTENCY-HANDOFF`).

Validation notes from this run:

1. `go test ./pkg/backend ./cmd/plz-confirm -count=1` passed.
2. `go test ./... -count=1` passed in `plz-confirm` during pre-commit hook.
3. `go test ./... -count=1` passed in `go-go-os/go-inventory-chat` with workspace module resolution.
4. `go-inventory-chat` currently resolves `github.com/go-go-golems/plz-confirm/pkg/backend` via local workspace composition; published `plz-confirm` `v0.0.3` does not yet include this package.

## File-Level Change Sketch (Expected)

### plz-confirm repo

1. Implemented public backend package:
   - `pkg/backend/backend.go`
   - `pkg/backend/backend_test.go`
2. Internal store/scriptengine remain wrapped behind public `pkg/backend`.
3. CLI wiring in `cmd/plz-confirm/main.go` now points to `pkg/backend`.

### go-inventory-chat backend

1. `go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main.go`
   - mounts confirm handler under `/confirm/*` via `plzconfirmbackend.NewServer().Mount(...)`
2. Add integration tests in server test file for confirm endpoints.

### go-go-os frontend (package-first)

1. New package `go-go-os/packages/confirm-runtime/*`:
   - request store slice and selectors
   - ws manager + api client
   - request host and widget adapters
   - script-specific orchestration
2. Engine additions in `go-go-os/packages/engine/src/components/widgets/*`:
   - `SelectableList`
   - `SelectableDataTable`
   - `SchemaFormRenderer`
   - `FilePickerDropzone`
   - `ImageChoiceGrid`
   - `RequestActionBar`
3. `go-go-os/apps/inventory/src/App.tsx` + store:
   - thin integration layer only (registration, window open callbacks, app-key delegation).

## Testing Strategy

### Unit tests

1. Serialization tests for each widget output payload.
2. Script section rendering invariants:
   - exactly one interactive section
   - display sections allowed
3. Deduplication behavior for WS events.

### Integration tests (backend)

1. `go-inventory-chat` route coexistence test (`/chat` + `/confirm/*`).
2. WS handshake and pending replay for confirm channel.
3. Script event endpoint behavior under mount prefix.

### End-to-end tests

Scenario matrix:

1. Confirm widget request -> approve/reject.
2. Select single/multi with search.
3. Form submit validation.
4. Table single/multi selection.
5. Image select + confirm modes.
6. Script flow with:
   - progress updates,
   - back button,
   - display+interactive sections,
   - terminal completion.

### Performance tests

1. Burst of 100 pending requests should not freeze desktop UI.
2. Multiple concurrent request windows remain interactive.

## Risks and Mitigations

### Risk 1: Internal package extraction churn in plz-confirm

Impact:

1. Medium to high; could destabilize existing plz-confirm command behavior.

Mitigation:

1. Extract with wrappers first (minimal movement), then deeper refactor later.
2. Run existing plz-confirm test suite before and after extraction.

### Risk 2: Route namespace confusion

Impact:

1. Medium; frontend might accidentally call wrong `/api` endpoints.

Mitigation:

1. Centralize base URLs in one confirm API client module.
2. Add explicit tests for URL generation.

### Risk 3: Script UI parity gaps

Impact:

1. Medium; script flows may behave differently from existing plz-confirm UI.

Mitigation:

1. Mirror current `WidgetRenderer` logic exactly first.
2. Keep conformance fixtures and golden tests for script views.

### Risk 4: Upload/image behavior mismatch

Impact:

1. Medium; existing plz-confirm browser upload component currently simulates flows in some paths.

Mitigation:

1. Define and test real backend upload path requirements early.
2. Explicitly mark unsupported modes until completed.

## Alternatives Considered

### Alternative A: Redirect to existing plz-confirm web frontend in separate tab/window

Pros:

1. Lowest implementation effort.

Cons:

1. Fails requirement of native macOS desktop windowing frontend.
2. Creates context switch and UX fragmentation.

Decision:

1. Rejected.

### Alternative B: Re-implement plz-confirm backend logic directly in go-inventory-chat

Pros:

1. Full local control.

Cons:

1. Duplicates a mature system.
2. High long-term maintenance and divergence risk.
3. Violates stated requirement to keep script functionality in plz-confirm backend.

Decision:

1. Rejected.

### Alternative C: Convert plz-confirm scripts to go-go-os plugin runtime immediately

Pros:

1. One runtime model eventually.

Cons:

1. High complexity and high migration risk.
2. Not required for current objective.

Decision:

1. Deferred to possible future phase.

## Design Decisions

1. **Use route namespace `/confirm/*`** to avoid collisions and support side-by-side chat.
2. **Extract embeddable plz-confirm public package** instead of illegal internal imports.
3. **Frontend-only projection of script views**; no script runtime execution in go-go-os for this phase.
4. **One request = one desktop window** for clarity and parallel interactions.
5. **Mirror plz-confirm WS lifecycle semantics** (`new_request`, `request_updated`, `request_completed`) with idempotent client reconciliation.
6. **Use package-first frontend architecture**: build `packages/confirm-runtime` for reuse from day 1, with `apps/inventory` as the first host adapter.
7. **Upgrade engine with core-generic widgets first** (`SelectableList`, `SelectableDataTable`, `SchemaFormRenderer`, `FilePickerDropzone`, `ImageChoiceGrid`, `RequestActionBar`) and keep plz-confirm protocol logic out of engine.

## Intern Onboarding Runbook (First 5 Days)

### Day 1: Read and run

1. Read this document fully.
2. Read:
   - `go-go-os` desktop architecture overview docs.
   - `plz-confirm` `js-script-development.md` and `js-script-api.md`.
3. Run both systems locally and capture baseline behavior.

### Day 2: Contract fixtures

1. Build fixture JSON for each plz-confirm widget input/output pair.
2. Add script fixtures covering sections/progress/back/toast.
3. Write tiny parser tests on frontend side.

### Day 3: Backend mounting spike

1. Spike embeddable plz-confirm package extraction (minimal wrapper path).
2. Mount under `/confirm` in `go-inventory-chat`.
3. Add route coexistence tests.

### Day 4: Frontend bridge skeleton

1. Create `packages/confirm-runtime` scaffold and wire package exports.
2. Add confirm WS manager + Redux slice in package.
3. Define host adapter interfaces for window opening and endpoint/session resolution.

### Day 5: First widget end-to-end

1. Add first core widget upgrades in engine (`SelectableList` + `RequestActionBar` minimum).
2. Implement confirm widget fully in confirm-runtime using those engine primitives.
3. Demo full create/respond/complete cycle in inventory host and document gaps.

## Open Questions

1. Should confirm route namespace be `/confirm` or `/plz-confirm`? (recommend `/confirm` for brevity)
2. Do we need auth/session bridging between chat and confirm channels now, or can both remain `global` session initially?
3. Should completed request windows auto-close immediately or show a short success state?
4. What is the long-term plan for upload APIs beyond images in plz-confirm backend?
5. Should any of the new core widgets remain internal initially, or should all six ship as stable engine exports immediately?
6. How should feature flags control rollout (env var in backend, frontend toggle, or both)?

## References

1. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/go-inventory-chat/cmd/hypercard-inventory-server/main.go`
2. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/go-inventory-chat/internal/pinoweb/hypercard_events.go`
3. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/shell/windowing/useDesktopShellController.tsx`
4. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/desktop/core/state/windowingSlice.ts`
5. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/shell/windowing/defaultWindowContentAdapters.tsx`
6. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/components/shell/windowing/PluginCardSessionHost.tsx`
7. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/plugin-runtime/runtimeService.ts`
8. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/hypercard/artifacts/artifactProjectionMiddleware.ts`
9. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/go-go-os/packages/engine/src/chat/ws/wsManager.ts`
10. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/internal/server/server.go`
11. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/internal/server/script.go`
12. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/internal/server/ws.go`
13. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/internal/scriptengine/engine.go`
14. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/internal/store/store.go`
15. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/proto/plz_confirm/v1/request.proto`
16. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/proto/plz_confirm/v1/widgets.proto`
17. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/agent-ui-system/client/src/components/WidgetRenderer.tsx`
18. `/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/agent-ui-system/client/src/services/websocket.ts`
