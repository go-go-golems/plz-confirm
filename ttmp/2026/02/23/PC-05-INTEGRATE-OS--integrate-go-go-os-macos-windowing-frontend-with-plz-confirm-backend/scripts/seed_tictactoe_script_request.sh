#!/usr/bin/env bash
set -euo pipefail

# Seed a backend JS script request (tic-tac-toe) against the integrated
# go-go-os confirm mount.
#
# Usage:
#   BASE_URL="http://127.0.0.1:8091/confirm" \
#   SESSION_ID="global" \
#   bash ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/scripts/seed_tictactoe_script_request.sh

BASE_URL="${BASE_URL:-http://127.0.0.1:8091/confirm}"
SESSION_ID="${SESSION_ID:-global}"
TITLE="${TITLE:-tic-tac-toe}"
SCRIPT_FILE="${SCRIPT_FILE:-/home/manuel/workspaces/2026-02-23/plz-confirm-hypercard/plz-confirm/ttmp/2026/02/22/PC-01-ADD-JS-API--add-js-api-describe-extension/scripts/tictactoe.js}"

require_bin() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "ERROR: missing required binary: $1" >&2
    exit 1
  }
}

require_bin curl
require_bin jq

if [[ ! -f "$SCRIPT_FILE" ]]; then
  echo "ERROR: script file not found: $SCRIPT_FILE" >&2
  exit 1
fi

probe_code="$(
  curl -sS -o /dev/null -w '%{http_code}' \
    -X POST "${BASE_URL}/api/requests" \
    -H 'content-type: application/json' \
    -d '{}'
)"
if [[ "$probe_code" == "000" ]]; then
  echo "ERROR: backend not reachable at ${BASE_URL}" >&2
  exit 1
fi

payload="$(
  jq -nc \
    --arg session "$SESSION_ID" \
    --arg title "$TITLE" \
    --arg script "$(cat "$SCRIPT_FILE")" \
    '{
      type: "script",
      sessionId: $session,
      scriptInput: {
        title: $title,
        script: $script
      }
    }'
)"

response="$(
  curl -sS -X POST "${BASE_URL}/api/requests" \
    -H 'content-type: application/json' \
    -d "$payload"
)"

request_id="$(echo "$response" | jq -r '.id // empty')"
if [[ -z "$request_id" ]]; then
  echo "ERROR: failed to create tic-tac-toe script request" >&2
  echo "$response" | jq . >&2 || echo "$response" >&2
  exit 1
fi

echo "$response" | jq .
echo
echo "Created script request: ${request_id}"
echo "Next:"
echo "  1) Open Confirm Queue in go-go-os inventory UI"
echo "  2) Open request ${request_id}"
echo "  3) Play through tic-tac-toe"
echo
echo "Optional WS watcher:"
echo "  go run ./cmd/plz-confirm ws --base-url \"${BASE_URL}\" --session-id \"${SESSION_ID}\" --pretty"
