# Tasks

## TODO

- [x] [Proposal 1] Fix script request sidebar display: render script title + widget badge + completed metadata; add request-card coverage tests.

- [x] [Proposal 2] Add grid widget support end-to-end: proto/schema validation, frontend grid dialog, click event payload handling, docs/tests.
- [x] [Proposal 3] Implement composite views using sections with one interactive section plus display sections; update runtime validation, proto mapping, renderer, and tests.
- [x] [Proposal 4] Add progress indicators to ScriptView and render progress UI in widget shell with step label and completion ratio.
- [x] [Proposal 5] Add back/undo navigation contract (showBack, backLabel) plus back event wiring from UI through server/engine to scripts.
- [x] [Proposal 6] Add rating/likert widget type with configurable scale, labels, defaults, and submit payload validation.
- [ ] [Proposal 7] Support prefilled defaults/initial values consistently across widgets and preserve user edits during rerenders.
- [ ] [Proposal 9] Add toast/flash message event type and frontend transient notification handling with severity + timeout.
- [ ] [Proposal 14] Add deterministic seeded randomness via ctx.random and ctx.randomInt with per-run seed and reproducibility tests.
- [ ] [Proposal 15] Add rich select options with description, disabled state, and metadata/icon rendering while preserving backward compatibility.
- [ ] [Proposal 11] Implement declarative branching helper API (branch tables/predicates) and runtime integration tests for multi-path flows.
