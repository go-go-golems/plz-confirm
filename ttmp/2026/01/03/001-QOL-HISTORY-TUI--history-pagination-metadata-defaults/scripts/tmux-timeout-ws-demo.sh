#!/usr/bin/env bash
set -euo pipefail

# tmux-timeout-ws-demo.sh
#
# Purpose:
# - Run a reproducible dev setup to test server-side timeouts over WebSocket:
#   - Go backend on :3001
#   - Vite UI on :3000 (proxy to :3001)
#   - CLI WS watcher prints WS events
#   - Control window shows the exact CLI command to trigger a short-timeout request
#
# Usage:
#   bash ttmp/2026/01/03/001-QOL-HISTORY-TUI--history-pagination-metadata-defaults/scripts/tmux-timeout-ws-demo.sh
#   tmux attach -t PLZ-TIMEOUT
#
# Then open:
#   http://localhost:3000/?sessionId=global
#
# Notes:
# - Kill port holders via lsof-who (preferred) if needed:
#     lsof-who -p 3000 -k
#     lsof-who -p 3001 -k

SESSION="${SESSION:-PLZ-TIMEOUT}"
API_ADDR="${API_ADDR:-:3001}"
UI_PORT="${UI_PORT:-3000}"
SESSION_ID="${SESSION_ID:-global}"

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../../.." && pwd)"
VITE="$REPO/agent-ui-system"

SERVER_CMD="cd \"$REPO\" && go run ./cmd/plz-confirm serve --addr \"$API_ADDR\" 2>&1 | tee /tmp/plz-confirm-timeout-server.log"
VITE_CMD="cd \"$VITE\" && (test -d node_modules || pnpm install) && pnpm dev --host --port \"$UI_PORT\" 2>&1 | tee /tmp/plz-confirm-timeout-vite.log; echo \"vite exited\"; exec bash"
WS_CMD="cd \"$REPO\" && go run ./cmd/plz-confirm ws --base-url \"http://localhost${API_ADDR}\" --session-id \"$SESSION_ID\" --pretty 2>&1 | tee /tmp/plz-confirm-timeout-ws.log; echo \"ws watcher exited\"; exec bash"

TRIGGER_CMD="cd \"$REPO\" && go run ./cmd/plz-confirm confirm --base-url \"http://localhost${API_ADDR}\" --session-id \"$SESSION_ID\" --timeout 20 --wait-timeout 120 --title \"TIMEOUT_DEMO\" --message \"Click approve within 20s, or let it expire to see status=completed + comment=AUTO_TIMEOUT.\""

if tmux has-session -t "$SESSION" 2>/dev/null; then
  echo "tmux session already exists: $SESSION"
  echo "attach with: tmux attach -t $SESSION"
  exit 0
fi

tmux new-session -d -s "$SESSION" -n control

tmux send-keys -t "$SESSION:control" "echo \"Repo: $REPO\"" C-m
tmux send-keys -t "$SESSION:control" "echo \"UI:   http://localhost:$UI_PORT/?sessionId=$SESSION_ID\"" C-m
tmux send-keys -t "$SESSION:control" "echo \"Logs: tail -f /tmp/plz-confirm-timeout-server.log /tmp/plz-confirm-timeout-ws.log\"" C-m
tmux send-keys -t "$SESSION:control" "echo \"Trigger (paste in a shell):\"" C-m
tmux send-keys -t "$SESSION:control" "echo \"$TRIGGER_CMD\"" C-m

tmux new-window -t "$SESSION" -n server "$SERVER_CMD"
tmux new-window -t "$SESSION" -n vite "bash -lc '$VITE_CMD'"
tmux new-window -t "$SESSION" -n ws "bash -lc '$WS_CMD'"
tmux new-window -t "$SESSION" -n trigger "bash"
tmux send-keys -t "$SESSION:trigger" "$TRIGGER_CMD" C-m

tmux select-window -t "$SESSION:control"

echo "Started tmux session: $SESSION"
echo "Attach with: tmux attach -t $SESSION"
