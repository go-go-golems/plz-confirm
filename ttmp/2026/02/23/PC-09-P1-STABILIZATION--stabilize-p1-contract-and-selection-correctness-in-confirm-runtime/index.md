---
Title: Stabilize P1 contract and selection correctness in confirm-runtime
Ticket: PC-09-P1-STABILIZATION
Status: complete
Topics:
    - architecture
    - frontend
    - backend
    - javascript
    - go
    - api
    - bug
    - ux
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/02/23/PC-09-P1-STABILIZATION--stabilize-p1-contract-and-selection-correctness-in-confirm-runtime/design-doc/01-implementation-plan-p1-stabilization.md
      Note: Execution blueprint for resolving all P1 findings
    - Path: ttmp/2026/02/23/PC-09-P1-STABILIZATION--stabilize-p1-contract-and-selection-correctness-in-confirm-runtime/reference/01-diary.md
      Note: Detailed chronological implementation diary for this stabilization ticket
ExternalSources: []
Summary: Follow-up stabilization ticket to resolve all P1 contract and selection correctness findings from the PC-05 inspector audit.
LastUpdated: 2026-02-24T20:38:00-05:00
WhatFor: Drive and track code/test/documentation work needed to eliminate high-severity integration correctness risks before broader rollout.
WhenToUse: Use as the entry point for planning, executing, and reviewing P1 stabilization work.
---


# Stabilize P1 contract and selection correctness in confirm-runtime

## Overview

This ticket implements and verifies fixes for the three high-severity (`P1`) findings from the PC-05 inspector review:

1. mode-aware oneof output encoding for multi-capable widgets,
2. table selection key correctness for rows without `id`,
3. upload `maxSize` numeric-string normalization.

## Key Links

- Implementation plan: [design-doc/01-implementation-plan-p1-stabilization.md](./design-doc/01-implementation-plan-p1-stabilization.md)
- Detailed diary: [reference/01-diary.md](./reference/01-diary.md)
- Task tracker: [tasks.md](./tasks.md)
- Changelog: [changelog.md](./changelog.md)

## Status

Current status: **complete**

## Topics

- architecture
- frontend
- backend
- javascript
- go
- api
- bug
- ux

## Structure

- `design-doc/` — implementation design and execution plans
- `reference/` — step-by-step diary and technical evidence
- `scripts/` — ticket-specific helper scripts (if needed)
- `archive/` — deprecated ticket artifacts
