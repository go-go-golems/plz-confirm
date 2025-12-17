---
Title: Bug report
Ticket: 002-FIX-FORM-DISPLAY
Status: active
Topics:
    - plz-confirm
    - frontend
    - bug
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Form widget UI shows only raw fields (property keys) and omits schema/context (instructions, field titles/descriptions) when using plz-confirm form."
LastUpdated: 2025-12-17T16:23:02.557770064-05:00
---

# Bug report — Form widget missing schema/context display

## Summary

When running `plz-confirm form` with a JSON Schema that includes human-facing context (schema-level instructions and per-field titles/descriptions), the web UI renders **only a list of bare input fields** with labels derived from the raw property keys (e.g. `q1_total_files`). The contextual content is not displayed, making the form hard/impossible to complete correctly.

## Reproduction

### Prerequisites
- `plz-confirm` server + web UI running (default `http://localhost:3000`)

### Command

```bash
plz-confirm form \
  --title "📚 Documentation Cleanup Quiz - Typed Turn Keys" \
  --schema @/tmp/doc-cleanup-quiz.json \
  --output json \
  --wait-timeout 300 \
  --base-url http://localhost:3000
```

### Observed behavior
- The UI shows the dialog title.
- The UI renders a sequence of fields named like `q10_commits`, `q1_total_files`, `q2_severity`, … with generic placeholders like `ENTER_Q1_TOTAL_FILES...`.
- There is **no visible schema-level instructions/context** and **no per-field question text/help** (e.g. field `title`/`description`).

(See also the DOM snippet provided in the ticket description; form element id `schema-form`.)

## Expected behavior

At minimum:
- Render schema-level context if present:
  - `schema.title` (optional, if distinct from `input.title`)
  - `schema.description` (instructions)
- Render per-field context if present:
  - Use `properties[<name>].title` as the label (fallback to `<name>`)
  - Show `properties[<name>].description` as helper text under the label
- Avoid replacing meaningful context with generic placeholders when better options exist (e.g. use example/default/description).

## Impact
- **High usability impact**: for “quiz” style schemas (like the documentation cleanup quiz), the user can’t see the question/instructions, only internal field identifiers.
- In practice this defeats the purpose of using the web UI to collect structured, human-friendly answers.

## Notes / scope
- This appears to be a **frontend rendering gap**, not a request transport issue: CLI/server transmit the schema as `any` and the client receives it intact (see analysis doc).

