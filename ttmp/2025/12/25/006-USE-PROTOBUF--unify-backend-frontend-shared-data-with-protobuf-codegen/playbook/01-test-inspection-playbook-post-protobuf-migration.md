---
Title: Test inspection playbook (post-protobuf migration)
Ticket: 006-USE-PROTOBUF
Status: active
Topics:
    - backend
    - frontend
    - api
    - protobuf
    - codegen
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-26T09:19:52.74968909-05:00
WhatFor: ""
WhenToUse: ""
---

# Test inspection playbook (post-protobuf migration)

## Purpose

Provide a repeatable, team-friendly inspection procedure to validate **all critical behaviors** of plz-confirm after the protobuf migration:

- Backend REST + WebSocket contracts (protojson, legacy enum strings, oneof shapes)
- CLI end-to-end behavior for each widget
- Frontend rendering + submission for each widget
- Images endpoint and image widget variants
- Tooling contracts (codegen, lint, tests)

## Environment Assumptions

- You have the repo checked out and dependencies installed (Go toolchain + pnpm).
- Ports available:
  - Backend API on `:3001`
  - Vite UI on `:3000` (proxying `/api` and `/ws` → `:3001`)
- Tools available:
  - `go`, `pnpm`, `tmux`, `curl`, `jq`, `protoc`

### Canonical dev topology

- Backend: `http://localhost:3001`
- UI: `http://localhost:3000`

This matches `agent-ui-system/vite.config.ts` proxy settings.

## Commands

```bash
REPO="/home/manuel/workspaces/2025-12-24/add-img-widget-plz-confirm/plz-confirm"
cd "$REPO"

# 0) Fast correctness gates (must be green)
go test ./... -count=1
buf lint .
cd agent-ui-system && pnpm run check && cd ..

# 1) Start dev stack (tmux)
bash scripts/tmux-up.sh
tmux attach -t PLZ-CONFIRM
```

## Exit Criteria

- **All gates pass**:
  - `go test ./... -count=1`
  - `buf lint .`
  - `pnpm -C agent-ui-system check`
- **Backend/API smoke passes** (UI-less):
  - `bash scripts/curl-inspector-smoke.sh` succeeds
- **CLI manual suite passes** (UI-driven):
  - `bash ttmp/2025/12/15/DESIGN-PLZ-CONFIRM-001--port-agent-ui-system-cli-backend-to-go-using-glazed-framework/scripts/test-all-commands.sh`
- **CLI auto suite passes** (API-driven submit):
  - `bash ttmp/2025/12/24/001-ADD-IMG-WIDGET--add-image-widget-to-cli-and-web-interface/scripts/auto-e2e-cli-via-api.sh`
  - `bash ttmp/2025/12/24/004-ADD-WIDGET-COMMENT--add-freeform-comment-field-to-all-widgets/scripts/auto-e2e-comment-via-api.sh`
- **Frontend manual checks pass** (see below).

## Notes

## What changed with the protobuf migration (testing implications)

### 1) Outputs now use protobuf oneof JSON shapes

For some widgets, the JSON submitted to `/api/requests/{id}/response` changed from “union-ish” shapes to explicit oneof fields:

- **SelectOutput**
  - Single: `{"selectedSingle":"staging"}`
  - Multi: `{"selectedMulti":{"values":["a","b"]}}`
- **TableOutput**
  - Single: `{"selectedSingle":{...row...}}`
  - Multi: `{"selectedMulti":{"values":[...rows...]}}`
- **ImageOutput**
  - Confirm: `{"selectedBool":true,"timestamp":"..."}`
  - Pick single image: `{"selectedNumber":0,"timestamp":"..."}`
  - Pick multiple images: `{"selectedNumbers":{"values":[0,2]},"timestamp":"..."}`
  - Options single: `{"selectedString":"Wrong theme","timestamp":"..."}`
  - Options multi: `{"selectedStrings":{"values":["Wrong theme","Missing icon"]},"timestamp":"..."}`

### 2) Enums are emitted as legacy strings

The server uses `protojson` and the enum value names are chosen so JSON still uses:
- `type: "confirm" | "select" | ...`
- `status: "pending" | "completed" | ...`

## Inspector checklist (manual UI)

### A) WebSocket connectivity
- Open `http://localhost:3000`
- Confirm the UI shows “connected” state (or no WS error banner).
- In browser devtools console, verify incoming WS messages:
  - `type: "new_request"`
  - `type: "request_completed"`

### B) Widget-by-widget manual interaction (UI + CLI)

Run:

```bash
cd "$REPO"
bash ttmp/2025/12/15/DESIGN-PLZ-CONFIRM-001--port-agent-ui-system-cli-backend-to-go-using-glazed-framework/scripts/test-all-commands.sh
```

For each widget:
- Confirm UI renders correctly from protobuf-shaped request (`confirmInput`, `selectInput`, etc).
- Submit action.
- Confirm CLI unblocks and prints result.

### C) Images pipeline
- CLI image widget using local files uploads to `/api/images`
- UI can fetch the served image bytes from `/api/images/{id}`
- Validate Variant A (pick image) and Variant B (options).

## Inspector checklist (UI-less / curl emulation)

Run:

```bash
cd "$REPO"
API_BASE_URL=http://localhost:3001 bash scripts/curl-inspector-smoke.sh
```

This emulates the inspector “clicks” by submitting `/response` with correct oneof shapes.

### 3) Note on 64-bit integers in protojson

`protojson` serializes 64-bit integers as **strings** in JSON (to preserve precision). You may see:
- `maxSize: "1234"`
- `size: "1"`
- `selectedNumber: "0"`

This is expected and should not be treated as a failure.

## Troubleshooting quick hits

- **Server not reachable**: start `go run ./cmd/plz-confirm serve --addr :3001`
- **UI not reachable**: `pnpm -C agent-ui-system dev --host --port 3000`
- **Auto E2E scripts failing**: ensure server logs are being teed to `/tmp/plz-confirm-server.log`
- **Enum string mismatch**: validate `.type` is `confirm` etc (not `WIDGET_TYPE_CONFIRM`).
