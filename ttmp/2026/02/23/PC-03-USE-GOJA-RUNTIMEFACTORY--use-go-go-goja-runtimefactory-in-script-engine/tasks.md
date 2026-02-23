# Tasks

## Analysis and Planning

- [x] A1. Produce baseline impact analysis of RuntimeFactory migration.
- [x] A2. Produce initial phased design plan.
- [x] A3. Produce updated hard-cut implementation plan with require and console capture requirements.

## Execution Plan (Detailed)

- [x] E1. Add `script_logs` to `UIRequest` in `proto/plz_confirm/v1/request.proto`.
- [x] E2. Regenerate protobuf code (`make codegen`) and verify generated Go/TS updates are limited to expected files.
- [x] E3. Implement script engine runtime hard-cut: use factory-only runtime creation path and remove direct per-call `goja.New()` usage in run methods.
- [x] E4. Implement per-run console capture (`log/info/warn/error`) with bounded collector and truncation marker.
- [x] E5. Populate `InitAndViewResult.Logs` and `UpdateAndViewResult.Logs` from collector.
- [x] E6. Extend store patch path to persist latest script run logs (`PatchScript` signature and callsites).
- [x] E7. Thread logs into create/update/complete responses via top-level `scriptLogs` and terminal `scriptOutput.logs`.
- [x] E8. Introduce typed script engine error categories and migrate server status mapping to `errors.Is`-first classification.
- [x] E9. Update and add tests for runtime behavior and HTTP contracts:
  - `require` availability in script runtime
  - console capture in create/update responses
  - terminal log behavior in `scriptOutput.logs`
  - timeout/cancel mapping under wrapped errors
  - log truncation behavior
- [x] E10. Update docs (`pkg/doc/js-script-development.md` and `pkg/doc/js-script-api.md`) to reflect new runtime and `scriptLogs` semantics.
- [x] E11. Run validation test set and record results in diary and changelog.

## Post-Implementation Stabilization (Build Pipeline)

- [x] S1. Reproduce `make build`/`ui-build` hang and capture the blocking interactive pnpm prompt in the Dagger frontend build step.
- [x] S2. Update `internal/server/generate_build.go` to remove prompt conditions and force non-interactive install semantics.
- [x] S3. Re-validate `make ui-build` and full `make build` on branch `task/use-runtime-factory` and record results.

## Commit Plan

- [x] C1. Commit schema + codegen changes (E1-E2).
- [x] C2. Commit runtime and server/store plumbing changes (E3-E8).
- [x] C3. Commit tests and docs updates (E9-E11).
- [x] C4. Commit ticket diary/changelog/task updates if separated from code commits.
- [x] C5. Commit build-pipeline stabilization fix for non-interactive Dagger pnpm install (S2).
- [x] C6. Commit ticket task/diary/changelog updates for stabilization work (S1-S3).

## Documentation and Delivery

- [x] D1. Keep detailed diary updated after each implementation step and commit.
- [x] D2. Record decision points and validation outputs in ticket changelog.
- [x] D3. Upload refreshed ticket bundle to reMarkable after implementation completes.
