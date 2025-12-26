---
Title: Add Image Widget to CLI and Web Interface
Ticket: 001-ADD-IMG-WIDGET
Status: complete
Topics:
    - cli
    - backend
    - agent-ui-system
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: agent-ui-system/client/src/components/WidgetRenderer.tsx
      Note: Widget router - need to add image case
    - Path: agent-ui-system/client/src/components/widgets/ConfirmDialog.tsx
      Note: Reference for confirmation UI patterns
    - Path: agent-ui-system/client/src/components/widgets/SelectDialog.tsx
      Note: Reference for selection UI patterns
    - Path: agent-ui-system/client/src/types/schemas.ts
      Note: TypeScript type definitions - need to add ImageInput/ImageOutput
    - Path: cmd/plz-confirm/main.go
      Note: Command registration - need to add imageCmd
    - Path: internal/cli/confirm.go
      Note: Reference implementation for CLI command pattern
    - Path: internal/cli/select.go
      Note: Reference implementation for selection-based widget
    - Path: internal/server/server.go
      Note: Backend server - widget-agnostic
    - Path: internal/types/types.go
      Note: Defines widget types and Input/Output structs - need to add WidgetImage
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-24T22:21:40.325038249-05:00
WhatFor: ""
WhenToUse: ""
---



# Add Image Widget to CLI and Web Interface

## Overview

This ticket adds a new **image widget** to plz-confirm that allows AI models to:
- Display a text prompt with one or more images
- Present selection options (single-select, multi-select) or confirmation buttons
- Receive user feedback about image similarity, selection, or confirmation

The widget supports two interaction modes:
- **Select mode**: Display images with labels/options, allow user to select one or more
- **Confirm mode**: Display images with approve/reject buttons for similarity checks or confirmations

**Current Status**: Analysis phase complete. Comprehensive analysis document created covering CLI, backend, and frontend implementation requirements.

**Key Documents**:
- [Design Document](./design-doc/01-image-widget-design-specification.md) - Complete design specification with all decisions and rationale
- [Analysis Document](./analysis/01-image-widget-implementation-analysis.md) - Complete architecture analysis and implementation requirements
- [Diary](./reference/01-diary.md) - Step-by-step research and decision tracking

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **complete**

## Testing (dev stack in tmux)

The most reliable local dev topology we used while building/testing this widget is:

- **Go backend** on `:3001`
- **Vite UI** on `:3000` proxying `/api` and `/ws` → `:3001`
- CLI commands use `--base-url http://localhost:3000`

### Start server + Vite in tmux (with logs)

```bash
cd /home/manuel/workspaces/2025-12-24/add-img-widget-plz-confirm/plz-confirm && \
tmux kill-session -t plzimg 2>/dev/null || true && \
rm -f /tmp/plz-confirm-server.log /tmp/plz-confirm-vite.log && \
tmux new-session -d -s plzimg -n server "cd /home/manuel/workspaces/2025-12-24/add-img-widget-plz-confirm/plz-confirm && go run ./cmd/plz-confirm serve --addr :3001 2>&1 | tee /tmp/plz-confirm-server.log" && \
tmux new-window -t plzimg -n vite "cd /home/manuel/workspaces/2025-12-24/add-img-widget-plz-confirm/plz-confirm && pnpm -C agent-ui-system dev --host 2>&1 | tee /tmp/plz-confirm-vite.log" && \
tmux attach -t plzimg
```

### Run widget commands (manual browser validation)

Open `http://localhost:3000` in your browser, then run (examples):

```bash
cd /home/manuel/workspaces/2025-12-24/add-img-widget-plz-confirm/plz-confirm && \
go run ./cmd/plz-confirm image \
  --base-url http://localhost:3000 \
  --title "Manual check: image widget" \
  --message "Pick an image and submit." \
  --image /absolute/path/to/a.png --image-label "A" \
  --image /absolute/path/to/b.png --image-label "B" \
  --output yaml
```

### Inspect logs

- Server: `tail -n 200 /tmp/plz-confirm-server.log`
- Vite: `tail -n 200 /tmp/plz-confirm-vite.log`

### Scripted smoke tests (no browser clicks)

This ticket also includes scripts that validate plumbing by auto-submitting responses via the API:

- `scripts/auto-e2e-cli-via-api.sh`
- `scripts/manual-test-image-widget-all-options.sh`

## Topics

- cli
- backend
- agent-ui-system

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
