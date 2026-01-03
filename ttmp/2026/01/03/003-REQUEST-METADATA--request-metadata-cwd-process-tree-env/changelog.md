# Changelog

## 2026-01-03

- Initial workspace created


## 2026-01-03

Create ticket for request metadata; documented post-protojson REST/WS state and outlined Linux-only parent process collection plan

### Related Files

- /home/manuel/workspaces/2026-01-03/plz-confirm-improvements/plz-confirm/ttmp/2026/01/03/003-REQUEST-METADATA--request-metadata-cwd-process-tree-env/analysis/01-request-metadata-architecture-post-protojson-cutover.md — New analysis doc
- /home/manuel/workspaces/2026-01-03/plz-confirm-improvements/plz-confirm/ttmp/2026/01/03/003-REQUEST-METADATA--request-metadata-cwd-process-tree-env/tasks.md — Implementation task breakdown


## 2026-01-03

Implement UIRequest.metadata (cwd + process tree) and wire through CLI/server/store; add smoke coverage (commit 865bcf1d4d7a1f862cce7dd3ce03a20c1ef1bd56).

### Related Files

- /home/manuel/workspaces/2026-01-03/plz-confirm-improvements/plz-confirm/internal/client/client.go — Attach metadata on create
- /home/manuel/workspaces/2026-01-03/plz-confirm-improvements/plz-confirm/internal/metadata/metadata.go — Client-side collector
- /home/manuel/workspaces/2026-01-03/plz-confirm-improvements/plz-confirm/proto/plz_confirm/v1/request.proto — Metadata schema


## 2026-01-03

Closed: request metadata implemented and wired end-to-end (commit 865bcf1d4d7a1f862cce7dd3ce03a20c1ef1bd56).

