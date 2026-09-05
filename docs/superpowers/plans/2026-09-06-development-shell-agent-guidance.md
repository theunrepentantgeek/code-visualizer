# Development Shell Agent Guidance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Tell future AI agents to use `dev.sh` when a required command-line tool is missing.

**Architecture:** Add one focused subsection to the repository's existing AI instruction file. The change documents the current development-shell behavior without changing scripts, dependencies, or CI.

**Tech Stack:** Markdown, Bash (`dev.sh`)

---

### Task 1: Document the development shell fallback

**Files:**
- Modify: `.github/copilot-instructions.md:153`

- [ ] **Step 1: Add the development shell guidance**

Insert this subsection immediately before `## Building, Testing, and Linting`:

```markdown
## Development Shell

If a command-line tool you need is missing, run `./dev.sh` and retry the command
inside the shell it starts. The first invocation installs any missing dependencies
and may take a few minutes; later invocations are fast.
```

- [ ] **Step 2: Verify the documentation diff**

Run:

```bash
git diff --check &&
git --no-pager diff -- .github/copilot-instructions.md
```

Expected: `git diff --check` reports no errors, and the diff contains only the new
`Development Shell` subsection in `.github/copilot-instructions.md`.

- [ ] **Step 3: Commit the guidance**

```bash
git add .github/copilot-instructions.md
git commit -m "docs: explain development shell fallback" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```
