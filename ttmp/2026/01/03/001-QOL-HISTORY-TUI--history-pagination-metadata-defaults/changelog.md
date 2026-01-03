# Changelog

## 2026-01-03

- Initial workspace created
- Added initial research diary and architecture notes for history/metadata/defaults
- Fixed web UI history duplication by making completion updates idempotent and by queueing multiple pending requests (code: 9f32913)
- Added dev targets (`make dev-backend`, `make dev-frontend`, `make dev-tmux`) and a ticket-local seed script for UI reproduction (code: 9f32913)
