# Filter Files by Git Range Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an opt-in `changedOnly` mode that renders only current-tree files modified within the effective Git range.

**Architecture:** Add one Git-provider query that reuses `HistoryRange`, then apply its result in a shared post-scan pipeline stage before metric providers run. Expose the setting through the root config and every command, preserving the current default and existing range syntax.

**Tech Stack:** Go 1.26, Kong, go-git, Gomega, YAML/JSON configuration, Taskfile

---

## File Structure

- `internal/provider/git/changed_paths.go`: discover current tracked paths modified in a history range.
- `internal/provider/git/changed_paths_test.go`: provider range, rename, deletion, and empty-selection coverage.
- `internal/stages/changed_only.go`: validate and apply changed-only tree pruning.
- `internal/stages/changed_only_test.go`: stage no-op, pruning, error, and filter-intersection coverage.
- `internal/config/config.go`, `internal/config/config_test.go`: root `changedOnly` configuration and round trips.
- `cmd/codeviz/*_cmd.go`, `cmd/codeviz/main.go`: CLI switch and shared stage flags.
- `cmd/codeviz/main_test.go`, `cmd/codeviz/render_cmd_test.go`: command and preset propagation tests.
- `internal/{treemap,radialtree,donuttree,bubbletree,spiral,scatter}/pipeline.go`: run the shared filter after scanning and before providers.
- `docs/content/docs/visualizations/*.md`: document the switch consistently.

### Task 1: Provider-Level Modified Path Discovery

**Files:**
- Create: `internal/provider/git/changed_paths.go`
- Create: `internal/provider/git/changed_paths_test.go`

- [ ] **Step 1: Write failing provider tests**

Create repository fixtures with commits that add, modify, rename, and delete
files. Assert that `ChangedPathsInHistoryRange(repoPath, currentPaths, range)`
returns only current path names touched by selected commits. Cover a date range,
`tag:` bounds, rename destination behavior, deletion exclusion, and an empty
range.

- [ ] **Step 2: Run tests and verify failure**

Run:

```bash
go test ./internal/provider/git -run 'TestChangedPathsInHistoryRange' -count=1
```

Expected: compilation fails because `ChangedPathsInHistoryRange` is undefined.

- [ ] **Step 3: Implement the provider API**

Implement:

```go
func ChangedPathsInHistoryRange(
    repoPath string,
    currentPaths map[string]bool,
    historyRange HistoryRange,
) (map[string]bool, error)
```

Normalize path separators and reuse `walkTrackedHistoryInHistoryRange`. Add each
reported `trackedChange.path` to the result set. Do not invent separate range,
rename, or merge semantics.

- [ ] **Step 4: Run provider tests**

Run:

```bash
go test ./internal/provider/git -run 'TestChangedPathsInHistoryRange' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

Commit the provider API and tests with message:

```text
Add Git range changed-path discovery
```

### Task 2: Configuration and CLI Surface

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Modify: `internal/stages/common.go`
- Modify: `cmd/codeviz/main.go`
- Modify: `cmd/codeviz/treemap_cmd.go`
- Modify: `cmd/codeviz/radialtree_cmd.go`
- Modify: `cmd/codeviz/donuttree_cmd.go`
- Modify: `cmd/codeviz/bubbletree_cmd.go`
- Modify: `cmd/codeviz/spiral_cmd.go`
- Modify: `cmd/codeviz/scatter_cmd.go`
- Modify: `cmd/codeviz/render_cmd.go`
- Modify: `cmd/codeviz/main_test.go`
- Modify: `cmd/codeviz/render_cmd_test.go`

- [ ] **Step 1: Write failing config and CLI tests**

Add YAML and JSON tests for:

```yaml
changedOnly: true
```

Add parsing coverage proving every visualization and `render` accepts
`--changed-only`. Add preset-construction assertions proving `RenderCmd` forwards
the switch.

- [ ] **Step 2: Run tests and verify failure**

Run:

```bash
go test ./internal/config ./cmd/codeviz -run 'ChangedOnly|changed.only' -count=1
```

Expected: failures because the config field and CLI switch are absent.

- [ ] **Step 3: Implement configuration and CLI wiring**

Add:

```go
ChangedOnly *bool `yaml:"changedOnly,omitempty" json:"changedOnly,omitempty"`
```

to `config.Config`, with a helper that treats nil as false and an override that
sets true only when the CLI switch is present. Include it in `ForExport`.

Add this field to every command:

```go
ChangedOnly bool `help:"Show only files modified in the selected Git range." name:"changed-only" optional:""`
```

Apply the override before validation, carry the effective value in
`stages.Flags`, and forward it through every preset constructor.

- [ ] **Step 4: Run config and CLI tests**

Run:

```bash
go test ./internal/config ./cmd/codeviz -run 'ChangedOnly|changed.only' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

Commit with message:

```text
Expose changed-only filtering in config and CLI
```

### Task 3: Shared Tree Filter Stage

**Files:**
- Create: `internal/stages/changed_only.go`
- Create: `internal/stages/changed_only_test.go`
- Modify: `internal/stages/errors.go`
- Modify: `internal/treemap/pipeline.go`
- Modify: `internal/radialtree/pipeline.go`
- Modify: `internal/donuttree/pipeline.go`
- Modify: `internal/bubbletree/pipeline.go`
- Modify: `internal/spiral/pipeline.go`
- Modify: `internal/scatter/pipeline.go`

- [ ] **Step 1: Write failing stage tests**

Construct model trees and temporary Git repositories to assert:

- disabled mode is a no-op without Git;
- enabled mode rejects an unconstrained range;
- enabled mode requires Git;
- unmatched files and recursively empty directories are removed;
- files absent after include/exclude or binary scan filtering cannot reappear;
- an empty result returns `*NoFilesAfterFilterError` with a changed-only message.

- [ ] **Step 2: Run tests and verify failure**

Run:

```bash
go test ./internal/stages -run 'TestFilterChangedOnly' -count=1
```

Expected: compilation fails because `FilterChangedOnly` is undefined.

- [ ] **Step 3: Implement tree pruning**

Implement `FilterChangedOnly(*CommonState) error`. Return immediately when the
effective setting is false. Validate that `HistoryRange.From` or
`HistoryRange.Until` is non-empty, resolve the repository root, build the
current-tree path set, call `git.ChangedPathsInHistoryRange`, and recursively
filter `Directory.Files` and `Directory.Dirs`. Update directory count metadata
to remain consistent with the pruned tree.

Return `NoFilesAfterFilterError` with:

```text
no files available for visualization after filtering to files changed in the selected Git range
```

when no files remain.

- [ ] **Step 4: Wire all pipelines**

Call `stages.FilterChangedOnly` immediately after `stages.ScanFilesystem` and
before `stages.CheckGitRequirement` or `stages.LoadGitHistory`. This preserves
scan-time include/exclude and binary ordering while reducing provider work.

- [ ] **Step 5: Run stage and visualization pipeline tests**

Run:

```bash
go test ./internal/stages ./internal/treemap ./internal/radialtree ./internal/donuttree ./internal/bubbletree ./internal/spiral ./internal/scatter -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

Commit with message:

```text
Filter visualization trees to changed files
```

### Task 4: Documentation and Manual Verification

**Files:**
- Modify: `docs/content/docs/visualizations/tree-map.md`
- Modify: `docs/content/docs/visualizations/radial-tree.md`
- Modify: `docs/content/docs/visualizations/donut-tree.md`
- Modify: `docs/content/docs/visualizations/bubble-tree.md`
- Modify: `docs/content/docs/visualizations/spiral.md`
- Modify: `docs/content/docs/visualizations/scatter.md`

- [ ] **Step 1: Document the option**

Add `--changed-only` to each command table and explain that it requires at least
one `--from` or `--until` bound, filters the current tree after existing scan
filters, excludes deleted files, and uses current names for renames.

- [ ] **Step 2: Run targeted tests**

Run:

```bash
go test ./internal/provider/git ./internal/stages ./internal/config ./cmd/codeviz -count=1
```

Expected: PASS.

- [ ] **Step 3: Manually exercise the CLI**

Build the command, run a normal visualization and a `--changed-only` visualization
against a temporary Git fixture, export their computed data, and verify the
changed-only output contains only the expected current-tree file. Also verify
unconstrained use fails with the expected message and exit classification.

- [ ] **Step 4: Commit**

Commit with message:

```text
Document changed-only range filtering
```

### Task 5: Full Verification

**Files:**
- Review all modified files.

- [ ] **Step 1: Format and inspect**

Run `gofumpt` on changed Go files, `git diff --check`, and inspect
`git diff --stat` plus the complete diff. Confirm `.custom-gcl.yml` retains the
pre-existing user modification and is not committed.

- [ ] **Step 2: Run repository CI**

Run `task ci` through the required low-noise task subagent.

Expected: build, all tests, lint, and no-change verification pass.

- [ ] **Step 3: Scan changed files for secrets**

Run the repository secret scanner over every modified or created file.

Expected: no secrets.

- [ ] **Step 4: Run CodeQL**

Run CodeQL with a non-trivial change declaration. Investigate and fix any
change-related alert, then rerun it.

- [ ] **Step 5: Request code review**

Run the code-review workflow against the completed diff. Fix any high-confidence
finding and repeat targeted and full verification as needed.

- [ ] **Step 6: Final commit and progress update**

Commit and push all verified changes with the progress reporter, marking every
checklist item complete.
