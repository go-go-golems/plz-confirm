# Tasks

## Planned

- [x] WP-01: Capture current websocket broadcaster invariants and failure modes.
- [x] WP-02: Implement per-connection websocket writer pump (`wsClient`) with bounded queue.
- [x] WP-03: Route connect-time pending replay through the same queue/pump path.
- [x] WP-04: Add regression tests for queue behavior and existing websocket lifecycle ordering.
- [x] WP-05: Run `go test ./internal/server -count=1` and fix regressions.
- [ ] WP-06: Update ticket docs/changelog/diary and commit.
