#!/usr/bin/env bash
set -euo pipefail

# manual-test-image-widget-all-options.sh
#
# Purpose:
# - Manual test runner for the *full surface area* of the new `plz-confirm image` widget.
# - Covers:
#   - mode: select vs confirm
#   - select Variant A (image-pick): multi on/off, labels/alts/captions
#   - select Variant B (images-as-context + checkbox question): options[] + multi on/off
#   - image sources: local file paths, absolute URLs, data:image/... URIs
#   - base-url: via Vite (3000) and direct backend (3001)
#   - output formats: yaml/json/table/csv
#   - negative tests: invalid mode, mismatched metadata counts
#
# Prereqs (recommended dev topology):
# - Go backend: `go run ./cmd/plz-confirm serve --addr :3001`
# - Vite UI:     `pnpm -C agent-ui-system dev --host` (proxy /api and /ws to :3001)
# - Browser open to: http://localhost:3000
#
# Tips:
# - This script runs commands sequentially. Each command will block until you answer in the browser.
# - Set NONINTERACTIVE=1 to skip the pauses between cases (still blocks for UI interaction).

UI_BASE_URL="${UI_BASE_URL:-http://localhost:3000}"
API_BASE_URL="${API_BASE_URL:-http://localhost:3001}"
WAIT_TIMEOUT="${WAIT_TIMEOUT:-600}"
TIMEOUT_S="${TIMEOUT_S:-300}"
NONINTERACTIVE="${NONINTERACTIVE:-0}"

require_bin() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "ERROR: missing required binary: $1" >&2
    exit 1
  }
}

require_bin curl
require_bin jq
require_bin base64
require_bin go

# Use the current source tree by default to avoid a stale installed binary.
PLZ_CONFIRM_CMD_DEFAULT=(go run ./cmd/plz-confirm)
if [ -n "${PLZ_CONFIRM_BIN:-}" ]; then
  PLZ_CONFIRM_CMD_DEFAULT=("${PLZ_CONFIRM_BIN}")
fi

say() { printf "\n== %s ==\n" "$*"; }

pause() {
  if [ "$NONINTERACTIVE" = "1" ]; then return 0; fi
  read -r -p "Press Enter to run the next case (Ctrl+C to stop)..." _
}

mk_png_fixtures() {
  # Tiny 1x1 PNG
  local png_b64
  png_b64='iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO7+qK0AAAAASUVORK5CYII='

  printf '%s' "$png_b64" | base64 -d > /tmp/plz-img-1.png
  printf '%s' "$png_b64" | base64 -d > /tmp/plz-img-2.png
  printf '%s' "$png_b64" | base64 -d > /tmp/plz-img-3.png

  printf '%s' "$png_b64" | base64 -d > /tmp/plz-img-a.png
  printf '%s' "$png_b64" | base64 -d > /tmp/plz-img-b.png

  echo "$png_b64" > /tmp/plz-img.b64
}

upload_to_api_images() {
  local file="$1"
  curl -sS -F "file=@${file}" "${API_BASE_URL}/api/images" | jq -r '.url' | sed -E "s|^|${API_BASE_URL}|"
}

run_case() {
  local title="$1"; shift
  local output="$1"; shift

  say "$title"
  echo "UI: ${UI_BASE_URL}"
  echo "Hint: watch the browser; this command will block until you submit."
  echo
  echo "Command:"
  printf "  %q" "${PLZ_CONFIRM_CMD_DEFAULT[@]}" "$@"
  echo " --base-url ${UI_BASE_URL} --timeout ${TIMEOUT_S} --wait-timeout ${WAIT_TIMEOUT} --output ${output}"
  echo

  "${PLZ_CONFIRM_CMD_DEFAULT[@]}" "$@" \
    --base-url "${UI_BASE_URL}" \
    --timeout "${TIMEOUT_S}" \
    --wait-timeout "${WAIT_TIMEOUT}" \
    --output "${output}"
}

run_case_direct_backend() {
  local title="$1"; shift
  local output="$1"; shift

  say "$title"
  echo "Direct backend mode: CLI uses API_BASE_URL=${API_BASE_URL}"
  echo "Note: browser UI can still be on ${UI_BASE_URL}; CLI talks to ${API_BASE_URL}"
  echo

  "${PLZ_CONFIRM_CMD_DEFAULT[@]}" "$@" \
    --base-url "${API_BASE_URL}" \
    --timeout "${TIMEOUT_S}" \
    --wait-timeout "${WAIT_TIMEOUT}" \
    --output "${output}"
}

negative_case() {
  local title="$1"; shift
  say "NEGATIVE: $title"
  set +e
  "${PLZ_CONFIRM_CMD_DEFAULT[@]}" "$@" --base-url "${UI_BASE_URL}" --wait-timeout 3 --output yaml
  local rc=$?
  set -e
  echo "exit_code=${rc} (expected non-zero)"
}

say "Config"
echo "UI_BASE_URL=${UI_BASE_URL}"
echo "API_BASE_URL=${API_BASE_URL}"
echo "WAIT_TIMEOUT=${WAIT_TIMEOUT}"
echo "TIMEOUT_S=${TIMEOUT_S}"
echo "PLZ_CONFIRM_BIN=${PLZ_CONFIRM_BIN:-'(default: go run ./cmd/plz-confirm)'}"
echo

mk_png_fixtures

say "Sanity: /api/images works (upload + HEAD)"
UP_JSON="$(curl -sS -F "file=@/tmp/plz-img-1.png" "${API_BASE_URL}/api/images")"
echo "$UP_JSON"
IMG_URL="$(echo "$UP_JSON" | jq -r '.url')"
curl -sS -I "${API_BASE_URL}${IMG_URL}" | head -n 8

pause

# --- Image widget cases ---

run_case "Image select (Variant A, single) - labels + alts + captions" yaml \
  image \
  --title "Variant A: pick one image" \
  --message "Pick the best candidate." \
  --image /tmp/plz-img-1.png --image-label "Candidate A" --image-alt "Tiny PNG A" --image-caption "Caption A" \
  --image /tmp/plz-img-2.png --image-label "Candidate B" --image-alt "Tiny PNG B" --image-caption "Caption B"

pause

run_case "Image select (Variant A, multi) - 3 images" json \
  image \
  --title "Variant A: pick multiple images" \
  --message "Select all that apply." \
  --multi \
  --image /tmp/plz-img-1.png --image-label "One" \
  --image /tmp/plz-img-2.png --image-label "Two" \
  --image /tmp/plz-img-3.png --image-label "Three"

pause

run_case "Image select (Variant B, single) - images as context + checkbox question (options)" table \
  image \
  --title "Variant B: choose one issue" \
  --message "Which issue is most severe?" \
  --image /tmp/plz-img-a.png \
  --image /tmp/plz-img-b.png \
  --option "Text is too small" \
  --option "Button alignment is off" \
  --option "Wrong color theme" \
  --option "Missing icon"

pause

run_case "Image select (Variant B, multi) - images as context + checkbox question (options + --multi)" yaml \
  image \
  --title "Variant B: choose all issues" \
  --message "Select every issue you see." \
  --multi \
  --image /tmp/plz-img-a.png \
  --image /tmp/plz-img-b.png \
  --option "Text is too small" \
  --option "Button alignment is off" \
  --option "Wrong color theme" \
  --option "Missing icon"

pause

run_case "Image confirm (mode=confirm) - approve/reject buttons" csv \
  image \
  --title "Confirm: are these images similar?" \
  --message "Approve if similar; reject otherwise." \
  --mode confirm \
  --image /tmp/plz-img-a.png \
  --image /tmp/plz-img-b.png

pause

say "Image sources: URL + data URI"
echo "We’ll test:"
echo "- URL sources by uploading to /api/images and using the absolute URL"
echo "- data URI source using a small base64 string"
pause

IMG_URL_A="$(upload_to_api_images /tmp/plz-img-1.png)"
IMG_URL_B="$(upload_to_api_images /tmp/plz-img-2.png)"
DATA_URI="data:image/png;base64,$(cat /tmp/plz-img.b64)"

run_case "Image select (Variant A) with URL images" yaml \
  image \
  --title "URL images" \
  --message "These images are served from /api/images/{id} (absolute URLs)." \
  --image "${IMG_URL_A}" --image-label "URL A" \
  --image "${IMG_URL_B}" --image-label "URL B"

pause

run_case "Image select (Variant A) with a data: URI image (plus one URL)" yaml \
  image \
  --title "data: URI image" \
  --message "One image is a data URI; one is a URL." \
  --image "${DATA_URI}" --image-label "DATA_URI" \
  --image "${IMG_URL_A}" --image-label "URL"

pause

run_case_direct_backend "Direct backend base-url (CLI talks to :3001 directly)" yaml \
  image \
  --title "Direct backend mode" \
  --message "CLI uses --base-url http://localhost:3001." \
  --image /tmp/plz-img-1.png --image-label "A" \
  --image /tmp/plz-img-2.png --image-label "B"

pause

# --- Negative tests (do not block on UI) ---

negative_case "Invalid mode value" image \
  --title "Bad mode" \
  --mode banana \
  --image /tmp/plz-img-1.png

negative_case "Mismatched --image-label count" image \
  --title "Bad labels" \
  --image /tmp/plz-img-1.png --image /tmp/plz-img-2.png \
  --image-label "only-one-label"

say "Done"
echo "If you want to re-run just one case, copy/paste the printed command."
