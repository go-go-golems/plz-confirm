#!/usr/bin/env bash
set -euo pipefail

SESSION="DESIGN-PLZ-CONFIRM-001"
ROOT="/home/manuel/workspaces/2025-12-15/package-llm-notification-tool"
REPO="$ROOT/plz-confirm"
SERVER_CMD="cd \"$REPO\" && go run ./cmd/agentui serve --addr :3001"

tmux respawn-pane -k -t "$SESSION:server" "$SERVER_CMD"


