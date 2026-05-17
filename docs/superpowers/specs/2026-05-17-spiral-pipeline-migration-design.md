# Spiral Pipeline Migration — Design

**Date:** 2026-05-17
**Status:** Approved (pending implementation plan).
**Predecessor:** [`2026-05-17-bubbletree-pipeline-migration-design.md`](2026-05-17-bubbletree-pipeline-migration-design.md)
**Branch:** `feature/spiral-pipeline` (off `main` after bubbletree PR #248 merged).

---

## 1. Goal

Migrate the spiral visualization to the pipeline scaffold already used by treemap and bubbletree, and extract three reusable git-history stages into `internal/stages` so future time-aware visualizations don't re-implement the commit walk.

After this work:

- `cmd/codeviz/spiral_cmd.go` contains only the Kong struct, validation, config merging, override application, and a short `pipeline.Run` composition.
- All spiral-specific render / inks / bucketing / aggregation / disc-sizing / layout-glue code lives in `internal/spiral`.
- Loading commit history, grouping commits by file, and extracting per-file time ranges are three independent, composable stages in `internal/stages` operating on a rich `git.Commit` data type.

## 2. Non-goals

- **No CLI / output / log changes.** Spiral's flags, rendered pixels, and log lines stay byte-identical.
- **No promotion of `buildBucketInk` to `internal/inks`.** Bubble / treemap / radial share a per-file ink-construction pattern; spiral's bucket-aggregated form has one consumer today. Promote when a second bucket-based viz appears.
- **No render-test scope expansion.** Port the existing `cmd/codeviz/spiral_canvas_test.go` to `internal/spiral/render_test.go`, no Goldie-free PNG/SVG/JPG matrix beyond what's already there.
- **No removal of `git.BulkFileHistory`.** The richer `BulkCommitHistory` is additive. Existing callers stay on `BulkFileHistory` until / unless they need the richer data.

## 3. Architecture

### 3.1 New shared building blocks

#### 3.1.1 `internal/provider/git` — richer commit type

```go
// Signature mirrors go-git's object.Signature: an author or committer record
// captured at the moment a commit was made.
type Signature struct {
    Name  string
    Email string
    When  time.Time
}

// Commit is a single commit in the project history, carrying enough metadata
// for any downstream consumer (timeline, churn, authorship, message-mining).
// ChangedPaths is restricted to the tracked path set passed to BulkCommitHistory
// so the slice size stays bounded.
type Commit struct {
    Hash         string
    Author       Signature
    Committer    Signature
    Message      string
    ParentHashes []string
    ChangedPaths []string // slash-separated, repo-relative
}

// BulkCommitHistory walks the commit graph once and returns one Commit per
// commit reachable from HEAD that touches at least one path in `tracked`.
// onCommit is invoked after each commit is examined (for progress reporting).
func BulkCommitHistory(
    repoPath string,
    tracked map[string]bool,
    onCommit func(),
) ([]Commit, error)
```

Internally `BulkCommitHistory` reuses `changedFilesInCommit` (the same helper `BulkFileHistory` uses). Commits with no tracked paths changed are omitted from the result, matching today's behaviour where empty timestamp entries never reach spiral.

#### 3.1.2 `internal/stages` — git-history stages and sidecars

New file `internal/stages/git_history.go`:

```go
// CommitRef points back into CommonState.GitHistory with the per-file
// when-touched timestamp. Storing a pointer avoids duplicating Author /
// Message / ParentHashes per file-commit pair.
type CommitRef struct {
    Commit *git.Commit
    When   time.Time
}

// TimeRange is the earliest and latest commit times observed for a file.
type TimeRange struct {
    Earliest time.Time
    Latest   time.Time
}
```

New `CommonState` fields:

```go
GitHistory    []git.Commit                  // LoadGitHistory
FileHistory   map[*model.File][]CommitRef   // GroupGitHistoryByFile
FileTimeRange map[*model.File]TimeRange     // ExtractFileHistory
```

New stages (generic over `VizState`, same shape as the existing shared stages):

| Stage                   | Reads                          | Writes          |
| ----------------------- | ------------------------------ | --------------- |
| `LoadGitHistory`        | `Root`, `Flags` (for progress) | `GitHistory`    |
| `GroupGitHistoryByFile` | `GitHistory`, `Root`           | `FileHistory`   |
| `ExtractFileHistory`    | `FileHistory`                  | `FileTimeRange` |

`LoadGitHistory` internally:
1. Resolves the repo root via `git.RepoRootFor(common.Root.Path)`.
2. Builds a `map[slashRelPath]bool` from `common.Root` (mirroring today's `loadCommitHistory`).
3. Builds an `onCommit` progress callback via the new `stages.BuildHistoryProgress` helper (moved from `cmd/codeviz/progress.go`).
4. Calls `git.BulkCommitHistory(repoRoot, tracked, onCommit)`.
5. Returns `eris.New("no commit history found; commit-history-dependent visualizations require git commits")` when the result is empty — spiral's existing precondition, hoisted into the shared stage so other commit-history-dependent vizes inherit it.

`GroupGitHistoryByFile`:
1. Walks `common.Root` once to build `map[slashRelPath]*model.File`.
2. Iterates `common.GitHistory`; for each commit, for each `ChangedPath` that resolves to a known file, appends `CommitRef{Commit: &commits[i], When: commit.Author.When}` to that file's slice.
3. Writes `common.FileHistory`.

`ExtractFileHistory`:
1. Iterates `common.FileHistory`; for each `(*File, []CommitRef)`, folds `min`/`max` over the `When` values.
2. Writes `common.FileTimeRange`.

`BuildHistoryProgress` and its `startHistoryTicker` helper move from `cmd/codeviz/progress.go` to `internal/stages/progress.go` (alongside `BuildScanProgress` / `BuildMetricProgress`). `startProgressTicker` either moves with them if it had no other users, or stays put if `cmd/codeviz` still needs it.

### 3.2 New `internal/spiral` package layout

```
internal/spiral/
  layout.go         (existing, unchanged)
  layout_test.go    (existing, unchanged)
  node.go           (existing, unchanged)
  timebucket.go     (existing, unchanged)
  timebucket_test.go (existing, unchanged)
  render.go         (new — moved from cmd/codeviz/spiral_canvas.go)
  inks.go           (new — Inks + BuildInks + buildBucketInk)
  bucketing.go      (new — BuildTimeBuckets wrapper, AssignFilesToBuckets)
  aggregation.go    (new — aggregateBucketMetrics, sumNumericMetric, modeCategory, aggregateColourMetric)
  discsize.go       (new — applyDiscSizes, minDiscRadius)
  state.go          (new — State embeds stages.CommonState)
  stages.go         (new — spiral-specific stages)
  render_test.go    (new — ported from cmd/codeviz/spiral_canvas_test.go)
  inks_test.go      (new — ported from cmd/codeviz/spiral_canvas_test.go)
```

`render.go` exports a single function `RenderToCanvas(layout SpiralLayout, buckets []TimeBucket, width, height int, inks Inks) *canvas.Canvas`. `inks.go` exports `Inks { Fill, Border canvas.Ink }` and `BuildInks(...)`. The helpers in `bucketing.go`, `aggregation.go`, and `discsize.go` stay unexported — they are called only from `stages.go`.

### 3.3 Spiral pipeline State

```go
type State struct {
    stages.CommonState

    Config             *config.Spiral
    IncludeBinaryFiles bool

    // Resolved during the pipeline:
    Size          metric.Name
    FillMetric    metric.Name
    FillPalette   palette.PaletteName
    BorderMetric  metric.Name
    BorderPalette palette.PaletteName
    Resolution    Resolution            // hourly | daily
    Labels        LabelMode             // all | laps | none

    Buckets       []TimeBucket
    Inks          Inks
    Layout        SpiralLayout          // contains Nodes + sizing metadata
    LegendConfig  *canvas.LegendConfig
}

func (s *State) Common() *stages.CommonState  { return &s.CommonState }
func (s *State) IncludeBinary() bool          { return s.IncludeBinaryFiles }
```

### 3.4 Spiral-specific stages (`internal/spiral/stages.go`)

| Stage                         | Reads                                                                   | Writes                                                                   |
| ----------------------------- | ----------------------------------------------------------------------- | ------------------------------------------------------------------------ |
| `ResolveMetrics`              | `Config`                                                                | `Size`, `Fill*`, `Border*`, `Resolution`, `Labels`, `Common().Requested` |
| `BuildTimeBucketsStage`       | `Resolution`, `Common().FileTimeRange`, `Common().FileHistory`          | `Buckets` (built + files assigned)                                       |
| `AggregateBucketMetricsStage` | `Size`, `FillMetric`, `BorderMetric`, `Buckets`                         | `Buckets` (each bucket's aggregated values populated)                    |
| `BuildInksStage`              | `Buckets`, `FillMetric`, `FillPalette`, `BorderMetric`, `BorderPalette` | `Inks`, emits "Rendering image" log                                      |
| `BuildLegendStage`            | `Inks`, `FillMetric`, `BorderMetric`, `Size`, `Config.Legend*`          | `LegendConfig`                                                           |
| `LayoutStage`                 | `Buckets`, `Common().Width/Height`, `Resolution`, `Labels`              | `Layout` (after `applyDiscSizes`)                                        |
| `RenderStage`                 | `Layout`, `Buckets`, `Inks`, `Common().Width/Height`, `LegendConfig`    | `Common().Canvas`                                                        |
| `LogResult`                   | `Common().Root`, dimensions, metrics                                    | (logs only)                                                              |

`BuildTimeBucketsStage` derives the global `(startTime, endTime)` by folding over `Common().FileTimeRange`. `AggregateBucketMetricsStage` and `LayoutStage` are direct adaptations of today's `aggregateBucketMetrics` / `applySpiralDiscSizes` calls.

### 3.5 Pipeline composition

`SpiralCmd.Run`:

```go
pipeline.Run[*spiral.State](state,
    // Pre-scan
    stages.ValidatePaths,
    stages.ExportConfig,
    stages.BuildFilterRules,
    spiral.ResolveMetrics,
    stages.ScanFilesystem,
    stages.CheckGitRequirement,
    stages.RunProviders,
    stages.FilterBinaryFiles,
    stages.ExportData,

    // Git history (new shared stages)
    stages.LoadGitHistory,
    stages.GroupGitHistoryByFile,
    stages.ExtractFileHistory,

    // Spiral-specific
    stages.ResolveDimensions,
    spiral.BuildTimeBucketsStage,
    spiral.AggregateBucketMetricsStage,
    spiral.BuildInksStage,
    spiral.BuildLegendStage,
    spiral.LayoutStage,
    spiral.RenderStage,
    stages.WriteCanvas,
    spiral.LogResult,
)
```

## 4. Files to delete

- `cmd/codeviz/spiral_canvas.go` (~288 lines) — moved to `internal/spiral/render.go` + `inks.go`.
- `cmd/codeviz/spiral_canvas_test.go` (~191 lines) — ported to `internal/spiral/render_test.go` + `inks_test.go`.
- `cmd/codeviz/spiral_githistory.go` (~73 lines) — behaviour now lives in `stages.LoadGitHistory` + `stages.GroupGitHistoryByFile`.

`cmd/codeviz/progress.go` loses `buildHistoryProgress` and `startHistoryTicker` (they move to `internal/stages`). `startProgressTicker` stays only if other `cmd/codeviz` code uses it; otherwise it moves too.

`cmd/codeviz/spiral_cmd.go` shrinks from ~486 lines to roughly the size of `bubbletree_cmd.go` (~150 lines). Removed: `Run`'s open-coded orchestration, `scanAndRunProviders`, `buildTimeBuckets`, `layoutAndRender`, `logRendered`, `aggregateBucketMetrics`, `aggregateBucket`, `aggregateColourMetric`, `sumNumericMetric`, `modeCategory`, `commitTimeRange`, `assignFilesToBuckets`, `applySpiralDiscSizes`, `resolveResolution`, `resolveLabels`, `resolveFillMetric`, `collectSpiralMetrics`.

## 5. Testing strategy

- **Existing tests stay green:** `cmd/codeviz` validation and config-merge tests, `internal/spiral/*_test.go`, treemap / bubbletree / radial tests, `internal/inks` tests.
- **Relocated tests:** `cmd/codeviz/spiral_canvas_test.go` content is split into `internal/spiral/render_test.go` and `internal/spiral/inks_test.go` (same Gomega style, same coverage surface — no expansion).
- **New tests in `internal/stages/git_history_test.go`** cover:
  - `LoadGitHistory` against a small in-tree fixture repo (re-use any existing fixture helper) — returns at least one `Commit` with non-empty `ChangedPaths`, populated `Author.When`, populated `Hash`.
  - `LoadGitHistory` errors with "no commit history found" when the tracked set is empty or the repo has no matching commits.
  - `GroupGitHistoryByFile` produces a non-empty map and each `CommitRef.Commit` is non-nil and points back into `CommonState.GitHistory` (verify by pointer identity).
  - `ExtractFileHistory` computes correct min/max for a file touched by multiple commits at known timestamps.
- **Smoke test:** `task build` then `./bin/codeviz render spiral . -o /tmp/spiral.png` exits 0, produces a valid PNG, and the final log line is `Rendered spiral …` with the same keys as today.

## 6. Sequencing

The implementation plan derived from this spec will land in commits along these boundaries (no checkpoint left red):

1. **Add `git.BulkCommitHistory`** + `Commit` / `Signature` types. Existing `BulkFileHistory` callers untouched.
2. **Add `stages.LoadGitHistory`, `GroupGitHistoryByFile`, `ExtractFileHistory`** + new CommonState fields + `BuildHistoryProgress`. Spiral still uses the old path; new stages have unit tests but no production caller yet.
3. **Move spiral render + inks into `internal/spiral`** (`render.go`, `inks.go`); delete `cmd/codeviz/spiral_canvas.go` and port `spiral_canvas_test.go` into the package.
4. **Move spiral helpers** (`bucketing.go`, `aggregation.go`, `discsize.go`) into `internal/spiral`. Spiral still uses them via `cmd/codeviz/spiral_cmd.go`.
5. **Add spiral pipeline `State` + stages** (`state.go`, `stages.go`); not wired yet.
6. **Rewrite `SpiralCmd.Run`** as a `pipeline.Run` composition. Delete `spiral_githistory.go` and the now-dead helpers in `spiral_cmd.go`. Move `buildHistoryProgress` out of `cmd/codeviz/progress.go`.

Each step ends on a green `task ci`.

## 7. Risks and mitigations

- **Risk:** `BulkCommitHistory` allocates more per commit than `BulkFileHistory` (full message, parent hashes). For very large repos this could noticeably increase memory.
  **Mitigation:** Message is the only unbounded field. If profiling shows a regression, store it lazily (e.g. defer reading until first access) — but defer the optimization until evidence warrants it. Document the trade-off in the package doc on `Commit`.

- **Risk:** Pointer-back into `GitHistory` (`CommitRef.Commit *Commit`) means the slice must not be mutated after `LoadGitHistory` completes.
  **Mitigation:** The stage writes the slice once and no other stage touches it. Make this an invariant in the package doc on `GitHistory`.

- **Risk:** Three stages where one would suffice may feel like over-decomposition to future readers.
  **Mitigation:** Each stage has a single, named responsibility that maps directly to a verb a future viz would want to invoke (load / group / extract). The composability is the value: a churn viz needs `Load` + `Group` but not `Extract`; an author-timeline viz needs `Load` only. The decomposition pays for itself the first time a viz needs only one of the three.

## 8. Open questions

None — all clarifying questions resolved during brainstorming.
