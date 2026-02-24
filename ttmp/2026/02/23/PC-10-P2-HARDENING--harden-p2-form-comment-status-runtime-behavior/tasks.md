# Tasks

## P2 Hardening Scope

- [x] A1. Add boolean field support end-to-end (`FieldType`, schema inference, `FieldRow` control rendering)
- [x] A2. Fix required-field validation semantics for false/zero values in form submit path
- [x] A3. Ensure `SchemaFormRenderer` uncontrolled internal state resyncs when schema/value inputs change
- [x] A4. Prevent comment leakage between sequential requests/script steps by resetting comment state per request scope
- [x] A5. Align runtime status mapping with proto contract (`pending`, `completed`, `timeout`, `error`) and normalize legacy `expired`
- [x] A6. Preserve completion output payloads from websocket `request_completed` events in runtime event mapping

## Regression Tests

- [x] T1. Add engine test: boolean schema field maps to boolean control type
- [x] T2. Add engine test: required field validation allows valid `false`/`0` values
- [x] T3. Add runtime/host regression test for request-scoped action bar reset key behavior
- [x] T4. Add adapter regression test: `timeout` and `error` statuses map deterministically
- [x] T5. Add adapter regression test: `request_completed` websocket event preserves widget output payload

## Verification

- [x] V1. Run focused frontend tests for touched engine + confirm-runtime files
- [x] V2. Run broader engine package tests as integration guardrail
- [x] V3. Run plz-confirm backend tests to ensure no integration regressions
- [x] V4. Add and execute ticket validation script in `scripts/`

## Ticket Closure

- [x] D1. Update implementation diary with detailed steps, failures, and commands
- [x] D2. Update changelog with commit hashes and related files
- [x] D3. Mark all tasks complete and close ticket/doc statuses
