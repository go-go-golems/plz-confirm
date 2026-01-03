---
Title: Create a PR with prescribe
Ticket: 004-BUG-LANDING-PAGE
Status: complete
Topics:
    - web
    - backend
    - static
    - bug
DocType: playbooks
Intent: long-term
Owners: []
RelatedFiles:
    - Path: .github/workflows/lint.yml
      Note: Example CI gotcha (go:embed requires generated files)
    - Path: .gitignore
      Note: Ignore .pr-builder/ local prescribe state
    - Path: ttmp/2026/01/03/004-BUG-LANDING-PAGE--bug-returns-404-instead-of-react-app/playbooks/01-create-a-pr-with-prescribe.md
      Note: Prescribe PR workflow
    - Path: ttmp/2026/01/03/004-BUG-LANDING-PAGE--bug-returns-404-instead-of-react-app/reference/01-diary.md
      Note: Ticket diary that used this workflow
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-03T15:36:44.255640801-05:00
WhatFor: ""
WhenToUse: ""
---


# Playbook: Create a PR with `prescribe`

## Goal

Generate a high-quality PR title/body from the current branch diff using `prescribe`, then create the PR on GitHub (via `gh pr create`), using `PINOCCHIO_PROFILE=gemini-2.5-pro`.

## Preconditions

- You are on the branch you want to PR: `git status -sb`
- You have a clean working tree (recommended): `git status --porcelain`
- You are authenticated with GitHub CLI: `gh auth status -h github.com`
- The branch is pushed (or you’re willing to let `prescribe create` push it)

## Workflow (CLI)

### 1) Initialize a session (persist it)

```bash
prescribe session init --save \
  --title "Short PR title placeholder" \
  --description "1–2 sentences of intent"
```

Check what you’re about to feed to the model:

```bash
prescribe session show
```

### 2) Add filters to reduce noise (high leverage)

Common examples:

```bash
# Exclude ticket docs / large blobs
prescribe filter add --name "Exclude ticket docs" --exclude "ttmp/**"

# Exclude lockfiles unless you’re intentionally changing deps
prescribe filter add --name "Exclude lockfile" --exclude "agent-ui-system/pnpm-lock.yaml"
```

Re-check token count and included file counts:

```bash
prescribe session show
```

### 3) Generate PR data (Gemini 2.5 Pro via Pinocchio profile)

```bash
PINOCCHIO_PROFILE=gemini-2.5-pro \
prescribe generate \
  --ai-api-type gemini \
  --ai-engine gemini-2.5-pro \
  --stream \
  --output-file .pr-builder/generated-pr.md
```

Notes:
- This also writes structured PR YAML to `.pr-builder/last-generated-pr.yaml`.
- If the story is wrong, iterate by adjusting filters and/or adding a more specific `--title` / `--description` / `--prompt`.

### 4) Dry-run PR creation (recommended)

```bash
PINOCCHIO_PROFILE=gemini-2.5-pro \
prescribe create --use-last --dry-run --base main
```

### 5) Create the PR

```bash
PINOCCHIO_PROFILE=gemini-2.5-pro \
prescribe create --use-last --base main
```

This runs:
- `git push`
- `gh pr create --title ... --body ... --base main`

### 6) Post-create hygiene

`prescribe` stores local state under `.pr-builder/`. Keep it out of git:

```bash
rg -n "\\.pr-builder/" .gitignore || echo ".pr-builder/" >> .gitignore
```

## Troubleshooting

### “Warning: uncommitted change”

You can still create a PR, but it’s easy to accidentally describe changes that aren’t committed. Prefer:

```bash
git status --porcelain
```

Commit or stash first, then regenerate.

### Too many tokens / too much noise

Add filters (docs, generated files, lockfiles, dist output) and re-run generate:

```bash
prescribe filter add --name "Exclude dist" --exclude "dist/**"
prescribe session show
```

### Want to inspect exact model payload (no inference)

```bash
prescribe generate --export-rendered --separator markdown --output-file rendered.md
```
