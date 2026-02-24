#!/usr/bin/env bash
set -euo pipefail

PLZ_ROOT="$(git rev-parse --show-toplevel)"
GO_OS_ROOT="$(cd "${PLZ_ROOT}/../go-go-os" && pwd)"

echo "[PC-09] Running confirm-runtime P1 regression tests"
cd "${GO_OS_ROOT}"
npx vitest run \
  packages/confirm-runtime/src/proto/confirmProtoAdapter.test.ts \
  packages/confirm-runtime/src/components/ConfirmRequestWindowHost.test.ts

echo
echo "[PC-09] Running engine package tests (integration guardrail)"
npm run test -w packages/engine

echo
echo "[PC-09] Running plz-confirm backend tests"
cd "${PLZ_ROOT}"
go test ./...

echo
echo "[PC-09] Validation complete"
