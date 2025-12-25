#!/usr/bin/env bash
set -euo pipefail

# auto-e2e-cli-via-api.sh
#
# Purpose:
# - Exercise the CLI verbs (confirm/select/form/table/upload/image)
# - WITHOUT manual browser interaction, by auto-submitting responses through:
#     POST /api/requests/{id}/response
#
# This is a pragmatic smoke test for plumbing:
# - CLI creates request -> server logs "Created request <id> (<type>)"
# - We parse the request id from the server logfile
# - We submit a response payload
# - CLI should unblock and print output
#
# Prereqs:
# - Go server running on :3001 (and writing a log file we can parse)
# - Vite UI running on :3000 (not strictly required for the API-driven submit, but keeps parity with dev setup)
# - jq, curl
#
# Recommended setup (what this ticket uses):
# - Go server logs: /tmp/plz-confirm-server.log
# - Vite logs:      /tmp/plz-confirm-vite.log
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

if [ ! -f "$SERVER_LOG" ]; then
  echo "ERROR: server log not found: $SERVER_LOG" >&2
  echo "Hint: start the server with logging, e.g.:" >&2
  echo "  go run ./cmd/plz-confirm serve --addr :3001 2>&1 | tee /tmp/plz-confirm-server.log" >&2
  exit 1
fi

echo "== Config =="
echo "UI_BASE_URL=$UI_BASE_URL"
echo "API_BASE_URL=$API_BASE_URL"
echo "SERVER_LOG=$SERVER_LOG"
echo

# Create tiny 1x1 PNG fixtures for image-widget tests (in /tmp; easy to clean).
PNG_B64='iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO7+qK0AAAAASUVORK5CYII='
printf '%s' "$PNG_B64" | base64 -d > /tmp/plz-img-1.png
printf '%s' "$PNG_B64" | base64 -d > /tmp/plz-img-2.png
printf '%s' "$PNG_B64" | base64 -d > /tmp/plz-img-a.png
printf '%s' "$PNG_B64" | base64 -d > /tmp/plz-img-b.png

echo "== Sanity: /api/images upload works =="
UP_JSON="$(curl -sS -F "file=@/tmp/plz-img-1.png" "${API_BASE_URL}/api/images")"
echo "$UP_JSON"
IMG_URL="$(echo "$UP_JSON" | jq -r '.url')"
curl -sS -I "${API_BASE_URL}${IMG_URL}" | head -n 5
echo

run_and_answer() {
  local name="$1"; shift
  local widget_type="$1"; shift
  local output_json="$1"; shift

  local out="/tmp/plz-${name}-out.json"
  local submit_out="/tmp/plz-${name}-submit.json"

  # Log cursor: only consider new lines after this.
  local start
  start="$(wc -l < "$SERVER_LOG")"

  echo "== Running: $name (type=$widget_type) =="
  echo "CLI: plz-confirm $* --base-url ${UI_BASE_URL} --wait-timeout 30 --output json"

  ( go run ./cmd/plz-confirm "$@" --base-url "$UI_BASE_URL" --wait-timeout 30 --output json > "$out" ) &
  local pid=$!

  # Find request id in server log.
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

  echo "Request ID: $id"

  curl -sS -X POST "${API_BASE_URL}/api/requests/${id}/response" \
    -H 'Content-Type: application/json' \
    -d "{\"output\": ${output_json}}" > "$submit_out"

  wait "$pid"

  echo "--- ${name} output ---"
  cat "$out"
  echo
}

# confirm
run_and_answer confirm confirm '{"approved":true,"timestamp":"2025-12-25T00:00:00Z"}' \
  confirm --title 'TEST confirm' --message 'Auto-answered via /response'

# select
run_and_answer select select '{"selected":"staging"}' \
  select --title 'TEST select' --option production --option staging --option development

# form (needs schema file)
cat > /tmp/plz-form-schema.json <<'JSON'
{"type":"object","properties":{"host":{"type":"string"}},"required":["host"]}
JSON
run_and_answer form form '{"data":{"host":"example"}}' \
  form --title 'TEST form' --schema @/tmp/plz-form-schema.json

# table (needs data file)
cat > /tmp/plz-table.json <<'JSON'
[{"id":"srv-1","name":"alpha"},{"id":"srv-2","name":"beta"}]
JSON
run_and_answer table table '{"selected":{"id":"srv-2","name":"beta"}}' \
  table --title 'TEST table' --data @/tmp/plz-table.json --columns id,name

# upload (note: current UI upload is simulated; we still validate CLI/server plumbing by directly submitting output)
run_and_answer upload upload '{"files":[{"name":"a.txt","size":1,"path":"/tmp/a.txt","mimeType":"text/plain"}]}' \
  upload --title 'TEST upload' --accept .txt --multiple

# image (Variant A: pick an image)
run_and_answer image_pick image '{"selected":0,"timestamp":"2025-12-25T00:00:00Z"}' \
  image --title 'TEST image pick' --message 'Pick one image' \
  --image /tmp/plz-img-1.png --image /tmp/plz-img-2.png

# image (Variant B: images as context + checkbox question)
run_and_answer image_opts image '{"selected":["Wrong color theme","Missing icon"],"timestamp":"2025-12-25T00:00:00Z"}' \
  image --title 'TEST image options' --message 'Which issues are present?' \
  --image /tmp/plz-img-a.png --image /tmp/plz-img-b.png \
  --multi \
  --option 'Text is too small' \
  --option 'Wrong color theme' \
  --option 'Missing icon'

# image (confirm mode)
run_and_answer image_confirm image '{"selected":true,"timestamp":"2025-12-25T00:00:00Z"}' \
  image --title 'TEST image confirm' --message 'Are these similar?' --mode confirm \
  --image /tmp/plz-img-a.png --image /tmp/plz-img-b.png

echo "== Done =="
echo "Recent server log lines:"
tail -n 40 "$SERVER_LOG"


