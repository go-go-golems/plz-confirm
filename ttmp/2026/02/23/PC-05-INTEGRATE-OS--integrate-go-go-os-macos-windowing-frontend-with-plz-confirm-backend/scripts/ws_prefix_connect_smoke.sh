#!/usr/bin/env bash
set -euo pipefail

# Verifies ws command preserves /confirm prefix by consuming one event.
# Expected connection URL: ws://<host>/confirm/ws?sessionId=<id>

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLZ_CONFIRM_DIR="$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel)"

BASE_URL="${BASE_URL:-http://127.0.0.1:8091/confirm}"
SESSION_ID="${SESSION_ID:-global}"

cd "$PLZ_CONFIRM_DIR"
TMP_DIR="$(mktemp -d /tmp/pc05-ws-smoke.XXXXXX)"
WS_LOG="$TMP_DIR/ws.log"

go run ./cmd/plz-confirm ws \
  --base-url "$BASE_URL" \
  --session-id "$SESSION_ID" \
  --count 1 >"$WS_LOG" 2>&1 &
WS_PID=$!

cleanup() {
  kill "${WS_PID:-}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

sleep 0.5
TITLE="ws-prefix-smoke-$(date +%s%N)"
curl -sS -X POST "$BASE_URL/api/requests" \
  -H 'content-type: application/json' \
  -d "{\"type\":\"confirm\",\"sessionId\":\"$SESSION_ID\",\"confirmInput\":{\"title\":\"$TITLE\"}}" >/dev/null

wait "$WS_PID"
OUTPUT="$(cat "$WS_LOG")"
printf '%s\n' "$OUTPUT"

if grep -q "connected: ws://127.0.0.1:8091/confirm/ws?sessionId=$SESSION_ID" <<<"$OUTPUT"; then
  echo "ok: ws prefix preserved"
  exit 0
fi

echo "ws prefix/connect smoke failed"
exit 1
