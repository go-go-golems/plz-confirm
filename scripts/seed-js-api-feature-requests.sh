#!/usr/bin/env bash
set -euo pipefail

# Seed one request per JS API feature flow so humans can validate in the browser.
#
# Usage:
#   bash scripts/seed-js-api-feature-requests.sh
#   SESSION_ID=global API_BASE_URL=http://localhost:3001 bash scripts/seed-js-api-feature-requests.sh

API_BASE_URL="${API_BASE_URL:-http://localhost:3001}"
SESSION_ID="${SESSION_ID:-global}"
TITLE_PREFIX="${TITLE_PREFIX:-PC02 Human}"

SCRIPT_DIR_REL="ttmp/2026/02/22/PC-02-JS-API-IMPROVEMENTS--js-script-api-improvements/scripts"
SCRIPT_DIR="${SCRIPT_DIR:-$SCRIPT_DIR_REL}"

if ! command -v jq >/dev/null 2>&1; then
  echo "error: jq is required" >&2
  exit 1
fi
if ! command -v curl >/dev/null 2>&1; then
  echo "error: curl is required" >&2
  exit 1
fi

if [[ ! -d "$SCRIPT_DIR" ]]; then
  echo "error: script directory not found: $SCRIPT_DIR" >&2
  exit 1
fi

create_request() {
  local title="$1"
  local script_path="$2"
  local response_file
  response_file="$(mktemp)"

  local http_code
  http_code="$(
    jq -n \
      --arg session "$SESSION_ID" \
      --arg title "$title" \
      --rawfile script "$script_path" \
      '{type:"script",sessionId:$session,scriptInput:{title:$title,timeoutMs:3000,script:$script}}' \
    | curl -sS -o "$response_file" -w "%{http_code}" \
      -X POST "$API_BASE_URL/api/requests" \
      -H 'Content-Type: application/json' \
      -d @-
  )"

  if [[ "$http_code" != "201" ]]; then
    echo "FAILED  $title  http=$http_code" >&2
    cat "$response_file" >&2
    rm -f "$response_file"
    exit 1
  fi

  local id widget step
  id="$(jq -r '.id' "$response_file")"
  widget="$(jq -r '.scriptView.widgetType' "$response_file")"
  step="$(jq -r '.scriptView.stepId // "-"' "$response_file")"
  printf "%-44s  %-36s  %-10s  %s\n" "$title" "$id" "$widget" "$step"
  rm -f "$response_file"
}

echo "API_BASE_URL=$API_BASE_URL"
echo "SESSION_ID=$SESSION_ID"
echo "SCRIPT_DIR=$SCRIPT_DIR"
echo
printf "%-44s  %-36s  %-10s  %s\n" "TITLE" "REQUEST_ID" "WIDGET" "STEP"
printf "%-44s  %-36s  %-10s  %s\n" "-----" "----------" "------" "----"

shopt -s nullglob
scripts=( "$SCRIPT_DIR"/[0-9][0-9]-*.js )
shopt -u nullglob

if [[ "${#scripts[@]}" -eq 0 ]]; then
  echo "error: no scripts matching '$SCRIPT_DIR/[0-9][0-9]-*.js'" >&2
  exit 1
fi

for script_path in "${scripts[@]}"; do
  base="$(basename "$script_path" .js)"
  suffix="${base#*-}"
  title="$TITLE_PREFIX ${base%%-*} ${suffix} ($SESSION_ID)"
  create_request "$title" "$script_path"
done
