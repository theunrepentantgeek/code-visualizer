# Golden Tests Restoration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restore byte-perfect golden-file testing for every visualization (PNG + SVG) and for the metric-expression layer, driven entirely from synthetic in-memory data.

**Architecture:** Extract a shared, exported `RenderPipeline` from each of the 5 visualization commands so tests exercise the exact wiring the CLI ships (no drift). A new `internal/goldentest` package builds synthetic `model.Directory` trees in memory, runs the real resolve → aggregate → render stages, and compares output bytes with Goldie. A second suite enumerates every valid metric expression from the registry, runs the real aggregation, and snapshots the result tree as JSON via the existing `export` package.

**Tech Stack:** Go 1.26, Goldie v2 (`github.com/sebdah/goldie/v2`), Gomega, the project's `pipeline`/`stages`/viz packages, `fogleman/gg` (PNG) + custom SVG backend, embedded `goregular.TTF` font.

**Spec:** `docs/superpowers/specs/2026-06-28-golden-tests-restoration-design.md`

---

## Background the implementer must know

- **The repo uses `Taskfile` targets:** `task test` (= `go test ./... -count=1`),
  `task build`, `task lint`, `task ci`, and crucially
  `task update-golden-files` (= `GOLDIE_UPDATE=1 go test ./... -count=1`).
  Goldie honors the `GOLDIE_UPDATE` env var, so new goldens are created/updated
  by `task update-golden-files` with **no Taskfile change**.
- **Do not modify or delete any existing test.** Only add new files, plus the
  pure pipeline extraction in the viz command/packages described below.
- **The pipeline** (`internal/pipeline`) is a type-keyed value store. Stages are
  plain functions applied with `pipeline.ApplyFuncX` (one typed arg),
  `ApplyFuncXY` (two), `ApplyFuncXYZ` (three). Each viz command's `Run` builds a
  `*pipeline.State` from a `*stages.CommonState`, a `*config.<Viz>`, and a
  `*<viz>.State`, then applies a list of stages in order. The first stage to
  return an error short-circuits the rest; `s.Err()` returns it.
- **Provider registration:** base metrics are registered globally by
  `filesystem.Register()`, `git.Register()`, `golang.Register()` (see
  `cmd/codeviz/main_test.go`). Tests that resolve metrics must call these in
  `TestMain`.
- **Determinism rules (from the spec):** no randomness exists in production;
  SVG already formats coordinates with fixed precision; PNG is deterministic on
  the single CI platform; Suite 2 rounds every `Measure` to 6 decimal places
  test-side before serialization. Synthetic fixtures use **distinct** sibling
  size values (no exact ties) to avoid unstable-sort sensitivity.

---

## File Structure

**Production changes (pure pipeline extraction — one pair of functions per viz):**

- `internal/treemap/pipeline.go` (new) — `acquireData` + exported `RenderPipeline`.
- `internal/radialtree/pipeline.go` (new) — same pattern.
- `internal/bubbletree/pipeline.go` (new) — same pattern.
- `internal/scatter/pipeline.go` (new) — same pattern.
- `internal/spiral/pipeline.go` (new) — same pattern (includes git-history stages
  in `acquireData`).
- `cmd/codeviz/treemap_cmd.go`, `radialtree_cmd.go`, `bubbletree_cmd.go`,
  `scatter_cmd.go`, `spiral_cmd.go` — each `Run` replaces its inline stage list
  with calls to the two new functions.

**New test package `internal/goldentest`:**

- `main_test.go` — `TestMain` registering providers.
- `model_builder.go` — synthetic `*model.Directory` builder for Suite 1 (file-level base metrics).
- `viz_golden_test.go` — Suite 1: 5 vizzes × {png, svg}.
- `metric_tree_builder.go` — synthetic tree builder for Suite 2 (registry-driven base values across file/declaration/commit levels).
- `metric_golden_test.go` — Suite 2: enumerate expressions → aggregate → round → export JSON snapshot.
- `testdata/` — generated golden files (`*.golden`).

**Dependency:** `go.mod` / `go.sum` — re-add `github.com/sebdah/goldie/v2`.

---

## Task 1: Re-add the Goldie dependency

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add the dependency**

Run:
```bash
go get github.com/sebdah/goldie/v2@v2.8.0
```
Expected: `go.mod` gains `github.com/sebdah/goldie/v2 v2.8.0` (and `go.sum` updated).

- [ ] **Step 2: Verify it resolves and the module stays tidy**

Run:
```bash
go mod tidy && go build ./...
```
Expected: builds cleanly, no errors. (Goldie is unused so far; that is fine —
`go mod tidy` keeps it because the next tasks import it. If `tidy` removes it
because nothing imports it yet, that's expected; it will be re-added when the
first test imports it. To avoid churn, proceed directly to Task 7+ which import
it, then run tidy. For now just confirm `go get` succeeded.)

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "build: re-add goldie v2 golden-file testing dependency"
```

---

## Task 2: Extract treemap RenderPipeline (refactor)

**Files:**
- Create: `internal/treemap/pipeline.go`
- Modify: `cmd/codeviz/treemap_cmd.go` (the `Run` method body)

This is a behavior-preserving extraction. The current `treemap_cmd.go` `Run`
applies this stage list (verbatim): `ValidatePaths, ExportConfig,
BuildFilterRules, RegisterSelectionMetrics, treemap.ResolveMetrics,
ScanFilesystem, CheckGitRequirement, RunProviders, PopulateDeclarations,
RunAggregations, FilterBinaryFiles, ExportData, ResolveDimensions,
InitDrawingBounds, ReserveTitleBounds, ReserveFooterBounds,
treemap.BuildInksStage, treemap.BuildLegendStage, treemap.LayoutStage,
treemap.RenderStage, treemap.LabelStage, treemap.ApplyCanvasBlockLabels,
ApplyTitle, ApplyFooter, WriteCanvas, treemap.LogResult`.

We split it after `PopulateDeclarations` into `acquireData` (data) and
`RenderPipeline` (everything from `RunAggregations` onward). The pre-scan
resolve stages stay in `Run`.

- [ ] **Step 1: Create the extraction file**

Create `internal/treemap/pipeline.go`:

```go
package treemap

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// acquireData runs the data-acquisition stages: scan the filesystem, run
// providers, and populate declarations. Tests that supply a pre-built model
// tree skip this function and inject CommonState.Root directly.
func acquireData(s *pipeline.State) {
	pipeline.ApplyFuncX(s, stages.ScanFilesystem)
	pipeline.ApplyFuncX(s, stages.CheckGitRequirement)
	pipeline.ApplyFuncX(s, stages.RunProviders)
	pipeline.ApplyFuncX(s, stages.PopulateDeclarations)
}

// RenderPipeline runs every stage from aggregation through writing the canvas.
// It assumes CommonState.Root is populated (by acquireData in production, or by
// a test harness in golden tests) and that metrics have been resolved into
// CommonState.Requested. Shared by the CLI command and the golden-test harness
// so both exercise identical wiring.
func RenderPipeline(s *pipeline.State) {
	pipeline.ApplyFuncX(s, stages.RunAggregations)
	pipeline.ApplyFuncX(s, stages.FilterBinaryFiles)
	pipeline.ApplyFuncX(s, stages.ExportData)
	pipeline.ApplyFuncX(s, stages.ResolveDimensions)
	pipeline.ApplyFuncX(s, stages.InitDrawingBounds)
	pipeline.ApplyFuncX(s, stages.ReserveTitleBounds)
	pipeline.ApplyFuncX(s, stages.ReserveFooterBounds)
	pipeline.ApplyFuncXY(s, BuildInksStage)
	pipeline.ApplyFuncXYZ(s, BuildLegendStage)
	pipeline.ApplyFuncXY(s, LayoutStage)
	pipeline.ApplyFuncXY(s, RenderStage)
	pipeline.ApplyFuncXYZ(s, LabelStage)
	pipeline.ApplyFuncXY(s, ApplyCanvasBlockLabels)
	pipeline.ApplyFuncX(s, stages.ApplyTitle)
	pipeline.ApplyFuncX(s, stages.ApplyFooter)
	pipeline.ApplyFuncX(s, stages.WriteCanvas)
	pipeline.ApplyFuncXY(s, LogResult)
}

// _ keeps the config import referenced if future stages need it; remove if unused.
var _ = config.New
```

NOTE: if `goimports`/`gofumpt` flags the unused `config` import, delete the
`config` import line and the `var _ = config.New` line. They are only present in
case the implementer wants to reference config; the stages above do not need it.
Prefer removing them for a clean build.

- [ ] **Step 2: Rewrite the command's Run to use the extracted functions**

In `cmd/codeviz/treemap_cmd.go`, replace the stage list inside `Run` (from the
first `pipeline.ApplyFuncX(s, stages.ValidatePaths)` through
`pipeline.ApplyFuncXY(s, treemap.LogResult)`) with:

```go
	pipeline.ApplyFuncX(s, stages.ValidatePaths)
	pipeline.ApplyFuncX(s, stages.ExportConfig)
	pipeline.ApplyFuncX(s, stages.BuildFilterRules)
	pipeline.ApplyFuncX(s, stages.RegisterSelectionMetrics)
	pipeline.ApplyFuncXYZ(s, treemap.ResolveMetrics)

	treemap.AcquireData(s)
	treemap.RenderPipeline(s)
```

Then export `acquireData` as `AcquireData` in `internal/treemap/pipeline.go`
(rename the function and its doc comment) so the command package can call it.

(Why export `AcquireData`: the command package needs it. The golden harness only
needs `RenderPipeline`.)

- [ ] **Step 3: Build**

Run: `go build ./...`
Expected: compiles. If the `config` import in `pipeline.go` is unused, remove it.

- [ ] **Step 4: Run the existing CLI tests to prove behavior is unchanged**

Run: `go test ./cmd/codeviz/... ./internal/treemap/... -count=1`
Expected: PASS (the existing `render_matrix_test.go` and treemap tests still
pass — output is byte-identical because the stage order is unchanged).

- [ ] **Step 5: Commit**

```bash
git add internal/treemap/pipeline.go cmd/codeviz/treemap_cmd.go
git commit -m "refactor(treemap): extract AcquireData + RenderPipeline from command Run"
```

---

## Task 3: Extract radial RenderPipeline (refactor)

**Files:**
- Create: `internal/radialtree/pipeline.go`
- Modify: `cmd/codeviz/radialtree_cmd.go`

The current radial `Run` stage list (verbatim) matches treemap except the
viz-specific stages are: `radialtree.BuildInksStage` (XY),
`radialtree.BuildLegendStage` (XY), `radialtree.LayoutStage` (XY),
`radialtree.RenderStage` (XY), `radialtree.LogResult` (XY). Radial has **no**
`LabelStage`/`ApplyCanvasBlockLabels`.

- [ ] **Step 1: Create `internal/radialtree/pipeline.go`**

```go
package radialtree

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// AcquireData runs scan, providers, and declaration population.
func AcquireData(s *pipeline.State) {
	pipeline.ApplyFuncX(s, stages.ScanFilesystem)
	pipeline.ApplyFuncX(s, stages.CheckGitRequirement)
	pipeline.ApplyFuncX(s, stages.RunProviders)
	pipeline.ApplyFuncX(s, stages.PopulateDeclarations)
}

// RenderPipeline runs aggregation through writing the canvas, assuming
// CommonState.Root and CommonState.Requested are populated.
func RenderPipeline(s *pipeline.State) {
	pipeline.ApplyFuncX(s, stages.RunAggregations)
	pipeline.ApplyFuncX(s, stages.FilterBinaryFiles)
	pipeline.ApplyFuncX(s, stages.ExportData)
	pipeline.ApplyFuncX(s, stages.ResolveDimensions)
	pipeline.ApplyFuncX(s, stages.InitDrawingBounds)
	pipeline.ApplyFuncX(s, stages.ReserveTitleBounds)
	pipeline.ApplyFuncX(s, stages.ReserveFooterBounds)
	pipeline.ApplyFuncXY(s, BuildInksStage)
	pipeline.ApplyFuncXY(s, BuildLegendStage)
	pipeline.ApplyFuncXY(s, LayoutStage)
	pipeline.ApplyFuncXY(s, RenderStage)
	pipeline.ApplyFuncX(s, stages.ApplyTitle)
	pipeline.ApplyFuncX(s, stages.ApplyFooter)
	pipeline.ApplyFuncX(s, stages.WriteCanvas)
	pipeline.ApplyFuncXY(s, LogResult)
}
```

- [ ] **Step 2: Rewrite `radialtree_cmd.go` Run** — replace the stage list with:

```go
	pipeline.ApplyFuncX(s, stages.ValidatePaths)
	pipeline.ApplyFuncX(s, stages.ExportConfig)
	pipeline.ApplyFuncX(s, stages.BuildFilterRules)
	pipeline.ApplyFuncX(s, stages.RegisterSelectionMetrics)
	pipeline.ApplyFuncXYZ(s, radialtree.ResolveMetrics)

	radialtree.AcquireData(s)
	radialtree.RenderPipeline(s)
```

- [ ] **Step 3: Build** — `go build ./...` → compiles.

- [ ] **Step 4: Test** — `go test ./cmd/codeviz/... ./internal/radialtree/... -count=1` → PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/radialtree/pipeline.go cmd/codeviz/radialtree_cmd.go
git commit -m "refactor(radial): extract AcquireData + RenderPipeline from command Run"
```

---

## Task 4: Extract bubble-tree RenderPipeline (refactor)

**Files:**
- Create: `internal/bubbletree/pipeline.go`
- Modify: `cmd/codeviz/bubbletree_cmd.go`

Bubble-tree's viz stages are `bubbletree.BuildInksStage` (XY),
`bubbletree.BuildLegendStage` (XY), `bubbletree.LayoutStage` (XY),
`bubbletree.RenderStage` (XY), `bubbletree.LogResult` (XY). No label stage.

- [ ] **Step 1: Create `internal/bubbletree/pipeline.go`**

```go
package bubbletree

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// AcquireData runs scan, providers, and declaration population.
func AcquireData(s *pipeline.State) {
	pipeline.ApplyFuncX(s, stages.ScanFilesystem)
	pipeline.ApplyFuncX(s, stages.CheckGitRequirement)
	pipeline.ApplyFuncX(s, stages.RunProviders)
	pipeline.ApplyFuncX(s, stages.PopulateDeclarations)
}

// RenderPipeline runs aggregation through writing the canvas, assuming
// CommonState.Root and CommonState.Requested are populated.
func RenderPipeline(s *pipeline.State) {
	pipeline.ApplyFuncX(s, stages.RunAggregations)
	pipeline.ApplyFuncX(s, stages.FilterBinaryFiles)
	pipeline.ApplyFuncX(s, stages.ExportData)
	pipeline.ApplyFuncX(s, stages.ResolveDimensions)
	pipeline.ApplyFuncX(s, stages.InitDrawingBounds)
	pipeline.ApplyFuncX(s, stages.ReserveTitleBounds)
	pipeline.ApplyFuncX(s, stages.ReserveFooterBounds)
	pipeline.ApplyFuncXY(s, BuildInksStage)
	pipeline.ApplyFuncXY(s, BuildLegendStage)
	pipeline.ApplyFuncXY(s, LayoutStage)
	pipeline.ApplyFuncXY(s, RenderStage)
	pipeline.ApplyFuncX(s, stages.ApplyTitle)
	pipeline.ApplyFuncX(s, stages.ApplyFooter)
	pipeline.ApplyFuncX(s, stages.WriteCanvas)
	pipeline.ApplyFuncXY(s, LogResult)
}
```

- [ ] **Step 2: Rewrite `bubbletree_cmd.go` Run** — replace the stage list with:

```go
	pipeline.ApplyFuncX(s, stages.ValidatePaths)
	pipeline.ApplyFuncX(s, stages.ExportConfig)
	pipeline.ApplyFuncX(s, stages.BuildFilterRules)
	pipeline.ApplyFuncX(s, stages.RegisterSelectionMetrics)
	pipeline.ApplyFuncXYZ(s, bubbletree.ResolveMetrics)

	bubbletree.AcquireData(s)
	bubbletree.RenderPipeline(s)
```

- [ ] **Step 3: Build** — `go build ./...` → compiles.

- [ ] **Step 4: Test** — `go test ./cmd/codeviz/... ./internal/bubbletree/... -count=1` → PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/bubbletree/pipeline.go cmd/codeviz/bubbletree_cmd.go
git commit -m "refactor(bubble-tree): extract AcquireData + RenderPipeline from command Run"
```

---

## Task 5: Extract scatter RenderPipeline (refactor)

**Files:**
- Create: `internal/scatter/pipeline.go`
- Modify: `cmd/codeviz/scatter_cmd.go`

Scatter is imported in the command as `scatterviz "…/internal/scatter"`; the
package name is `scatter`. Its viz stages are `BuildInksStage` (XY),
`BuildLegendStage` (XY), `LayoutStage` (XY), `RenderStage` (XY), `LogResult`
(XY). No label stage.

- [ ] **Step 1: Create `internal/scatter/pipeline.go`**

```go
package scatter

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// AcquireData runs scan, providers, and declaration population.
func AcquireData(s *pipeline.State) {
	pipeline.ApplyFuncX(s, stages.ScanFilesystem)
	pipeline.ApplyFuncX(s, stages.CheckGitRequirement)
	pipeline.ApplyFuncX(s, stages.RunProviders)
	pipeline.ApplyFuncX(s, stages.PopulateDeclarations)
}

// RenderPipeline runs aggregation through writing the canvas, assuming
// CommonState.Root and CommonState.Requested are populated.
func RenderPipeline(s *pipeline.State) {
	pipeline.ApplyFuncX(s, stages.RunAggregations)
	pipeline.ApplyFuncX(s, stages.FilterBinaryFiles)
	pipeline.ApplyFuncX(s, stages.ExportData)
	pipeline.ApplyFuncX(s, stages.ResolveDimensions)
	pipeline.ApplyFuncX(s, stages.InitDrawingBounds)
	pipeline.ApplyFuncX(s, stages.ReserveTitleBounds)
	pipeline.ApplyFuncX(s, stages.ReserveFooterBounds)
	pipeline.ApplyFuncXY(s, BuildInksStage)
	pipeline.ApplyFuncXY(s, BuildLegendStage)
	pipeline.ApplyFuncXY(s, LayoutStage)
	pipeline.ApplyFuncXY(s, RenderStage)
	pipeline.ApplyFuncX(s, stages.ApplyTitle)
	pipeline.ApplyFuncX(s, stages.ApplyFooter)
	pipeline.ApplyFuncX(s, stages.WriteCanvas)
	pipeline.ApplyFuncXY(s, LogResult)
}
```

- [ ] **Step 2: Rewrite `scatter_cmd.go` Run** — replace the stage list with
(note the package alias `scatterviz`):

```go
	pipeline.ApplyFuncX(s, stages.ValidatePaths)
	pipeline.ApplyFuncX(s, stages.ExportConfig)
	pipeline.ApplyFuncX(s, stages.BuildFilterRules)
	pipeline.ApplyFuncX(s, stages.RegisterSelectionMetrics)
	pipeline.ApplyFuncXYZ(s, scatterviz.ResolveMetrics)

	scatterviz.AcquireData(s)
	scatterviz.RenderPipeline(s)
```

- [ ] **Step 3: Build** — `go build ./...` → compiles.

- [ ] **Step 4: Test** — `go test ./cmd/codeviz/... ./internal/scatter/... -count=1` → PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/scatter/pipeline.go cmd/codeviz/scatter_cmd.go
git commit -m "refactor(scatter): extract AcquireData + RenderPipeline from command Run"
```

---

## Task 6: Extract spiral RenderPipeline (refactor)

**Files:**
- Create: `internal/spiral/pipeline.go`
- Modify: `cmd/codeviz/spiral_cmd.go`

Spiral is special: between `ExportData` and `ResolveDimensions` the current
`Run` runs three git-history stages: `LoadGitHistory`,
`GroupGitHistoryByFile`, `ExtractFileHistory`. These are **data acquisition**
(they read a git repo and populate `CommonState.GitHistory`/`FileHistory`/
`FileTimeRange`) and are independent of `RunAggregations`/`FilterBinaryFiles`/
`ExportData` (those touch metric values, not git fields). We therefore move
them into `AcquireData`, which is a behavior-preserving reorder. Spiral's
golden test injects `FileHistory`/`FileTimeRange` directly and skips
`AcquireData`.

Spiral's render-only stages (verbatim, after the git stages): `ResolveDimensions,
InitDrawingBounds, ReserveTitleBounds, ReserveFooterBounds,
BuildTimeBucketsStage (XY), AggregateBucketMetricsStage (XY), BuildInksStage
(XY), BuildLegendStage (XY), LayoutStage (XY), RenderStage (XY), ApplyTitle,
ApplyFooter, WriteCanvas, LogResult (XY)`.

- [ ] **Step 1: Create `internal/spiral/pipeline.go`**

```go
package spiral

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// AcquireData runs scan, providers, declaration population, and git-history
// loading. The git-history stages populate CommonState.FileHistory and
// FileTimeRange, which the render pipeline's time-bucket stages consume. Tests
// that supply synthetic history set those fields directly and skip AcquireData.
func AcquireData(s *pipeline.State) {
	pipeline.ApplyFuncX(s, stages.ScanFilesystem)
	pipeline.ApplyFuncX(s, stages.CheckGitRequirement)
	pipeline.ApplyFuncX(s, stages.RunProviders)
	pipeline.ApplyFuncX(s, stages.PopulateDeclarations)
	pipeline.ApplyFuncX(s, stages.LoadGitHistory)
	pipeline.ApplyFuncX(s, stages.GroupGitHistoryByFile)
	pipeline.ApplyFuncX(s, stages.ExtractFileHistory)
}

// RenderPipeline runs aggregation through writing the canvas, assuming
// CommonState.Root, CommonState.Requested, CommonState.FileHistory and
// CommonState.FileTimeRange are populated.
func RenderPipeline(s *pipeline.State) {
	pipeline.ApplyFuncX(s, stages.RunAggregations)
	pipeline.ApplyFuncX(s, stages.FilterBinaryFiles)
	pipeline.ApplyFuncX(s, stages.ExportData)
	pipeline.ApplyFuncX(s, stages.ResolveDimensions)
	pipeline.ApplyFuncX(s, stages.InitDrawingBounds)
	pipeline.ApplyFuncX(s, stages.ReserveTitleBounds)
	pipeline.ApplyFuncX(s, stages.ReserveFooterBounds)
	pipeline.ApplyFuncXY(s, BuildTimeBucketsStage)
	pipeline.ApplyFuncXY(s, AggregateBucketMetricsStage)
	pipeline.ApplyFuncXY(s, BuildInksStage)
	pipeline.ApplyFuncXY(s, BuildLegendStage)
	pipeline.ApplyFuncXY(s, LayoutStage)
	pipeline.ApplyFuncXY(s, RenderStage)
	pipeline.ApplyFuncX(s, stages.ApplyTitle)
	pipeline.ApplyFuncX(s, stages.ApplyFooter)
	pipeline.ApplyFuncX(s, stages.WriteCanvas)
	pipeline.ApplyFuncXY(s, LogResult)
}
```

- [ ] **Step 2: Rewrite `spiral_cmd.go` Run** — replace the stage list with:

```go
	pipeline.ApplyFuncX(s, stages.ValidatePaths)
	pipeline.ApplyFuncX(s, stages.ExportConfig)
	pipeline.ApplyFuncX(s, stages.BuildFilterRules)
	pipeline.ApplyFuncX(s, stages.RegisterSelectionMetrics)
	pipeline.ApplyFuncXYZ(s, spiral.ResolveMetrics)

	spiral.AcquireData(s)
	spiral.RenderPipeline(s)
```

- [ ] **Step 3: Build** — `go build ./...` → compiles.

- [ ] **Step 4: Test** — `go test ./cmd/codeviz/... ./internal/spiral/... -count=1` → PASS.
  (This confirms the git-stage reorder is behavior-preserving: spiral's existing
  tests and any CLI spiral test still pass.)

- [ ] **Step 5: Commit**

```bash
git add internal/spiral/pipeline.go cmd/codeviz/spiral_cmd.go
git commit -m "refactor(spiral): extract AcquireData + RenderPipeline from command Run"
```

---

## Task 7: Golden-test package skeleton + synthetic model builder

**Files:**
- Create: `internal/goldentest/main_test.go`
- Create: `internal/goldentest/model_builder.go`
- Test: `internal/goldentest/model_builder_test.go`

- [ ] **Step 1: Write a failing test for the model builder**

Create `internal/goldentest/model_builder_test.go`:

```go
package goldentest

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

func TestBuildVizModel_IsDeterministicAndPopulated(t *testing.T) {
	g := NewGomegaWithT(t)

	root := buildVizModel()

	g.Expect(root).NotTo(BeNil())
	g.Expect(root.Dirs).NotTo(BeEmpty(), "expected nested directories")
	g.Expect(root.Files).NotTo(BeEmpty(), "expected root-level files")

	// Every file carries the file-level base metrics the visualizations use.
	f := root.Files[0]
	lines, ok := f.Quantity(filesystem.FileLines)
	g.Expect(ok).To(BeTrue(), "file-lines must be set")
	g.Expect(lines).To(BeNumerically(">", 0))

	_, ok = f.Classification(filesystem.FileType)
	g.Expect(ok).To(BeTrue(), "file-type must be set")
}
```

- [ ] **Step 2: Run it to confirm it fails to compile/run**

Run: `go test ./internal/goldentest/... -run TestBuildVizModel -count=1`
Expected: FAIL — `buildVizModel` undefined.

- [ ] **Step 3: Create `main_test.go` registering providers**

Create `internal/goldentest/main_test.go`:

```go
package goldentest

import (
	"testing"

	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/golang"
)

// TestMain registers all base-metric providers exactly as the CLI does, so the
// global metric registry is populated before any test resolves a metric.
func TestMain(m *testing.M) {
	filesystem.Register()
	git.Register()
	golang.Register()
	m.Run()
}
```

- [ ] **Step 4: Implement `buildVizModel`**

Create `internal/goldentest/model_builder.go`. The tree is fixed and small,
with **distinct** `file-lines` values per file (no exact ties, per the spec's
determinism rule). `file-size` and `file-type` are also set so any viz that
colours by file type has data.

```go
package goldentest

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

// synthFile builds a model.File with the file-level base metrics every
// visualization may read. lines must be distinct across siblings to keep the
// (unstable) radius sort deterministic.
func synthFile(path, name, ext, fileType string, lines, size int64) *model.File {
	f := &model.File{
		Path:      path,
		Name:      name,
		Extension: ext,
	}
	f.SetQuantity(filesystem.FileLines, lines)
	f.SetQuantity(filesystem.FileSize, size)
	f.SetClassification(filesystem.FileType, fileType)

	return f
}

// buildVizModel returns a fixed, deterministic directory tree for the
// visualization golden tests. Three levels deep with a spread of file types and
// distinct sizes so layouts and colour scales are non-trivial.
func buildVizModel() *model.Directory {
	return &model.Directory{
		Path: "root",
		Name: "root",
		Files: []*model.File{
			synthFile("root/readme.md", "readme.md", "md", "Markdown", 40, 800),
			synthFile("root/main.go", "main.go", "go", "Go", 120, 2400),
		},
		Dirs: []*model.Directory{
			{
				Path: "root/src",
				Name: "src",
				Files: []*model.File{
					synthFile("root/src/app.go", "app.go", "go", "Go", 210, 4100),
					synthFile("root/src/util.go", "util.go", "go", "Go", 75, 1500),
					synthFile("root/src/styles.css", "styles.css", "css", "CSS", 33, 660),
				},
				Dirs: []*model.Directory{
					{
						Path: "root/src/deep",
						Name: "deep",
						Files: []*model.File{
							synthFile("root/src/deep/big.go", "big.go", "go", "Go", 305, 6000),
							synthFile("root/src/deep/note.txt", "note.txt", "txt", "Text", 12, 240),
						},
					},
				},
			},
			{
				Path: "root/docs",
				Name: "docs",
				Files: []*model.File{
					synthFile("root/docs/guide.md", "guide.md", "md", "Markdown", 88, 1760),
				},
			},
		},
	}
}
```

NOTE: confirm the exported metric-name constants exist with these names:
`filesystem.FileLines`, `filesystem.FileSize`, `filesystem.FileType` (they are
referenced in `internal/provider/filesystem/register.go`). If a constant has a
different identifier, adjust the references; do not invent names.

- [ ] **Step 5: Run the test to confirm it passes**

Run: `go test ./internal/goldentest/... -run TestBuildVizModel -count=1`
Expected: PASS.

- [ ] **Step 6: Tidy + commit**

```bash
go mod tidy
git add internal/goldentest/ go.mod go.sum
git commit -m "test(goldentest): add package skeleton and synthetic viz model builder"
```

---

## Task 8: Suite 1 — treemap PNG + SVG goldens (establish harness)

**Files:**
- Create: `internal/goldentest/viz_golden_test.go`
- Create (generated): `internal/goldentest/testdata/treemap-png.golden`, `treemap-svg.golden`

This task builds the reusable harness and proves it on treemap. The harness
constructs the pipeline `State` with the synthetic `Root` injected, runs the
resolve-prefix stages plus the viz's exported `RenderPipeline`, then reads and
golden-compares the output bytes.

- [ ] **Step 1: Write the harness + treemap test**

Create `internal/goldentest/viz_golden_test.go`:

```go
package goldentest

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sebdah/goldie/v2"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
	"github.com/theunrepentantgeek/code-visualizer/internal/treemap"
)

// vizFixtureWidth/Height keep golden images small and fast while still
// exercising layout, legend, title and footer.
const (
	vizFixtureWidth  = 320
	vizFixtureHeight = 240
)

// newCommonState builds a CommonState with the synthetic model injected and a
// config whose dimensions are the small fixture size. outputPath drives the
// WriteCanvas format (png/svg) via its extension.
func newCommonState(outputPath string, cfg *config.Config) *stages.CommonState {
	w, h := vizFixtureWidth, vizFixtureHeight
	cfg.ImageSize = &config.ImageSize{Width: &w, Height: &h}

	return &stages.CommonState{
		Output:     outputPath,
		Flags:      &stages.Flags{Config: cfg},
		RootConfig: cfg,
		VizName:    "golden",
		Root:       buildVizModel(),
	}
}

// runViz writes the visualization to outputPath using the supplied render
// closure, then returns the bytes.
func runViz(t *testing.T, outputPath string, render func(*stages.CommonState) error) []byte {
	t.Helper()
	g := NewGomegaWithT(t)

	cfg := config.New()
	common := newCommonState(outputPath, cfg)

	g.Expect(render(common)).To(Succeed())

	data, err := os.ReadFile(outputPath)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(data).NotTo(BeEmpty())

	return data
}

// renderTreemap resolves metrics and runs treemap.RenderPipeline against the
// pre-built model. size=file-lines, fill=file-type mirrors the structure preset.
func renderTreemap(common *stages.CommonState) error {
	cfg := common.RootConfig
	size := "file-lines"
	cfg.Treemap = &config.Treemap{
		Size: &size,
		Fill: &config.MetricSpec{Metric: "file-type"},
	}

	viz := &treemap.State{}
	s := pipeline.NewState(common, cfg.Treemap, viz)

	pipeline.ApplyFuncX(s, stages.BuildFilterRules)
	pipeline.ApplyFuncX(s, stages.RegisterSelectionMetrics)
	pipeline.ApplyFuncXYZ(s, treemap.ResolveMetrics)
	treemap.RenderPipeline(s)

	return s.Err()
}

func TestGolden_Treemap(t *testing.T) {
	cases := []struct {
		name string
		ext  string
	}{
		{"treemap-png", "png"},
		{"treemap-svg", "svg"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "out."+tc.ext)
			data := runViz(t, out, renderTreemap)

			g := goldie.New(t)
			g.Assert(t, tc.name, data)
		})
	}
}
```

NOTES for the implementer:
- Confirm the `config.MetricSpec` literal field is `Metric` (a `metric.Name`).
  Check `internal/config` for the exact struct (it is referenced as
  `config.MetricSpec{Metric: metric.Name(...)}` in `cmd/codeviz/render_matrix_test.go`).
  If `Metric` is typed `metric.Name`, write `Metric: "file-type"` (untyped
  string constant converts) or `Metric: metric.Name("file-type")`.
- Confirm `stages.Flags` has a `Config *config.Config` field (it does — see
  `internal/stages/common.go`). `ExportData`/`ExportConfig` no-op when the
  export paths are empty.

- [ ] **Step 2: Run the test WITHOUT goldens to confirm it fails**

Run: `go test ./internal/goldentest/ -run TestGolden_Treemap -count=1`
Expected: FAIL — `Golden file does not exist` (goldie reports missing fixtures).

- [ ] **Step 3: Generate the goldens**

Run: `GOLDIE_UPDATE=1 go test ./internal/goldentest/ -run TestGolden_Treemap -count=1`
Expected: PASS; creates `internal/goldentest/testdata/treemap-png.golden` and
`treemap-svg.golden`.

- [ ] **Step 4: Re-run normally to confirm stability**

Run: `go test ./internal/goldentest/ -run TestGolden_Treemap -count=1`
Expected: PASS with no diff (proves determinism across runs).

- [ ] **Step 5: Eyeball the SVG golden (sanity)**

Run: `head -c 400 internal/goldentest/testdata/treemap-svg.golden`
Expected: starts with `<svg xmlns="http://www.w3.org/2000/svg" width="320" height="240">`
and contains `<rect .../>` elements — confirming a real render, not an empty image.

- [ ] **Step 6: Commit**

```bash
git add internal/goldentest/viz_golden_test.go internal/goldentest/testdata/treemap-png.golden internal/goldentest/testdata/treemap-svg.golden
git commit -m "test(goldentest): add byte-perfect treemap PNG+SVG golden tests"
```

---

## Task 9: Suite 1 — radial, bubble-tree, scatter goldens

**Files:**
- Modify: `internal/goldentest/viz_golden_test.go`
- Create (generated): `radial-png.golden`, `radial-svg.golden`,
  `bubbletree-png.golden`, `bubbletree-svg.golden`,
  `scatter-png.golden`, `scatter-svg.golden`.

- [ ] **Step 1: Add the three render closures + their tests**

Append to `internal/goldentest/viz_golden_test.go`. Add these imports to the
existing import block: `"…/internal/radialtree"`, `"…/internal/bubbletree"`,
and `scatterviz "…/internal/scatter"`.

```go
// renderRadial: discSize=file-lines, fill=file-type.
func renderRadial(common *stages.CommonState) error {
	cfg := common.RootConfig
	discSize := "file-lines"
	if cfg.Radial == nil {
		cfg.Radial = &config.Radial{}
	}
	cfg.Radial.DiscSize = &discSize
	cfg.Radial.Fill = &config.MetricSpec{Metric: "file-type"}

	viz := &radialtree.State{}
	s := pipeline.NewState(common, cfg.Radial, viz)

	pipeline.ApplyFuncX(s, stages.BuildFilterRules)
	pipeline.ApplyFuncX(s, stages.RegisterSelectionMetrics)
	pipeline.ApplyFuncXYZ(s, radialtree.ResolveMetrics)
	radialtree.RenderPipeline(s)

	return s.Err()
}

// renderBubbletree: size=file-lines, fill=file-type.
func renderBubbletree(common *stages.CommonState) error {
	cfg := common.RootConfig
	size := "file-lines"
	if cfg.Bubbletree == nil {
		cfg.Bubbletree = &config.Bubbletree{}
	}
	cfg.Bubbletree.Size = &size
	cfg.Bubbletree.Fill = &config.MetricSpec{Metric: "file-type"}

	viz := &bubbletree.State{}
	s := pipeline.NewState(common, cfg.Bubbletree, viz)

	pipeline.ApplyFuncX(s, stages.BuildFilterRules)
	pipeline.ApplyFuncX(s, stages.RegisterSelectionMetrics)
	pipeline.ApplyFuncXYZ(s, bubbletree.ResolveMetrics)
	bubbletree.RenderPipeline(s)

	return s.Err()
}

// renderScatter: x-axis=file-size, y-axis=file-lines, size=file-lines, fill=file-type.
func renderScatter(common *stages.CommonState) error {
	cfg := common.RootConfig
	x := "file-size"
	y := "file-lines"
	size := "file-lines"
	if cfg.Scatter == nil {
		cfg.Scatter = &config.Scatter{}
	}
	cfg.Scatter.XAxis = &x
	cfg.Scatter.YAxis = &y
	cfg.Scatter.Size = &size
	cfg.Scatter.Fill = &config.MetricSpec{Metric: "file-type"}

	viz := &scatterviz.State{}
	s := pipeline.NewState(common, cfg.Scatter, viz)

	pipeline.ApplyFuncX(s, stages.BuildFilterRules)
	pipeline.ApplyFuncX(s, stages.RegisterSelectionMetrics)
	pipeline.ApplyFuncXYZ(s, scatterviz.ResolveMetrics)
	scatterviz.RenderPipeline(s)

	return s.Err()
}

func TestGolden_Radial(t *testing.T)     { runVizGolden(t, "radial", renderRadial) }
func TestGolden_Bubbletree(t *testing.T) { runVizGolden(t, "bubbletree", renderBubbletree) }
func TestGolden_Scatter(t *testing.T)    { runVizGolden(t, "scatter", renderScatter) }

// runVizGolden renders the named viz to PNG and SVG and golden-compares both.
func runVizGolden(t *testing.T, name string, render func(*stages.CommonState) error) {
	for _, ext := range []string{"png", "svg"} {
		t.Run(name+"-"+ext, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "out."+ext)
			data := runViz(t, out, render)

			g := goldie.New(t)
			g.Assert(t, name+"-"+ext, data)
		})
	}
}
```

NOTE: also refactor `TestGolden_Treemap` (Task 8) to call `runVizGolden(t,
"treemap", renderTreemap)` for consistency, OR leave it as-is. If you refactor,
re-run `GOLDIE_UPDATE=1` only if the golden *names* change (they do not:
`treemap-png`/`treemap-svg` are preserved), so no regeneration is needed.

Confirm each viz's config field names against `internal/config/radialtree.go`
(`DiscSize`, `Fill`), `internal/config/bubbletree.go` (`Size`, `Fill`),
`internal/config/scatter.go` (`XAxis`, `YAxis`, `Size`, `Fill`). These are
verified in the spec research; adjust only if a name differs.

- [ ] **Step 2: Confirm the tests fail without goldens**

Run: `go test ./internal/goldentest/ -run 'TestGolden_(Radial|Bubbletree|Scatter)' -count=1`
Expected: FAIL — missing golden files.

- [ ] **Step 3: Generate goldens**

Run: `GOLDIE_UPDATE=1 go test ./internal/goldentest/ -run 'TestGolden_(Radial|Bubbletree|Scatter)' -count=1`
Expected: PASS; creates 6 new `.golden` files.

- [ ] **Step 4: Re-run normally**

Run: `go test ./internal/goldentest/ -count=1`
Expected: PASS, no diffs, all viz goldens stable.

- [ ] **Step 5: Commit**

```bash
git add internal/goldentest/viz_golden_test.go internal/goldentest/testdata/
git commit -m "test(goldentest): add radial, bubble-tree, scatter PNG+SVG golden tests"
```

---

## Task 10: Suite 1 — spiral goldens (synthetic git history)

**Files:**
- Modify: `internal/goldentest/viz_golden_test.go`
- Create: `internal/goldentest/git_fixture.go`
- Create (generated): `spiral-png.golden`, `spiral-svg.golden`

Spiral needs `CommonState.FileHistory` and `CommonState.FileTimeRange`. We build
them synthetically from the model files with pinned timestamps, then call
`spiral.RenderPipeline` (which begins at `RunAggregations` and runs the
time-bucket stages).

- [ ] **Step 1: Create the synthetic git-history fixture**

Create `internal/goldentest/git_fixture.go`:

```go
package goldentest

import (
	"time"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// buildSpiralHistory assigns each file a small, deterministic set of commits
// with pinned dates spread across a fixed window, and returns the FileHistory
// and FileTimeRange maps the spiral pipeline consumes. Pinned dates keep the
// time-bucketing reproducible.
func buildSpiralHistory(root *model.Directory) (
	map[*model.File][]stages.CommitRef,
	map[*model.File]stages.TimeRange,
) {
	history := make(map[*model.File][]stages.CommitRef)
	ranges := make(map[*model.File]stages.TimeRange)

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	var idx int
	model.WalkFiles(root, func(f *model.File) {
		// Two commits per file at deterministic offsets.
		first := base.AddDate(0, idx%6, 0)         // months 0..5
		second := first.AddDate(0, 0, 10+idx%5)    // 10..14 days later

		c1 := &git.Commit{Hash: "c1-" + f.Path, Message: "create " + f.Name}
		c2 := &git.Commit{Hash: "c2-" + f.Path, Message: "update " + f.Name}

		history[f] = []stages.CommitRef{
			{Commit: c1, When: first},
			{Commit: c2, When: second},
		}
		ranges[f] = stages.TimeRange{Earliest: first, Latest: second}

		idx++
	})

	return history, ranges
}
```

NOTES:
- Confirm `model.WalkFiles(root, func(*model.File))` exists (used in
  `internal/stages/aggregation.go`). It does.
- Confirm `git.Commit` fields `Hash`, `Message` exist (they do — see
  `internal/provider/git/commit.go`). `Author`/`Committer` are `git.Signature`;
  leave them zero unless a render stage dereferences them. If a spiral stage
  needs author data, set `Author: git.Signature{...}` — check
  `internal/spiral` for any `.Author` usage before generating goldens.
- Confirm `stages.CommitRef{Commit, When}` and `stages.TimeRange{Earliest,
  Latest}` field names (see `internal/stages/git_history.go`). They match.

- [ ] **Step 2: Add the spiral render closure + test**

Append to `internal/goldentest/viz_golden_test.go` (add `"…/internal/spiral"`
to imports):

```go
// renderSpiral: size=file-lines, fill=file-type, with synthetic git history
// injected so the time-bucket stages have data.
func renderSpiral(common *stages.CommonState) error {
	cfg := common.RootConfig
	size := "file-lines"
	if cfg.Spiral == nil {
		cfg.Spiral = &config.Spiral{}
	}
	cfg.Spiral.Size = &size
	cfg.Spiral.Fill = &config.MetricSpec{Metric: "file-type"}

	common.FileHistory, common.FileTimeRange = buildSpiralHistory(common.Root)

	viz := &spiral.State{}
	s := pipeline.NewState(common, cfg.Spiral, viz)

	pipeline.ApplyFuncX(s, stages.BuildFilterRules)
	pipeline.ApplyFuncX(s, stages.RegisterSelectionMetrics)
	pipeline.ApplyFuncXYZ(s, spiral.ResolveMetrics)
	spiral.RenderPipeline(s)

	return s.Err()
}

func TestGolden_Spiral(t *testing.T) { runVizGolden(t, "spiral", renderSpiral) }
```

NOTE: `config.Spiral` is created by `config.New()` with `Resolution` and
`Labels` already set, so reuse `cfg.Spiral` rather than replacing it — only set
`Size` and `Fill`. The `if cfg.Spiral == nil` guard above is defensive; since
`config.New()` initializes it, the existing `Resolution`/`Labels` defaults are
preserved.

- [ ] **Step 3: Confirm failure without goldens**

Run: `go test ./internal/goldentest/ -run TestGolden_Spiral -count=1`
Expected: FAIL — missing golden files. (If it fails earlier with a pipeline
error, inspect `s.Err()` — most likely a missing field the spiral stages read;
fix the fixture per the NOTES above, then continue.)

- [ ] **Step 4: Generate goldens**

Run: `GOLDIE_UPDATE=1 go test ./internal/goldentest/ -run TestGolden_Spiral -count=1`
Expected: PASS; creates `spiral-png.golden`, `spiral-svg.golden`.

- [ ] **Step 5: Re-run + sanity-check SVG**

Run: `go test ./internal/goldentest/ -run TestGolden_Spiral -count=1 && head -c 200 internal/goldentest/testdata/spiral-svg.golden`
Expected: PASS; SVG header present with width="320" height="240".

- [ ] **Step 6: Commit**

```bash
git add internal/goldentest/git_fixture.go internal/goldentest/viz_golden_test.go internal/goldentest/testdata/spiral-png.golden internal/goldentest/testdata/spiral-svg.golden
git commit -m "test(goldentest): add spiral PNG+SVG golden tests with synthetic git history"
```

---

## Task 11: Suite 2 — synthetic metric tree builder (registry-driven)

**Files:**
- Create: `internal/goldentest/metric_tree_builder.go`
- Test: `internal/goldentest/metric_tree_builder_test.go`

This builds a tree whose nodes carry deterministic synthetic base values for
**every** base metric in `provider.AllBase()`, keyed by the descriptor's level
(file / declaration / commit). A newly added base metric is auto-populated.

- [ ] **Step 1: Write a failing test**

Create `internal/goldentest/metric_tree_builder_test.go`:

```go
package goldentest

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

func TestBuildMetricTree_PopulatesEveryFileBaseMetric(t *testing.T) {
	g := NewGomegaWithT(t)

	root := buildMetricTree()
	g.Expect(root.Files).NotTo(BeEmpty())

	// Every file-level base metric must have a value on the first file.
	f := root.Files[0]
	for _, desc := range provider.AllBaseForLevelFile() {
		switch desc.Kind {
		case metric.Quantity:
			_, ok := f.Quantity(desc.Name)
			g.Expect(ok).To(BeTrue(), "file metric %q (quantity) must be set", desc.Name)
		case metric.Measure:
			_, ok := f.Measure(desc.Name)
			g.Expect(ok).To(BeTrue(), "file metric %q (measure) must be set", desc.Name)
		case metric.Classification:
			_, ok := f.Classification(desc.Name)
			g.Expect(ok).To(BeTrue(), "file metric %q (classification) must be set", desc.Name)
		}
	}

	g.Expect(model.CountFiles(root)).To(BeNumerically(">", 1))
	g.Expect(model.CountDeclarations(root)).To(BeNumerically(">", 0))
	g.Expect(model.CountCommits(root)).To(BeNumerically(">", 0))
}
```

NOTE: this test references a small helper `provider.AllBaseForLevelFile()`. The
registry already exposes `provider.AllBaseForLevel(level)` (see
`internal/provider/base_registry.go`). Use that instead — replace
`provider.AllBaseForLevelFile()` with
`provider.AllBaseForLevel(metric.LevelFile)`. (Do **not** add a new exported
function to production; use the existing one.)

- [ ] **Step 2: Run to confirm failure**

Run: `go test ./internal/goldentest/ -run TestBuildMetricTree -count=1`
Expected: FAIL — `buildMetricTree` undefined.

- [ ] **Step 3: Implement the builder**

Create `internal/goldentest/metric_tree_builder.go`:

```go
package goldentest

import (
	"hash/fnv"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// classificationValues is a small fixed vocabulary used for synthetic
// classification base values, chosen deterministically by hash.
var classificationValues = []string{"alpha", "beta", "gamma", "delta"}

// synthInt returns a deterministic int64 in [1, 1000] derived from a seed.
func synthInt(seed string) int64 {
	return int64(hashOf(seed)%1000) + 1
}

// synthFloat returns a deterministic float64 in [0, 100) derived from a seed.
func synthFloat(seed string) float64 {
	return float64(hashOf(seed)%10000) / 100.0
}

// synthClass returns a deterministic classification value derived from a seed.
func synthClass(seed string) string {
	return classificationValues[hashOf(seed)%uint64(len(classificationValues))]
}

func hashOf(seed string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(seed))

	return h.Sum64()
}

// setBaseMetric writes a deterministic synthetic value for desc onto the
// container, keyed by the descriptor's kind. nodeID makes the value unique per
// node so aggregation produces non-trivial results.
func setBaseMetric(mc baseMetricSetter, desc provider.BaseMetricDescriptor, nodeID string) {
	seed := string(desc.Name) + "|" + nodeID
	switch desc.Kind {
	case metric.Quantity:
		mc.SetQuantity(desc.Name, synthInt(seed))
	case metric.Measure:
		mc.SetMeasure(desc.Name, synthFloat(seed))
	case metric.Classification:
		mc.SetClassification(desc.Name, synthClass(seed))
	}

	// File-level metrics that declare filters also need filter.base values so
	// filtered file-level aggregation has data to read.
	for _, fn := range desc.Filters {
		filtered := metric.MetricExpression{Filter: fn, Base: desc.Name}.ResultName()
		switch desc.Kind {
		case metric.Quantity:
			mc.SetQuantity(filtered, synthInt(seed+"|"+string(fn)))
		case metric.Measure:
			mc.SetMeasure(filtered, synthFloat(seed+"|"+string(fn)))
		case metric.Classification:
			mc.SetClassification(filtered, synthClass(seed+"|"+string(fn)))
		}
	}
}

// baseMetricSetter is the subset of model.MetricContainer used above.
type baseMetricSetter interface {
	SetQuantity(metric.Name, int64)
	SetMeasure(metric.Name, float64)
	SetClassification(metric.Name, string)
}

// declarationKinds gives a representative spread covering both visibilities and
// several kinds so declaration filters and kind-matching are exercised.
var declarationSpecs = []struct {
	name       string
	kind       string
	visibility string
}{
	{"PublicType", model.DeclKindType, "public"},
	{"privateType", model.DeclKindType, "private"},
	{"PublicFunc", model.DeclKindFunction, "public"},
	{"privateFunc", model.DeclKindFunction, "private"},
	{"PublicMethod", model.DeclKindMethod, "public"},
	{"privateConst", model.DeclKindConstant, "private"},
}

// newDeclarations builds a fixed set of declarations for a file, each carrying
// every declaration-level base metric.
func newDeclarations(fileID string, declLevel []provider.BaseMetricDescriptor) []*model.Declaration {
	decls := make([]*model.Declaration, 0, len(declarationSpecs))
	for _, ds := range declarationSpecs {
		d := &model.Declaration{Name: ds.name, Kind: ds.kind, Visibility: ds.visibility}
		for _, desc := range declLevel {
			setBaseMetric(d, desc, fileID+"/"+ds.name)
		}
		decls = append(decls, d)
	}

	return decls
}

// newCommits builds a fixed set of commits for a file, each carrying every
// commit-level base metric.
func newCommits(fileID string, commitLevel []provider.BaseMetricDescriptor) []*model.Commit {
	commits := make([]*model.Commit, 0, 2)
	for i := 0; i < 2; i++ {
		c := &model.Commit{Hash: fileID + "-commit"}
		for _, desc := range commitLevel {
			setBaseMetric(c, desc, fileID+"/commit")
		}
		commits = append(commits, c)
	}

	return commits
}

// newMetricFile builds a file populated with all file-level base metrics plus
// declarations and commits carrying their level's base metrics.
func newMetricFile(path, name, ext string,
	fileLevel, declLevel, commitLevel []provider.BaseMetricDescriptor,
) *model.File {
	f := &model.File{Path: path, Name: name, Extension: ext}
	for _, desc := range fileLevel {
		setBaseMetric(f, desc, path)
	}
	f.Declarations = newDeclarations(path, declLevel)
	f.Commits = newCommits(path, commitLevel)

	return f
}

// buildMetricTree returns a fixed nested directory tree where every node level
// carries deterministic synthetic values for every base metric in the registry.
func buildMetricTree() *model.Directory {
	fileLevel := provider.AllBaseForLevel(metric.LevelFile)
	declLevel := provider.AllBaseForLevel(metric.LevelDeclaration)
	commitLevel := provider.AllBaseForLevel(metric.LevelCommit)

	mk := func(path, name, ext string) *model.File {
		return newMetricFile(path, name, ext, fileLevel, declLevel, commitLevel)
	}

	return &model.Directory{
		Path: "root",
		Name: "root",
		Files: []*model.File{
			mk("root/a.go", "a.go", "go"),
			mk("root/b.go", "b.go", "go"),
		},
		Dirs: []*model.Directory{
			{
				Path:  "root/sub",
				Name:  "sub",
				Files: []*model.File{mk("root/sub/c.go", "c.go", "go")},
				Dirs: []*model.Directory{
					{
						Path:  "root/sub/deep",
						Name:  "deep",
						Files: []*model.File{mk("root/sub/deep/d.go", "d.go", "go")},
					},
				},
			},
		},
	}
}
```

NOTES:
- Confirm `model.CountFiles`, `model.CountDeclarations`, `model.CountCommits`,
  `model.WalkFiles` exist (referenced in `internal/stages/aggregation.go`).
- Confirm `model.DeclKind*` constants exist (see `internal/model/declaration.go`):
  `DeclKindType`, `DeclKindFunction`, `DeclKindMethod`, `DeclKindConstant`,
  `DeclKindInterface`, `DeclKindStruct`, `DeclKindVariable`.
- `metric.MetricExpression{Filter, Base}.ResultName()` produces the
  `filter.base` key (see `internal/metric/expression.go`).
- `*model.Declaration` and `*model.Commit` both embed `model.MetricContainer`,
  so they satisfy `baseMetricSetter`.

- [ ] **Step 4: Run the test to confirm it passes**

Run: `go test ./internal/goldentest/ -run TestBuildMetricTree -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/goldentest/metric_tree_builder.go internal/goldentest/metric_tree_builder_test.go
git commit -m "test(goldentest): add registry-driven synthetic metric tree builder"
```

---

## Task 12: Suite 2 — expression enumeration → aggregate → JSON golden

**Files:**
- Create: `internal/goldentest/metric_golden_test.go`
- Create (generated): `internal/goldentest/testdata/metric-expressions.golden`

This enumerates every valid expression from the registry, runs the real
`stages.ComputeAggregations`, rounds measures to 6 dp, and snapshots the tree as
JSON via the `export` package.

- [ ] **Step 1: Write the test**

Create `internal/goldentest/metric_golden_test.go`:

```go
package goldentest

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sebdah/goldie/v2"

	"github.com/theunrepentantgeek/code-visualizer/internal/export"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// candidateExpressions builds every expression worth probing for each base
// metric: the bare metric, base×aggregation, filter×base, and
// filter×base×aggregation. Mirrors cmd/codeviz/render_matrix_test.go so the set
// tracks the registry automatically.
func candidateExpressions() []string {
	names := make([]string, 0)
	for _, desc := range provider.AllBase() {
		base := string(desc.Name)
		names = append(names, base)
		for _, agg := range desc.Aggregations {
			names = append(names, base+"."+string(agg))
		}
		for _, fn := range desc.Filters {
			filtered := string(fn) + "." + base
			names = append(names, filtered)
			for _, agg := range desc.Aggregations {
				names = append(names, filtered+"."+string(agg))
			}
		}
	}

	return names
}

// validExpressions resolves each candidate at directory level and keeps the
// ones the registry accepts, de-duplicated and deterministic.
func validExpressions(t *testing.T) []provider.ResolvedMetric {
	t.Helper()

	seen := make(map[string]bool)
	resolved := make([]provider.ResolvedMetric, 0)
	for _, name := range candidateExpressions() {
		if seen[name] {
			continue
		}
		seen[name] = true

		r, err := provider.ResolveForValidation(metric.Name(name))
		if err != nil {
			continue
		}
		resolved = append(resolved, r)
	}

	return resolved
}

// requestedNames returns every metric name to include in the JSON snapshot:
// the file-level base names (so file rows show their inputs) plus every
// resolved expression's ResultName (so directory aggregates appear).
func requestedNames(resolved []provider.ResolvedMetric) []metric.Name {
	seen := make(map[metric.Name]bool)
	names := make([]metric.Name, 0)
	add := func(n metric.Name) {
		if !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
	}
	for _, desc := range provider.AllBase() {
		add(desc.Name)
	}
	for _, r := range resolved {
		add(r.ResultName)
	}

	return names
}

// roundMeasures rounds every Measure value in the tree to 6 decimal places, to
// keep the JSON snapshot robust to last-bit floating-point differences from
// aggregation-order changes. Quantities and classifications are exact already.
func roundMeasures(root *model.Directory, names []metric.Name) {
	round := func(mc interface {
		Measure(metric.Name) (float64, bool)
		SetMeasure(metric.Name, float64)
	}) {
		for _, n := range names {
			if v, ok := mc.Measure(n); ok {
				mc.SetMeasure(n, math.Round(v*1e6)/1e6)
			}
		}
	}

	var walkDir func(d *model.Directory)
	walkDir = func(d *model.Directory) {
		round(&d.MetricContainer)
		for _, f := range d.Files {
			round(&f.MetricContainer)
		}
		for _, sub := range d.Dirs {
			walkDir(sub)
		}
	}
	walkDir(root)
}

func TestGolden_MetricExpressions(t *testing.T) {
	g := NewGomegaWithT(t)

	root := buildMetricTree()
	resolved := validExpressions(t)
	g.Expect(resolved).NotTo(BeEmpty(), "registry should yield valid expressions")

	g.Expect(stages.ComputeAggregations(root, resolved)).To(Succeed())

	names := requestedNames(resolved)
	roundMeasures(root, names)

	out := filepath.Join(t.TempDir(), "metrics.json")
	g.Expect(export.Export(root, names, out)).To(Succeed())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	gold := goldie.New(t)
	gold.Assert(t, "metric-expressions", data)
}
```

NOTE: add `"github.com/theunrepentantgeek/code-visualizer/internal/stages"` to
the imports (used for `stages.ComputeAggregations`). Confirm the signature
`stages.ComputeAggregations(root *model.Directory, expressions []provider.ResolvedMetric) error`
(see `internal/stages/aggregation.go`). Confirm
`export.Export(root *model.Directory, requested []metric.Name, outputPath string) error`
(see `internal/export/export.go`).

- [ ] **Step 2: Confirm failure without the golden**

Run: `go test ./internal/goldentest/ -run TestGolden_MetricExpressions -count=1`
Expected: FAIL — missing golden file.

- [ ] **Step 3: Generate the golden**

Run: `GOLDIE_UPDATE=1 go test ./internal/goldentest/ -run TestGolden_MetricExpressions -count=1`
Expected: PASS; creates `internal/goldentest/testdata/metric-expressions.golden`.

- [ ] **Step 4: Inspect the golden for coverage (sanity)**

Run:
```bash
grep -c '"name"' internal/goldentest/testdata/metric-expressions.golden
grep -o '\.sum"\|\.mean"\|\.distinct"\|\.mode"\|public\.\|private\.' internal/goldentest/testdata/metric-expressions.golden | sort -u
```
Expected: the file contains many metric keys including aggregated forms
(`.sum`, `.mean`, etc.) and filtered forms (`public.`, `private.`), confirming
cross-level and filter coverage. Measure values should show at most 6 decimal
places.

- [ ] **Step 5: Re-run normally to confirm stability**

Run: `go test ./internal/goldentest/ -run TestGolden_MetricExpressions -count=1`
Expected: PASS, no diff.

- [ ] **Step 6: Commit**

```bash
git add internal/goldentest/metric_golden_test.go internal/goldentest/testdata/metric-expressions.golden
git commit -m "test(goldentest): add registry-driven metric-expression JSON golden"
```

---

## Task 13: Full verification + prove the tests bite

**Files:** none (verification only), plus a throwaway edit reverted before commit.

- [ ] **Step 1: Run the whole suite**

Run: `task test`
Expected: PASS, including all `internal/goldentest` goldens and every
pre-existing test (unchanged).

- [ ] **Step 2: Prove a viz golden actually bites (temporary tamper)**

Temporarily change one synthetic value in `model_builder.go` (e.g. change the
first `synthFile(... 40, 800)` to `... 41, 800`), then:

Run: `go test ./internal/goldentest/ -run TestGolden_Treemap -count=1`
Expected: FAIL — golden mismatch (the render changed). This proves the PNG/SVG
goldens detect output changes. **Revert the edit** afterwards:

```bash
git checkout -- internal/goldentest/model_builder.go
```

- [ ] **Step 3: Prove a metric golden actually bites (temporary tamper)**

Temporarily change the rounding precision in `metric_golden_test.go` from `1e6`
to `1e2` and re-run:

Run: `go test ./internal/goldentest/ -run TestGolden_MetricExpressions -count=1`
Expected: FAIL — golden mismatch (values rounded differently). **Revert**:

```bash
git checkout -- internal/goldentest/metric_golden_test.go
```

- [ ] **Step 4: Confirm `task update-golden-files` finds the new goldens with no Taskfile change**

Run: `task update-golden-files && git status --porcelain`
Expected: PASS and **no** changed golden files (regenerating produces identical
bytes → determinism holds, and the existing target already covers the new
tests).

- [ ] **Step 5: Full CI**

Run: `task ci`
Expected: `fmt:check`, `mod:check`, `build`, `test`, `lint` all green. If
`lint`/`fmt:check` flags the new files, run `task fmt` and fix lints, then
re-run. (Run `task lint` / `task ci` via an Explore subagent per the repo's
agent workflow rules, returning only failures.)

- [ ] **Step 6: Final commit (if fmt/lint produced changes)**

```bash
git add -A
git commit -m "test(goldentest): finalize golden tests; gofumpt + lint clean"
```

---

## Self-Review (completed by plan author)

**Spec coverage:**
- Visualization goldens (PNG + SVG, all 5 vizzes) → Tasks 8, 9, 10. ✓
- Synthetic in-memory directory structure → Task 7 (`buildVizModel`) + Task 11
  (`buildMetricTree`); scanner bypassed by injecting `CommonState.Root`. ✓
- Shared pipeline seam (approach B) → Tasks 2–6 export `RenderPipeline`, used by
  both the CLI command and the harness. ✓
- Metric-expression goldens covering all providers/aggregations/filters/
  cross-level → Task 11 (registry-driven base values at file/declaration/commit
  levels) + Task 12 (registry-driven enumeration + `ComputeAggregations` +
  `export` JSON). ✓
- Reuse `export` for snapshot → Task 12. ✓
- Re-add goldie; Taskfile unchanged → Task 1; `update-golden-files` honored via
  `GOLDIE_UPDATE` (Tasks 8–12 generation steps; Task 13 Step 4). ✓
- Determinism: no randomness (verified), SVG fixed-precision (inherent), measure
  rounding to 6 dp → Task 12 `roundMeasures`; distinct sibling sizes → Task 7. ✓
- No existing test modified/removed → only new files + pure extraction; Tasks
  2–6 Step 4 re-run existing tests to prove behavior unchanged. ✓

**Placeholder scan:** No "TBD"/"implement later". Each code step contains
complete code. NOTES flag identifiers to confirm against the codebase (exact
constant/field names) rather than leaving blanks.

**Type consistency:** `AcquireData`/`RenderPipeline` exported uniformly across
all five viz packages; harness closures use `config.New()` + per-viz config
sections; `stages.CommitRef`/`TimeRange`, `git.Commit`,
`provider.ResolvedMetric`, `provider.AllBaseForLevel`, `export.Export`,
`stages.ComputeAggregations` signatures referenced consistently.
