# Git Metrics Performance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring the Azure Service Operator cold spiral render median below 60 seconds by loading only requested Git metrics and eliminating repeated tree diffs for line-churn metrics.

**Architecture:** `provider.RunLoaders` will pass each consolidated loader exactly the requested metric names it owns. The Git loader will derive a requirements value from that subset, retain the one-pass commit-history prewarm, skip patch generation unless churn is requested, and reuse the tracked change from each already-computed tree diff when churn is needed.

**Tech Stack:** Go 1.26.1, go-git v5, Gomega, Task.

Spec: [Git Metrics Performance Design](../specs/2026-08-09-git-metrics-performance-design.md).

---

## File structure

| File | Responsibility |
| --- | --- |
| `internal/provider/loader.go` | Define a loader function that accepts its requested metric subset. |
| `internal/provider/run.go` | Intersect a selected loader's advertised metrics with the invocation request, then pass that intersection through lifecycle and progress handling. |
| `internal/provider/run_test.go` | Verify subset forwarding and subset-scoped progress. |
| `internal/provider/loader_test.go` | Adapt registry test loaders to the new function signature. |
| `internal/provider/classification/provider.go` | Adapt the existing classification loader at registration. |
| `internal/provider/filesystem/register.go` | Adapt existing filesystem loaders at registration. |
| `internal/provider/golang/register.go` | Adapt the existing Go loader at registration. |
| `internal/provider/git/loader.go` | Turn requested Git metric names into processor and churn requirements, prewarm once, and write only selected values. |
| `internal/provider/git/metrics.go` | Hold the ordered file-level Git metric list shared by registration and all-metrics test helpers. |
| `internal/provider/git/service.go` | Pass requirements into the bulk prewarm and update cached data from tracked changes. |
| `internal/provider/git/commit_data.go` | Update cached metadata and calculate churn from an already-computed `object.Change`. |
| `internal/provider/git/bulk_history.go` | Represent tracked changes with their source `object.Change`, preserving current root and merge TREESAME rules. |
| `internal/provider/git/metrics_test.go` | Cover selected writes, churn correctness, and the multi-file one-diff path. |
| `internal/provider/git/base_metrics_test.go` | Assert the registered consolidated loader advertises the shared ordered metric list. |

### Task 1: Pass selected metric subsets to loaders

**Files:**
- Modify: `internal/provider/loader.go`
- Modify: `internal/provider/run.go`
- Modify: `internal/provider/run_test.go`
- Modify: `internal/provider/loader_test.go`
- Modify: `internal/provider/classification/provider.go`
- Modify: `internal/provider/filesystem/register.go`
- Modify: `internal/provider/golang/register.go`
- Modify: `internal/provider/git/loader.go`

- [ ] **Step 1: Write failing subset-forwarding tests**

In `internal/provider/run_test.go`, add a recording loader and tests that
assert a consolidated loader receives only selected names, in loader metric
order, and that progress reports those same names:

```go
type requestedMetricsLoader struct {
    received []metric.Name
}

func (l *requestedMetricsLoader) Load(_ *model.Directory, requested []metric.Name) error {
    l.received = slices.Clone(requested)
    return nil
}

//nolint:paralleltest // mutates global base registry
func TestRunLoadersPassesRequestedSubsetToLoader(t *testing.T) {
    g := NewGomegaWithT(t)
    resetBaseRegistry(t)

    loader := &requestedMetricsLoader{}
    provider.RegisterLoader(provider.BaseMetricLoader{
        Metrics: []metric.Name{"first", "second", "third"},
        Load:    loader.Load,
    })

    g.Expect(provider.RunLoaders(nil, []metric.Name{"third", "first"}, nil)).To(Succeed())
    g.Expect(loader.received).To(Equal([]metric.Name{"first", "third"}))
}

//nolint:paralleltest // mutates global base registry
func TestRunLoadersReportsOnlyRequestedLoaderMetrics(t *testing.T) {
    g := NewGomegaWithT(t)
    resetBaseRegistry(t)

    progress := &progressTracker{}
    loader := &requestedMetricsLoader{}
    provider.RegisterLoader(provider.BaseMetricLoader{
        Metrics: []metric.Name{"first", "second"},
        Load:    loader.Load,
    })

    g.Expect(provider.RunLoaders(nil, []metric.Name{"second"}, progress)).To(Succeed())
    g.Expect(progress.started).To(Equal([]metric.Name{"second"}))
    g.Expect(progress.finished).To(Equal([]metric.Name{"second"}))
}
```

Add `"slices"` to the test imports.

- [ ] **Step 2: Run the focused tests to verify they fail**

Run:

```sh
go test ./internal/provider -run 'TestRunLoaders(PassesRequestedSubsetToLoader|ReportsOnlyRequestedLoaderMetrics)' -count=1
```

Expected: FAIL to compile because `LoadFunc` and `runSingleLoader` do not
accept a requested metric slice.

- [ ] **Step 3: Change the loader contract and execute subsets**

In `internal/provider/loader.go`, replace the function type with:

```go
// LoadFunc populates requested base metrics in the directory tree.
type LoadFunc func(root *model.Directory, requested []metric.Name) error
```

In `internal/provider/run.go`:

1. Thread `requested []metric.Name` from `RunLoaders` through
   `runLoaderLevel` and `runSingleLoader`.
2. Add this helper, preserving the order in `loader.Metrics`:

```go
func requestedMetricsForLoader(loader BaseMetricLoader, requested []metric.Name) []metric.Name {
    wanted := make(map[metric.Name]bool, len(requested))
    for _, name := range requested {
        wanted[name] = true
    }

    result := make([]metric.Name, 0, len(loader.Metrics))
    for _, name := range loader.Metrics {
        if wanted[name] {
            result = append(result, name)
        }
    }

    return result
}
```

3. In `runSingleLoader`, compute `selected := requestedMetricsForLoader(loader,
   requested)`, call `notifyStarted(selected, progress)`, wire per-file
   progress using `selected`, invoke `loader.Load(root, selected)`, and call
   `notifyFinished(selected, progress)`.
4. Change `notifyStarted`, `notifyFinished`, and `wireFileProgress` to accept
   `[]metric.Name`, rather than `BaseMetricLoader`; their loops continue to
   iterate their new slice argument.
5. Update every test-only `Load` closure and
   `fileProgressLoader.Load`/`blockingFileProgressLoader.Load` to accept
   `_ []metric.Name` as a second parameter. They retain their current test
   behavior.
6. In `internal/provider/loader_test.go`, update both registered closures to
   accept `_ []metric.Name`.
7. Keep the existing filesystem, Go, and classification `Load` methods
   unchanged for their direct package tests. At each registration site, wrap
   the existing method in the new contract. In
   `internal/provider/filesystem/register.go`, use:

```go
Load: func(root *model.Directory, _ []metric.Name) error {
    return FileSizeProvider{}.Load(root)
},
```

   Use the same form with `fileLinesProvider.Load(root)` and
   `FileTypeProvider{}.Load(root)` for the other two filesystem registrations.
   In `internal/provider/golang/register.go`, use:

```go
Load: func(root *model.Directory, _ []metric.Name) error {
    return loadFileMetrics(root)
},
```

   In `internal/provider/classification/provider.go`, use:

```go
Load: func(root *model.Directory, _ []metric.Name) error {
    return l.Load(root)
},
```
8. In `internal/provider/git/loader.go`, change the method signature now so
   the repository builds after this commit:

```go
func (l *metricsLoader) Load(root *model.Directory, _ []metric.Name) error {
    return walkGitFilesAll(root, l.onFile)
}
```

   Add the `metric` import. Task 2 will consume this parameter and replace
   `walkGitFilesAll`.

The loader set returned by `LoadersFor` is already filtered to at least one
requested metric, so `selected` is non-empty for every invocation.

- [ ] **Step 4: Run provider tests**

Run:

```sh
go test ./internal/provider/... -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit the loader-contract change**

```sh
git add internal/provider/loader.go internal/provider/run.go internal/provider/run_test.go \
  internal/provider/loader_test.go internal/provider/classification/provider.go \
  internal/provider/filesystem/register.go internal/provider/golang/register.go \
  internal/provider/git/loader.go
git commit -m "refactor(provider): pass selected metrics to loaders"
```

### Task 2: Make the Git loader selection-aware

**Files:**
- Modify: `internal/provider/git/metrics.go`
- Modify: `internal/provider/git/register.go`
- Modify: `internal/provider/git/loader.go`
- Modify: `internal/provider/git/metrics_test.go`
- Modify: `internal/provider/git/base_metrics_test.go`

- [ ] **Step 1: Write failing Git selection tests**

In `internal/provider/git/metrics_test.go`, add:

```go
func TestLoadGitMetrics_PopulatesOnlyRequestedMetrics(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    dir := setupTestGitRepo(t)
    root := buildTree(dir, "shared.go")

    resetService()
    g.Expect(loadGitMetrics(root, []metric.Name{CommitCount}, nil)).To(Succeed())

    count, ok := root.Files[0].Quantity(CommitCount)
    g.Expect(ok).To(BeTrue())
    g.Expect(count).To(Equal(int64(2)))

    _, ok = root.Files[0].Quantity(FileAge)
    g.Expect(ok).To(BeFalse())
    _, ok = root.Files[0].Quantity(TotalLinesAdded)
    g.Expect(ok).To(BeFalse())
    _, ok = root.Files[0].Measure(CommitDensity)
    g.Expect(ok).To(BeFalse())
}

func TestLoadGitMetrics_ChurnRequestPopulatesOnlyRequestedChurnMetric(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    dir := setupDiffRepo(t)
    root := buildTree(dir, "churn.go")

    resetService()
    g.Expect(loadGitMetrics(root, []metric.Name{TotalLinesAdded}, nil)).To(Succeed())

    added, ok := root.Files[0].Quantity(TotalLinesAdded)
    g.Expect(ok).To(BeTrue())
    g.Expect(added).To(Equal(int64(3)))
    _, ok = root.Files[0].Quantity(TotalLinesRemoved)
    g.Expect(ok).To(BeFalse())
    _, ok = root.Files[0].Quantity(CommitCount)
    g.Expect(ok).To(BeFalse())
}
```

Add the `metric` package import. Change
`TestMetricsLoaderReportsFileProgress` to call:

```go
g.Expect(loader.Load(root, []metric.Name{CommitCount})).To(Succeed())
```

- [ ] **Step 2: Run the focused Git tests to verify they fail**

Run:

```sh
go test ./internal/provider/git -run 'TestLoadGitMetrics_(PopulatesOnlyRequestedMetrics|ChurnRequestPopulatesOnlyRequestedChurnMetric)' -count=1
```

Expected: FAIL to compile because `loadGitMetrics` does not exist and
`metricsLoader.Load` has the old signature.

- [ ] **Step 3: Add ordered Git metric names and requirements**

In `internal/provider/git/metrics.go`, add the shared ordered list:

```go
var fileMetricNames = []metric.Name{
    FileAge,
    FileFreshness,
    AuthorCount,
    CommitCount,
    TotalLinesAdded,
    TotalLinesRemoved,
    CommitDensity,
}
```

In `internal/provider/git/register.go`, use `Metrics: fileMetricNames` in the
registered `BaseMetricLoader`.

In `internal/provider/git/loader.go`:

1. Change `metricsLoader.Load` to:

```go
func (l *metricsLoader) Load(root *model.Directory, requested []metric.Name) error {
    return loadGitMetrics(root, requested, l.onFile)
}
```

2. Keep `loadAllFileMetrics` as a test helper, but delegate it to:

```go
func loadAllFileMetrics(root *model.Directory) error {
    return loadGitMetrics(root, fileMetricNames, nil)
}
```

3. Add:

```go
type metricRequirements struct {
    processors     []providerDef
    needsLineStats bool
}

func newMetricRequirements(requested []metric.Name) metricRequirements {
    requirements := metricRequirements{
        processors: make([]providerDef, 0, len(requested)),
    }

    for _, name := range requested {
        def, ok := providerDefs[name]
        if !ok {
            continue
        }

        requirements.processors = append(requirements.processors, def)
        if name == TotalLinesAdded || name == TotalLinesRemoved {
            requirements.needsLineStats = true
        }
    }

    return requirements
}
```

4. Rename `walkGitFilesAll` to `loadGitMetrics`, with signature:

```go
func loadGitMetrics(
    root *model.Directory,
    requested []metric.Name,
    onFile func(),
) error
```

It creates `requirements := newMetricRequirements(requested)`, calls the
existing `s.bulkPrewarm(pathSet, onFile)`, and applies only
`requirements.processors` in the file walk. Task 3 will pass
`requirements` into the prewarm so it can skip churn calculation.
5. Replace `anyFileHasGitMetric` with a service helper that checks the
prewarmed `commitCache` for a requested scanned path whose `count > 0`.
It must lock `commitMu` for reading and must not depend on `FileAge` having
been written. Keep the existing "no metrics" error text when no scanned path
has Git history.

- [ ] **Step 4: Update registration coverage**

In `internal/provider/git/base_metrics_test.go`, replace the duplicated
seven-name literal in `TestRegister_RegistersConsolidatedGitLoader` with:

```go
loaders := provider.LoadersFor(fileMetricNames)

g.Expect(loaders).To(HaveLen(1))
g.Expect(loaders[0].Metrics).To(Equal(fileMetricNames))
g.Expect(loaders[0].Load).ToNot(BeNil())
```

- [ ] **Step 5: Run Git package tests**

Run:

```sh
go test ./internal/provider/git -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit selection-aware Git loading**

```sh
git add internal/provider/git/metrics.go internal/provider/git/register.go \
  internal/provider/git/loader.go internal/provider/git/metrics_test.go \
  internal/provider/git/base_metrics_test.go
git commit -m "perf(git): load only requested file metrics"
```

### Task 3: Reuse commit diffs for line-churn statistics

**Files:**
- Modify: `internal/provider/git/bulk_history.go`
- Modify: `internal/provider/git/service.go`
- Modify: `internal/provider/git/commit_data.go`
- Modify: `internal/provider/git/metrics_test.go`

- [ ] **Step 1: Write a multi-file churn regression test**

Add this helper and test to `internal/provider/git/metrics_test.go`:

```go
func setupMultiFileDiffRepo(t *testing.T) string {
    t.Helper()

    dir := setupDiffRepo(t)
    write := func(name, content string) {
        t.Helper()
        g := NewGomegaWithT(t)
        g.Expect(os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600)).To(Succeed())
    }
    run := func(args ...string) {
        t.Helper()
        cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // test helper
        cmd.Dir = dir
        out, err := cmd.CombinedOutput()
        if err != nil {
            t.Fatalf("command %v failed: %s\n%s", args, err, out)
        }
    }

    write("other.go", "one\ntwo\n")
    run("git", "add", "other.go")
    run("git", "commit", "-m", "add other")
    write("churn.go", "line1\nlineX\nline3\nline4\nline5\nline6\n")
    write("other.go", "one\ntwo\nthree\n")
    run("git", "add", "churn.go", "other.go")
    run("git", "commit", "-m", "change both")

    return dir
}

func TestLoadGitMetrics_ChurnForMultiFileCommit(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    dir := setupMultiFileDiffRepo(t)
    root := buildTree(dir, "churn.go", "other.go")

    resetService()
    g.Expect(loadGitMetrics(
        root,
        []metric.Name{TotalLinesAdded, TotalLinesRemoved},
        nil,
    )).To(Succeed())

    churnAdded, ok := root.Files[0].Quantity(TotalLinesAdded)
    g.Expect(ok).To(BeTrue())
    g.Expect(churnAdded).To(Equal(int64(4)))
    otherAdded, ok := root.Files[1].Quantity(TotalLinesAdded)
    g.Expect(ok).To(BeTrue())
    g.Expect(otherAdded).To(Equal(int64(1)))
}
```

The expected `churn.go` total is its prior three additions plus the final
one-line addition. The test protects correctness for multiple paths in one
commit; the implementation below ensures that case does not repeat
`object.DiffTree`.

- [ ] **Step 2: Run the churn tests to establish current behavior**

Run:

```sh
go test ./internal/provider/git -run 'Test(LoadGitMetrics_ChurnForMultiFileCommit|TotalLines(Added|Removed)Provider)' -count=1
```

Expected: PASS before the refactor, establishing the expected churn values.

- [ ] **Step 3: Retain tracked `object.Change` values**

In `internal/provider/git/bulk_history.go`, add:

```go
type trackedChange struct {
    path   string
    change *object.Change
}
```

Add a `trackedChangesInCommit` helper that implements the same root,
single-parent, and merge rules as `changedFilesInCommit`, but returns
`[]trackedChange`.

- For a root commit, return tracked existing paths with `change: nil`.
- For a single-parent commit, call `object.DiffTree` once, filter its
  `object.Changes` to tracked paths, and retain pointers to the corresponding
  `object.Change`.
- For a merge, compute one diff per parent, retain
  `map[string]*object.Change` per parent, and include a path only when it is
  present in every parent map. Use its first-parent change as `change`, which
  preserves the current `computeFileDiffStats` first-parent behavior.

Rewrite `changedFilesInCommit` as a projection of
`trackedChangesInCommit`, so `BulkFileHistory` and `BulkCommitHistory`
retain exactly their current public behavior. Replace the old
`diffTrackedFiles` helper with a helper that returns the path-to-change map
needed by both the merge rule and prewarm.

- [ ] **Step 4: Consume retained changes in the prewarm**

In `internal/provider/git/service.go`, change:

```go
func (s *repoService) bulkPrewarm(
    paths map[string]bool,
    requirements metricRequirements,
    onFileProcessed func(),
) error
```

and pass `requirements` into `doBulkPrewarm` and `prewarmCommit`.

In `prewarmCommit`, replace:

```go
changed := changedFilesInCommit(c, paths)

for _, relPath := range changed {
    data := cache[relPath]
    data.updateFrom(c, relPath)
```

with:

```go
changed := trackedChangesInCommit(c, paths)

for _, entry := range changed {
    data := cache[entry.path]
    data.updateFrom(c, entry.change, requirements.needsLineStats)
```

Keep the `onFileProcessed` callback once per changed tracked path.

In `internal/provider/git/commit_data.go`, change `updateFrom` to:

```go
func (data *commitData) updateFrom(
    c *object.Commit,
    change *object.Change,
    needsLineStats bool,
) {
    when := c.Author.When
    if data.oldest.IsZero() || when.Before(data.oldest) {
        data.oldest = when
    }
    if data.newest.IsZero() || when.After(data.newest) {
        data.newest = when
    }
    data.authors[c.Author.Email] = true
    data.count++

    if !needsLineStats || change == nil || change.From.Name == "" {
        return
    }

    patch, err := object.Changes{change}.Patch()
    if err != nil {
        return
    }
    for _, stat := range patch.Stats() {
        data.linesAdded += int64(stat.Addition)
        data.linesRemoved += int64(stat.Deletion)
    }
}
```

Remove the bulk-prewarm call to `computeFileDiffStats`; retain that function
and `filterChangesForFile` only for the existing per-file
`processCommitForFile` fallback. Move the fallback's prior call into:

```go
func updateLineStatsForFile(data *commitData, c *object.Commit, relPath string) {
    added, removed := computeFileDiffStats(c, relPath)
    data.linesAdded += added
    data.linesRemoved += removed
}
```

Place `computeFileDiffStats` and `filterChangesForFile` with that fallback
helper in `service.go`; they are not called by the bulk prewarm. This avoids
changing behavior when a concurrent cache miss uses the historical per-file
log path.

- [ ] **Step 5: Run focused Git tests**

Run:

```sh
go test ./internal/provider/git -count=1
```

Expected: PASS, including merge-history, selected-metric, and multi-file churn
coverage.

- [ ] **Step 6: Commit the one-diff churn implementation**

```sh
git add internal/provider/git/bulk_history.go internal/provider/git/service.go \
  internal/provider/git/commit_data.go internal/provider/git/metrics_test.go
git commit -m "perf(git): reuse commit diffs for churn metrics"
```

### Task 4: Verify the repository and external performance acceptance

**Files:**
- No source changes expected.

- [ ] **Step 1: Run the affected package suite**

Run:

```sh
go test ./internal/provider/... -count=1
```

Expected: PASS.

- [ ] **Step 2: Run the full test suite**

Run:

```sh
task test
```

Expected: PASS.

- [ ] **Step 3: Build and run three cold acceptance measurements**

Run:

```sh
task build
for run in 1 2 3; do
  output="$(mktemp --suffix=.svg)"
  /usr/bin/time -f '%e' -o "/tmp/codeviz-aso-spiral-${run}.seconds" \
    bin/codeviz spiral --config samples/spiral/code-visualizer.yml \
      /home/bevan/github/azure-service-operator --output "$output" --quiet
  rm -f "$output"
done
sort -n /tmp/codeviz-aso-spiral-{1,2,3}.seconds | sed -n '2p'
rm -f /tmp/codeviz-aso-spiral-{1,2,3}.seconds
```

Expected: the printed median is strictly less than `60`.

- [ ] **Step 4: Run repository formatting and lint gates**

Dispatch the noisy CI command through an Explore-equivalent subagent as
required by repository guidance:

```text
Run `task ci` in the repository. Return only its exit status, the identity and
count of failing tests or linters, each offending file:line and message, or a
one-line success note.
```

Expected: exit status 0 with no failing tests or linters.

### Task 5: Share the spiral history and Git-metric traversal

**Files:**
- Modify: `internal/provider/git/commit.go`
- Modify: `internal/provider/git/service.go`
- Modify: `internal/provider/git/commit_test.go`
- Modify: `internal/provider/git/metrics_test.go`
- Modify: `internal/stages/git_history.go`
- Modify: `internal/stages/git_history_test.go`
- Modify: `internal/spiral/pipeline.go`

- [ ] **Step 1: Write failing combined-pass provider tests**

In `internal/provider/git/commit_test.go`, add a test that uses the existing
Git fixture and verifies the new combined API returns history while warming
the selected metric cache:

```go
func TestBulkCommitHistoryAndPrewarm_ReturnsHistoryAndWarmsCommitCount(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)
	tracked := map[string]bool{"old.go": true, "shared.go": true}

	resetService()
	commits, err := BulkCommitHistoryAndPrewarm(
		dir,
		tracked,
		[]metric.Name{CommitCount},
		nil,
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(commits).NotTo(BeEmpty())

	s, err := getService(dir)
	g.Expect(err).NotTo(HaveOccurred())
	data := s.commitCache["shared.go"]
	g.Expect(data).NotTo(BeNil())
	g.Expect(data.count).To(Equal(int64(2)))
	g.Expect(data.hasLineStats).To(BeFalse())
}
```

Add the `metric` package import.

In `internal/provider/git/metrics_test.go`, add a reuse test that proves the
provider does not repeat a completed prewarm:

```go
func TestLoadGitMetrics_ReusesCombinedHistoryPrewarm(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupTestGitRepo(t)
	root := buildTree(dir, "shared.go")
	tracked := map[string]bool{"shared.go": true}

	resetService()
	_, err := BulkCommitHistoryAndPrewarm(
		dir,
		tracked,
		[]metric.Name{CommitCount},
		nil,
	)
	g.Expect(err).NotTo(HaveOccurred())

	s, err := getService(dir)
	g.Expect(err).NotTo(HaveOccurred())
	before := s.commitCache["shared.go"]

	g.Expect(loadGitMetrics(root, []metric.Name{CommitCount}, nil)).To(Succeed())
	g.Expect(s.commitCache["shared.go"]).To(BeIdenticalTo(before))
}
```

- [ ] **Step 2: Run the new tests to verify they fail**

Run:

```sh
go test ./internal/provider/git \
  -run 'Test(BulkCommitHistoryAndPrewarm_ReturnsHistoryAndWarmsCommitCount|LoadGitMetrics_ReusesCombinedHistoryPrewarm)' \
  -count=1
```

Expected: FAIL to compile because `BulkCommitHistoryAndPrewarm` does not
exist.

- [ ] **Step 3: Add the combined history-and-prewarm API**

In `internal/provider/git/commit.go`, add:

```go
func BulkCommitHistoryAndPrewarm(
	repoPath string,
	tracked map[string]bool,
	requested []metric.Name,
	onCommitProcessed func(),
) ([]Commit, error)
```

It obtains the service with `getService`, constructs requirements through
`newMetricRequirements(requested)`, and delegates to:

```go
func (s *repoService) bulkCommitHistoryAndPrewarm(
	tracked map[string]bool,
	requirements metricRequirements,
	onCommitProcessed func(),
) ([]Commit, error)
```

Implement the service method in `internal/provider/git/service.go`:

1. Allocate `cache` with one `commitData` per tracked path only when
   `len(requirements.processors) != 0`. Set `hasLineStats` from
   `requirements.needsLineStats`.
2. Open `HEAD` and one log iterator using the same error wrapping as
   `doBulkPrewarm`.
3. For each commit, call `trackedChangesInCommit(c, tracked)` once. Call
   `onCommitProcessed` once for every examined commit. If no tracked change
   exists, do not append a `Commit`.
4. For a changed commit, append the same `Commit` shape currently produced by
   `BulkCommitHistory`: hash, author, committer, message, parent hashes, and
   ordered changed paths.
5. When `cache` is non-nil, update the matching `commitData` with
   `updateMetadata(c)` and, only when `requirements.needsLineStats`, with
   `updateChangeStats(entry.change)`.
6. After successful iteration, publish the full cache through
   `mergeBulkPrewarmCache(cache, requirements)`. Do not publish it if the
   iterator returns an error.

Leave `BulkCommitHistory` unchanged for its existing callers and tests. The
new API is the only API that produces rich history and metric cache entries in
one traversal.

- [ ] **Step 4: Run Git provider tests**

Run:

```sh
go test ./internal/provider/git -count=1
```

Expected: PASS.

- [ ] **Step 5: Write a failing stage integration test**

In `internal/stages/git_history_test.go`, add:

```go
func TestLoadGitHistory_ThenRunProviders_WritesRequestedGitMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := setupHistoryRepo(t)
	state := buildHistoryState(dir)
	state.Requested.BaseMetrics = []metric.Name{git.CommitCount}

	g.Expect(LoadGitHistory(state)).To(Succeed())
	g.Expect(RunProviders(state)).To(Succeed())

	count, ok := state.Root.Files[1].Quantity(git.CommitCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(count).To(Equal(int64(2)))
}
```

Add the `metric` and `git` imports. The provider-level reuse test from Step 1
proves the second caller does not rerun the prewarm; this stage test proves
the reordered production stages write the expected metric value after history
loading without adding a cache-inspection API.

- [ ] **Step 6: Reorder spiral acquisition and use the combined API**

In `internal/stages/git_history.go`, replace the `BulkCommitHistory` call in
`LoadGitHistory` with:

```go
commits, err := git.BulkCommitHistoryAndPrewarm(
	repoRoot,
	tracked,
	c.Requested.BaseMetrics,
	onCommit,
)
```

Keep `stop()` immediately after the call and preserve all existing error
wrapping and empty-history behavior.

In `internal/spiral/pipeline.go`, move `LoadGitHistory` to immediately after
`CheckGitRequirement` and before `RunProviders`. Keep
`GroupGitHistoryByFile` and `ExtractFileHistory` after
`PopulateDeclarations`. The resulting production sequence is:

```go
ScanFilesystem
CheckGitRequirement
LoadGitHistory
RunProviders
PopulateDeclarations
GroupGitHistoryByFile
ExtractFileHistory
```

This is safe because `ResolveMetrics` has already populated
`CommonState.Requested` before `AcquireData` begins, and `LoadGitHistory`
needs only the scanned root and requested metric names.

- [ ] **Step 7: Run affected tests and commit**

Run:

```sh
go test ./internal/provider/git ./internal/stages ./internal/spiral -count=1
```

Expected: PASS.

Commit:

```sh
git add internal/provider/git/commit.go internal/provider/git/service.go \
  internal/provider/git/commit_test.go internal/provider/git/metrics_test.go \
  internal/stages/git_history.go internal/stages/git_history_test.go \
  internal/spiral/pipeline.go
git commit -m "perf(spiral): share git history metric traversal"
```

### Task 6: Revalidate end-to-end performance

**Files:**
- No source changes expected.

- [ ] **Step 1: Run focused and full tests**

Run:

```sh
go test ./internal/provider/... ./internal/stages ./internal/spiral -count=1
task test
```

Expected: PASS.

- [ ] **Step 2: Measure three cold acceptance renders**

Run:

```sh
task build
for run in 1 2 3; do
  output="$(mktemp --suffix=.svg)"
  /usr/bin/time -f '%e' -o "/tmp/codeviz-aso-spiral-${run}.seconds" \
    bin/codeviz spiral --config samples/spiral/code-visualizer.yml \
      /home/bevan/github/azure-service-operator --output "$output" --quiet
  rm -f "$output"
done
sort -n /tmp/codeviz-aso-spiral-{1,2,3}.seconds | sed -n '2p'
rm -f /tmp/codeviz-aso-spiral-{1,2,3}.seconds
```

Expected: the printed median is strictly less than `60`.

- [ ] **Step 3: Run the repository CI gate**

Dispatch `task ci` through an Explore-equivalent subagent and return only its
exit status, failing test or linter identities, offending file:line messages,
or a one-line success note.

Expected: exit status 0 with no failures.
