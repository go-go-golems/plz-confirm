---
Title: Static assets build + serving for plz-confirm
Ticket: 004-BUG-LANDING-PAGE
Status: active
Topics:
    - web
    - backend
    - static
    - bug
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-03T14:54:28.846381363-05:00
WhatFor: ""
WhenToUse: ""
---

# Static assets compilation + serving (and why `/` 404s on `go run ... serve`)

## Executive summary

`go run ./cmd/plz-confirm serve` currently compiles the backend without the `embed` build tag, which means the embedded static filesystem (`embeddedPublicFS`) is nil. The server therefore does not register a `/` handler, so `/` returns `404 page not found` even though the React SPA exists in the repo.

Static assets are only served when:
- the frontend is built (Vite output), and
- those built files are copied into `internal/server/embed/public/`, and
- the backend is compiled (it always embeds `internal/server/embed/public/`).

## What the frontend build produces

The frontend lives under `agent-ui-system/` and is built with Vite (via `pnpm run build`).

Build output (expected):
- `agent-ui-system/dist/public/index.html`
- `agent-ui-system/dist/public/assets/*`

The `index.html` in this project references assets at absolute paths like `/assets/index-*.js` and `/assets/index-*.css`, so the backend must serve `/assets/*` for the SPA to load.

## How those files end up in the Go server

The embedding flow is:

1. `go generate ./internal/server` (or `go generate ./...`) runs:
   - `internal/server/generate.go` → `go run generate_build.go`
2. `internal/server/generate_build.go` does:
   - runs `pnpm run build` (in `agent-ui-system/`)
   - copies `agent-ui-system/dist/public/*` into `internal/server/embed/public/*`

So after codegen, the Go package contains the frontend artifacts at:
- `internal/server/embed/public/index.html`
- `internal/server/embed/public/assets/*`

## How the Go server serves the SPA

Serving is implemented in `internal/server/server.go`:

- `(*Server).Handler()` registers API routes first (`/api/*`, `/ws`), then calls `s.handleStaticFiles(mux)`.
- `handleStaticFiles` returns early unless it can open `index.html` from `embeddedPublicFS`.
- When enabled, it registers a handler at `/` which:
  - refuses `/api*` and `/ws*` (so static serving can’t shadow API routes)
  - serves a file directly if it exists in the embedded FS
  - otherwise falls back to serving `index.html` (SPA client-side routing support)

## Where `embeddedPublicFS` comes from

The server embeds static assets via `internal/server/embed.go`:
- `//go:embed embed/public`
- `embeddedPublicFS = fs.Sub(embeddedFS, "embed/public")`

Historical note (root cause for this ticket): previously the repo also had `internal/server/embed_none.go` which compiled by default and set `embeddedPublicFS = nil`, requiring `-tags embed` to serve the SPA.

## Why `/` returns 404 today

When `embeddedPublicFS` is nil:
- `handleStaticFiles` exits without registering a `/` handler
- no other handler matches `/`
- `net/http` responds with a 404

This is exactly what happens on:
- `go run ./cmd/plz-confirm serve` (default invocation), and also on
- `go run ./cmd/plz-confirm serve --addr :3000` (same build behavior)

## Fix direction

To restore the expected “single-process serves the SPA” behavior for `go run ./cmd/plz-confirm serve`, we should ensure that `embeddedPublicFS` is available in the default build (i.e., remove the build-tag gating), while still keeping the “missing assets” failure mode clear if `internal/server/embed/public/index.html` is absent.

## Current behavior (after fix)

`go run ./cmd/plz-confirm serve` serves the embedded SPA on `/` (including `/assets/*`) as long as `internal/server/embed/public/index.html` exists in the tree at build time. `go generate ./internal/server` refreshes those embedded assets from the Vite build output.
