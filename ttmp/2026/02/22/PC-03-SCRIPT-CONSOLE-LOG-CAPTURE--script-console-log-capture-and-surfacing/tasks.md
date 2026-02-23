# Tasks

## TODO

- [ ] Define runtime logging contract for scripts: supported console methods (`log`, `warn`, `error`), entry format, truncation marker semantics, and default limits (entry count, bytes per entry, total bytes).
- [ ] Implement a bounded log collector in `internal/scriptengine/engine.go` that is isolated per VM run and cannot panic on malformed JS values.
- [ ] Inject a `console` object into the VM before script execution and wire each method to the bounded collector.
- [ ] Implement safe argument serialization for console calls (primitive handling, object stringification fallback, `undefined`/`null`, cyclic object fallback, join strategy).
- [ ] Include phase metadata in each log line (`describe`, `init`, `view`, `update`) so troubleshooting can correlate log lines to lifecycle steps.
- [ ] Populate `InitAndViewResult.Logs` and `UpdateAndViewResult.Logs` from the collector and verify server completion path preserves `ScriptOutput.logs`.
- [ ] Decide and implement lifecycle behavior for non-terminal step logs (drop, return per-step only, or bounded accumulation across steps); document the chosen policy.
- [ ] Add scriptengine unit tests for console availability, multi-argument logging, truncation limits, cyclic values, and non-crashing behavior.
- [ ] Add server integration tests that assert logs appear in completed `scriptOutput.logs` and that excessive logging remains bounded.
- [ ] Update `pkg/doc/js-script-api.md` and `pkg/doc/js-script-development.md` with console logging behavior, limits, and examples.
- [ ] Add a ticket-local smoke script under `scripts/` that emits representative logs and verify output manually against a running server.
