#!/usr/bin/env bash
set -euo pipefail

# End-to-end check:
# 1) Start ws listener on /confirm/ws
# 2) Start plz-confirm CLI confirm command
# 3) Capture request id from ws event
# 4) Submit confirmOutput via REST
# 5) Verify CLI unblocks with tabular output

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLZ_CONFIRM_DIR="$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel)"

BASE_URL="${BASE_URL:-http://127.0.0.1:8091/confirm}"
SESSION_ID="${SESSION_ID:-global}"
WAIT_TIMEOUT="${WAIT_TIMEOUT:-20}"

TMP_DIR="$(mktemp -d /tmp/pc05-e2e.XXXXXX)"
TITLE="E2E-$(date +%s%N)"
WS_LOG="$TMP_DIR/ws.log"
CLI_LOG="$TMP_DIR/cli.log"
RESP_JSON="$TMP_DIR/response.json"

cleanup() {
  kill "${WS_PID:-}" >/dev/null 2>&1 || true
  kill "${CLI_PID:-}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

cd "$PLZ_CONFIRM_DIR"

go run ./cmd/plz-confirm ws \
  --base-url "$BASE_URL" \
  --session-id "$SESSION_ID" >"$WS_LOG" 2>&1 &
WS_PID=$!

sleep 0.6

go run ./cmd/plz-confirm confirm \
  --base-url "$BASE_URL" \
  --session-id "$SESSION_ID" \
  --title "$TITLE" \
  --message "compat test" \
  --wait-timeout "$WAIT_TIMEOUT" >"$CLI_LOG" 2>&1 &
CLI_PID=$!

REQ_ID=""
for _ in $(seq 1 40); do
  LINE="$(grep "$TITLE" "$WS_LOG" | head -n1 || true)"
  if [[ -n "$LINE" ]]; then
    REQ_ID="$(printf '%s' "$LINE" | sed -n 's/.*"id":"\([^"]\+\)".*/\1/p')"
    if [[ -n "$REQ_ID" ]]; then
      break
    fi
  fi
  sleep 0.5
done

if [[ -z "$REQ_ID" ]]; then
  echo "FAILED: request id not found in ws log"
  cat "$WS_LOG" || true
  exit 1
fi

TS="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
RESP_CODE="$(curl -sS -o "$RESP_JSON" -w '%{http_code}' \
  -X POST "$BASE_URL/api/requests/$REQ_ID/response" \
  -H 'content-type: application/json' \
  -d "{\"confirmOutput\":{\"approved\":true,\"timestamp\":\"$TS\",\"comment\":\"approved-from-e2e\"}}")"

wait "$CLI_PID"

if [[ "$RESP_CODE" != "200" ]]; then
  echo "FAILED: response endpoint returned HTTP $RESP_CODE"
  cat "$RESP_JSON" || true
  exit 1
fi

if ! grep -q "$REQ_ID" "$CLI_LOG"; then
  echo "FAILED: CLI output did not contain request id $REQ_ID"
  cat "$CLI_LOG" || true
  exit 1
fi

echo "ok: e2e completed"
echo "request_id=$REQ_ID"
echo "tmp_dir=$TMP_DIR"
cat "$CLI_LOG"
