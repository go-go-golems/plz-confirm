# Tasks

## TODO

- [x] Define protobuf schema for request metadata
- [x] Add `ProcessInfo` + `RequestMetadata` messages to `proto/plz_confirm/v1/request.proto`
- [x] Add `metadata` field to `UIRequest`
- [x] Run `make codegen` and confirm Go + TS types regenerate cleanly

- [x] Implement Linux process-tree collector (linux build tag)
- [x] Add `internal/metadata/process_linux.go` (`//go:build linux`)
- [x] Read `/proc/<pid>/comm` for process name
- [x] Read `/proc/<pid>/cmdline` for argv (NUL-separated)
- [x] Read parent PID (from `/proc/<pid>/stat` or `/proc/<pid>/status`)
- [x] Walk parent chain until PID 1 / failure / cycle
- [x] Add `internal/metadata/process_other.go` (`//go:build !linux`) stub (no parent chain)
- [x] Add unit tests for parsing helpers (pure functions) where feasible

- [x] Attach metadata on request creation (CLI)
- [x] Update `internal/client/client.go:CreateRequest` to set `reqProto.Metadata`:
- [x] `cwd` (best effort)
- [x] `self` (pid/name/argv)
- [x] `parents` (Linux-only)
- [x] Ensure metadata collection failures do not break request creation

- [x] Preserve metadata through server storage and WS/REST emission
- [x] Update `internal/store/store.go:Create` cloning logic to keep `Metadata`
- [x] (Optional) Enrich metadata server-side:
- [x] `remote_addr` and `user_agent` fields merged into `reqProto.Metadata` in `internal/server/server.go:handleCreateRequest`

- [x] Display metadata in the web UI (optional but recommended)
- [x] Show `cwd` and a compact process label in history rows (`agent-ui-system/client/src/pages/Home.tsx`)
- [x] Add a “details” affordance for full process chain (tooltip or expandable block)

- [x] Add/extend smoke tests
- [x] Extend `scripts/curl-inspector-smoke.sh` to include a create request with `metadata` and assert it is preserved on GET
- [x] Add a Linux-only test that checks parent-chain fields are populated (best effort; allow variability)

- [x] Update docs
- [x] Add a short “request metadata” section to `pkg/doc/adding-widgets.md` describing the `UIRequest.metadata` field and intended usage
