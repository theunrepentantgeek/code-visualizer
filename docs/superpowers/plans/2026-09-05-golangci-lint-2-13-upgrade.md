# golangci-lint v2.13.2 Upgrade Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade the repository's custom golangci-lint build to v2.13.2 and enable the new high-signal `iface.unusedmethod` analyzer wherever its package-local model is valid.

**Architecture:** Keep the existing custom-build pipeline and nilaway plugin unchanged while synchronizing its three version pins. Existing enabled linters automatically receive the v2.13 analyzer updates; add only the opt-in `iface.unusedmethod` analyzer, with narrow exclusions for cross-package contract interfaces that the analyzer cannot evaluate correctly.

**Tech Stack:** Go 1.26.1, golangci-lint v2.13.2, golangci-lint custom module plugins, nilaway, Task

---

### Task 1: Pin golangci-lint v2.13.2

**Files:**
- Modify: `.custom-gcl.yml:2`
- Modify: `.devcontainer/.custom-gcl.template.yml:1`
- Modify: `.devcontainer/install-dependencies.sh:150`

- [ ] **Step 1: Confirm the current pins**

Run:

```bash
rg -n 'v2\.(12\.2|13\.2)' .custom-gcl.yml .devcontainer/.custom-gcl.template.yml .devcontainer/install-dependencies.sh
```

Expected: exactly three matches, all containing `v2.12.2`.

- [ ] **Step 2: Update every pin**

Apply these exact replacements:

```yaml
# .custom-gcl.yml
version: v2.13.2
```

```yaml
# .devcontainer/.custom-gcl.template.yml
version: v2.13.2
```

```bash
# .devcontainer/install-dependencies.sh
curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b "$TOOL_DEST" v2.13.2 2>&1
```

- [ ] **Step 3: Verify the pins stay synchronized**

Run:

```bash
rg -n 'v2\.(12\.2|13\.2)' .custom-gcl.yml .devcontainer/.custom-gcl.template.yml .devcontainer/install-dependencies.sh
```

Expected: exactly three matches, all containing `v2.13.2`, and no `v2.12.2` match.

### Task 2: Enable the new iface analyzer

**Files:**
- Modify: `.golangci.yml:79-80`

- [ ] **Step 1: Create an isolated analyzer configuration**

Use `apply_patch` to create `.golangci-iface-audit.yml` with:

```yaml
version: "2"
linters:
  default: none
  enable:
    - iface
  settings:
    iface:
      enable:
        - identical
        - unusedmethod
```

- [ ] **Step 2: Install v2.13.2 and build the custom binary**

Run:

```bash
mkdir -p tools
curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b "$PWD/tools" v2.13.2
./tools/golangci-lint custom --version v2.13.2 --destination ./tools --name golangci-lint-custom -v
./tools/golangci-lint-custom version
```

Expected: the build succeeds and the version output begins with `v2.13.2-custom-gcl-`.

- [ ] **Step 3: Run the analyzer without exclusions and record its failure**

Run:

```bash
./tools/golangci-lint-custom run --config .golangci-iface-audit.yml
```

Expected: FAIL with 11 `unusedmethod` findings in `internal/canvas/model` and `internal/inks`. These are false positives caused by the analyzer's documented package-local scope: the reported interface methods are consumed from other packages.

- [ ] **Step 4: Configure the analyzer with narrow package exclusions**

Add this block at the start of `linters.settings` in `.golangci.yml`:

```yaml
    iface:
      enable:
        - identical
        - unusedmethod
      settings:
        unusedmethod:
          exclude:
            - github.com/theunrepentantgeek/code-visualizer/internal/canvas/model
            - github.com/theunrepentantgeek/code-visualizer/internal/inks
```

Including `identical` preserves the analyzer that `iface` enables by default when an explicit analyzer list is supplied. The two exclusions retain valid cross-package contract interfaces while keeping `unusedmethod` active for every other package.

- [ ] **Step 5: Verify the configuration**

Run:

```bash
./tools/golangci-lint-custom config verify
```

Expected: exit status 0 and no configuration errors.

- [ ] **Step 6: Run the scoped iface audit**

Run:

```bash
./tools/golangci-lint-custom run --enable-only iface
```

Expected: `0 issues`.

- [ ] **Step 7: Remove the temporary audit configuration**

Use `apply_patch` to delete `.golangci-iface-audit.yml`.

- [ ] **Step 8: Commit the upgrade**

```bash
git add .custom-gcl.yml .devcontainer/.custom-gcl.template.yml .devcontainer/install-dependencies.sh .golangci.yml
git commit -m "Upgrade golangci-lint to 2.13.2" -m "Enable the new iface unusedmethod analyzer outside packages whose interfaces are intentionally consumed cross-package." -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 3: Validate the complete upgrade

**Files:**
- Verify only; no planned source changes

- [ ] **Step 1: Run full CI**

Dispatch `task ci` through an Explore or equivalent subagent because repository guidance requires summarized handling of verbose lint output. Require:

- exit status 0;
- no failing tests;
- no failing linters;
- no files modified by `task tidy`.

- [ ] **Step 2: Verify the final diff**

Run:

```bash
git status --short
git --no-pager diff main...HEAD --check
git --no-pager diff main...HEAD -- .custom-gcl.yml .devcontainer/.custom-gcl.template.yml .devcontainer/install-dependencies.sh .golangci.yml
```

Expected: the worktree is clean, `diff --check` exits 0, all three pins are v2.13.2, and the only linter configuration addition enables `iface.unusedmethod` with the two documented package exclusions.
