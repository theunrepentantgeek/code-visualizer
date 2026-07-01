# Golden Tests Restoration — Design

**Date:** 2026-06-28
**Status:** Approved
**Branch:** `restore/golden-tests`

## Problem

Output verification regressed silently. The visualization tests that survive
(`cmd/codeviz/render_matrix_test.go`) only assert that output files are
*non-empty* — they never check the actual bytes. There are no golden-file tests
left, and the `goldie` dependency was removed. As a result, a rendering or
metric-computation regression can pass CI unnoticed, which destroys confidence
that the tool produces correct output.

We need byte-level verification of:

1. **Visualization output** — PNG *and* SVG for every supported visualization
   type.
2. **Metric expressions** — the resolution/aggregation/filter/cross-level layer
   that turns base metric values into the numbers the visualizations consume.

Both must run from **synthetic, in-memory data** (not the repository's own
files) so the goldens are stable across trivial commits, and both must be found
and updated by the existing `Taskfile` with no modification to it.

### Constraints

- Do **not** remove or change any existing test.
- New goldens must be created/updated by the existing
  `task update-golden-files` target (which already sets `GOLDIE_UPDATE=1`) and
  exercised by `task test`, with **no** Taskfile changes.
- The filesystem scanner is `os`-bound (`os.ReadDir`/`os.Stat`), so
  "in-memory directory structure" means building a `model.Directory` tree
  directly and bypassing the scanner.

## Background — relevant architecture

- **Visualizations.** Five `render` subcommands: `tree-map`, `radial`,
  `bubble-tree`, `spiral`, `scatter` (`cmd/codeviz/*_cmd.go`). Each assembles a
  near-identical pipeline inline in its `Run` method:
  `resolve → ScanFilesystem → RunProviders → PopulateDeclarations →
  RunAggregations → … → render stages → WriteCanvas`. Spiral additionally runs
  `LoadGitHistory` / `GroupGitHistoryByFile` / `ExtractFileHistory` and
  time-bucket stages.
- **Pipeline.** `pipeline.State` is a type-keyed value store;
  `stages.CommonState` carries `Root *model.Directory`, `Width`/`Height`,
  `Canvas`, and (for spiral) `GitHistory` / `FileHistory` / `FileTimeRange`.
- **Rendering.** `canvas.Canvas.Render(path)` selects a backend from the file
  extension (PNG/JPG raster, SVG vector). Text uses the **embedded**
  `goregular.TTF` font (`internal/canvas/textlayout`), so raster output is
  deterministic — byte-perfect PNG goldens are feasible (there was prior
  precedent with the removed palette golden test).
- **Metrics.** Base metric descriptors come from `provider.AllBase()`. A user
  expression `[filter.]base[.aggregation]` resolves via
  `provider.ResolveForValidation` / `provider.ResolveExpression`. Aggregation
  across levels (file/declaration/commit → directory) is performed by
  `stages.ComputeAggregations`.
- **Export.** `internal/export` deterministically serializes the model tree plus
  a requested set of metrics to JSON (`json.MarshalIndent`, which sorts map
  keys). This is reused as the Suite 2 snapshot format.

## Goals

- Byte-perfect golden coverage of all five visualizations in both PNG and SVG.
- Byte-perfect golden coverage of metric-expression results across all
  providers, aggregations, filters, and cross-level aggregation paths.
- Tests exercise the **same** rendering and aggregation code the CLI ships, so
  wiring changes cannot silently bypass verification.
- Registry-driven coverage: newly-added base metrics, aggregations, and filters
  are picked up automatically without editing the tests.

## Non-goals

- Re-testing each provider's base-metric *computation* from real files/git —
  that is already covered by existing provider tests, which are retained. Suite 2
  pre-populates synthetic base values and verifies the expression/aggregation
  layer on top of them.
- Changing the `Taskfile`, the CLI behavior, or any output format.
- Cross-architecture byte-stability guarantees. Goldens are generated and
  verified in the project's single CI/devcontainer environment; embedded fonts
  and fixed inputs make output deterministic there.

## Design

### Approach selected

**Shared pipeline seam (approach B).** Rather than a standalone harness that
re-lists render stages (which could drift from production and re-introduce the
exact "change slipped past review" failure mode), each viz pipeline is split so
that tests and the CLI share one copy of the render wiring.

### Part 1 — Visualization golden tests

**Production refactor (pure extraction).** For each of the five viz packages
(`treemap`, `radialtree`, `bubbletree`, `spiral`, `scatterviz`), extract the
inline pipeline from the command's `Run` into two functions in the viz package:

- `acquireData(s *pipeline.State)` — the data-acquisition prefix:
  `ScanFilesystem`, `CheckGitRequirement`, `RunProviders`,
  `PopulateDeclarations` (and for spiral, `LoadGitHistory`,
  `GroupGitHistoryByFile`, `ExtractFileHistory`). May remain unexported.
- `RenderPipeline(s *pipeline.State)` — everything after data acquisition:
  `RunAggregations`, `FilterBinaryFiles`, `ExportData`, `ResolveDimensions`,
  `InitDrawingBounds`, `ReserveTitleBounds`, `ReserveFooterBounds`, the
  viz-specific inks/legend/layout/render/label stages, `ApplyTitle`,
  `ApplyFooter`, `WriteCanvas`, `LogResult`. **Exported** so the
  `internal/goldentest` package can call it.

Each command's `Run` becomes: build state → resolve-prefix stages
(`ValidatePaths`, `ExportConfig`, `BuildFilterRules`,
`RegisterSelectionMetrics`, `<viz>.ResolveMetrics`) → `acquireData(s)` →
`<viz>.RenderPipeline(s)`. This is a behavior-preserving extraction; existing
tests (`render_matrix_test.go`, `run_cmd_test.go`, etc.) continue to pass
unchanged.

**Harness (`internal/goldentest`, test-only).** A helper builds one fixed
synthetic `model.Directory` tree (a few nested directories and files) with
file-level base metrics pre-populated deterministically (e.g. `file-lines`,
`file-size`, `file-type`). For spiral, synthetic `GitHistory` / `FileHistory` /
`FileTimeRange` are constructed with pinned commit dates and injected into
`CommonState`.

For each visualization the harness:

1. Builds the `pipeline.State` from a `*stages.CommonState` with `Root`
   pre-set, the viz config, and the viz state.
2. Applies the resolve-prefix stages needed for metric resolution
   (`BuildFilterRules`, `RegisterSelectionMetrics`, `<viz>.ResolveMetrics`) —
   `ValidatePaths`/`ExportConfig` are skipped because there is no real
   target path.
3. Calls `<viz>.RenderPipeline(s)`, writing to a temp `.png` and a temp `.svg`.
4. Reads the bytes and compares them with `goldie`.

This yields one test per `viz × {png, svg}` = **10 golden files** under
`internal/goldentest/testdata/`.

Each viz is configured with simple, file-level + classification metrics
(e.g. `size = file-lines`, `fill = file-type`) so the synthetic base values are
sufficient and no provider/git/declaration computation is required (spiral's
git input is injected directly).

### Part 2 — Metric-expression golden tests

**Synthetic tree.** A deterministic builder constructs a nested directory tree
containing directories, files, per-file declarations (a spread of kinds and
public/private visibility, to exercise declaration filters and kind matching),
and per-file commits (with pinned dates). Base values are **registry-driven**:
for every descriptor in `provider.AllBase()`, each node at the descriptor's
level receives a deterministic synthetic value derived from
`(metricName, nodeID)` — a stable hash mapped into a bounded numeric value, or a
value drawn from a small fixed set for classifications. For file-level base
metrics that declare filters, the `filter.base` keys are also populated so
filtered file-level aggregation has data to read. This means a newly-added base
metric is automatically given synthetic data and covered.

**Expression enumeration.** Candidate expressions are generated from the
registry exactly as `render_matrix_test.go` does — each base, each
`base.aggregation`, each `filter.base`, and each `filter.base.aggregation` —
then filtered to the valid set via `provider.ResolveForValidation`. This is full
registry-driven enumeration (chosen over a hand-curated subset): it inherently
covers all providers, all aggregations, all filters, and all cross-level paths,
costs nothing extra at snapshot time, and auto-tracks future registry additions.

**Snapshot.** All valid expressions are resolved and `stages.ComputeAggregations`
is run once to populate file- and directory-level results across the whole tree.
Every `Measure` value in the tree is then rounded to 6 decimal places (test-side,
leaving `export` unchanged; see Determinism). The tree is serialized to JSON via
`export` (passing every resolved `ResultName` as the requested set) and compared
with `goldie`. The JSON captures every node's file-level base values and
directory-level aggregates, so all cross-level aggregation results are verified
in one snapshot.

### Part 3 — Shared infrastructure

- **Dependency.** Re-add `github.com/sebdah/goldie/v2` (last used at v2.8.0) to
  `go.mod`/`go.sum` via `go get` + `go mod tidy`.
- **Goldie usage.** Tests use `goldie.New(t, …)` and `g.Assert(t, name, bytes)`.
  Goldie honors the `GOLDIE_UPDATE` environment variable, which
  `task update-golden-files` already sets, so new goldens are created and
  updated with no Taskfile change. `task test` runs the comparisons.
- **Placement.** All new code lives in a new `internal/goldentest` package
  (builders + both suites + `testdata/`). No production viz package gains test
  fixtures; the only production change is the pure pipeline extraction described
  in Part 1.
- **No existing test is modified or removed.**

## Determinism

The golden outputs must be reproducible across runs and across trivial commits.
The relevant sources of non-determinism were reviewed:

### Randomness

There is **no** use of `math/rand` (or any other randomness) in production
code. The layout algorithms are pure functions of their input. The bubble-tree
layout — the most likely candidate — uses deterministic front-chain circle
packing plus Welzl's minimum-enclosing-circle algorithm; its only ordering
input is `dir.Dirs` followed by `dir.Files`, then a `slices.SortFunc` by radius
descending. Because the synthetic builders fix node order, output is fully
reproducible.

`slices.SortFunc` is not a *stable* sort, so circles of exactly equal radius
could in principle reorder if Go's sort implementation changes across a Go
upgrade. To remove even this theoretical risk, the synthetic fixtures are
constructed so that sibling size metrics are distinct (no exact ties).

### Floating point

The three output formats are affected differently:

- **SVG — already safe.** The SVG backend formats every coordinate with fixed
  precision (`%.2f` for positions/radii, `%.1f` for stroke widths and
  font sizes, `%.3f` for alpha). Sub-hundredth floating-point noise is quantized
  away by `fmt`, so SVG output is robust to last-bit differences without any
  change. Fixtures are chosen to avoid values sitting exactly on a rounding
  boundary.
- **PNG — accepted single-platform risk.** Raster output cannot be rounded.
  It is deterministic on a fixed platform (fixed-point rasterizer + embedded
  font) but is not guaranteed byte-identical across CPU architectures. Goldens
  are authored and verified in the single CI/devcontainer environment, matching
  the prior project precedent for PNG golden comparison.
- **JSON (Suite 2) — round measures.** `Measure` values are `float64` at full
  precision. The arithmetic is deterministic, but aggregations such as `mean`
  and `range` produce long trailing decimals whose final bit can shift if
  summation order changes — and this repository actively performs
  summation-reordering performance refactors. To keep the goldens clean and
  robust to such trivial changes, Suite 2 **rounds every `Measure` value to 6
  decimal places before serialization**. This is done test-side (walking the
  tree after `ComputeAggregations`, before calling `export`), so the production
  `export` package is left unchanged. `Quantity` (`int64`) and `Classification`
  (string) values are already exact and need no rounding.

### Other

- Embedded `goregular.TTF` font → stable text rasterization.
- Fixed canvas dimensions and fixed synthetic inputs.
- Pinned commit dates for git-derived data (spiral + commit-level metrics).
- Map serialization is key-sorted by `json.MarshalIndent`; model traversal is
  slice-ordered, so no map-iteration ordering leaks into output.

## Testing strategy

- Generate goldens with `task update-golden-files`; verify a clean
  `task test` re-run passes (no diffs).
- Confirm a deliberate, temporary tweak to a render or aggregation stage
  produces a golden diff (manual sanity check during implementation), proving
  the tests actually bite.
- Run `task ci` (`fmt:check`, `mod:check`, `build`, `test`, `lint`) green.

## Files affected (anticipated)

- `cmd/codeviz/treemap_cmd.go`, `radialtree_cmd.go`, `bubbletree_cmd.go`,
  `spiral_cmd.go`, `scatter_cmd.go` — extract pipeline; `Run` calls the new
  functions.
- `internal/treemap/`, `internal/radialtree/`, `internal/bubbletree/`,
  `internal/spiral/`, `internal/scatter/` (viz packages) — new
  `acquireData` + exported `RenderPipeline`.
- `internal/goldentest/` — new package: synthetic builders, Suite 1, Suite 2,
  `testdata/` goldens.
- `go.mod`, `go.sum` — re-add goldie.
