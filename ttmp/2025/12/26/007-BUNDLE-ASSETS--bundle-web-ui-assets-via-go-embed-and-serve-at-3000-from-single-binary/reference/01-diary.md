---
Title: Diary
Ticket: 007-BUNDLE-ASSETS
Status: active
Topics:
    - backend
    - build
    - release
    - embed
    - assets
    - ui
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-26T13:27:01.221542379-05:00
WhatFor: ""
WhenToUse: ""
---

# Diary

## Goal

Step-by-step narrative of making the plz-confirm Go server serve the Web UI from a single binary on `http://localhost:3000/` (no Vite required), using `go:embed` in production builds and a sane local-dev workflow.

## Context

- The server already exposes REST under `/api/*` and WS under `/ws`.
- The frontend is built by Vite in `agent-ui-system/`.
- The codebase already has a bundling pipeline (`go generate` builds UI and copies into `internal/server/embed/public`), but embedding is gated by a Go build tag.

## Quick Reference

### Key files / symbols

- `internal/server/server.go`: `(*Server).handleStaticFiles` (static serving + SPA fallback)
- `internal/server/embed.go`: `embeddedPublicFS` (only when building with `-tags embed`)
- `internal/server/embed_none.go`: `embeddedPublicFS = nil` in default builds
- `internal/server/generate.go`: `//go:generate` entrypoint
- `internal/server/generate_build.go`: runs `pnpm run build` and copies output into `internal/server/embed/public`

## Usage Examples

### “No Vite” run (embedded build)

```bash
make build
./dist/plz-confirm serve --addr :3000
```

### Smoke checks

```bash
curl -sSf http://localhost:3000/ | head
curl -sSfI http://localhost:3000/api/requests || true
```

## Related

- `analysis/01-current-bundling-pipeline-and-recommended-approach.md`
