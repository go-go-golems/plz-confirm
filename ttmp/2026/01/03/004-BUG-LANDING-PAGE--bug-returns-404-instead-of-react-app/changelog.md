# Changelog

## 2026-01-03

- Initial workspace created


## 2026-01-03

Fix static serving: go run ./cmd/plz-confirm serve now serves the embedded React SPA on / (no build tags); add regression test; update docs (commit f83d67c41a04a5b4fb263379f6ba42900aea4af4)

### Related Files

- /home/manuel/workspaces/2026-01-03/plz-confirm-improvements/plz-confirm/Makefile — Stop using -tags embed for build/install
- /home/manuel/workspaces/2026-01-03/plz-confirm-improvements/plz-confirm/internal/server/embed.go — Removed build-tag gating so embeddedPublicFS is always available
- /home/manuel/workspaces/2026-01-03/plz-confirm-improvements/plz-confirm/internal/server/server_static_test.go — Guard against regressions to 404 on /
