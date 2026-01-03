---
Title: Create PR Diary (prescribe)
Ticket: 004-BUG-LANDING-PAGE
Status: active
Topics:
    - web
    - backend
    - static
    - bug
    - tooling
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/01/03/004-BUG-LANDING-PAGE--bug-returns-404-instead-of-react-app/playbooks/01-create-a-pr-with-prescribe.md
      Note: Playbook followed to generate/create PRs via prescribe
    - Path: .gitignore
      Note: Ensure `.pr-builder/` is ignored (prescribe local state)
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-03T16:46:07-05:00
WhatFor: ""
WhenToUse: ""
---

# Create PR Diary (prescribe)

## Goal

Backfill the exact steps taken while following the “Create a PR with `prescribe`” playbook, including the practical gotchas encountered (diff noise, base branch mismatch, and model hallucinations in the generated PR body).

## Step 1: Read the playbook and validate prerequisites

This step loaded the ticket playbook and confirmed the local environment supports the prescribed flow (GitHub CLI auth + `prescribe` installed). The impact is that we can run `prescribe generate`/`prescribe create` without needing additional setup.

### What I did
- Read `ttmp/2026/01/03/004-BUG-LANDING-PAGE--bug-returns-404-instead-of-react-app/playbooks/01-create-a-pr-with-prescribe.md`.
- Verified `gh auth status -h github.com` shows an active authenticated account:
  - `✓ Logged in to github.com account wesen (keyring)`
  - `Active account: true`
- Verified `prescribe` is installed (`command -v prescribe` → `~/.local/bin/prescribe`).

### Why
- The playbook expects `gh` auth and optionally pushes branches; failing late here wastes time.

### What worked
- GitHub CLI was already authenticated.
- `prescribe` was already available at `~/.local/bin/prescribe`.

### What didn't work
- N/A

### What I learned
- N/A

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- N/A

### What should be done in the future
- N/A

### Code review instructions
- Start with `ttmp/2026/01/03/004-BUG-LANDING-PAGE--bug-returns-404-instead-of-react-app/playbooks/01-create-a-pr-with-prescribe.md` and confirm the prerequisites match your local environment.

## Step 2: Inspect git state and identify the effective PR base

This step checked the current branch state and the diff against `origin/main` to understand what the PR would actually contain. The impact is catching “wrong base” situations early, before we generate PR text against the wrong comparison.

### What I did
- Ran `git status -sb` and confirmed we were on `task/plz-confirm-improvements...origin/task/plz-confirm-improvements` with a clean working tree.
- Ran `git log -5 --oneline --decorate` and confirmed the head commits were:
  - `4404411` “📝 docs: close request metadata ticket”
  - `865bcf1` “✨ metadata: attach request provenance to UIRequest”
  - `9443ddf` “👷 ci: remove nancy scan”
- Checked remotes with `git remote -v`:
  - `origin git@github.com:go-go-golems/plz-confirm`
  - `wesen git@github.com:wesen/plz-confirm.git`
- Fetched base with `git fetch origin main`.
- Compared against the intended base:
  - `git log --oneline origin/main..HEAD` → 2 commits (`4404411`, `865bcf1`)
  - `git diff --stat origin/main...HEAD` → `20 files changed, 800 insertions(+), 93 deletions(-)`
  - `git merge-base HEAD origin/main` → `9443ddf` (this commit is already on `origin/main`)
- Confirmed local `main` was stale vs upstream:
  - `git rev-parse main` → `9991704cf4d9929f02e51dffd46e5d60ce1d04f8`
  - `git rev-parse origin/main` → `d98f7b68a2c3838d5c25084420310b7149b3b6be`

### Why
- `prescribe session init` defaults to `main` as base; if local `main` is stale, the session includes unrelated commits/files and the generated PR copy becomes misleading.

### What worked
- Using `origin/main` directly showed the correct “what would be merged” diff.

### What didn't work
- Relying on local `main` as “the base” produced extra noise because `main` was behind `origin/main`.

### What I learned
- Always prefer `origin/main` (or `upstream/main`) as the compare base when generating PR text, unless you’ve explicitly fast-forwarded local `main`.

### What was tricky to build
- It’s easy to miss this because the branch can be “up to date” with its remote (`origin/task/...`) while local `main` is still behind.

### What warrants a second pair of eyes
- Confirm the intended PR base is `origin/main` and that we aren’t accidentally including already-merged work.

### What should be done in the future
- Update the playbook to explicitly recommend `--target origin/main` (or to `git fetch origin main && git switch main && git pull --ff-only`) before `prescribe session init`.

### Code review instructions
- Validate base selection with `git log --oneline origin/main..HEAD` and ensure it matches what you expect the PR to contain.

## Step 3: Initialize a prescribe session and reduce context noise

This step initialized `.pr-builder/session.yaml` and then reduced included files via filters to keep token count reasonable and avoid feeding irrelevant docs/generated output to the model. The impact is cheaper, faster inference and less chance of the model keying off unrelated documents.

### What I did
- Ran:
  - `prescribe session init --save --title "metadata: attach request provenance to UIRequest" --description "Capture request cwd/process info, persist it, and surface a compact label in the UI."`
  - Output: `Target: main` and `Files: 74`
- Ran `prescribe session show` and captured the key numbers:
  - `total_files: 74`, `visible_files: 74`, `included_files: 74`, `token_count: 91993`
- Added filters to exclude:
  - `prescribe filter add --name "Exclude ticket docs" --exclude "ttmp/**"` (after this: `Files now filtered: 44`)
  - `prescribe filter add --name "Exclude generated proto" --exclude "proto/generated/**"` (after this: `Files now filtered: 45`)
  - `prescribe filter add --name "Exclude generated TS proto" --exclude "agent-ui-system/client/src/proto/generated/**"` (after this: `Files now filtered: 46`)
  - `prescribe filter add --name "Exclude commit msg config" --exclude ".git-commit-message.yaml"` (after this: `Files now filtered: 47`)
- Re-ran `prescribe session show` and confirmed:
  - `visible_files: 27`, `included_files: 27`, `filtered_files: 47`, `token_count: 44094`
- Verified `.pr-builder/` is already ignored (`.gitignore` contains `.pr-builder/` at line 22).

### Why
- The branch diff included both code and extensive ticket docs; without filtering, the model context was large and noisy.

### What worked
- Token count dropped substantially after excluding ticket docs and generated files.

### What didn't work
- N/A

### What I learned
- Excluding `ttmp/**` and generated artifacts is high leverage for PR description quality.

### What was tricky to build
- Choosing filters that remove noise without hiding review-relevant changes requires knowing which files are regenerated vs authored.

### What warrants a second pair of eyes
- Confirm we didn’t filter out an authored file that should be described/reviewed (e.g., hand-edited generated code).

### What should be done in the future
- Consider adding a standard “repo preset” of filters for `prescribe` (docs, lockfiles, generated code).

### Code review instructions
- Inspect `.pr-builder/session.yaml` to confirm the filtered file list matches review intent (and that `.pr-builder/` remains gitignored).

## Step 4: Generate PR title/body and correct hallucinated content

This step ran `prescribe generate` using the requested `PINOCCHIO_PROFILE=gemini-2.5-pro` and iterated on prompts after the first generation included claims not supported by the diff. The impact is preventing a PR body that promises refactors/removals that didn’t happen.

### What I did
- Generated PR YAML with:
  - `PINOCCHIO_PROFILE=gemini-2.5-pro prescribe generate --ai-api-type gemini --ai-engine gemini-2.5-pro --stream --output-file .pr-builder/generated-pr.md`
  - This wrote `.pr-builder/generated-pr.md` and `.pr-builder/last-generated-pr.yaml`.
- Observed the first generation’s YAML body included unsupported claims, including:
  - “Unified API Contract … removing custom JSON wrappers”
  - “Removed Node.js Server … Go binary is now the sole server”
- Ran `prescribe generate --help` to look for a “load session”/dry-run path; noted `--create`, `--create-dry-run`, and `-s, --load-session`.
- Tried a stricter prompt (only use diff context); that run produced non-YAML prose and failed parsing with:
  - `failed to parse PR YAML: yaml: line 3: did not find expected alphabetic or numeric character`
  - Output still wrote to `.pr-builder/generated-pr.md`.
- Re-ran generation with an explicit “ONLY YAML; specific keys” prompt:
  - YAML parsing succeeded and updated `.pr-builder/last-generated-pr.yaml`
  - The output still repeated the same unsupported “architecture simplification” claims.

### Why
- PR descriptions that include incorrect claims create review thrash and can mislead downstream users (release notes, changelog).

### What worked
- The “ONLY YAML with keys” constraint restored YAML parsing and updated `.pr-builder/last-generated-pr.yaml`.

### What didn't work
- The model still hallucinated significant refactors/removals that were not in the provided diff context; prompting alone didn’t eliminate it reliably.

### What I learned
- For high-stakes PR bodies, treat LLM output as a draft and sanity-check against `git log origin/main..HEAD` / `git diff --stat origin/main...HEAD`.

### What was tricky to build
- Getting a concise, accurate PR body is sensitive to (a) base branch selection and (b) which files are included; either mistake amplifies hallucinations.

### What warrants a second pair of eyes
- Validate the generated PR body in `.pr-builder/generated-pr.md` against the actual diff; remove any “refactor/removal” bullets that aren’t explicitly present.

### What should be done in the future
- Add a playbook guardrail: before creating the PR, explicitly verify the generated bullets map to concrete file changes (e.g., `git diff --name-only origin/main...HEAD`).

### Code review instructions
- Start by reading `.pr-builder/generated-pr.md` and cross-check each claim against `git diff --stat origin/main...HEAD`.

## Step 5: Attempt to rebase the session on `origin/main` and note an apparent session/show mismatch

This step attempted to re-initialize the session with `--target origin/main` to avoid local-`main` drift. The impact is uncovering that `prescribe session show` may not reflect the saved session file unless explicitly loaded (or there may be a bug in how the default session is discovered).

### What I did
- Ran `prescribe session init --save --target origin/main ...`:
  - Output: `Target: origin/main` and `Files: 20`
- Inspected `.pr-builder/session.yaml` and confirmed it recorded `target_branch: origin/main` and exactly these 20 files:
  - `.git-commit-message.yaml`
  - `agent-ui-system/client/src/pages/Home.tsx`
  - `agent-ui-system/client/src/proto/generated/plz_confirm/v1/request.ts`
  - `internal/client/client.go`
  - `internal/metadata/metadata.go`
  - `internal/metadata/metadata_linux_test.go`
  - `internal/metadata/metadata_test.go`
  - `internal/metadata/process_linux.go`
  - `internal/metadata/process_other.go`
  - `internal/metadata/process_shims.go`
  - `internal/server/server.go`
  - `internal/store/store.go`
  - `pkg/doc/adding-widgets.md`
  - `proto/generated/go/plz_confirm/v1/request.pb.go`
  - `proto/plz_confirm/v1/request.proto`
  - `scripts/curl-inspector-smoke.sh`
  - `ttmp/2026/01/03/003-REQUEST-METADATA--request-metadata-cwd-process-tree-env/changelog.md`
  - `ttmp/2026/01/03/003-REQUEST-METADATA--request-metadata-cwd-process-tree-env/index.md`
  - `ttmp/2026/01/03/003-REQUEST-METADATA--request-metadata-cwd-process-tree-env/reference/01-diary.md`
  - `ttmp/2026/01/03/003-REQUEST-METADATA--request-metadata-cwd-process-tree-env/tasks.md`
- Ran `prescribe session show` and observed it still reported `target_branch` as `main` and `total_files` as `74`, which didn’t match the saved session file contents.

### Why
- If `prescribe session show` doesn’t reflect the on-disk session, it’s easy to generate/create against the wrong base without realizing.

### What worked
- Verifying `.pr-builder/session.yaml` directly provided ground truth for what was persisted.

### What didn't work
- Using `prescribe session show` as the sole verification signal was unreliable in this run.

### What I learned
- When in doubt, read `.pr-builder/session.yaml` directly and/or use `prescribe generate -s .pr-builder/session.yaml` to force loading the intended session.

### What was tricky to build
- The tool output gave conflicting signals (“init saved with target origin/main” vs “show reports target main”), which is easy to miss mid-flow.

### What warrants a second pair of eyes
- Confirm whether `prescribe session show` requires an explicit “load session” flag (or whether this is a bug worth filing).

### What should be done in the future
- Update the playbook with an explicit verification step that checks `.pr-builder/session.yaml` (not just `session show`), and/or always invoke `prescribe generate -s .pr-builder/session.yaml`.

### Code review instructions
- Compare `prescribe session show` output with the contents of `.pr-builder/session.yaml` and confirm which one is used by `prescribe generate` in this environment.
