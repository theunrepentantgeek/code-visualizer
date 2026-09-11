# Historical Filesystem Snapshot Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `--until` render the exact committed filesystem snapshot at the selected endpoint while preserving live-working-tree behavior when `--until` is omitted.

**Architecture:** Add a project-local `io/fs` adapter over go-git v5 `object.Tree`, then represent every scan input as a `source.Tree` with separate logical and repository-relative paths. Resolve an optional snapshot before scanning; content providers read through the source while Git providers use explicit repository-relative identities and a snapshot reference clock.

**Tech Stack:** Go 1.26.1, `io/fs`, go-git v5, Gomega, `testing/fstest`, existing pipeline/provider framework

---

## File Structure

- Create `internal/source/tree.go`: common read-only content-source contract and OS-backed implementation.
- Create `internal/source/gitfs.go`: `io/fs` adapter over a go-git `object.Tree`.
- Create `internal/source/gitfs_file.go`: file, directory, `FileInfo`, and `DirEntry` implementations.
- Create `internal/source/gitfs_test.go`: `fstest.TestFS` and Git-mode/error tests.
- Create `internal/provider/git/snapshot.go`: resolve `--until` to a commit and scoped Git tree.
- Create `internal/provider/git/snapshot_test.go`: revision/date/subtree resolution tests.
- Modify `internal/model/file.go` and `internal/model/directory.go`: attach content source and repository-relative identity without changing display paths.
- Modify `internal/scan/scanner.go`, `internal/scan/walker.go`, `internal/scan/node_builder.go`, and `internal/scan/filter_policy.go`: scan `fs.FS`, safely resolve symlinks, and populate source-aware model nodes.
- Modify `internal/stages/common.go` and `internal/stages/scan.go`: select working-tree or snapshot source before scanning.
- Modify `internal/provider/filesystem/metrics.go`: read file contents through model source handles.
- Modify `internal/provider/golang/analysis.go`, `internal/provider/golang/file_stats.go`, `internal/provider/golang/declarations.go`, and `internal/provider/golang/imports.go`: parse source bytes and locate `go.mod` through the source.
- Modify `internal/provider/git/metrics.go`, `internal/provider/git/loader.go`, `internal/provider/git/authorship_loader.go`, `internal/stages/git_history.go`, and `internal/stages/author_history.go`: use explicit repository roots and identities rather than deriving them from physical paths.
- Modify `internal/provider/git/changed_paths.go` and `internal/stages/changed_only.go`: intersect changed paths with snapshot identities without consulting the current index.
- Modify `internal/provider/git/service.go` and `internal/provider/git/git_provider.go`: calculate age/freshness/density from an injected reference time.
- Modify all visualization `pipeline.go` files: replace `ScanFilesystem` with the source-resolving scan stage.
- Modify `docs/content/docs/usage.md` and visualization docs: document snapshot semantics.

### Task 1: Implement the Git tree `fs.FS` adapter

**Files:**
- Create: `internal/source/gitfs.go`
- Create: `internal/source/gitfs_file.go`
- Create: `internal/source/gitfs_test.go`

- [ ] **Step 1: Write failing conformance and mode tests**

Create an in-memory go-git repository fixture with regular, executable, nested,
binary, symlink, and gitlink entries. Assert:

```go
func TestGitFSConforms(t *testing.T) {
	t.Parallel()
	fsys := newFixtureGitFS(t)
	g := NewWithT(t)
	g.Expect(fstest.TestFS(fsys,
		"README.md",
		"cmd/tool/main.go",
		"script.sh",
		"link",
	)).To(Succeed())
}

func TestGitFSExposesSymlinkAndSkipsGitlink(t *testing.T) {
	t.Parallel()
	fsys := newFixtureGitFS(t)
	g := NewWithT(t)

	target, err := fs.ReadLink(fsys, "link")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(target).To(Equal("README.md"))

	_, err = fs.Stat(fsys, "vendor/module")
	g.Expect(err).To(MatchError(fs.ErrNotExist))
}
```

- [ ] **Step 2: Run the adapter tests and verify failure**

Run:

```bash
go test ./internal/source -run 'TestGitFS' -count=1
```

Expected: FAIL because `internal/source` and the Git tree adapter do not exist.

- [ ] **Step 3: Implement the adapter**

Define:

```go
func NewGitFS(tree *object.Tree, modTime time.Time) fs.FS
```

The concrete type must implement `fs.FS`, `fs.ReadDirFS`, `fs.ReadFileFS`,
`fs.StatFS`, and `fs.ReadLinkFS`. Use slash-separated `fs.ValidPath` names,
map `filemode.Executable` to executable permission bits, return symlink target
blob text from `ReadLink`, and omit `filemode.Submodule` entries from directory
results.

Every returned error must be wrapped in `*fs.PathError` with the operation and
requested path. Directory handles implement `fs.ReadDirFile`; regular file
handles wrap the blob reader and reject `ReadDir`.

- [ ] **Step 4: Run adapter tests**

Run:

```bash
go test ./internal/source -count=1
```

Expected: PASS, including `fstest.TestFS`.

- [ ] **Step 5: Commit**

```bash
git add internal/source/gitfs.go internal/source/gitfs_file.go internal/source/gitfs_test.go
git commit -m "Add Git tree filesystem adapter" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 2: Define source-aware model identities

**Files:**
- Create: `internal/source/tree.go`
- Create: `internal/source/tree_test.go`
- Modify: `internal/model/file.go`
- Modify: `internal/model/directory.go`
- Modify: `internal/model/file_test.go`
- Modify: `internal/model/directory_test.go`

- [ ] **Step 1: Write failing source identity tests**

Cover OS-backed reads and model identity:

```go
func TestFileOpenUsesAttachedSource(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fsys := fstest.MapFS{"src/main.go": {Data: []byte("package main\n")}}
	file := model.File{Path: "src/main.go", RepoPath: "project/src/main.go", Source: fsys}

	data, err := file.ReadAll()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(data).To(Equal([]byte("package main\n")))
}
```

- [ ] **Step 2: Run tests and verify failure**

Run:

```bash
go test ./internal/source ./internal/model -run 'Test(FileOpen|OSTree)' -count=1
```

Expected: FAIL because the source contract and model fields do not exist.

- [ ] **Step 3: Implement the source contract**

Add:

```go
type Tree struct {
	FS       fs.FS
	RootName string
	RepoRoot string
	RepoBase string
	Clock    time.Time
}

func WorkingTree(path string) (Tree, error)
func (t Tree) RepoPath(name string) string
```

Model nodes store logical slash paths, `RepoPath`, and the source `fs.FS`.
Provide `(*model.File).Open()` and `(*model.File).ReadAll()` helpers so
providers never branch on backing-store type.

- [ ] **Step 4: Run source and model tests**

Run:

```bash
go test ./internal/source ./internal/model -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/source/tree.go internal/source/tree_test.go internal/model
git commit -m "Add source-aware model identities" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 3: Resolve historical snapshot commits

**Files:**
- Create: `internal/provider/git/snapshot.go`
- Create: `internal/provider/git/snapshot_test.go`
- Modify: `internal/provider/git/history_reference.go`
- Modify: `internal/provider/git/history_reference_test.go`

- [ ] **Step 1: Write failing snapshot-resolution tests**

Exercise:

```go
snapshot, err := ResolveSnapshot(repoDir, "date:2025-03-01")
g.Expect(err).NotTo(HaveOccurred())
g.Expect(snapshot.Commit.Author.When).To(Equal(latestOnMarchFirst))

snapshot, err = ResolveSnapshot(repoDir, "tag:v1.0")
g.Expect(err).NotTo(HaveOccurred())
g.Expect(snapshot.Commit.Hash.String()).To(Equal(v1Hash))
```

Add fixtures with commits on parallel branches and equal author timestamps.
Assert date-only normalization includes commits through the final nanosecond,
the latest timestamp wins across the reachable graph, equal timestamps use
lexicographically smallest hash, and no matching commit returns a clear error.

- [ ] **Step 2: Run tests and verify failure**

Run:

```bash
go test ./internal/provider/git -run 'TestResolveSnapshot|TestParseHistoryDate' -count=1
```

Expected: FAIL because `ResolveSnapshot` does not exist and upper date-only
bounds currently normalize to second precision.

- [ ] **Step 3: Implement resolution**

Export:

```go
type Snapshot struct {
	RepoRoot string
	Commit   *object.Commit
	Tree     *object.Tree
}

func ResolveSnapshot(repoPath, until string) (Snapshot, error)
func (s Snapshot) Subtree(repoRelativeDir string) (*object.Tree, error)
```

Reuse `resolveHistoryReference`. For revisions, load that commit. For dates,
walk from `HEAD`, select commits whose author time is not after the bound, and
choose deterministically by timestamp then hash. Preserve existing history
iteration semantics: date-filtered history still walks all matching reachable
commits rather than being retipped to the selected snapshot.

- [ ] **Step 4: Run resolution tests**

Run:

```bash
go test ./internal/provider/git -run 'TestResolveSnapshot|TestParseHistoryDate|TestHistoryRange' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/provider/git/snapshot.go internal/provider/git/snapshot_test.go internal/provider/git/history_reference.go internal/provider/git/history_reference_test.go
git commit -m "Resolve historical filesystem snapshots" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 4: Refactor scanning onto `io/fs`

**Files:**
- Modify: `internal/scan/scanner.go`
- Modify: `internal/scan/walker.go`
- Modify: `internal/scan/node_builder.go`
- Modify: `internal/scan/filter_policy.go`
- Modify: `internal/scan/scanner_test.go`
- Modify: `internal/scan/collaborators_test.go`
- Modify: `internal/scan/scanner_unix_test.go`

- [ ] **Step 1: Write failing source-backed scanner tests**

Add tests that scan `fstest.MapFS` and `source.NewGitFS` and assert identical
model paths, sizes, file types, include/exclude behavior, and binary filtering.
Add symlink tests proving safe internal regular-file links are followed while
directory, broken, cyclic, and escaping links are skipped.

- [ ] **Step 2: Run scanner tests and verify failure**

Run:

```bash
go test ./internal/scan -run 'TestScan(Source|Git|Symlink)' -count=1
```

Expected: FAIL because `Scan` accepts only an OS path.

- [ ] **Step 3: Implement source-backed scanning**

Change the public entry point to:

```go
func Scan(tree source.Tree, rules []filter.Rule, progress Progress, includeBinary bool) (*model.Directory, error)
```

Replace `os.ReadDir`, `os.Stat`, and absolute-path filtering with `fs.ReadDir`,
`fs.Stat`, `fs.ReadLink`, and slash-relative paths. Keep `ScanPath` as a narrow
compatibility helper for tests and direct provider fixtures:

```go
func ScanPath(path string, rules []filter.Rule, progress Progress, includeBinary bool) (*model.Directory, error)
```

Resolve symlinks component-by-component with a visited-path set and fixed
maximum depth. Reject absolute targets and any clean target that escapes with
`..`. Preserve existing logging and no-files errors.

- [ ] **Step 4: Run scanner tests**

Run:

```bash
go test ./internal/scan -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/scan
git commit -m "Scan working and Git tree sources" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 5: Select the source in the shared pipeline

**Files:**
- Modify: `internal/stages/common.go`
- Modify: `internal/stages/scan.go`
- Create: `internal/stages/source.go`
- Create: `internal/stages/source_test.go`
- Modify: `internal/bubbletree/pipeline.go`
- Modify: `internal/donuttree/pipeline.go`
- Modify: `internal/radialtree/pipeline.go`
- Modify: `internal/scatter/pipeline.go`
- Modify: `internal/spiral/pipeline.go`
- Modify: `internal/treemap/pipeline.go`

- [ ] **Step 1: Write failing source-selection tests**

Create a repository where `old.go` exists at `--until`, `new.go` exists only in
the working tree, and `old.go` has changed contents. Assert the shared stage
selects historical bytes with `--until` and live bytes without it.

- [ ] **Step 2: Run tests and verify failure**

Run:

```bash
go test ./internal/stages -run 'TestResolveSource|TestScanSource' -count=1
```

Expected: FAIL because `CommonState` has no source or snapshot fields.

- [ ] **Step 3: Implement source selection**

Add to `CommonState`:

```go
Source       source.Tree
Snapshot     *git.Snapshot
RepoRoot     string
ReferenceNow time.Time
```

Implement `ResolveSource(c *CommonState) error`. It builds an OS source when
`HistoryRange.Until` is blank; otherwise it resolves a snapshot, scopes it to
the repository-relative target directory, and builds a Git source. Make
`ScanFilesystem` consume `c.Source`.

Insert `ResolveSource` immediately before `ScanFilesystem` in every
visualization pipeline.

- [ ] **Step 4: Run stage and visualization pipeline tests**

Run:

```bash
go test ./internal/stages ./internal/bubbletree ./internal/donuttree ./internal/radialtree ./internal/scatter ./internal/spiral ./internal/treemap -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/stages internal/bubbletree/pipeline.go internal/donuttree/pipeline.go internal/radialtree/pipeline.go internal/scatter/pipeline.go internal/spiral/pipeline.go internal/treemap/pipeline.go
git commit -m "Select historical sources before scanning" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 6: Make content providers source-aware

**Files:**
- Modify: `internal/provider/filesystem/metrics.go`
- Modify: `internal/provider/filesystem/metrics_test.go`
- Modify: `internal/provider/golang/analysis.go`
- Modify: `internal/provider/golang/file_loader.go`
- Modify: `internal/provider/golang/file_stats.go`
- Modify: `internal/provider/golang/declarations.go`
- Modify: `internal/provider/golang/imports.go`
- Modify: corresponding tests under `internal/provider/golang`

- [ ] **Step 1: Write failing virtual-content tests**

Use a `fstest.MapFS` model whose logical paths do not exist on disk. Assert
line count, declarations, imports, comment ratio, complexity, and module-path
classification are populated from source bytes.

- [ ] **Step 2: Run tests and verify failure**

Run:

```bash
go test ./internal/provider/filesystem ./internal/provider/golang -run 'Test.*Source' -count=1
```

Expected: FAIL because providers call `os.Open` and `os.ReadFile`.

- [ ] **Step 3: Refactor providers to consume model file readers**

Change line counting and Go parsing helpers to accept `io.Reader`, `[]byte`, or
`*model.File`. Cache keys must combine source identity and logical path rather
than physical path alone. Replace upward OS `go.mod` discovery with source-root
lookups:

```go
func findModulePath(fsys fs.FS, start string) string
func analyzeFile(name string, src []byte, modulePath string) (*fileStats, error)
```

Keep exported provider registration unchanged.

- [ ] **Step 4: Run provider tests**

Run:

```bash
go test ./internal/provider/filesystem ./internal/provider/golang -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/provider/filesystem internal/provider/golang
git commit -m "Read content metrics from selected sources" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 7: Decouple Git providers from physical model paths

**Files:**
- Modify: `internal/provider/git/metrics.go`
- Modify: `internal/provider/git/loader.go`
- Modify: `internal/provider/git/authorship_loader.go`
- Modify: `internal/provider/git/changed_paths.go`
- Modify: `internal/stages/git_history.go`
- Modify: `internal/stages/author_history.go`
- Modify: `internal/stages/changed_only.go`
- Modify: related tests

- [ ] **Step 1: Write failing snapshot-identity tests**

Build model files with virtual logical paths and explicit repository paths.
Assert Git metrics, commit grouping, authorship, and changed-only filtering
operate without resolving `Root.Path` as a physical repository.

- [ ] **Step 2: Run tests and verify failure**

Run:

```bash
go test ./internal/provider/git ./internal/stages -run 'Test.*(RepoPath|Snapshot|ChangedOnly)' -count=1
```

Expected: FAIL because loaders derive repository-relative paths from physical
model paths and changed-only intersects the current index.

- [ ] **Step 3: Pass explicit repository context**

Add repository-root parameters to range-aware Git loaders and use each
`model.File.RepoPath` directly. Change `ChangedPathsInHistoryRange` to accept
the scanned path set as authoritative; perform current-index intersection only
for live working-tree sources.

Update history and authorship stages to use `CommonState.RepoRoot` and snapshot
identities. Preserve existing range traversal and progress totals.

- [ ] **Step 4: Run Git and stage tests**

Run:

```bash
go test ./internal/provider/git ./internal/stages -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/provider/git internal/stages
git commit -m "Use snapshot identities for Git metrics" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 8: Apply the snapshot reference clock

**Files:**
- Modify: `internal/provider/git/service.go`
- Modify: `internal/provider/git/git_provider.go`
- Modify: `internal/provider/git/loader.go`
- Modify: `internal/provider/git/authorship_loader.go`
- Modify: `internal/provider/git/metrics_test.go`
- Modify: `internal/provider/git/author_history_range_test.go`
- Modify: `internal/stages/providers.go`

- [ ] **Step 1: Write failing clock tests**

Use fixed commits years in the past and a fixed snapshot time. Assert file age,
freshness, commit density, current-maintainer, orphan-risk, and recent-window
metrics are calculated relative to the snapshot time rather than `time.Now()`.

- [ ] **Step 2: Run tests and verify failure**

Run:

```bash
go test ./internal/provider/git ./internal/stages -run 'Test.*ReferenceClock' -count=1
```

Expected: FAIL because file metrics call `time.Since` and range authorship uses
the first iterated commit as its implicit clock.

- [ ] **Step 3: Inject a reference time**

Thread `CommonState.ReferenceNow` through the range-aware Git loader entry
points. Replace `time.Since` with `referenceTime.Sub`. Set
`AuthorHistoryResult.HeadDate` to the supplied snapshot clock for historical
sources while preserving current behavior for live sources.

- [ ] **Step 4: Run clock and provider tests**

Run:

```bash
go test ./internal/provider/git ./internal/stages -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/provider/git internal/stages/providers.go
git commit -m "Use snapshot time for historical metrics" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 9: Add end-to-end historical snapshot coverage

**Files:**
- Create: `internal/stages/historical_snapshot_test.go`
- Modify: `cmd/codeviz/main_test.go`
- Modify: `cmd/codeviz/render_matrix_test.go`

- [ ] **Step 1: Write integration tests**

Construct a repository fixture with:

- a historical file later deleted;
- a file added after the bound;
- a Go file whose historical and current content produce different metrics;
- a safe symlink;
- a submodule entry; and
- an untracked working-tree file.

Assert direct visualization state and preset-render state use the historical
tree with `--until`, while an omitted `--until` sees the current and untracked
files. Assert date, tag, and SHA forms select equivalent snapshots.

- [ ] **Step 2: Run integration tests and verify behavior**

Run:

```bash
go test ./internal/stages ./cmd/codeviz -run 'TestHistoricalSnapshot|TestRender.*Until' -count=1
```

Expected: PASS after Tasks 1-8; any failure identifies a missing command or
preset wiring surface.

- [ ] **Step 3: Fix uncovered wiring with minimal changes**

Apply only the forwarding changes identified by Step 2. Do not add a new CLI
flag: `--until` itself activates snapshot selection.

- [ ] **Step 4: Re-run integration tests**

Run:

```bash
go test ./internal/stages ./cmd/codeviz -run 'TestHistoricalSnapshot|TestRender.*Until' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/stages/historical_snapshot_test.go cmd/codeviz
git commit -m "Cover historical snapshots end to end" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 10: Document and verify the behavior

**Files:**
- Modify: `docs/content/docs/usage.md`
- Modify: `docs/content/docs/visualizations/bubble-tree.md`
- Modify: `docs/content/docs/visualizations/donut-tree.md`
- Modify: `docs/content/docs/visualizations/radial-tree.md`
- Modify: `docs/content/docs/visualizations/scatter.md`
- Modify: `docs/content/docs/visualizations/spiral.md`
- Modify: `docs/content/docs/visualizations/tree-map.md`

- [ ] **Step 1: Update documentation**

Document that `--until` selects both history and committed file contents,
date-only bounds include the entire date, date bounds choose the latest matching
reachable commit for the snapshot, `--from` alone remains live, and historical
submodules are not expanded.

- [ ] **Step 2: Run focused tests**

Run:

```bash
go test ./internal/source ./internal/scan ./internal/provider/filesystem ./internal/provider/golang ./internal/provider/git ./internal/stages ./cmd/codeviz -count=1
```

Expected: PASS.

- [ ] **Step 3: Run formatting**

Run:

```bash
task fmt
git --no-pager diff --check
```

Expected: formatting succeeds and `diff --check` prints no output.

- [ ] **Step 4: Run repository CI through the required delegated runner**

Run `task ci` through an Explore/task subagent and request only its exit status,
failing tests or linters, offending `file:line` messages, or a one-line success
summary.

Expected: PASS with no failing tests or linters.

- [ ] **Step 5: Commit**

```bash
git add docs internal cmd
git commit -m "Document historical snapshot semantics" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 11: Review and open the pull request

**Files:**
- Review all changes since `origin/main`

- [ ] **Step 1: Run focused code review**

Use the `requesting-code-review` skill to review correctness against
`docs/superpowers/specs/2026-09-09-historical-filesystem-snapshot-design.md`,
paying particular attention to symlink containment, date resolution, virtual
path identity, and live-mode compatibility.

- [ ] **Step 2: Resolve review findings**

For each confirmed finding, add a failing regression test, implement the
smallest correction, and rerun the affected package tests.

- [ ] **Step 3: Re-run repository CI**

Run `task ci` through the required delegated runner.

Expected: PASS.

- [ ] **Step 4: Push and create the PR**

Invoke the `open-pull-request` skill. Use a title such as:

```text
Render historical filesystem snapshots for --until
```

Reference the GitHub issue in the PR body and summarize the intentional
behavior change, source abstraction, test coverage, and compatibility behavior.
