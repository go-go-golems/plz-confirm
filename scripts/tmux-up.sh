#!/usr/bin/env bash
set -euo pipefail

# Repo-agnostic tmux dev session for plz-confirm.
#
# Windows:
# - control: reminders + common commands
# - server:  Go backend server on :3001
# - vite:    Vite dev server on :3000 (proxies /api and /ws to :3001)
# - tests:   optional smoke runner window (does not auto-run; you can paste/run)
#
# Usage:
#   bash scripts/tmux-up.sh
#   tmux attach -t PLZ-CONFIRM
#
# Optional env:
#   SESSION=PLZ-CONFIRM
#   API_ADDR=:3001
#   UI_PORT=3000

SESSION="${SESSION:-PLZ-CONFIRM}"
API_ADDR="${API_ADDR:-:3001}"
UI_PORT="${UI_PORT:-3000}"

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VITE="$REPO/agent-ui-system"

SERVER_CMD="cd \"$REPO\" && go run ./cmd/plz-confirm serve --addr \"$API_ADDR\" 2>&1 | tee /tmp/plz-confirm-server.log"
VITE_CMD="cd \"$VITE\" && (test -d node_modules || pnpm install) && pnpm dev --host --port \"$UI_PORT\" 2>&1 | tee /tmp/plz-confirm-vite.log; echo \"vite exited\"; exec bash"

if tmux has-session -t "$SESSION" 2>/dev/null; then
  echo "tmux session already exists: $SESSION"
  echo "attach with: tmux attach -t $SESSION"
  exit 0
fi

tmux new-session -d -s "$SESSION" -n control

tmux send-keys -t "$SESSION:control" "echo \"Control window for $SESSION\"" C-m
tmux send-keys -t "$SESSION:control" "echo \"Repo: $REPO\"" C-m
tmux send-keys -t "$SESSION:control" "echo \"- Restart server: tmux respawn-pane -k -t $SESSION:server \\\"$SERVER_CMD\\\"\"" C-m
tmux send-keys -t "$SESSION:control" "echo \"- Restart vite:   tmux respawn-pane -k -t $SESSION:vite   \\\"bash -lc '$VITE_CMD'\\\"\"" C-m
tmux send-keys -t "$SESSION:control" "echo \"- Logs: tail -f /tmp/plz-confirm-server.log /tmp/plz-confirm-vite.log\"" C-m
tmux send-keys -t "$SESSION:control" "echo \"- CLI manual suite: bash ttmp/2025/12/15/DESIGN-PLZ-CONFIRM-001--port-agent-ui-system-cli-backend-to-go-using-glazed-framework/scripts/test-all-commands.sh\"" C-m
tmux send-keys -t "$SESSION:control" "echo \"- CLI auto suite (API-driven): bash ttmp/2025/12/24/001-ADD-IMG-WIDGET--add-image-widget-to-cli-and-web-interface/scripts/auto-e2e-cli-via-api.sh\"" C-m
tmux send-keys -t "$SESSION:control" "echo \"- Kill session:   tmux kill-session -t $SESSION\"" C-m

tmux new-window -t "$SESSION" -n server "$SERVER_CMD"
tmux new-window -t "$SESSION" -n vite "bash -lc '$VITE_CMD'"

tmux new-window -t "$SESSION" -n tests "bash"
tmux send-keys -t "$SESSION:tests" "cd \"$REPO\" && echo \"Run one:\" && echo \"bash ttmp/2025/12/24/001-ADD-IMG-WIDGET--add-image-widget-to-cli-and-web-interface/scripts/auto-e2e-cli-via-api.sh\" && echo \"bash ttmp/2025/12/15/DESIGN-PLZ-CONFIRM-001--port-agent-ui-system-cli-backend-to-go-using-glazed-framework/scripts/test-all-commands.sh\"" C-m

tmux select-window -t "$SESSION:control"

echo "Started tmux session: $SESSION"
echo "Attach with: tmux attach -t $SESSION"


