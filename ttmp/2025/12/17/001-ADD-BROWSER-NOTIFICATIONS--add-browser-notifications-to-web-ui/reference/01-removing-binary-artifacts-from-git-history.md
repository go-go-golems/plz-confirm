---
Title: Removing Binary Artifacts from Git History
Slug: removing-binary-artifacts-from-git-history
Short: Step-by-step guide to finding and removing large binary files from git history to reduce repository size
Topics:
- git
- repository-management
- binary-files
- git-history
IsTemplate: false
IsTopLevel: false
ShowPerDefault: false
SectionType: Reference
---

# Removing Binary Artifacts from Git History

Binary files accidentally committed to git can bloat repository size and slow down clones. This guide provides a systematic approach to identify large files in your git history and remove them completely, even from past commits. The process involves finding the files, removing them from tracking, rewriting history, and cleaning up the repository.

## Finding Large Files in Git History

Before removing files, you need to identify which files are taking up space in your repository. Git stores all versions of all files in its object database, so a large file committed once will remain in history even if deleted later.

### Check Recent Commits for Large Files

Start by examining recent commits to see what files were added:

```bash
# Show file sizes in recent commits
git log --stat --oneline -20

# Show only commits that added large files
git log --all --pretty=format: --name-only --diff-filter=A | \
  sort -u | \
  xargs -I {} sh -c 'git log --all --pretty=format: --diff-filter=A -- {} | head -1 | xargs -I [] git cat-file -s [] 2>/dev/null | awk "{print \$1, \"{}\"}"' | \
  sort -rn | head -20
```

### Find Large Objects in Git Database

A more direct approach is to examine git's object database directly:

```bash
# List all objects sorted by size (shows object hash and size)
git rev-list --objects --all | \
  git cat-file --batch-check='%(objecttype) %(objectsize) %(rest)' | \
  awk '/^blob/ {print substr($0,6)}' | \
  sort -n -k2 | \
  tail -20
```

This command shows the 20 largest files in your repository. The output format is: `size filename`, making it easy to identify problematic files.

### Check Specific File Types

If you suspect certain file types (like images, binaries, or archives), search for them:

```bash
# Find all PNG files in git history
git log --all --full-history --oneline -- "*.png"

# Find all binary executables
git log --all --full-history --oneline -- "*.exe" "*.bin"

# Check if specific files exist in history
git log --all --full-history --oneline -- path/to/large-file.bin
```

## Removing Files from Git History

Once you've identified the files to remove, you have two options: stop tracking them going forward (keeps them in history) or remove them completely from all commits (rewrites history).

### Option 1: Stop Tracking (Keep in History)

If the files are already committed and you just want to prevent future tracking:

```bash
# Remove from git tracking but keep files on disk
git rm --cached path/to/file.bin

# Add to .gitignore to prevent future commits
echo "*.bin" >> .gitignore

# Commit the changes
git add .gitignore
git commit -m "Remove binary files from tracking"
```

**When to use:** When you don't need to rewrite history (e.g., files haven't been pushed to a shared remote, or you're okay with them remaining in history).

### Option 2: Remove from All History (Rewrite History)

To completely remove files from all commits, you need to rewrite git history. This is more complex but removes the files entirely.

**Warning:** Rewriting history changes commit hashes. Only do this if:
- You haven't pushed to a shared remote, OR
- You coordinate with your team and force-push (they'll need to re-clone)

#### Step 1: Remove Files from All Commits

Use `git filter-branch` to remove files from every commit:

```bash
# Remove specific files from all branches and tags
git filter-branch --force --index-filter \
  'git rm --cached --ignore-unmatch path/to/file1.bin path/to/file2.bin' \
  --prune-empty --tag-name-filter cat -- --all
```

**Explanation:**
- `--force`: Overwrite existing backup refs
- `--index-filter`: Modify the index (staging area) for each commit
- `git rm --cached --ignore-unmatch`: Remove files from index, ignore if they don't exist in that commit
- `--prune-empty`: Remove commits that become empty after filtering
- `--tag-name-filter cat`: Update tags to point to rewritten commits
- `-- --all`: Process all branches and tags

#### Step 2: Clean Up Backup Refs

`git filter-branch` creates backup refs. Remove them:

```bash
# Remove backup refs
rm -rf .git/refs/original/

# Expire reflog entries
git reflog expire --expire=now --all

# Run garbage collection to actually remove the objects
git gc --prune=now --aggressive
```

#### Step 3: Verify Files Are Removed

Confirm the files are gone from history:

```bash
# Should return no results
git log --all --full-history --oneline -- path/to/file.bin

# Check repository size (should be smaller)
git count-objects -vH
```

#### Step 4: Update Remote (If Applicable)

If you have a remote repository, force-push the rewritten history:

```bash
# Safer than --force (fails if remote has new commits)
git push --force-with-lease origin main

# Or for all branches
git push --force-with-lease origin --all
```

**Important:** Coordinate with your team before force-pushing. They'll need to re-clone or reset their local repositories.

## Complete Example: Removing Image Files

Here's a real-world example of removing large PNG images that were accidentally committed:

```bash
# 1. Find the large files
git rev-list --objects --all | \
  git cat-file --batch-check='%(objecttype) %(objectsize) %(rest)' | \
  awk '/^blob/ {print substr($0,6)}' | \
  sort -n -k2 | tail -5

# Output shows:
# 1772440 agent-ui-system/client/public/images/logo-placeholder.png
# 1799452 agent-ui-system/client/public/images/hero-bg.png
# 2071124 agent-ui-system/client/public/images/empty-state.png

# 2. Check if files are actually used in codebase
grep -r "logo-placeholder.png\|hero-bg.png\|empty-state.png" agent-ui-system/

# 3. Remove from all history
git filter-branch --force --index-filter \
  'git rm --cached --ignore-unmatch \
    agent-ui-system/client/public/images/empty-state.png \
    agent-ui-system/client/public/images/hero-bg.png \
    agent-ui-system/client/public/images/logo-placeholder.png' \
  --prune-empty --tag-name-filter cat -- --all

# 4. Clean up
rm -rf .git/refs/original/
git reflog expire --expire=now --all
git gc --prune=now --aggressive

# 5. Verify
git log --all --full-history --oneline -- "*.png"
# Should return no results

# 6. Check size reduction
git count-objects -vH
# Before: size-pack: ~5.6MB
# After: size-pack: 263.86 KiB
```

## Preventing Future Binary Commits

To avoid committing binaries in the future:

### Update .gitignore

Add patterns for common binary file types:

```bash
# Add to .gitignore
cat >> .gitignore << EOF

# Binary files
*.exe
*.bin
*.dll
*.so
*.dylib

# Large image files (if not needed)
*.png
*.jpg
*.jpeg
*.gif
*.webp

# Archives
*.zip
*.tar
*.tar.gz
*.rar
EOF
```

### Pre-commit Checks

Use git hooks to prevent large files:

```bash
# .git/hooks/pre-commit
#!/bin/bash
# Prevent commits of files larger than 1MB
max_size=1048576  # 1MB in bytes

while read -r file; do
    size=$(git cat-file -s ":$file" 2>/dev/null || echo 0)
    if [ "$size" -gt "$max_size" ]; then
        echo "Error: $file is larger than 1MB ($size bytes)"
        exit 1
    fi
done < <(git diff --cached --name-only)

exit 0
```

## Troubleshooting

### Filter-Branch Takes Too Long

For very large repositories, consider using `git filter-repo` instead (requires installation):

```bash
# Install git-filter-repo
pip install git-filter-repo

# Remove files (faster than filter-branch)
git filter-repo --path path/to/file.bin --invert-paths
```

### Files Still Appear After Filter-Branch

If files still appear, ensure you:
1. Removed backup refs: `rm -rf .git/refs/original/`
2. Expired reflog: `git reflog expire --expire=now --all`
3. Ran garbage collection: `git gc --prune=now --aggressive`

### Team Members Have Old History

After force-pushing, team members need to update:

```bash
# Option 1: Re-clone (safest)
cd ..
rm -rf repo-name
git clone <url> repo-name

# Option 2: Reset local (if no local changes)
git fetch origin
git reset --hard origin/main
```

## Summary

Removing binary artifacts from git history involves:
1. **Finding** large files using `git rev-list` and `git log`
2. **Removing** them with `git filter-branch` or `git filter-repo`
3. **Cleaning up** backup refs and running garbage collection
4. **Verifying** the files are gone
5. **Updating** remotes with force-push (if applicable)
6. **Preventing** future commits with `.gitignore` and hooks

Always coordinate with your team before rewriting shared history, and prefer `--force-with-lease` over `--force` when updating remotes.

