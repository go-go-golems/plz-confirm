---
Title: Current bundling pipeline and recommended approach
Ticket: 007-BUNDLE-ASSETS
Status: active
Topics:
    - backend
    - build
    - release
    - embed
    - assets
    - ui
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-26T13:27:01.102925492-05:00
WhatFor: ""
WhenToUse: ""
---


# Current bundling pipeline and recommended approach

## What exists today (important discovery)

The codebase already has the **full serving path** for bundling the Vite UI into a single Go binary, but it is **gated by a build tag**:

- `internal/server/embed.go` is only compiled with `-tags embed`, and it embeds `internal/server/embed/public/**` via `//go:embed`.
- `internal/server/embed_none.go` (default builds) sets `embeddedPublicFS = nil`, which disables static serving entirely.
- `internal/server/server.go` already registers a root handler (`/`) that:
  - avoids `/api*` and `/ws`,
  - serves files from `embeddedPublicFS`,
  - falls back to `index.html` for SPA routes.

Separately, there is a generator that *produces* the files to embed:

- `internal/server/generate.go`: `//go:generate go run generate_build.go`
- `internal/server/generate_build.go`: runs `pnpm run build` in `agent-ui-system/` and copies `agent-ui-system/dist/public` → `internal/server/embed/public`.

## Why it doesn’t fully satisfy “single binary” yet

The current behavior depends on building with **`-tags embed`**.

If the build pipeline produces a binary without that tag, the server will still listen on `:3000` but **will not serve the UI** because `embeddedPublicFS == nil`.

So the core ticket work is to make “production-ish builds” consistently:

1. run `go generate ./...` (to populate `internal/server/embed/public`)
2. compile the server/binary with `-tags embed` (so the assets are embedded)

## Recommended approach

### 1) Build contracts

- **Makefile**:
  - `make build` should generate assets and build with `-tags embed`.
  - `make install` should also build with `-tags embed`.
- **goreleaser**:
  - set `tags: [embed]` (or equivalent flags) so release artifacts always include the UI.

### 2) Dev ergonomics (optional but valuable)

Add a disk fallback path for local dev:

- if `embeddedPublicFS == nil` but `internal/server/embed/public/index.html` exists on disk (after a previous `go generate`), serve from disk.
- This makes “no vite” dev possible even without `-tags embed`, while still keeping the “single binary” goal for release builds.

## How to test

### Manual

- Build embedded binary and run server:
  - `make build`
  - `./dist/plz-confirm serve --addr :3000` (or whatever build outputs)
- Validate:
  - `curl -sSf http://localhost:3000/ | head` returns HTML
  - `curl -sSfI http://localhost:3000/assets/*` returns `200`
  - `curl -sSf http://localhost:3000/api/requests` returns method error (proves API route still wins)

### Automated smoke script

Add a small script (curl-based) that runs against a running embedded server and asserts the behaviors above.

