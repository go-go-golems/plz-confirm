# Changelog

## 2026-01-03

- Initial workspace created


## 2026-01-03

Define cut-over ticket to remove legacy REST wrapper JSON and switch clients/server to protojson(UIRequest) with no backwards compatibility

### Related Files

- /home/manuel/workspaces/2026-01-03/plz-confirm-improvements/plz-confirm/ttmp/2026/01/03/002-REMOVE-LEGACY-REST-JSON--remove-legacy-rest-json-use-protojson-uirequest/index.md — Added goal + related files
- /home/manuel/workspaces/2026-01-03/plz-confirm-improvements/plz-confirm/ttmp/2026/01/03/002-REMOVE-LEGACY-REST-JSON--remove-legacy-rest-json-use-protojson-uirequest/tasks.md — Added detailed task breakdown for no-BC cutover


## 2026-01-03

Breaking cutover: REST endpoints now accept protojson(UIRequest) bodies (no legacy wrapper JSON); updated CLI, UI response submit, and smoke scripts

### Related Files

- /home/manuel/workspaces/2026-01-03/plz-confirm-improvements/plz-confirm/agent-ui-system/client/src/services/websocket.ts — submitResponse now sends UIRequest output oneof
- /home/manuel/workspaces/2026-01-03/plz-confirm-improvements/plz-confirm/internal/client/client.go — CreateRequest now sends protojson(UIRequest)
- /home/manuel/workspaces/2026-01-03/plz-confirm-improvements/plz-confirm/internal/server/server.go — Decode protojson(UIRequest) for create/response
- /home/manuel/workspaces/2026-01-03/plz-confirm-improvements/plz-confirm/scripts/curl-inspector-smoke.sh — Updated for new REST body shapes

