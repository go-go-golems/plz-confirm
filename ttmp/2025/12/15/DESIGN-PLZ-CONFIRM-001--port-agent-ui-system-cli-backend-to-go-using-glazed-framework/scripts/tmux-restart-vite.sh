#!/usr/bin/env bash
set -euo pipefail

SESSION="DESIGN-PLZ-CONFIRM-001"
ROOT="/home/manuel/workspaces/2025-12-15/package-llm-notification-tool"
VITE="$ROOT/plz-confirm/agent-ui-system"
VITE_CMD="cd \"$VITE\" && (test -d node_modules || pnpm install) && pnpm dev --host --port 3000; echo \"vite exited\"; exec bash"

tmux respawn-pane -k -t "$SESSION:vite" "bash -lc '$VITE_CMD'"


