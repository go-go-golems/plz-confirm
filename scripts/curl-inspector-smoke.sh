#!/usr/bin/env bash
set -euo pipefail

# curl-inspector-smoke.sh
#
# Purpose:
# - Emulate "inspector actions" WITHOUT the web UI:
#   - Create requests via POST /api/requests (like the CLI does)
#   - Submit responses via POST /api/requests/{id}/response (like the UI does)
#   - Validate via GET /api/requests/{id} and GET /api/requests/{id}/wait
#
# This validates:
# - JSON <-> protobuf conversion (server side)
# - protojson enum strings (type/status) remain legacy-compatible (confirm/pending/etc)
# - output oneof JSON shapes are accepted (selectedSingle/selectedMulti/etc)
#
# Prereqs:
# - server running on API_BASE_URL (default http://localhost:3001)
# - curl + jq
#
# Usage:
#   API_BASE_URL=http://localhost:3001 bash scripts/curl-inspector-smoke.sh

API_BASE_URL="${API_BASE_URL:-http://localhost:3001}"

require_bin() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "ERROR: missing required binary: $1" >&2
    exit 1
  }
}

require_bin curl
require_bin jq
require_bin base64

say() { printf "\n== %s ==\n" "$*"; }

post_json() {
  local url="$1"
  local data="$2"
  curl -sS -X POST "$url" -H 'Content-Type: application/json' -d "$data"
}

get_json() {
  local url="$1"
  curl -sS "$url"
}

assert_eq() {
  local got="$1"
  local want="$2"
  local msg="$3"
  if [ "$got" != "$want" ]; then
    echo "ASSERT FAILED: $msg" >&2
    echo "  got:  $got" >&2
    echo "  want: $want" >&2
    exit 1
  fi
}

create_request() {
  local type="$1"
  local input_json="$2"
  local timeout="${3:-60}"
  post_json "${API_BASE_URL}/api/requests" "{\"type\":\"${type}\",\"sessionId\":\"global\",\"input\":${input_json},\"timeout\":${timeout}}"
}

submit_response() {
  local id="$1"
  local output_json="$2"
  post_json "${API_BASE_URL}/api/requests/${id}/response" "{\"output\":${output_json}}"
}

wait_request() {
  local id="$1"
  local timeout="${2:-10}"
  get_json "${API_BASE_URL}/api/requests/${id}/wait?timeout=${timeout}"
}

say "Config"
echo "API_BASE_URL=${API_BASE_URL}"

say "Sanity: server reachable"
curl -sS -I "${API_BASE_URL}/" | head -n 5 || true

say "Image API: upload + fetch"
PNG_B64='iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO7+qK0AAAAASUVORK5CYII='
printf '%s' "$PNG_B64" | base64 -d > /tmp/plz-inspector-1x1.png
UP="$(curl -sS -F "file=@/tmp/plz-inspector-1x1.png" "${API_BASE_URL}/api/images")"
echo "$UP" | jq .
IMG_URL="$(echo "$UP" | jq -r '.url')"
curl -sS -I "${API_BASE_URL}${IMG_URL}" | head -n 5

say "confirm: create -> submit -> wait"
REQ="$(create_request "confirm" '{"title":"INSPECTOR confirm","message":"approve via API"}')"
ID="$(echo "$REQ" | jq -r '.id')"
assert_eq "$(echo "$REQ" | jq -r '.type')" "confirm" "confirm.type"
assert_eq "$(echo "$REQ" | jq -r '.status')" "pending" "confirm.status"
submit_response "$ID" "{\"approved\":true,\"timestamp\":\"$(date -Is)\",\"comment\":\"CURL_OK\"}" >/dev/null
DONE="$(wait_request "$ID" 10)"
assert_eq "$(echo "$DONE" | jq -r '.status')" "completed" "confirm.completed.status"
echo "$DONE" | jq -e '.confirmOutput.approved == true' >/dev/null

say "select: selectedSingle"
REQ="$(create_request "select" '{"title":"INSPECTOR select","options":["a","b","c"],"multi":false,"searchable":true}')"
ID="$(echo "$REQ" | jq -r '.id')"
submit_response "$ID" "{\"selectedSingle\":\"b\",\"comment\":\"CURL_OK\"}" >/dev/null
DONE="$(wait_request "$ID" 10)"
echo "$DONE" | jq -e '.selectOutput.selectedSingle == "b"' >/dev/null

say "select: selectedMulti"
REQ="$(create_request "select" '{"title":"INSPECTOR select multi","options":["a","b","c"],"multi":true,"searchable":true}')"
ID="$(echo "$REQ" | jq -r '.id')"
submit_response "$ID" "{\"selectedMulti\":{\"values\":[\"a\",\"c\"]},\"comment\":\"CURL_OK\"}" >/dev/null
DONE="$(wait_request "$ID" 10)"
echo "$DONE" | jq -e '.selectOutput.selectedMulti.values | length == 2' >/dev/null

say "form: struct schema + struct output"
REQ="$(create_request "form" '{"title":"INSPECTOR form","schema":{"type":"object","properties":{"host":{"type":"string"}},"required":["host"]}}')"
ID="$(echo "$REQ" | jq -r '.id')"
submit_response "$ID" '{"data":{"host":"example"},"comment":"CURL_OK"}' >/dev/null
DONE="$(wait_request "$ID" 10)"
echo "$DONE" | jq -e '.formOutput.data.host == "example"' >/dev/null

say "table: data[] + selectedSingle"
REQ="$(create_request "table" '{"title":"INSPECTOR table","data":[{"id":"srv-1","name":"alpha"},{"id":"srv-2","name":"beta"}],"columns":["id","name"],"multiSelect":false,"searchable":true}')"
ID="$(echo "$REQ" | jq -r '.id')"
submit_response "$ID" '{"selectedSingle":{"id":"srv-2","name":"beta"},"comment":"CURL_OK"}' >/dev/null
DONE="$(wait_request "$ID" 10)"
echo "$DONE" | jq -e '.tableOutput.selectedSingle.id == "srv-2"' >/dev/null

say "upload: files[]"
REQ="$(create_request "upload" '{"title":"INSPECTOR upload","accept":[".txt"],"multiple":true,"maxSize":1234}')"
ID="$(echo "$REQ" | jq -r '.id')"
submit_response "$ID" '{"files":[{"name":"a.txt","size":1,"path":"/tmp/a.txt","mimeType":"text/plain"}],"comment":"CURL_OK"}' >/dev/null
DONE="$(wait_request "$ID" 10)"
echo "$DONE" | jq -e '.uploadOutput.files[0].name == "a.txt"' >/dev/null

say "image: variant A (selectedNumber)"
REQ="$(create_request "image" "{\"title\":\"INSPECTOR image\",\"message\":\"pick\",\"images\":[{\"src\":\"${API_BASE_URL}${IMG_URL}\",\"label\":\"A\"},{\"src\":\"${API_BASE_URL}${IMG_URL}\",\"label\":\"B\"}],\"mode\":\"select\",\"options\":[],\"multi\":false}")"
ID="$(echo "$REQ" | jq -r '.id')"
submit_response "$ID" "{\"selectedNumber\":0,\"timestamp\":\"$(date -Is)\",\"comment\":\"CURL_OK\"}" >/dev/null
DONE="$(wait_request "$ID" 10)"
echo "$DONE" | jq -e '(.imageOutput.selectedNumber|tonumber) == 0' >/dev/null

say "image: confirm mode (selectedBool)"
REQ="$(create_request "image" "{\"title\":\"INSPECTOR image confirm\",\"message\":\"similar?\",\"images\":[{\"src\":\"${API_BASE_URL}${IMG_URL}\"}],\"mode\":\"confirm\",\"options\":[],\"multi\":false}")"
ID="$(echo "$REQ" | jq -r '.id')"
submit_response "$ID" "{\"selectedBool\":true,\"timestamp\":\"$(date -Is)\",\"comment\":\"CURL_OK\"}" >/dev/null
DONE="$(wait_request "$ID" 10)"
echo "$DONE" | jq -e '.imageOutput.selectedBool == true' >/dev/null

say "Done"
echo "OK: curl inspector smoke passed"


