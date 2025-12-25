#!/usr/bin/env bash
set -euo pipefail

# Smoke test script for the image widget.
#
# Prereqs:
# - backend server running (Go): `plz-confirm serve --addr :3001`
# - UI running (Vite) on :3000 (proxying /api + /ws to :3001), or production server serving UI
# - browser open to the UI so you can respond to the request
#
# NOTE: Replace ./img*.png with real local files on your machine.

BASE_URL="${BASE_URL:-http://localhost:3000}"

echo "BASE_URL=${BASE_URL}"
echo

echo "== Variant A: image-pick (single select) =="
plz-confirm image \
  --base-url "${BASE_URL}" \
  --title "Select the best screenshot" \
  --message "Pick the image that matches the final UI." \
  --image ./img1.png --image-label "Candidate A" \
  --image ./img2.png --image-label "Candidate B" \
  --output yaml

echo
echo "== Variant B: images as context + multi-select question =="
plz-confirm image \
  --base-url "${BASE_URL}" \
  --title "Review these screenshots" \
  --message "Which issues are present?" \
  --image ./before.png \
  --image ./after.png \
  --multi \
  --option "Text is too small" \
  --option "Button alignment is off" \
  --option "Wrong color theme" \
  --option "Missing icon" \
  --output yaml

echo
echo "== Confirm: similarity check =="
plz-confirm image \
  --base-url "${BASE_URL}" \
  --title "Are these images similar?" \
  --message "Compare the two images and answer yes/no." \
  --mode confirm \
  --image ./imgA.png \
  --image ./imgB.png \
  --output yaml


