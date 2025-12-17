#!/usr/bin/env bash
set -euo pipefail

SESSION="DESIGN-PLZ-CONFIRM-001"
ROOT="/home/manuel/workspaces/2025-12-15/package-llm-notification-tool"
VITE="$ROOT/vibes/2025-12-15/agent-ui-system"
VITE_CMD="cd \"$VITE\" && pnpm dev --host --port 3000"

tmux respawn-pane -k -t "$SESSION:vite" "$VITE_CMD"


