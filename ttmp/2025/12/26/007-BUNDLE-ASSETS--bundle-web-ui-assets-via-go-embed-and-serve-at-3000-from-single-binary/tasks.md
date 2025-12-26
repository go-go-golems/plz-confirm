# Tasks

## TODO

- [ ] Step 1: Analyze current asset bundling + serving code paths (go:generate, build tags, server handler)
- [ ] Step 2: Make `make build` / `make install` produce an embedded-asset binary (compile with `-tags embed`)
- [ ] Step 3: Ensure releases bundle the UI (update `.goreleaser.yaml` to build with embed tags; validate GitHub release workflow prerequisites)
- [ ] Step 4: Add a smoke test (curl-based) that validates `/`, `/assets/*`, `/api/*`, and `/ws` behavior on the embedded build
- [ ] Step 5: Update docs/dev ergonomics (README guidance for “no-vite” mode; optional disk fallback when embed tag isn’t used but `internal/server/embed/public` exists)

