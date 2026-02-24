# Tasks

## P1 Stabilization Scope

- [x] A1. Make select output encoding mode-aware (`multi` flag drives `selectedMulti` oneof)
- [x] A2. Make table output encoding mode-aware (`multiSelect` flag drives `selectedMulti` oneof)
- [x] A3. Make image output encoding mode-aware for multi select (`multi` drives `selectedStrings`)
- [x] A4. Normalize upload `maxSize` from protojson numeric-string to number in request adapter mapping
- [x] A5. Fix table row-key fallback in request host to avoid `id`-only collision behavior

## Regression Tests

- [x] T1. Add adapter regression test: select multi=true with single selected value emits `selectedMulti`
- [x] T2. Add adapter regression test: table multiSelect=true with single selected row emits `selectedMulti`
- [x] T3. Add adapter regression test: image multi=true with single selected id emits `selectedStrings`
- [x] T4. Add adapter regression test: upload maxSize string maps to numeric payload
- [x] T5. Add host/runtime regression test for table selection correctness when rows lack `id`

## Verification

- [x] V1. Run focused frontend tests for touched packages/files
- [x] V2. Run relevant backend tests to ensure no integration regressions
- [x] V3. Perform manual smoke check for select/table/image/upload request flows

## Ticket Closure

- [x] D1. Update implementation diary with step-by-step changes, commands, and outcomes
- [x] D2. Update changelog with commits and related files
- [x] D3. Mark ticket tasks complete and set ticket/docs status to closed
