#!/usr/bin/env bash
set -euo pipefail

# Seed the UI with multiple pending requests created via the CLI.
#
# Intended use: reproduce/verify history + queue behavior in the web UI.
#
# Prereqs:
# - backend running (default: http://localhost:3001)
# - web UI connected (vite on :3000 or backend UI)
#
# Usage:
#   API_BASE_URL=http://localhost:3001 bash ttmp/2026/01/03/001-QOL-HISTORY-TUI--history-pagination-metadata-defaults/scripts/seed-requests-with-metadata.sh
#
# Notes:
# - We intentionally set a small wait timeout so the command returns quickly
#   while the request stays pending server-side.

API_BASE_URL="${API_BASE_URL:-http://localhost:3001}"
WAIT_TIMEOUT="${WAIT_TIMEOUT:-1}"
REQUEST_TIMEOUT="${REQUEST_TIMEOUT:-600}"

RUN=(go run ./cmd/plz-confirm)

ts() { date -u +"%Y-%m-%dT%H:%M:%SZ"; }

say() { printf "\n== %s ==\n" "$*"; }

safe_run() {
  set +e
  "$@"
  local rc=$?
  set -e
  if [ $rc -ne 0 ]; then
    echo "(ok) command exited non-zero (expected for wait timeout): rc=$rc" >&2
  fi
}

say "Config"
echo "API_BASE_URL=$API_BASE_URL"
echo "WAIT_TIMEOUT=$WAIT_TIMEOUT"
echo "REQUEST_TIMEOUT=$REQUEST_TIMEOUT"

say "confirm x3"
safe_run "${RUN[@]}" confirm \
  --base-url "$API_BASE_URL" \
  --wait-timeout "$WAIT_TIMEOUT" \
  --timeout "$REQUEST_TIMEOUT" \
  --title "HISTORY_SEED confirm A $(ts)" \
  --message "Approve/reject to generate a completed history entry."

safe_run "${RUN[@]}" confirm \
  --base-url "$API_BASE_URL" \
  --wait-timeout "$WAIT_TIMEOUT" \
  --timeout "$REQUEST_TIMEOUT" \
  --title "HISTORY_SEED confirm B $(ts)" \
  --message "Second confirm request (check queue order)."

safe_run "${RUN[@]}" confirm \
  --base-url "$API_BASE_URL" \
  --wait-timeout "$WAIT_TIMEOUT" \
  --timeout "$REQUEST_TIMEOUT" \
  --title "HISTORY_SEED confirm C $(ts)" \
  --message "Third confirm request (check queue order)."

say "select x2"
safe_run "${RUN[@]}" select \
  --base-url "$API_BASE_URL" \
  --wait-timeout "$WAIT_TIMEOUT" \
  --timeout "$REQUEST_TIMEOUT" \
  --title "HISTORY_SEED select region $(ts)" \
  --option us-east-1 \
  --option us-west-2 \
  --option eu-central-1 \
  --option ap-northeast-1 \
  --searchable

safe_run "${RUN[@]}" select \
  --base-url "$API_BASE_URL" \
  --wait-timeout "$WAIT_TIMEOUT" \
  --timeout "$REQUEST_TIMEOUT" \
  --title "HISTORY_SEED select multi $(ts)" \
  --option alpha \
  --option beta \
  --option gamma \
  --multi \
  --searchable

say "table x1"
TABLE_DATA_JSON='[
  {"id":"srv-1","name":"alpha","env":"dev"},
  {"id":"srv-2","name":"beta","env":"staging"},
  {"id":"srv-3","name":"gamma","env":"prod"}
]'
safe_run bash -lc "printf '%s' '$TABLE_DATA_JSON' | ${RUN[*]} table --base-url '$API_BASE_URL' --wait-timeout '$WAIT_TIMEOUT' --timeout '$REQUEST_TIMEOUT' --title 'HISTORY_SEED table servers $(ts)' --data - --columns id --columns name --columns env"

say "form x1"
FORM_SCHEMA_JSON='{
  "type": "object",
  "properties": {
    "service": { "type": "string", "title": "Service" },
    "replicas": { "type": "integer", "minimum": 1, "maximum": 10, "title": "Replicas" },
    "dryRun": { "type": "boolean", "title": "Dry Run" }
  },
  "required": ["service", "replicas"]
}'
safe_run bash -lc "printf '%s' '$FORM_SCHEMA_JSON' | ${RUN[*]} form --base-url '$API_BASE_URL' --wait-timeout '$WAIT_TIMEOUT' --timeout '$REQUEST_TIMEOUT' --title 'HISTORY_SEED form deploy $(ts)' --schema -"

say "upload x1"
safe_run "${RUN[@]}" upload \
  --base-url "$API_BASE_URL" \
  --wait-timeout "$WAIT_TIMEOUT" \
  --timeout "$REQUEST_TIMEOUT" \
  --title "HISTORY_SEED upload logs $(ts)" \
  --accept .log \
  --accept .txt \
  --multiple \
  --max-size $((5 * 1024 * 1024))

say "Done"
echo "Open the UI and complete a few requests; history should not duplicate entries."

