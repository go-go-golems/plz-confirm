# Changelog

## 2025-12-17

- Initial workspace created


## 2025-12-17

Implemented browser notifications for new requests. Created notification service that requests permission on app load and shows notifications when new requests arrive via WebSocket. Notifications include request type and title, and clicking them focuses the window.

### Related Files

- /home/manuel/workspaces/2025-12-15/package-llm-notification-tool/plz-confirm/agent-ui-system/client/src/App.tsx — Request notification permission on app load
- /home/manuel/workspaces/2025-12-15/package-llm-notification-tool/plz-confirm/agent-ui-system/client/src/services/notifications.ts — New browser notification service
- /home/manuel/workspaces/2025-12-15/package-llm-notification-tool/plz-confirm/agent-ui-system/client/src/services/websocket.ts — Integrated notification service to show notifications on new_request messages


## 2025-12-17

Tested browser notifications by starting servers via tmux and sending a test confirm request. Ready for user verification in frontend.

