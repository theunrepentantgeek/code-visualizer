# Directory Partition Metrics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Alluvial's root-only direct-files workaround with reusable recursive directory partitioning and on-demand scoped metric evaluation.

**Architecture:** `internal/model` will define directory selections and recursively partition a source tree into full-subtree and direct-files selections. `internal/stages` will evaluate any selection into a fresh metric container using the same source-grounded logic as ordinary directory aggregation. Alluvial acquisition will eagerly materialize path-keyed band metrics so its snapshot and rendering model no longer contain source-tree directories or evaluation rules.

**Tech Stack:** Go 1.26+, Gomega, Goldie v2, Kong, go-git

**Spec:** `docs/superpowers/specs/2026-10-04-directory-partition-metrics-design.md`

## Global Constraints

- A directory partition includes every selected file exactly once.
- The target root is always partitioned; each reachable expansion recursively replaces one subtree with its child subtrees and optional direct-files remainder.
- Omit a direct-files remainder when `len(Directory.Files) == 0`; do not omit a non-empty selection merely because its chosen metric evaluates to zero.
- Direct-files remainders use the directory's repository-relative path; repository root uses `.`.
- Evaluate direct-only metrics from source nodes, never by subtracting child aggregates.
- Preserve shared scanner, provider, filtering, history, export, and unreachable-expansion behavior.
- Replace `Snapshot.Root` and `Snapshot.DirectFiles`; do not layer the new design over the existing workaround.
- Run repository commands through `./dev.sh -c '<command>'`.

## Review Focus

- An invalid or zero `DirectoryScope` returns an explicit error rather than silently selecting files.
- A nested expansion is applied only when its parent expansion makes that path reachable.
- A directory with files whose selected metric is absent or zero still produces a band; only a directory with no selected files is omitted.
- A missing or file-empty individual historical reference produces an empty band map and valid introduced/removed transitions.
- Map iteration never determines visual order; Alluvial sorts repository-relative paths before constructing columns.

---

### Task 1: Shared Directory Selections and Recursive Partitioning

**Files:**
- Create: `internal/model/directory_selection.go`
- Create: `internal/model/directory_selection_test.go`

**Interfaces:**
- Consumes: `model.Directory`, `model.File`, and existing `model.WalkFiles`.
- Produces:
  - `type DirectoryScope uint8`
  - `const DirectoryScopeInvalid`, `DirectorySubtree`, and `DirectoryDirectFiles`
  - `type DirectorySelection struct { Directory *Directory; Scope DirectoryScope }`
  - `func (s DirectorySelection) WalkFiles(fn func(*File)) error`
  - `func PartitionDirectories(root *Directory, expansions []string) map[string]DirectorySelection`

- [ ] **Step 1: Write failing scoped traversal tests**

Add:

```go
func TestDirectorySelection_WalkFiles_SubtreeIncludesDescendants(t *testing.T)
func TestDirectorySelection_WalkFiles_DirectFilesExcludesDescendants(t *testing.T)
func TestDirectorySelection_WalkFiles_RejectsInvalidScope(t *testing.T)
```

Use a root with `root.go` and a child containing `child.go`. Assert subtree traversal sees both files, direct traversal sees only `root.go`, and the zero-value scope returns an error containing `invalid directory scope`.

- [ ] **Step 2: Run scoped traversal tests and verify RED**

Run:

```bash
./dev.sh -c 'go test ./internal/model -run TestDirectorySelection_WalkFiles'
```

Expected: build failure because `DirectorySelection` and its scopes do not exist.

- [ ] **Step 3: Implement the scoped traversal interface**

Create the types and `WalkFiles` signature above. `DirectorySubtree` delegates to `WalkFiles`; `DirectoryDirectFiles` iterates only `s.Directory.Files`; nil directories and unknown scopes return explicit errors.

- [ ] **Step 4: Run scoped traversal tests and verify GREEN**

Run the command from Step 2.

Expected: PASS.

- [ ] **Step 5: Write failing recursive partition tests**

Add:

```go
func TestPartitionDirectories_PartitionsRootDirectFilesAndChildren(t *testing.T)
func TestPartitionDirectories_RecursivelyPartitionsExpandedDirectories(t *testing.T)
func TestPartitionDirectories_OmitsEmptyDirectFileSelections(t *testing.T)
func TestPartitionDirectories_IgnoresUnreachableNestedExpansion(t *testing.T)
func TestPartitionDirectories_NilAndFileEmptyRootsReturnEmpty(t *testing.T)
func TestPartitionDirectories_AssignsEveryFileExactlyOnce(t *testing.T)
```

Build a tree with paths `.`, `internal`, `internal/api`, and
`internal/api/private`, with direct files at multiple levels. Assert:

- the default map contains `.` as `DirectoryDirectFiles` plus `internal` as
  `DirectorySubtree`;
- expanding `internal` and `internal/api` produces direct selections for those
  paths only when they have direct files, plus child subtree selections;
- requesting only `internal/api` leaves `internal` collapsed;
- collecting files across every returned selection yields every source file
  exactly once.

- [ ] **Step 6: Run partition tests and verify RED**

Run:

```bash
./dev.sh -c 'go test ./internal/model -run TestPartitionDirectories'
```

Expected: build failure because `PartitionDirectories` does not exist.

- [ ] **Step 7: Implement recursive partitioning**

Implement the exact signature above. Normalize expansion paths with
`path.Clean`, key the result by `Directory.RepoPath`, partition the root
implicitly, and recursively replace only currently reachable subtree
selections. Use a map to enforce one selection per path; do not rely on map
iteration order.

- [ ] **Step 8: Run all model tests**

Run:

```bash
./dev.sh -c 'go test ./internal/model'
```

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/model/directory_selection.go internal/model/directory_selection_test.go
git commit -m "feat(model): add directory partition selections" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 2: Shared Scoped Metric Evaluation

**Files:**
- Modify: `internal/stages/aggregation.go`
- Modify: `internal/stages/aggregation_test.go`

**Interfaces:**
- Consumes:
  - `model.DirectorySelection`
  - `func (s model.DirectorySelection) WalkFiles(func(*model.File)) error`
  - `[]provider.ResolvedMetric`
- Produces:
  - `func EvaluateAggregations(selection model.DirectorySelection, expressions []provider.ResolvedMetric) (*model.MetricContainer, error)`
  - Existing `func ComputeAggregations(root *model.Directory, expressions []provider.ResolvedMetric) error` preserved with unchanged behavior.

- [ ] **Step 1: Write failing file-level scope tests**

Add:

```go
func TestEvaluateAggregations_DirectFilesExcludesDescendantValues(t *testing.T)
func TestEvaluateAggregations_PreservesNonAdditiveMeanAndDistinct(t *testing.T)
func TestEvaluateAggregations_NonEmptyZeroMetricSelectionIsRetained(t *testing.T)
```

Use a directory with direct file sizes `100` and `300`, plus a child file size
`900`, and direct/child file classifications that differ. Assert direct scope
has sum `400`, mean `200`, and only the direct classifications, while subtree
scope includes the child. For the zero case, assert evaluation succeeds and
stores a zero result rather than treating the selection as empty.

- [ ] **Step 2: Run file-level evaluator tests and verify RED**

Run:

```bash
./dev.sh -c 'go test ./internal/stages -run TestEvaluateAggregations'
```

Expected: build failure because `EvaluateAggregations` does not exist.

- [ ] **Step 3: Introduce the shared evaluator and refactor file-level aggregation**

Implement the exact exported signature above. Add one unexported evaluator that
writes to a target supporting `SetQuantity`, `SetMeasure`, and
`SetClassification`. Refactor numeric and classification collectors to obtain
files through `DirectorySelection.WalkFiles`. Keep result names and kind
handling unchanged.

- [ ] **Step 4: Run file-level evaluator tests and verify GREEN**

Run the command from Step 2.

Expected: PASS.

- [ ] **Step 5: Write failing declaration-, commit-, and validation tests**

Add:

```go
func TestEvaluateAggregations_DirectFilesLimitsDeclarationValues(t *testing.T)
func TestEvaluateAggregations_DirectFilesLimitsCommitValues(t *testing.T)
func TestEvaluateAggregations_RejectsInvalidScope(t *testing.T)
func TestComputeAggregations_StillPopulatesEveryDirectoryFromSubtrees(t *testing.T)
```

Construct direct and child files with distinct declaration and commit values.
Assert direct scope excludes the child values, invalid scope returns an error,
and `ComputeAggregations` still writes full-subtree values to the root and each
child.

- [ ] **Step 6: Run the new tests and verify RED**

Run:

```bash
./dev.sh -c 'go test ./internal/stages -run "Test(EvaluateAggregations|ComputeAggregations_Still)"'
```

Expected: declaration/commit scope assertions fail until their collectors use
the shared scoped traversal.

- [ ] **Step 7: Refactor all source levels onto scoped evaluation**

Make declaration and commit collectors traverse only files selected by
`DirectorySelection`. Preserve per-file declaration aggregation performed by
`ComputeAggregations`. Refactor `ComputeAggregations` to evaluate each real
directory with `DirectorySubtree`, so ordinary directory metrics and on-demand
metrics share one implementation.

- [ ] **Step 8: Run aggregation and provider tests**

Run:

```bash
./dev.sh -c 'go test ./internal/stages ./internal/provider/...'
```

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/stages/aggregation.go internal/stages/aggregation_test.go
git commit -m "refactor(metric): evaluate scoped directory selections" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 3: Eagerly Materialize Alluvial Snapshot Bands

**Files:**
- Modify: `internal/alluvial/state.go`
- Modify: `internal/alluvial/pipeline.go`
- Modify: `internal/alluvial/acquisition_plan_test.go`
- Modify: `internal/alluvial/data.go`
- Modify: `internal/alluvial/data_test.go`

**Interfaces:**
- Consumes:
  - `model.PartitionDirectories(root, expansions)`
  - `stages.EvaluateAggregations(selection, expressions)`
- Produces:
  - `type Snapshot struct { Reference string; Bands map[string]*model.MetricContainer }`
  - `func evaluateBands(root *model.Directory, expansions []string, expressions []provider.ResolvedMetric) (map[string]*model.MetricContainer, error)` as an unexported acquisition helper.
  - `AcquisitionPlan` retains a cloned expansion list prepared from `cfg.Expand`.
  - `Options` no longer contains `Expand`.

- [ ] **Step 1: Rewrite data tests against evaluated snapshot bands**

Replace test snapshots containing `Root`/`DirectFiles` with a helper:

```go
func testSnapshot(reference string, bands map[string]testBandMetrics) alluvial.Snapshot
```

The helper creates one `model.MetricContainer` per path and sets the requested
width/fill metrics. Preserve assertions for reference order, transitions,
temporal fills, constant bands, and all-references-empty errors. Add:

```go
func TestBuildData_SortsEvaluatedBandPaths(t *testing.T)
func TestBuildData_IncludesNonEmptyBandWithZeroWidth(t *testing.T)
```

- [ ] **Step 2: Run data tests and verify RED**

Run:

```bash
./dev.sh -c 'go test ./internal/alluvial -run "Test(BuildData|Data_)"'
```

Expected: build failures because `Snapshot.Bands` does not exist and
`Root`/`DirectFiles` still define the interface.

- [ ] **Step 3: Replace the snapshot and data interfaces**

Change `Snapshot` to the exact interface above. Update `BuildData` to:

- reject only when every snapshot band map is empty;
- sort map keys before building each `Column`;
- read width and fill values from each metric container;
- preserve zero-valued bands;
- remove `selectedDirectories`, `expandSelectedDirectory`, and all
  `Root`/`DirectFiles` handling;
- remove `Expand` from `Options` and `BuildDataStage`.

- [ ] **Step 4: Run data tests and verify GREEN**

Run:

```bash
./dev.sh -c 'go test ./internal/alluvial -run "Test(BuildData|Data_)"'
```

Expected: PASS.

- [ ] **Step 5: Write failing acquisition tests**

Add acquisition-level tests for the unexported behavior through
`PrepareReferences`/`AcquireReference` or the closest existing package seam:

```go
func TestAcquisitionPlan_MaterializesRootAndChildBands(t *testing.T)
func TestAcquisitionPlan_MaterializesRecursiveExpansionRemainders(t *testing.T)
func TestAcquisitionPlan_OmitsEmptyDirectFileRemainder(t *testing.T)
func TestAcquisitionPlan_StoresNoSourceDirectoriesInSnapshot(t *testing.T)
```

Assert evaluated maps contain the expected repository-relative paths and
metrics, including direct remainders at multiple expanded levels.

- [ ] **Step 6: Run acquisition tests and verify RED**

Run:

```bash
./dev.sh -c 'go test ./internal/alluvial -run TestAcquisitionPlan'
```

Expected: failures because acquisition still creates `DirectFiles` and does not
use the shared recursive partition.

- [ ] **Step 7: Materialize partitions during acquisition**

Clone `cfg.Expand` into `AcquisitionPlan` during `PrepareReferences`. After
`finishSnapshot`, call `evaluateBands` with the snapshot root, expansion list,
and `snapshotCommon.Requested.Expressions`; assign the result to
`Snapshot.Bands`. Remove `directFilesAggregate` and its synthetic directory.
Keep first-snapshot export on `snapshotCommon.Root`.

- [ ] **Step 8: Run all Alluvial tests**

Run:

```bash
./dev.sh -c 'go test ./internal/alluvial'
```

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/alluvial/state.go internal/alluvial/pipeline.go \
  internal/alluvial/acquisition_plan_test.go internal/alluvial/data.go \
  internal/alluvial/data_test.go
git commit -m "refactor(alluvial): materialize partition bands" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 4: Verify Recursive Expansion Through the CLI

**Files:**
- Modify: `cmd/codeviz/alluvial_cmd_test.go`
- Modify: `docs/content/docs/visualizations/alluvial.md`

**Interfaces:**
- Consumes: evaluated Alluvial snapshot bands from Task 3.
- Produces: user-visible root and recursively expanded direct-files remainders with documented semantics.

- [ ] **Step 1: Add failing root and nested expansion CLI tests**

Extend `createAlluvialSubdirectoryFixture` with:

- a repository-root file;
- a direct file under `module`;
- a direct file under `module/child`;
- a file under `module/child/nested`.

Add:

```go
func TestAlluvialCmd_Run_RendersRepositoryRootDirectFilesBand(t *testing.T)
func TestAlluvialCmd_Run_RendersRecursiveExpansionRemainders(t *testing.T)
func TestAlluvialCmd_Run_OmitsRemainderForExpandedDirectoryWithoutDirectFiles(t *testing.T)
```

For the recursive case, pass `Expand: []string{"module", "module/child"}` and
assert SVG labels contain `.`, `module`, `module/child`, and
`module/child/nested`. Add an empty-direct-files directory to the fixture and
assert its children render when expanded but its own path does not.
Retain `TestAlluvialCmd_Run_RendersTargetIntroducedAfterFirstReference` to pin
the empty-individual-reference behavior from the Review Focus section.

- [ ] **Step 2: Run CLI tests and verify RED**

Run:

```bash
./dev.sh -c 'go test ./cmd/codeviz -run "TestAlluvialCmd_Run_Renders(RepositoryRoot|Recursive)|TestAlluvialCmd_Run_OmitsRemainder"'
```

Expected: failures until acquisition applies recursive shared partitioning to
the command's expansion list.

- [ ] **Step 3: Complete CLI wiring and documentation**

Fix only command/acquisition wiring exposed by the failing tests. Update the
Alluvial documentation to replace “never adds an `Other` aggregate” with the
approved semantics: expansion replaces a directory with child subtrees plus an
optional direct-files remainder labelled by the directory path.

- [ ] **Step 4: Run CLI and documentation-adjacent tests**

Run:

```bash
./dev.sh -c 'go test ./cmd/codeviz ./internal/alluvial'
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/codeviz/alluvial_cmd_test.go \
  docs/content/docs/visualizations/alluvial.md
git commit -m "test(alluvial): cover recursive directory remainders" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 5: Final Verification and PR Update

**Files:**
- Verify only; modify files only to fix failures caused by Tasks 1-4.

**Interfaces:**
- Consumes: completed shared partitioning, scoped evaluation, and Alluvial migration.
- Produces: a verified branch and updated PR.

- [ ] **Step 1: Format and run focused suites**

Run:

```bash
./dev.sh -c 'gofumpt -w internal/model/directory_selection.go internal/model/directory_selection_test.go internal/stages/aggregation.go internal/stages/aggregation_test.go internal/alluvial/state.go internal/alluvial/pipeline.go internal/alluvial/acquisition_plan_test.go internal/alluvial/data.go internal/alluvial/data_test.go cmd/codeviz/alluvial_cmd_test.go && go test ./internal/model ./internal/stages ./internal/provider/... ./internal/alluvial ./cmd/codeviz'
```

Expected: PASS with no formatting diff left behind beyond the intended files.

- [ ] **Step 2: Run repository CI through a task agent**

Delegate exactly:

```bash
./dev.sh -c 'task ci'
```

The agent must return only exit status, failing tests/linters, offending
file:line messages, or a one-line success note. If `task tidy` removes
pre-existing valid suppressions as previously observed, restore only those
unrelated generated changes and separately run `task build && task test` plus
`task lint`.

- [ ] **Step 3: Inspect the final branch**

Run:

```bash
git diff --check
git status --short
git log --oneline origin/main..HEAD
```

Expected: no whitespace errors; only intentional changes or explicitly
unrelated pre-existing sample artifacts remain uncommitted.

- [ ] **Step 4: Request final code review**

Review the full branch against
`docs/superpowers/specs/2026-10-04-directory-partition-metrics-design.md`,
focusing on exact file partitioning, non-additive metric correctness, missing
historical targets, and removal of the synthetic directory workaround. Fix all
Critical and Important findings.

- [ ] **Step 5: Push and update PR #771**

Push all implementation commits to the existing upstream branch. Update the PR
description if necessary to explain recursive remainders and shared scoped
metric evaluation.
