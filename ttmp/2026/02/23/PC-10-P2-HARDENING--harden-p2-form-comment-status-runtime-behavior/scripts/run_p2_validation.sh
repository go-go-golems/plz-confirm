#!/usr/bin/env bash
set -euo pipefail

PLZ_ROOT="$(git rev-parse --show-toplevel)"
GO_OS_ROOT="$(cd "${PLZ_ROOT}/../go-go-os" && pwd)"

echo "[PC-10] Running focused P2 regression tests"
cd "${GO_OS_ROOT}"
npx vitest run \
  packages/engine/src/__tests__/schema-form-renderer.test.ts \
  packages/engine/src/__tests__/form-view.test.ts \
  packages/confirm-runtime/src/proto/confirmProtoAdapter.test.ts \
  packages/confirm-runtime/src/components/ConfirmRequestWindowHost.test.ts

echo
echo "[PC-10] Running full engine test suite"
npm run test -w packages/engine

echo
echo "[PC-10] Running plz-confirm backend tests"
cd "${PLZ_ROOT}"
go test ./...

echo
echo "[PC-10] Validation complete"
