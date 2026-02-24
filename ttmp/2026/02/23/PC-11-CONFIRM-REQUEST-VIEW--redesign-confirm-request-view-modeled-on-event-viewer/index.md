---
Title: Redesign confirm request view modeled on Event Viewer
Ticket: PC-11-CONFIRM-REQUEST-VIEW
Status: active
Topics:
    - architecture
    - frontend
    - javascript
    - ux
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../go-go-os/apps/inventory/src/App.tsx
      Note: Current queue window and app routing integration point
    - Path: ../../../../../../../go-go-os/packages/engine/src/chat/debug/EventViewerWindow.tsx
      Note: UX interaction model to mirror
    - Path: design-doc/01-design-event-viewer-modeled-confirm-request-view.md
      Note: Primary redesign blueprint
ExternalSources: []
Summary: Separate ticket for redesigning confirm request triage UX with Event Viewer-inspired interaction patterns and diagnostics.
LastUpdated: 2026-02-24T21:09:00-05:00
WhatFor: Provide a focused design-and-implementation track for upgrading confirm queue/request navigation UX quality.
WhenToUse: Use when planning or implementing confirm queue UX improvements in inventory host and potential reusable component extraction.
---

# Redesign confirm request view modeled on Event Viewer

## Overview

This ticket defines and tracks a redesigned confirm request viewer patterned on Event Viewer ergonomics. It focuses on triage quality (filters/search/expansion/follow state) while keeping request execution in dedicated request windows.

## Key Links

- Design doc: [design-doc/01-design-event-viewer-modeled-confirm-request-view.md](./design-doc/01-design-event-viewer-modeled-confirm-request-view.md)
- Task tracker: [tasks.md](./tasks.md)
- Changelog: [changelog.md](./changelog.md)

## Status

Current status: **active**

## Topics

- architecture
- frontend
- javascript
- ux

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
