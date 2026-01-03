# Tasks

## TODO

- [x] Remove Node legacy server (agent-ui-system/server)
  - [x] Remove `agent-ui-system/server/index.ts`
  - [x] Update `agent-ui-system/package.json` scripts to drop Node server build/start
  - [x] Remove Node-server-only dependencies from `agent-ui-system/package.json` (express/ws/cors/body-parser + @types)
  - [x] Ensure Go embedding pipeline still works (`go generate ./...` uses `pnpm -C agent-ui-system run build`)

- [x] Make history panel fixed-height and prevent page scroll
  - [x] Make `agent-ui-system/client/src/components/Layout.tsx` main area fixed-height (`h-screen`) and `overflow-hidden`
  - [x] Make `agent-ui-system/client/src/pages/Home.tsx` grid fill available height (`flex-1 min-h-0`) so internal ScrollAreas scroll
  - [x] Ensure history panel height is bounded to viewport and doesn’t push left column out of view

- [ ] Document current “request lifecycle” (CLI → server → WS → UI → response → wait)
- [ ] Identify current history mechanisms and limits (UI + server)
- [ ] Choose UX: bounded-only vs backend paging vs both
- [ ] Choose persistence backend (SQLite vs JSONL vs KV) and document decision
- [ ] Specify server-side history API: `GET /api/requests?...` (cursor, limit, filters)
- [ ] Specify request metadata schema (proto) and capture points:
  - CLI-provided (cwd/pid/ppid/parent pids)
  - server-enriched (remote addr / user-agent)
- [ ] Specify auto-default schema and semantics:
  - `autoCompleteAt` vs reuse `expiresAt`
  - completion kind (`user_submitted` vs `auto_default` vs `expired`)
  - how UI labels and summarizes defaulted results
- [ ] Identify required migrations:
  - proto changes + `make codegen`
  - store refactor (introduce interface; persistent impl)
  - new endpoints + UI fetch layer
