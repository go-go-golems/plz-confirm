# Tasks

## TODO

- [ ] Define protobuf schema for request metadata
  - [ ] Add `ProcessInfo` + `RequestMetadata` messages to `proto/plz_confirm/v1/request.proto`
  - [ ] Add `metadata` field to `UIRequest`
  - [ ] Run `make codegen` and confirm Go + TS types regenerate cleanly

- [ ] Implement Linux process-tree collector (linux build tag)
  - [ ] Add `internal/metadata/process_linux.go` (`//go:build linux`)
    - [ ] Read `/proc/<pid>/comm` for process name
    - [ ] Read `/proc/<pid>/cmdline` for argv (NUL-separated)
    - [ ] Read parent PID (from `/proc/<pid>/stat` or `/proc/<pid>/status`)
    - [ ] Walk parent chain until PID 1 / failure / cycle
  - [ ] Add `internal/metadata/process_other.go` (`//go:build !linux`) stub (no parent chain)
  - [ ] Add unit tests for parsing helpers (pure functions) where feasible

- [ ] Attach metadata on request creation (CLI)
  - [ ] Update `internal/client/client.go:CreateRequest` to set `reqProto.Metadata`:
    - [ ] `cwd` (best effort)
    - [ ] `self` (pid/name/argv)
    - [ ] `parents` (Linux-only)
  - [ ] Ensure metadata collection failures do not break request creation

- [ ] Preserve metadata through server storage and WS/REST emission
  - [ ] Update `internal/store/store.go:Create` cloning logic to keep `Metadata`
  - [ ] (Optional) Enrich metadata server-side:
    - [ ] `remote_addr` and `user_agent` fields merged into `reqProto.Metadata` in `internal/server/server.go:handleCreateRequest`

- [ ] Display metadata in the web UI (optional but recommended)
  - [ ] Show `cwd` and a compact process label in history rows (`agent-ui-system/client/src/pages/Home.tsx`)
  - [ ] Add a “details” affordance for full process chain (tooltip or expandable block)

- [ ] Add/extend smoke tests
  - [ ] Extend `scripts/curl-inspector-smoke.sh` to include a create request with `metadata` and assert it is preserved on GET
  - [ ] Add a Linux-only test that checks parent-chain fields are populated (best effort; allow variability)

- [ ] Update docs
  - [ ] Add a short “request metadata” section to `pkg/doc/adding-widgets.md` describing the `UIRequest.metadata` field and intended usage
