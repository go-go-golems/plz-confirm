# Changelog

## 2026-01-03

- Initial workspace created
- Added initial research diary and architecture notes for history/metadata/defaults
- Fixed web UI history duplication by making completion updates idempotent and by queueing multiple pending requests (code: 9f32913)
- Added dev targets (`make dev-backend`, `make dev-frontend`, `make dev-tmux`) and a ticket-local seed script for UI reproduction (code: 9f32913)
- Validated locally that history no longer duplicates entries (docs: c924478)
- Suppressed duplicate Redux completion actions by correlating completions by request id (code: eb41c20)
- Enforced `expiresAt` server-side (status transitions to `timeout`, WS broadcast, UI label) (code: b7fd7b5)
