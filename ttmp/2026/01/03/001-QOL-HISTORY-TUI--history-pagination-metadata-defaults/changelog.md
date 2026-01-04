# Changelog

## 2026-01-03

- Initial workspace created
- Added initial research diary and architecture notes for history/metadata/defaults
- Fixed web UI history duplication by making completion updates idempotent and by queueing multiple pending requests (code: 9f32913)
- Added dev targets (`make dev-backend`, `make dev-frontend`, `make dev-tmux`) and a ticket-local seed script for UI reproduction (code: 9f32913)
- Validated locally that history no longer duplicates entries (docs: c924478)
- Suppressed duplicate Redux completion actions by correlating completions by request id (code: eb41c20)
- Enforced `expiresAt` server-side (status transitions to `timeout`, WS broadcast, UI label) (code: b7fd7b5)
- Added `--session-id` to CLI widget commands + made UI sessionId configurable (`?sessionId=`) to support WS session scoping (code: 970404a)
- Changed expiry semantics: auto-complete expired requests with default outputs (`status=completed`, `comment=AUTO_TIMEOUT`) and label them as TIMEOUT in UI history (code: 3afe968)
- Added permanent expiry disable on first interaction via `POST /api/requests/{id}/touch` + `expiry_disabled`/`touched_at` fields (code: fdf1d15)
