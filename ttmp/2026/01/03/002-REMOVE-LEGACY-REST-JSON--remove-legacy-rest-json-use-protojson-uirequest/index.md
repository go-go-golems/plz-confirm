---
Title: Remove legacy REST JSON; use protojson UIRequest
Ticket: 002-REMOVE-LEGACY-REST-JSON
Status: active
Topics:
    - backend
    - api
    - protobuf
    - breaking-change
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: agent-ui-system/client/src/services/websocket.ts
      Note: UI submits responses via REST; change payload to protojson(UIRequest)
    - Path: internal/client/client.go
      Note: CLI currently sends legacy create wrapper; switch to protojson(UIRequest)
    - Path: internal/server/proto_convert.go
      Note: Legacy JSON wrapper conversion to delete/replace
    - Path: internal/server/server.go
      Note: REST handlers to change from wrapper JSON to protojson(UIRequest)
    - Path: internal/server/ws_events.go
      Note: WS already emits protojson(UIRequest); keep consistent
    - Path: pkg/doc/adding-widgets.md
      Note: Update docs for new REST payload shapes
    - Path: proto/plz_confirm/v1/request.proto
      Note: UIRequest envelope includes widget input/output oneofs
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-03T14:23:32.463018368-05:00
WhatFor: ""
WhenToUse: ""
---


# Remove legacy REST JSON; use protojson UIRequest

## Goal

Cut over the REST API from the legacy wrapper shapes:

- create: `{ type, sessionId, input, timeout }`
- respond: `{ output }`

to a single protobuf-backed JSON contract:

- create: `protojson(UIRequest)` (including the input oneof)
- respond: `protojson(UIRequest)` (including the output oneof)

No backwards compatibility: change the existing endpoints and update all clients in the same change.

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- backend
- api
- protobuf
- breaking-change

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
