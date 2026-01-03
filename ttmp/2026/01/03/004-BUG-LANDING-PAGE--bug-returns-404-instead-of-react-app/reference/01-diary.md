---
Title: Diary
Ticket: 004-BUG-LANDING-PAGE
Status: active
Topics:
    - web
    - backend
    - static
    - bug
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: internal/server/embed.go
      Note: Always embed SPA assets (fix for / 404)
    - Path: internal/server/server.go
      Note: Static serving + SPA fallback
    - Path: internal/server/server_static_test.go
      Note: Static serving test
    - Path: ttmp/2026/01/03/004-BUG-LANDING-PAGE--bug-returns-404-instead-of-react-app/analysis/01-static-assets-build-serving-for-plz-confirm.md
      Note: Analysis and decision record
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-03T14:54:28.792371577-05:00
WhatFor: ""
WhenToUse: ""
---


# Diary

## Goal

Track the investigation and fix for `004-BUG-LANDING-PAGE` (why `/` returns 404 when running `go run ./cmd/plz-confirm serve`, and how we restore serving the React SPA).

## Step 1: Reproduce 404 on `/` and identify why UI isn’t served

This step reproduced the reported behavior using the stock `serve` command and then traced the request routing to the static-file handler. The key finding is that `go run ./cmd/plz-confirm serve` compiles without the `embed` build tag, which makes `embeddedPublicFS` nil and causes the server to register no `/` handler at all.

The impact is that the backend works for `/api/*` and `/ws`, but the landing page is a plain 404 unless you build with `-tags embed` (and have generated/copied frontend assets into `internal/server/embed/public/`).

### What I did
- Ran `go run ./cmd/plz-confirm serve --addr :3005` in tmux and `curl http://localhost:3005/` to confirm the 404.
- Read `internal/server/server.go`, `internal/server/embed_none.go`, `internal/server/embed.go`, `internal/server/generate_build.go`, `README.md`, and `Makefile` to understand when static files are served.

### Why
- We need `/` to serve the React SPA when running the backend directly (or, at minimum, to make the failure mode explicit).

### What worked
- Repro is consistent: `/` responds `404 page not found` when running `go run ./cmd/plz-confirm serve` without build tags.
- Root cause is clear: `embeddedPublicFS` is nil under `!embed`, so `handleStaticFiles` returns early and nothing handles `/`.

### What didn't work
- Expecting the embedded SPA to serve on `/` when compiling via plain `go run` (no build tag).

### What I learned
- Current “prod vs dev” switch is implemented via build tags (`embed` vs `!embed`), not runtime detection.
- The server’s SPA fallback routing exists, but it is entirely gated behind `embeddedPublicFS != nil`.

### What was tricky to build
- The behavior is implicit: the server silently skips static serving when assets aren’t embedded, resulting in a generic 404 with no guidance.

### What warrants a second pair of eyes
- Confirm we actually want `go run ./cmd/plz-confirm serve` to serve the embedded SPA by default (vs forcing a two-process dev setup with Vite on :3000 and backend on :3001).

### What should be done in the future
- Decide and document the intended dev/prod contract for `serve` (single-process serving embedded assets vs Vite dev server + proxy), and ensure README examples match the contract.

### Code review instructions
- Start in `internal/server/server.go` (`handleStaticFiles`) and follow where `embeddedPublicFS` comes from (`internal/server/embed*.go`).
- Validate with `go run ./cmd/plz-confirm serve --addr :3005` then `curl -i http://localhost:3005/`.

## Step 2: Serve the embedded SPA on `/` for default `go run` builds

This step removes the build-tag gating that disabled static serving for the default build, so `go run ./cmd/plz-confirm serve` once again serves the React SPA on `/` (and `/assets/*`) without requiring `-tags embed`. It also adds a small regression test so we don’t reintroduce the 404-on-root behavior accidentally.

The impact is a smoother “single-process” workflow: running the backend directly now serves a usable UI. The two-process dev topology (Vite on :3000 + backend on :3001) remains valid, but it’s no longer required just to see a landing page.

**Commit (code):** f83d67c41a04a5b4fb263379f6ba42900aea4af4 — "🐛 server: serve embedded SPA on / for go run"

### What I did
- Removed the `embed`/`!embed` split by deleting `internal/server/embed_none.go` and making `internal/server/embed.go` compile unconditionally.
- Verified `/` and a referenced `/assets/*` URL return 200 when running `go run ./cmd/plz-confirm serve`.
- Added `internal/server/server_static_test.go` to assert `GET /` returns HTML that looks like the SPA `index.html`.
- Updated `Makefile`, `README.md`, and `pkg/doc/adding-widgets.md` to reflect that embedding is no longer controlled via `-tags embed`.
- Ran `gofmt` and `go test ./... -count=1`.

### Why
- The default `go run` path is the quickest way to run the server locally; returning a 404 on `/` is a footgun and makes the project feel broken even though the UI assets are present in-repo.

### What worked
- `GET /` now returns `200 OK` and includes `<div id="root"></div>`.
- `/assets/*` URLs referenced by `index.html` are served correctly.
- `go test ./... -count=1` passes (including the new static-serving test).

### What didn't work
- N/A

### What I learned
- Keeping `internal/server/embed/public/` present in the repo avoids the “fresh clone can’t compile because embed assets are missing” failure mode without needing build tags.

### What was tricky to build
- Making the change in a way that keeps both dev modes viable: embedded SPA serving must not shadow `/api/*` and `/ws`, and the SPA fallback behavior must remain intact.

### What warrants a second pair of eyes
- Confirm we’re comfortable with always embedding `internal/server/embed/public/` in all builds (binary size + always-on SPA serving), and that this aligns with the desired developer experience.

### What should be done in the future
- If we ever decide to stop committing `internal/server/embed/public/`, we’ll need a different “missing assets” contract (and corresponding docs/tests) to avoid returning to silent 404s on `/`.

### Code review instructions
- Start with `internal/server/embed.go`, then `internal/server/server.go:handleStaticFiles`, then `internal/server/server_static_test.go`.
- Validate locally:
  - `go run ./cmd/plz-confirm serve --addr :3005`
  - `curl -i http://localhost:3005/`
  - `go test ./... -count=1`
