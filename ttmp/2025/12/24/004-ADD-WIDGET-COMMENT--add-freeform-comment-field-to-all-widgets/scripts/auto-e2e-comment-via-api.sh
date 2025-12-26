#!/usr/bin/env bash
set -euo pipefail

# auto-e2e-comment-via-api.sh
#
# Purpose:
# - Validate the new optional "comment" field end-to-end across ALL widgets.
# - Runs CLI verbs and auto-submits responses via:
#     POST /api/requests/{id}/response
# - Ensures CLI prints the "comment" column for each command.
#
# Prereqs:
# - Go backend running on :3001 and logging to a file we can parse (default below).
# - jq, curl, go
#
# Recommended dev setup (as used in ticket 001):
# - Go server logs: /tmp/plz-confirm-server.log
# - UI base URL (only used by CLI): http://localhost:3000
#
# You can override:
#   SERVER_LOG=/path/to/server.log
#   UI_BASE_URL=http://localhost:3000
#   API_BASE_URL=http://localhost:3001

UI_BASE_URL="${UI_BASE_URL:-http://localhost:3000}"
API_BASE_URL="${API_BASE_URL:-http://localhost:3001}"
SERVER_LOG="${SERVER_LOG:-/tmp/plz-confirm-server.log}"

require_bin() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "ERROR: missing required binary: $1" >&2
    exit 1
  }
}

require_bin jq
require_bin curl
require_bin go
require_bin base64

if [ ! -f "$SERVER_LOG" ]; then
  echo "ERROR: server log not found: $SERVER_LOG" >&2
  echo "Hint: start server with logging, e.g.:" >&2
  echo "  go run ./cmd/plz-confirm serve --addr :3001 2>&1 | tee /tmp/plz-confirm-server.log" >&2
  exit 1
fi

echo "== Config =="
echo "UI_BASE_URL=$UI_BASE_URL"
echo "API_BASE_URL=$API_BASE_URL"
echo "SERVER_LOG=$SERVER_LOG"
echo

# Tiny 1x1 PNG fixtures (kept in /tmp).
PNG_B64='iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO7+qK0AAAAASUVORK5CYII='
printf '%s' "$PNG_B64" | base64 -d > /tmp/plz-img-1.png
printf '%s' "$PNG_B64" | base64 -d > /tmp/plz-img-2.png

assert_contains() {
  local hay="$1"
  local needle="$2"
  if ! grep -Fq "$needle" "$hay"; then
    echo "ERROR: expected output to contain: $needle" >&2
    echo "--- output file: $hay ---" >&2
    cat "$hay" >&2
    exit 1
  fi
}

run_and_answer() {
  local name="$1"; shift
  local widget_type="$1"; shift
  local output_json="$1"; shift

  local out="/tmp/plz-${name}-out.json"
  local start
  start="$(wc -l < "$SERVER_LOG")"

  echo "== Running: $name (type=$widget_type) =="

  ( go run ./cmd/plz-confirm "$@" --base-url "$UI_BASE_URL" --wait-timeout 30 --output json > "$out" ) &
  local pid=$!

  local id=""
  for _ in $(seq 1 200); do
    id="$(
      tail -n +"$((start+1))" "$SERVER_LOG" \
        | grep 'Created request' \
        | grep "(${widget_type})" \
        | tail -n 1 \
        | sed -E 's/.*Created request ([^ ]+) .*/\1/' \
        || true
    )"
    if [ -n "$id" ]; then break; fi
    sleep 0.05
  done
  if [ -z "$id" ]; then
    echo "ERROR: could not find request id for type=${widget_type} in $SERVER_LOG" >&2
    kill "$pid" 2>/dev/null || true
    exit 1
  fi

  curl -sS -X POST "${API_BASE_URL}/api/requests/${id}/response" \
    -H 'Content-Type: application/json' \
    -d "{\"output\": ${output_json}}" > "/tmp/plz-${name}-submit.json"

  wait "$pid"

  echo "--- ${name} output ---"
  cat "$out"
  echo

  assert_contains "$out" "\"comment\": \"AUTO_OK\""
}

run_and_answer confirm confirm '{"approved":true,"timestamp":"2025-12-25T00:00:00Z","comment":"AUTO_OK"}' \
  confirm --title 'TEST confirm comment' --message 'auto-answered'

run_and_answer select select '{"selectedSingle":"staging","comment":"AUTO_OK"}' \
  select --title 'TEST select comment' --option production --option staging

cat > /tmp/plz-form-schema.json <<'JSON'
{"type":"object","properties":{"host":{"type":"string"}},"required":["host"]}
JSON
run_and_answer form form '{"data":{"host":"example"},"comment":"AUTO_OK"}' \
  form --title 'TEST form comment' --schema @/tmp/plz-form-schema.json

cat > /tmp/plz-table.json <<'JSON'
[{"id":"srv-1","name":"alpha"},{"id":"srv-2","name":"beta"}]
JSON
run_and_answer table table '{"selectedSingle":{"id":"srv-2","name":"beta"},"comment":"AUTO_OK"}' \
  table --title 'TEST table comment' --data @/tmp/plz-table.json --columns id,name

run_and_answer upload upload '{"files":[{"name":"a.txt","size":1,"path":"/tmp/a.txt","mimeType":"text/plain"}],"comment":"AUTO_OK"}' \
  upload --title 'TEST upload comment' --accept .txt --multiple

run_and_answer image_pick image '{"selectedNumber":0,"timestamp":"2025-12-25T00:00:00Z","comment":"AUTO_OK"}' \
  image --title 'TEST image comment' --message 'pick one' --image /tmp/plz-img-1.png --image /tmp/plz-img-2.png

echo "== Done =="
tail -n 30 "$SERVER_LOG"


