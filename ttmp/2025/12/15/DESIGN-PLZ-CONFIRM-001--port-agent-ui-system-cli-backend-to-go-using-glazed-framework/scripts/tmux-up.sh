#!/usr/bin/env bash
set -euo pipefail

# tmux session with:
# - control: interactive shell + helper reminders
# - server:  Go backend server (plz-confirm/cmd/agentui serve)
# - vite:    Vite dev server for agent-ui-system frontend
#
# Usage:
#   ./tmux-up.sh
#   tmux attach -t DESIGN-PLZ-CONFIRM-001

SESSION="DESIGN-PLZ-CONFIRM-001"

ROOT="/home/manuel/workspaces/2025-12-15/package-llm-notification-tool"
REPO="$ROOT/plz-confirm"
VITE="$ROOT/vibes/2025-12-15/agent-ui-system"

SERVER_CMD="cd \"$REPO\" && go run ./cmd/agentui serve --addr :3001"
VITE_CMD="cd \"$VITE\" && pnpm dev --host --port 3000"

if tmux has-session -t "$SESSION" 2>/dev/null; then
  echo "tmux session already exists: $SESSION"
  echo "attach with: tmux attach -t $SESSION"
  exit 0
fi

tmux new-session -d -s "$SESSION" -n control

tmux send-keys -t "$SESSION:control" "echo \"Control window for $SESSION\"" C-m
tmux send-keys -t "$SESSION:control" "echo \"- Restart server:  tmux respawn-pane -k -t $SESSION:server \\\"$SERVER_CMD\\\"\"" C-m
tmux send-keys -t "$SESSION:control" "echo \"- Restart vite:    tmux respawn-pane -k -t $SESSION:vite   \\\"$VITE_CMD\\\"\"" C-m
tmux send-keys -t "$SESSION:control" "echo \"- Kill session:    tmux kill-session -t $SESSION\"" C-m

tmux new-window -t "$SESSION" -n server "$SERVER_CMD"
tmux new-window -t "$SESSION" -n vite "$VITE_CMD"

tmux select-window -t "$SESSION:control"

echo "Started tmux session: $SESSION"
echo "Attach with: tmux attach -t $SESSION"


