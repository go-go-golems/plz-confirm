# Changelog

## 2026-02-23

- Initial workspace created


- Replaced global websocket write lock with per-connection writer pump queue in `internal/server/ws.go`.
- Routed connect-time pending replay through per-client enqueue path.
- Added queue-closure and queue-full tests in `internal/server/ws_test.go`.
- Validation: `go test ./internal/server -count=1` passed.

## 2026-02-24

Ticket closed

