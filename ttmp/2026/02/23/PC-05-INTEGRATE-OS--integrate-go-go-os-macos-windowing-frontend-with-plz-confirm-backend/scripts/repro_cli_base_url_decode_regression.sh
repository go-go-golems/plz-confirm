#!/usr/bin/env bash
set -euo pipefail

# Reproduces the CLI base-url decode path for confirm command.
# Expected after fix:
# - request is created against --base-url
# - command waits and times out (since we do not auto-submit a response here)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLZ_CONFIRM_DIR="$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel)"

BASE_URL="${BASE_URL:-http://127.0.0.1:8091/confirm}"
SESSION_ID="${SESSION_ID:-global}"

cd "$PLZ_CONFIRM_DIR"
set +e
OUTPUT="$(go run ./cmd/plz-confirm confirm \
  --base-url "$BASE_URL" \
  --session-id "$SESSION_ID" \
  --title "decode-regression-check" \
  --wait-timeout 1 2>&1)"
EXIT_CODE=$?
set -e

printf '%s\n' "$OUTPUT"
if [[ $EXIT_CODE -eq 0 ]]; then
  echo "unexpected success: command should time out without a response"
  exit 1
fi

if grep -q 'unsupported outbound URL scheme ""' <<<"$OUTPUT"; then
  echo "regression present: base-url did not decode"
  exit 1
fi

if grep -Eq 'timeout waiting for response|context deadline exceeded' <<<"$OUTPUT"; then
  echo "ok: base-url decoded and request/wait path executed"
  exit 0
fi

echo "unexpected failure mode"
exit 1
