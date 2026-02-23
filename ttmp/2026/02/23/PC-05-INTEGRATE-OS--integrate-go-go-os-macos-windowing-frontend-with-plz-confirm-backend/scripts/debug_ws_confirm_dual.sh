#!/usr/bin/env bash
set -euo pipefail

# Debug helper used during integration:
# - starts ws with --count 1
# - starts confirm command with short wait
# - dumps both logs and exit codes

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLZ_CONFIRM_DIR="$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel)"

BASE_URL="${BASE_URL:-http://127.0.0.1:8091/confirm}"
SESSION_ID="${SESSION_ID:-global}"
TITLE="${TITLE:-debug}"

TMP_DIR="$(mktemp -d /tmp/pc05-e2e-debug.XXXXXX)"
WS_LOG="$TMP_DIR/ws.log"
CLI_LOG="$TMP_DIR/cli.log"
WS_EXIT="$TMP_DIR/ws.exit"
CLI_EXIT="$TMP_DIR/cli.exit"

cd "$PLZ_CONFIRM_DIR"

(
  go run ./cmd/plz-confirm ws \
    --base-url "$BASE_URL" \
    --session-id "$SESSION_ID" \
    --count 1 >"$WS_LOG" 2>&1
  echo "ws_exit=$?" >"$WS_EXIT"
) &
WS_PID=$!

sleep 0.4

(
  go run ./cmd/plz-confirm confirm \
    --base-url "$BASE_URL" \
    --session-id "$SESSION_ID" \
    --title "$TITLE" \
    --wait-timeout 2 >"$CLI_LOG" 2>&1
  echo "cli_exit=$?" >"$CLI_EXIT"
) &
CLI_PID=$!

wait "$WS_PID" || true
wait "$CLI_PID" || true

echo "tmp_dir=$TMP_DIR"
echo "--- ws.log ---"
cat "$WS_LOG" || true
echo "--- ws.exit ---"
cat "$WS_EXIT" || true
echo "--- cli.log ---"
cat "$CLI_LOG" || true
echo "--- cli.exit ---"
cat "$CLI_EXIT" || true
