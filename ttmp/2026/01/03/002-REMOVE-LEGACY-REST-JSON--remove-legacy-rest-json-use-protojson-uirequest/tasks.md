# Tasks

## TODO

- [x] Specify the new REST contract (protojson-only)
  - [x] Create request: `POST /api/requests` body is `protojson(UIRequest)` with `type` + input oneof set
  - [x] Submit response: `POST /api/requests/{id}/response` body is `protojson(UIRequest)` with output oneof set
  - [x] Define timeout semantics:
    - [x] Client may set `expiresAt` directly (RFC3339Nano); server defaults if omitted
    - [x] `sessionId` is still accepted but ignored (unchanged semantics)
  - [x] Update `pkg/doc/adding-widgets.md` and scripts with new payload shapes

- [x] Remove legacy REST JSON adapters (server-side)
  - [x] Delete legacy wrapper structs in `internal/server/server.go`
  - [x] Delete `internal/server/proto_convert.go` (legacy wrapper conversion layer)
  - [x] Update handlers in `internal/server/server.go` to decode protojson `UIRequest` directly
  - [x] Keep actionable 400 errors for missing/invalid oneofs

- [x] Update Go CLI client to speak protojson(UIRequest)
  - [x] Update `internal/client/client.go:CreateRequest` to send protojson `UIRequest` directly
  - [x] Keep `CreateRequestParams` and build `UIRequest` internally (sets input oneof + expiresAt)
  - [x] Validate widget commands still work (covered by e2e scripts)

- [x] Update web UI response submission to speak protojson(UIRequest)
  - [x] Update `agent-ui-system/client/src/services/websocket.ts:submitResponse` to send output oneof on `UIRequest`
  - [x] Ensure per-widget output shapes remain correct (oneof fields)

- [x] Update tests and smoke scripts
  - [x] Update `scripts/curl-inspector-smoke.sh` and repo e2e scripts to use protojson bodies
  - [x] Run `go test ./... -count=1`
  - [x] Run `pnpm -C agent-ui-system run check`
  - [x] Run `scripts/curl-inspector-smoke.sh`
  - [x] Run API-driven CLI smoke scripts (`auto-e2e-cli-via-api.sh`, `auto-e2e-comment-via-api.sh`)

- [x] Remove compatibility mentions
  - [x] Remove “compatibility with Node server shape” mentions in `internal/client/client.go`
  - [ ] Only if we remove `sessionId` from the protobuf contract: remove WS/session compatibility notes (not done; sessionId still present/ignored)
