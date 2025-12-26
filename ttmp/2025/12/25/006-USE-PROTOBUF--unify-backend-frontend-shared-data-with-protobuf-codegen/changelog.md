# Changelog

## 2025-12-25

- Initial workspace created


## 2025-12-25

Completed architecture analysis and protobuf design proposal. Documented all 11 shared data structures, API endpoints, and designed protobuf message hierarchy with oneof patterns for union types. Created 7-phase migration strategy.


## 2025-12-26

Updated design document: removed backward compatibility, changed to single-step migration, kept JSON wire format (no gRPC). Added 5 implementation tasks to ticket.

### Related Files

- /home/manuel/workspaces/2025-12-24/add-img-widget-plz-confirm/plz-confirm/ttmp/2025/12/25/006-USE-PROTOBUF--unify-backend-frontend-shared-data-with-protobuf-codegen/design-doc/01-protobuf-unification-codegen-proposal.md — Updated migration strategy and design decisions


## 2025-12-26

Step 1: Created protobuf definitions and build pipeline (commit db4b5f0)

### Related Files

- /home/manuel/workspaces/2025-12-24/add-img-widget-plz-confirm/plz-confirm/Makefile — Added proto generation target
- /home/manuel/workspaces/2025-12-24/add-img-widget-plz-confirm/plz-confirm/proto/plz_confirm/v1/request.proto — Core protobuf definitions
- /home/manuel/workspaces/2025-12-24/add-img-widget-plz-confirm/plz-confirm/proto/plz_confirm/v1/widgets.proto — Widget Input/Output types


## 2025-12-26

Step 2: Server now uses protobuf UIRequest internally and emits protojson for REST/WS (commit e335e58)

### Related Files

- /home/manuel/workspaces/2025-12-24/add-img-widget-plz-confirm/plz-confirm/internal/server/server.go — REST handlers now return protojson
- /home/manuel/workspaces/2025-12-24/add-img-widget-plz-confirm/plz-confirm/internal/server/ws.go — WS sends raw JSON bytes
- /home/manuel/workspaces/2025-12-24/add-img-widget-plz-confirm/plz-confirm/internal/server/ws_events.go — WS envelope marshaling


## 2025-12-26

Follow-up: protojson enum strings now match legacy wire contract (commit 6cf1da0)

### Related Files

- /home/manuel/workspaces/2025-12-24/add-img-widget-plz-confirm/plz-confirm/Makefile — Make proto marked phony
- /home/manuel/workspaces/2025-12-24/add-img-widget-plz-confirm/plz-confirm/buf.yaml — Buf module root + lint exceptions
- /home/manuel/workspaces/2025-12-24/add-img-widget-plz-confirm/plz-confirm/proto/plz_confirm/v1/request.proto — Enum value names adjusted for JSON compatibility

