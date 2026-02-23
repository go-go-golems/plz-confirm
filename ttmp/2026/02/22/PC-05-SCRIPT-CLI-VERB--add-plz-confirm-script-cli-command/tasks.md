# Tasks

## TODO

- [ ] Define CLI UX and flag contract for `plz-confirm script` including required/optional inputs, precedence rules, and output shape.
- [ ] Implement `internal/cli/script.go` command description, settings struct, and glazed flag definitions consistent with existing widget commands.
- [ ] Add script source loading logic with robust validation for `--script`, `--script-file`, and `--script-file -` (stdin) modes.
- [ ] Add props parsing and validation for `--props-file` and `--props-json`, with clear error messages on invalid JSON/object shape.
- [ ] Construct and submit `v1.ScriptInput` via `internal/client.CreateRequest` with support for `timeoutMs` mapping and standard request metadata/session handling.
- [ ] Wait for completion using `client.WaitRequest`, handle timeout/cancel/non-completed statuses correctly, and surface actionable errors.
- [ ] Emit structured output rows that include request id and script result payload in ways compatible with `table/json/yaml/csv` glazed outputs.
- [ ] Register the new command in `cmd/plz-confirm/main.go` and ensure help text and command discovery include the new verb.
- [ ] Add unit/integration tests for happy path, script input validation failures, props parse failures, wait timeout behavior, and server error propagation.
- [ ] Update user docs (`README.md`, `pkg/doc/how-to-use.md`, `pkg/doc/js-script-api.md`) with runnable command examples and common troubleshooting cases.
- [ ] Add a ticket-local smoke script under `scripts/` and document a manual validation checklist for local and CI execution.
