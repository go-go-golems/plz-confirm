---
Title: Integrate go-go-os macOS windowing frontend with plz-confirm backend
Ticket: PC-05-INTEGRATE-OS
Status: complete
Topics:
    - architecture
    - frontend
    - backend
    - go
    - javascript
    - ux
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/design-doc/01-integration-blueprint-plz-confirm-on-go-go-os-macos-windowing.md
      Note: |-
        Primary architecture and integration execution blueprint
        Primary architecture deliverable for this ticket
    - Path: ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/design-doc/02-postmortem-plz-confirm-integration-into-go-go-os.md
      Note: Deep retrospective and integration playbook for future external-system onboarding
    - Path: ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/design-doc/03-playbook-integrating-external-software-into-go-go-os.md
      Note: Reusable step-by-step integration playbook template for future external software onboarding
    - Path: ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/design-doc/04-inspector-review-plz-confirm-integration-quality-audit.md
      Note: Exhaustive inspector-style quality review report with prioritized findings and stabilization plan
    - Path: ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/design-doc/05-deep-dive-q-a-write-pump-duplication-409-and-confirmprotoadapter.md
      Note: Deep intern-focused Q&A covering write pump, duplication inventory, 409 reconciliation, and adapter architecture
    - Path: ttmp/2026/02/23/PC-05-INTEGRATE-OS--integrate-go-go-os-macos-windowing-frontend-with-plz-confirm-backend/reference/01-diary.md
      Note: |-
        Chronological investigation and delivery diary
        Detailed chronological implementation diary
ExternalSources: []
Summary: Ticket workspace for designing PC-05 integration of plz-confirm backend workflows into go-go-os macOS desktop windowing frontend in go-inventory-chat.
LastUpdated: 2026-02-24T10:00:39.310134751-05:00
WhatFor: Track architecture decisions, evidence, and delivery artifacts needed to implement and onboard PC-05 integration work.
WhenToUse: Use as the entry point for this ticket before reading subdocuments or starting implementation tasks.
---



# Integrate go-go-os macOS windowing frontend with plz-confirm backend

## Overview

This ticket captures the architecture analysis and implementation blueprint for replacing the current plz-confirm browser widget frontend with go-go-os macOS-style desktop windows inside `go-inventory-chat`, while keeping script execution in the plz-confirm backend.

## Key Links

- Design blueprint: [design-doc/01-integration-blueprint-plz-confirm-on-go-go-os-macos-windowing.md](./design-doc/01-integration-blueprint-plz-confirm-on-go-go-os-macos-windowing.md)
- Integration postmortem: [design-doc/02-postmortem-plz-confirm-integration-into-go-go-os.md](./design-doc/02-postmortem-plz-confirm-integration-into-go-go-os.md)
- Reusable integration playbook: [design-doc/03-playbook-integrating-external-software-into-go-go-os.md](./design-doc/03-playbook-integrating-external-software-into-go-go-os.md)
- Inspector quality audit: [design-doc/04-inspector-review-plz-confirm-integration-quality-audit.md](./design-doc/04-inspector-review-plz-confirm-integration-quality-audit.md)
- Deep-dive architecture Q&A: [design-doc/05-deep-dive-q-a-write-pump-duplication-409-and-confirmprotoadapter.md](./design-doc/05-deep-dive-q-a-write-pump-duplication-409-and-confirmprotoadapter.md)
- Detailed diary: [reference/01-diary.md](./reference/01-diary.md)
- Task tracker: [tasks.md](./tasks.md)
- Changelog: [changelog.md](./changelog.md)

## Status

Current status: **active**

## Topics

- architecture
- frontend
- backend
- go
- javascript
- ux

## Structure

- `design-doc/` — long-form architecture and integration blueprint
- `reference/` — chronological diary and implementation notes
- `scripts/` — ticket-specific helper scripts (if needed)
- `archive/` — deprecated ticket artifacts
