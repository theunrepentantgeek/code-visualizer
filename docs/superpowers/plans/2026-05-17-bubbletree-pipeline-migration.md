# Bubbletree Pipeline Migration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the bubbletree visualization command to the pipeline scaffold proven by the treemap migration, and extract the duplicated ink helpers into a new `internal/inks` package shared by treemap, bubbletree, and radial.

**Architecture:** Add `internal/inks` with the four ink helpers currently duplicated in `cmd/codeviz/ink_builder.go` and `internal/treemap/inks.go`. Move bubble render and ink code from `cmd/codeviz/bubble_canvas.go` into a new `internal/bubbletree/{render,inks}.go`. Add `internal/bubbletree/{state,stages}.go` with six viz-specific pipeline stages. Rewrite `BubbletreeCmd.Run` as a `pipeline.Run` composition that reuses `internal/stages` for the shared lifecycle steps. Delete `shape_inks.go`, `ink_builder.go`, `bubble_canvas.go`, and `bubble_canvas_test.go` once their contents have moved.

**Tech Stack:** Go 1.26.1, Kong, eris, Gomega, fogleman/gg. Toolchain via Taskfile (`task build`, `task test`, `task lint`, `task ci`).

**Reference spec:** [docs/superpowers/specs/2026-05-17-bubbletree-pipeline-migration-design.md](../specs/2026-05-17-bubbletree-pipeline-migration-design.md)

**Branch:** `feature/bubbletree-pipeline` (already created off `main`).

**Workflow reminder:** Run `task lint` and `task ci` via the Explore subagent — never inline — per the agent workflow rules in `.github/copilot-instructions.md`.

---

## File Structure

| Path                                      | Responsibility                                                                                                                  |
| ----------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `internal/inks/inks.go` (new)             | `BuildMetricInk`, `MetricValueForFile`, `CollectNumericValues`, `CollectDistinctTypes`. Free functions, no viz domain.          |
| `internal/inks/inks_test.go` (new)        | Table-driven tests for all four exported funcs.                                                                                 |
| `internal/treemap/inks.go` (modify)       | Keep `Inks` and `BuildInks`. Replace local copies of the four helpers with calls to `internal/inks`.                            |
| `internal/treemap/render.go` (modify)     | Two `metricValueForFile` call sites switch to `inks.MetricValueForFile`.                                                        |
| `internal/bubbletree/render.go` (new)     | All bubble render code, moved from `cmd/codeviz/bubble_canvas.go`. Exports `RenderToCanvas`.                                    |
| `internal/bubbletree/inks.go` (new)       | `Inks` struct + `BuildInks(...)` wrapping `inks.BuildMetricInk`.                                                                |
| `internal/bubbletree/state.go` (new)      | `State` struct + `Common()` + `IncludeBinary()` methods.                                                                        |
| `internal/bubbletree/stages.go` (new)     | Six stages: `ResolveMetrics`, `BuildInksStage`, `BuildLegendStage`, `LayoutStage`, `RenderStage`, `LogResult`.                  |
| `internal/bubbletree/render_test.go` (new)| Comprehensive PNG/SVG/JPG + labels modes + ink kinds, modelled on `internal/treemap/render_test.go`.                            |
| `internal/bubbletree/inks_test.go` (new)  | Bubble-specific ink + arc-font-size tests, ported from `cmd/codeviz/bubble_canvas_test.go`.                                     |
| `cmd/codeviz/bubble_canvas.go` (delete)   | Contents moved to `internal/bubbletree/render.go`.                                                                              |
| `cmd/codeviz/bubble_canvas_test.go` (delete) | Tests split between `internal/bubbletree/inks_test.go` and `render_test.go`.                                                 |
| `cmd/codeviz/ink_builder.go` (delete)     | All four helpers now in `internal/inks`.                                                                                        |
| `cmd/codeviz/shape_inks.go` (delete)      | Replaced by per-viz local structs.                                                                                              |
| `cmd/codeviz/radial_canvas.go` (modify)   | `buildMetricInk` → `inks.BuildMetricInk`; `metricValueForFile` → `inks.MetricValueForFile`; `shapeInks` → local `radialInks`.   |
| `cmd/codeviz/radial_canvas_test.go` (modify) | `metricValueForFile` → `inks.MetricValueForFile`; `shapeInks` references updated.                                            |
| `cmd/codeviz/spiral_canvas.go` (modify)   | `shapeInks` → local `spiralInks`. No ink-helper imports — spiral uses its own `buildBucketInk` / `spiralMetricValue`.            |
| `cmd/codeviz/bubbletree_cmd.go` (modify)  | `Run()` becomes a `pipeline.Run` composition. Kong struct, `Validate`, `validateConfig`, `mergeConfigAndValidate`, `applyOverrides` unchanged. `renderAndLog`, `resolveFillMetric`, `resolveLabels` removed. |
| `cmd/codeviz/main_test.go` (modify)       | `TestCollectDistinctTypes_ReturnsSortedTypes` removed (relocated to `internal/inks`).                                           |

---

## Task 1: Create `internal/inks` package

Extract the four duplicated helpers into a new package. No callers change yet — this task lands the package and its tests, leaving the existing copies in place so CI stays green.

**Files:**
- Create: `internal/inks/inks.go`
- Create: `internal/inks/inks_test.go`

- [ ] **Step 1: Create `internal/inks/inks.go`**

```go
// Package inks provides shared Ink construction helpers used by every
// visualization that derives colours from per-file model data.
package inks

import (
	"image/color"
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// BuildMetricInk creates an Ink for a given metric, using the appropriate
// constructor based on the metric kind (numeric vs categorical). Returns a
// fixed-colour ink when the metric is unknown or when no values are present.
func BuildMetricInk(
	root *model.Directory,
	m metric.Name,
	palName palette.PaletteName,
	fallback color.RGBA,
) canvas.Ink {
	d, ok := provider.GetDescriptor(m)
	if !ok {
		return canvas.FixedInk(fallback)
	}

	pal := palette.GetPalette(palName)

	if d.Kind == metric.Quantity || d.Kind == metric.Measure {
		values := CollectNumericValues(root, m)
		if len(values) == 0 {
			return canvas.FixedInk(fallback)
		}

		return canvas.NumericInk(m, values, pal)
	}

	types := CollectDistinctTypes(root, m)

	return canvas.CategoricalInk(m, types, pal)
}

// MetricValueForFile builds a MetricValue from a file's data for the given
// ink. Returns the zero MetricValue when file is nil, when the ink is fixed,
// or when the file has no value for the ink's metric.
func MetricValueForFile(file *model.File, ink canvas.Ink) canvas.MetricValue {
	if file == nil {
		return canvas.MetricValue{}
	}

	info := ink.Info()

	switch info.Kind {
	case canvas.InkNumeric:
		m := info.MetricName
		if v, ok := file.Quantity(m); ok {
			return canvas.MetricValue{Kind: metric.Quantity, Quantity: int(v)}
		}

		if v, ok := file.Measure(m); ok {
			return canvas.MetricValue{Kind: metric.Measure, Measure: v}
		}

		return canvas.MetricValue{}
	case canvas.InkCategorical:
		m := info.MetricName
		if v, ok := file.Classification(m); ok {
			return canvas.MetricValue{Kind: metric.Classification, Category: v}
		}

		return canvas.MetricValue{}
	default:
		return canvas.MetricValue{}
	}
}

// CollectNumericValues walks the directory tree and returns every file's
// numeric value for metric m (quantity preferred, then measure).
func CollectNumericValues(root *model.Directory, m metric.Name) []float64 {
	var values []float64

	model.WalkFiles(root, func(f *model.File) {
		values = append(values, extractNumeric(f, m))
	})

	return values
}

// CollectDistinctTypes returns the sorted distinct classification values
// observed for metric m across all files under root.
func CollectDistinctTypes(root *model.Directory, m metric.Name) []string {
	seen := map[string]bool{}

	model.WalkFiles(root, func(f *model.File) {
		if v, ok := f.Classification(m); ok {
			seen[v] = true
		}
	})

	types := make([]string, 0, len(seen))
	for t := range seen {
		types = append(types, t)
	}

	slices.Sort(types)

	return types
}

func extractNumeric(f *model.File, m metric.Name) float64 {
	if v, ok := f.Quantity(m); ok {
		return float64(v)
	}

	if v, ok := f.Measure(m); ok {
		return v
	}

	return 0
}
```

- [ ] **Step 2: Create `internal/inks/inks_test.go`**

```go
package inks_test

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

func TestMain(m *testing.M) {
	filesystem.Register()
	m.Run()
}

func fallbackColour() color.RGBA {
	return color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
}

func makeFile(name, ext string, size int64) *model.File {
	f := &model.File{Path: "/p/" + name, Name: name, Extension: ext}
	f.SetQuantity(filesystem.FileSize, size)
	f.SetClassification(filesystem.FileType, ext)

	return f
}

func TestCollectDistinctTypes_ReturnsSortedTypes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Path: "/p", Name: "p",
		Files: []*model.File{
			makeFile("z.go", "go", 1),
			makeFile("a.md", "md", 1),
			makeFile("m.txt", "txt", 1),
		},
	}

	g.Expect(inks.CollectDistinctTypes(root, filesystem.FileType)).
		To(Equal([]string{"go", "md", "txt"}))
}

func TestCollectNumericValues_WalksAllFiles(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Path: "/p",
		Files: []*model.File{
			makeFile("a.go", "go", 10),
			makeFile("b.go", "go", 20),
		},
		Dirs: []*model.Directory{
			{Path: "/p/sub", Files: []*model.File{makeFile("c.go", "go", 30)}},
		},
	}

	g.Expect(inks.CollectNumericValues(root, filesystem.FileSize)).
		To(ConsistOf(10.0, 20.0, 30.0))
}

func TestBuildMetricInk_NumericKind(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Path: "/p",
		Files: []*model.File{
			makeFile("a.go", "go", 100),
			makeFile("b.go", "go", 200),
		},
	}

	ink := inks.BuildMetricInk(root, filesystem.FileSize, palette.Temperature, fallbackColour())

	g.Expect(ink.Info().Kind).To(Equal(canvas.InkNumeric))
}

func TestBuildMetricInk_CategoricalKind(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Path: "/p",
		Files: []*model.File{
			makeFile("a.go", "go", 100),
			makeFile("b.rs", "rs", 200),
		},
	}

	ink := inks.BuildMetricInk(root, filesystem.FileType, palette.Categorization, fallbackColour())

	g.Expect(ink.Info().Kind).To(Equal(canvas.InkCategorical))
}

func TestBuildMetricInk_UnknownMetricFallsBackToFixed(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Path: "/p"}

	ink := inks.BuildMetricInk(root, metric.Name("does-not-exist"), palette.Temperature, fallbackColour())

	g.Expect(ink.Info().Kind).To(Equal(canvas.InkFixed))
}

func TestBuildMetricInk_EmptyNumericFallsBackToFixed(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Numeric descriptor exists, but the tree has no files → no values.
	root := &model.Directory{Path: "/p"}

	ink := inks.BuildMetricInk(root, filesystem.FileSize, palette.Temperature, fallbackColour())

	g.Expect(ink.Info().Kind).To(Equal(canvas.InkFixed))
}

func TestMetricValueForFile_NumericInk(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	file := makeFile("a.go", "go", 42)
	root := &model.Directory{Path: "/p", Files: []*model.File{file}}
	ink := inks.BuildMetricInk(root, filesystem.FileSize, palette.Temperature, fallbackColour())

	mv := inks.MetricValueForFile(file, ink)

	g.Expect(mv.Kind).To(Equal(metric.Quantity))
	g.Expect(mv.Quantity).To(Equal(42))
}

func TestMetricValueForFile_CategoricalInk(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	file := makeFile("a.go", "go", 1)
	root := &model.Directory{Path: "/p", Files: []*model.File{file}}
	ink := inks.BuildMetricInk(root, filesystem.FileType, palette.Categorization, fallbackColour())

	mv := inks.MetricValueForFile(file, ink)

	g.Expect(mv.Kind).To(Equal(metric.Classification))
	g.Expect(mv.Category).To(Equal("go"))
}

func TestMetricValueForFile_NilFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(inks.MetricValueForFile(nil, canvas.FixedInk(fallbackColour()))).
		To(Equal(canvas.MetricValue{}))
}

func TestMetricValueForFile_FixedInk(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	file := makeFile("a.go", "go", 1)

	g.Expect(inks.MetricValueForFile(file, canvas.FixedInk(fallbackColour()))).
		To(Equal(canvas.MetricValue{}))
}
```

- [ ] **Step 3: Run the new tests**

Run: `go test ./internal/inks/...`
Expected: all tests pass.

- [ ] **Step 4: Lint via Explore subagent**

Dispatch the `Explore` subagent to run `task lint` and report only failing linters / offending file:line / message (per the agent workflow rules). Fix any issues inline.

- [ ] **Step 5: Commit**

```bash
git add internal/inks/inks.go internal/inks/inks_test.go
git commit -m "feat(inks): add internal/inks package with shared ink helpers"
```

---

## Task 2: Convert `internal/treemap` to consume `internal/inks`

Replace treemap's local copies of the four helpers with calls into `internal/inks`. The treemap render tests (`internal/treemap/render_test.go`) are the safety net.

**Files:**
- Modify: `internal/treemap/inks.go`
- Modify: `internal/treemap/render.go`

- [ ] **Step 1: Replace the helper bodies in `internal/treemap/inks.go`**

Inside `BuildInks`, the two calls to `buildMetricInk` become `inks.BuildMetricInk`:

```go
inks.Fill = pkginks.BuildMetricInk(root, fillMetric, fillPaletteName, defaultFill)
if borderMetric != "" {
	inks.Border = pkginks.BuildMetricInk(root, borderMetric, borderPaletteName, structuralBorder)
}
```

(Import the new package as `pkginks "github.com/theunrepentantgeek/code-visualizer/internal/inks"` to avoid shadowing the local `inks` variable / package name. If the local variable rename is preferred, adjust accordingly — pick one and apply it consistently.)

Then delete from the same file:

- `func buildMetricInk(...)`
- `func metricValueForFile(...)`
- `func collectNumericValues(...)`
- `func collectDistinctTypes(...)`
- `func extractNumeric(...)`

Remove now-unused imports (`slices`, `provider`, possibly `metric`).

- [ ] **Step 2: Update `internal/treemap/render.go`**

The two call sites at lines ~128–129:

```go
fillMV := metricValueForFile(file, inks.Fill)
borderMV := metricValueForFile(file, inks.Border)
```

become:

```go
fillMV := pkginks.MetricValueForFile(file, inks.Fill)
borderMV := pkginks.MetricValueForFile(file, inks.Border)
```

Add the import `pkginks "github.com/theunrepentantgeek/code-visualizer/internal/inks"`.

- [ ] **Step 3: Run treemap tests**

Run: `go test ./internal/treemap/...`
Expected: all tests pass (including `TestRenderTreemapToCanvas_PNG`, `_SVG`, `_JPG`, `TestBuildTreemapInks_Numeric`, `TestBuildTreemapInks_Categorical`, `TestTreemapDynBorderWidth`).

- [ ] **Step 4: Lint via Explore subagent**

Dispatch `Explore` to run `task lint`. Fix any reported issues.

- [ ] **Step 5: Commit**

```bash
git add internal/treemap/inks.go internal/treemap/render.go
git commit -m "refactor(treemap): consume internal/inks; drop local helper copies"
```

---

## Task 3: Flip `cmd/codeviz` callers to `internal/inks`; delete `ink_builder.go`

Update bubble and radial code in `cmd/codeviz` to call the new package, then delete the old helper file. `shape_inks` stays for one more task.

**Files:**
- Modify: `cmd/codeviz/bubble_canvas.go`
- Modify: `cmd/codeviz/bubble_canvas_test.go`
- Modify: `cmd/codeviz/radial_canvas.go`
- Modify: `cmd/codeviz/radial_canvas_test.go`
- Modify: `cmd/codeviz/main_test.go`
- Delete: `cmd/codeviz/ink_builder.go`

- [ ] **Step 1: Update `cmd/codeviz/bubble_canvas.go` call sites**

In `buildBubbleInks`:

```go
inks.fill = pkginks.BuildMetricInk(root, fillMetric, fillPaletteName, bubbleDefaultFileFill)
if borderMetric != "" {
	inks.border = pkginks.BuildMetricInk(root, borderMetric, borderPaletteName, bubbleDefaultBorder)
}
```

In `addBubbleFileDiscsWalk`:

```go
fillMV := pkginks.MetricValueForFile(f, inks.fill)
borderMV := pkginks.MetricValueForFile(f, inks.border)
```

Add the import `pkginks "github.com/theunrepentantgeek/code-visualizer/internal/inks"`. Remove any imports that become unused.

- [ ] **Step 2: Update `cmd/codeviz/bubble_canvas_test.go`**

The line `fileMV := metricValueForFile(file, inks.border)` (around line 250) becomes:

```go
fileMV := pkginks.MetricValueForFile(file, inks.border)
```

Add the matching import.

- [ ] **Step 3: Update `cmd/codeviz/radial_canvas.go` call sites**

In `buildRadialInks`:

```go
inks.fill = pkginks.BuildMetricInk(root, fillMetric, fillPaletteName, radialDefaultFileFill)
if borderMetric != "" {
	inks.border = pkginks.BuildMetricInk(root, borderMetric, borderPaletteName, radialDefaultBorder)
}
```

In `addRadialDisc`:

```go
fillMV := pkginks.MetricValueForFile(e.file, inks.fill)
borderMV := pkginks.MetricValueForFile(e.file, inks.border)
```

Add the import, drop now-unused ones.

- [ ] **Step 4: Update `cmd/codeviz/radial_canvas_test.go`**

The line `fileMV := metricValueForFile(fileEntry.file, inks.border)` (around line 328) becomes:

```go
fileMV := pkginks.MetricValueForFile(fileEntry.file, inks.border)
```

Add the matching import.

- [ ] **Step 5: Remove `TestCollectDistinctTypes_ReturnsSortedTypes` from `cmd/codeviz/main_test.go`**

Delete the test function (lines ~192–215). The behaviour is now covered by the equivalent test in `internal/inks/inks_test.go`.

- [ ] **Step 6: Delete `cmd/codeviz/ink_builder.go`**

Run: `git rm cmd/codeviz/ink_builder.go`

- [ ] **Step 7: Run the full test suite**

Run: `go test ./...`
Expected: all tests pass.

- [ ] **Step 8: Lint via Explore subagent**

Dispatch `Explore` to run `task lint`. Fix issues.

- [ ] **Step 9: Commit**

```bash
git add cmd/codeviz/bubble_canvas.go cmd/codeviz/bubble_canvas_test.go \
        cmd/codeviz/radial_canvas.go cmd/codeviz/radial_canvas_test.go \
        cmd/codeviz/main_test.go cmd/codeviz/ink_builder.go
git commit -m "refactor(cmd): consume internal/inks; delete ink_builder.go"
```

---

## Task 4: Replace `shapeInks` with viz-local structs; delete `shape_inks.go`

The shared struct only existed to deduplicate `{fill, border}` pairs. Each viz now keeps its own identical struct. This is mechanical.

**Files:**
- Modify: `cmd/codeviz/bubble_canvas.go`
- Modify: `cmd/codeviz/bubble_canvas_test.go`
- Modify: `cmd/codeviz/radial_canvas.go`
- Modify: `cmd/codeviz/radial_canvas_test.go`
- Modify: `cmd/codeviz/spiral_canvas.go`
- Modify: any spiral tests that reference `shapeInks`
- Delete: `cmd/codeviz/shape_inks.go`

- [ ] **Step 1: Add a local struct at the top of `cmd/codeviz/bubble_canvas.go`**

Below the existing `const (...)` block:

```go
// bubbleInks holds the Ink instances for a bubble render pass.
type bubbleInks struct {
	fill   canvas.Ink
	border canvas.Ink
}
```

- [ ] **Step 2: Replace every `shapeInks` reference in bubble files**

In `cmd/codeviz/bubble_canvas.go` and `cmd/codeviz/bubble_canvas_test.go`, replace every occurrence of `shapeInks` with `bubbleInks`. Field accesses (`inks.fill`, `inks.border`) remain unchanged.

- [ ] **Step 3: Add `radialInks` to `cmd/codeviz/radial_canvas.go`**

Below the existing `const (...)` block:

```go
// radialInks holds the Ink instances for a radial render pass.
type radialInks struct {
	fill   canvas.Ink
	border canvas.Ink
}
```

- [ ] **Step 4: Replace every `shapeInks` reference in radial files**

In `cmd/codeviz/radial_canvas.go` and `cmd/codeviz/radial_canvas_test.go`, replace `shapeInks` with `radialInks`.

- [ ] **Step 5: Add `spiralInks` to `cmd/codeviz/spiral_canvas.go`**

Below the existing `const (...)` block:

```go
// spiralInks holds the Ink instances for a spiral render pass.
type spiralInks struct {
	fill   canvas.Ink
	border canvas.Ink
}
```

- [ ] **Step 6: Replace every `shapeInks` reference in spiral files**

In `cmd/codeviz/spiral_canvas.go` and any related test files, replace `shapeInks` with `spiralInks`. Verify via:

```bash
grep -rn "shapeInks" cmd/codeviz/
```

Expected: only `cmd/codeviz/shape_inks.go` matches.

- [ ] **Step 7: Delete `cmd/codeviz/shape_inks.go`**

Run: `git rm cmd/codeviz/shape_inks.go`

- [ ] **Step 8: Run the full test suite**

Run: `go test ./...`
Expected: all tests pass.

- [ ] **Step 9: Lint via Explore subagent**

Dispatch `Explore` to run `task lint`. Fix issues.

- [ ] **Step 10: Commit**

```bash
git add cmd/codeviz/bubble_canvas.go cmd/codeviz/bubble_canvas_test.go \
        cmd/codeviz/radial_canvas.go cmd/codeviz/radial_canvas_test.go \
        cmd/codeviz/spiral_canvas.go cmd/codeviz/shape_inks.go
# Add any other spiral test files that referenced shapeInks (verify via git status).
git commit -m "refactor(cmd): replace shapeInks with viz-local ink structs"
```

---

## Task 5: Move bubble render + ink code into `internal/bubbletree`

Relocate `bubble_canvas.go` into `internal/bubbletree/render.go` and create `internal/bubbletree/inks.go` with `Inks` + `BuildInks`. Exported symbols replace unexported ones at the package boundary.

**Files:**
- Create: `internal/bubbletree/render.go`
- Create: `internal/bubbletree/inks.go`
- Create: `internal/bubbletree/inks_test.go` (port of bubble canvas tests)
- Modify: `cmd/codeviz/bubbletree_cmd.go` (call sites switch to `bubbletree.BuildInks` / `bubbletree.RenderToCanvas`)
- Delete: `cmd/codeviz/bubble_canvas.go`
- Delete: `cmd/codeviz/bubble_canvas_test.go`

- [ ] **Step 1: Create `internal/bubbletree/inks.go`**

```go
package bubbletree

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	pkginks "github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

var (
	defaultFileFill = color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
	defaultDirFill  = color.RGBA{R: 0x66, G: 0x99, B: 0xCC, A: 0xFF}
	defaultBorder   = color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
	labelColour     = color.RGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF}
	bgColour        = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
)

// Inks holds the fill and border Ink instances for a bubble render pass.
type Inks struct {
	Fill   canvas.Ink
	Border canvas.Ink
}

// BuildInks creates fill and border inks from metric configuration.
// A zero borderMetric yields a fixed default border ink.
func BuildInks(
	root *model.Directory,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
	borderMetric metric.Name,
	borderPaletteName palette.PaletteName,
) Inks {
	inks := Inks{
		Border: canvas.FixedInk(defaultBorder),
	}

	inks.Fill = pkginks.BuildMetricInk(root, fillMetric, fillPaletteName, defaultFileFill)
	if borderMetric != "" {
		inks.Border = pkginks.BuildMetricInk(root, borderMetric, borderPaletteName, defaultBorder)
	}

	return inks
}
```

- [ ] **Step 2: Create `internal/bubbletree/render.go`**

Move the entire contents of `cmd/codeviz/bubble_canvas.go` into this new file with these changes:

- Package declaration becomes `package bubbletree`.
- Remove the imports for `internal/bubbletree` (the package is itself bubbletree now) and `internal/palette` (only used in `buildBubbleInks`, which has moved to `inks.go`).
- Remove the now-relocated `buildBubbleInks` function and the colour `var` block — those live in `inks.go`.
- Replace references to `bubbleInks` with `Inks` (exported), and field accesses `inks.fill` / `inks.border` with `inks.Fill` / `inks.Border`.
- Rename `renderBubbleToCanvas` to `RenderToCanvas` (exported). All other helpers (`addBubbleBackground`, `indexBubbleNodes`, `addBubbleDirDiscs`, `addBubbleFileDiscs`, `addBubbleLabels`, `addBubbleDirLabel`, `addBubbleFileLabel`, `bubbleArcFontSize`, `bubbleDirEntry`, `collectBubbleDirEntries`, `indexBubbleNodesWalk`, `addBubbleFileDiscsWalk`) stay unexported.
- Replace constants prefixed with `bubble*` with shorter unexported names where natural (e.g. `bubbleDirOpacity` → `dirOpacity`, `bubbleBorderWidth` → `borderWidth`, etc.). This is optional polish — if the rename feels noisy, keep the `bubble*` prefix; correctness is what matters.
- `pkginks.MetricValueForFile` calls remain as added in Task 3.

Verify the file compiles with:

```bash
go build ./internal/bubbletree/...
```

Expected: no errors. (Test code still lives in `cmd/codeviz` for the moment and references the soon-to-be-deleted unexported names — that will be fixed in step 4. CI is briefly red between steps 2 and 5 but each commit is at a green checkpoint.)

- [ ] **Step 3: Create `internal/bubbletree/inks_test.go`**

Port the relevant tests from `cmd/codeviz/bubble_canvas_test.go`:

```go
package bubbletree_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/bubbletree"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

func TestMain(m *testing.M) {
	filesystem.Register()
	m.Run()
}

func makeFile(name, ext string, size int64) *model.File {
	f := &model.File{Name: name, Extension: ext}
	f.SetQuantity(filesystem.FileSize, size)
	f.SetClassification(filesystem.FileType, ext)

	return f
}

func TestBuildInks_DefaultColours(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeFile("a.go", "go", 100)},
	}

	inks := bubbletree.BuildInks(root, "", "", "", "")

	g.Expect(inks.Fill.Info().Kind).To(Equal(canvas.InkFixed))
	g.Expect(inks.Border.Info().Kind).To(Equal(canvas.InkFixed))
}

func TestBuildInks_NumericFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeFile("a.go", "go", 100),
			makeFile("b.go", "go", 200),
		},
	}

	inks := bubbletree.BuildInks(root, filesystem.FileSize, palette.Temperature, "", "")

	g.Expect(inks.Fill.Info().Kind).To(Equal(canvas.InkNumeric))
	g.Expect(inks.Border.Info().Kind).To(Equal(canvas.InkFixed))
}

func TestBuildInks_BorderMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeFile("a.go", "go", 100),
			makeFile("b.rs", "rs", 200),
		},
	}

	inks := bubbletree.BuildInks(
		root,
		filesystem.FileSize, palette.Temperature,
		filesystem.FileType, palette.Categorization,
	)

	g.Expect(inks.Fill.Info().Kind).To(Equal(canvas.InkNumeric))
	g.Expect(inks.Border.Info().Kind).To(Equal(canvas.InkCategorical))
}
```

The four `TestBubbleArcFontSize_*` tests in `cmd/codeviz/bubble_canvas_test.go` reference the unexported `bubbleArcFontSize`. They cannot run from `_test` outside the package, so port them into an in-package test file:

Create `internal/bubbletree/render_internal_test.go` with `package bubbletree` (same-package test):

```go
package bubbletree

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestArcFontSize_EmptyLabel(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(bubbleArcFontSize("", 100)).To(Equal(0.0))
}

func TestArcFontSize_TinyRadius(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(bubbleArcFontSize("test", 10)).To(Equal(0.0))
}

func TestArcFontSize_NormalLabel(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fontSize := bubbleArcFontSize("normal", 100)

	g.Expect(fontSize).To(BeNumerically(">", 0))
	g.Expect(fontSize).To(BeNumerically(">=", bubbleMinArcFontSize))
	g.Expect(fontSize).To(BeNumerically("<=", bubbleDefaultFontSize))
}

func TestArcFontSize_LongLabel(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	longLabel := "this_is_a_very_long_label_that_cannot_fit_on_a_small_circle"

	g.Expect(bubbleArcFontSize(longLabel, 30)).To(Equal(0.0))
}
```

(If you renamed `bubbleArcFontSize` to `arcFontSize` and the constants to `minArcFontSize` / `defaultFontSize` in step 2, update the references here accordingly.)

- [ ] **Step 4: Update `cmd/codeviz/bubbletree_cmd.go`**

In the `renderAndLog` function:

```go
nodes := bubbletree.Layout(root, width, height, size, labels)
inks := bubbletree.BuildInks(root, fillMetric, fillPaletteName, borderMetric, borderPaletteName)
cv := bubbletree.RenderToCanvas(&nodes, root, width, height, inks)
```

Remove the now-unused imports `internal/canvas`, `internal/palette` if they were only used by the deleted helpers (re-check after the edit).

- [ ] **Step 5: Delete the old files**

```bash
git rm cmd/codeviz/bubble_canvas.go cmd/codeviz/bubble_canvas_test.go
```

- [ ] **Step 6: Run the full test suite**

Run: `go test ./...`
Expected: all tests pass.

- [ ] **Step 7: Lint via Explore subagent**

Dispatch `Explore` to run `task lint`. Fix issues.

- [ ] **Step 8: Commit**

```bash
git add internal/bubbletree/render.go internal/bubbletree/inks.go \
        internal/bubbletree/inks_test.go internal/bubbletree/render_internal_test.go \
        cmd/codeviz/bubbletree_cmd.go \
        cmd/codeviz/bubble_canvas.go cmd/codeviz/bubble_canvas_test.go
git commit -m "refactor(bubbletree): move render+inks into internal/bubbletree"
```

---

## Task 6: Add comprehensive render coverage

Add end-to-end render tests modelled on `internal/treemap/render_test.go`. This locks bubble render behaviour before the pipeline rewrite.

**Files:**
- Create: `internal/bubbletree/render_test.go`

- [ ] **Step 1: Create `internal/bubbletree/render_test.go`**

```go
package bubbletree_test

import (
	"bytes"
	"encoding/xml"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/bubbletree"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

func testRoot() *model.Directory {
	return &model.Directory{
		Path: "root",
		Files: []*model.File{
			makeFile("main.go", "go", 100),
			makeFile("style.css", "css", 50),
		},
		Dirs: []*model.Directory{
			{
				Path:  "root/pkg",
				Files: []*model.File{makeFile("lib.go", "go", 200)},
			},
		},
	}
}

func TestRenderBubbleToCanvas_PNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := testRoot()
	nodes := bubbletree.Layout(root, 1000, 800, filesystem.FileSize, bubbletree.LabelFoldersOnly)
	inks := bubbletree.BuildInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := bubbletree.RenderToCanvas(&nodes, root, 1000, 800, inks)

	out := filepath.Join(t.TempDir(), "bubble.png")
	g.Expect(cv.Render(out)).To(Succeed())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("png"))
}

func TestRenderBubbleToCanvas_SVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := testRoot()
	nodes := bubbletree.Layout(root, 800, 600, filesystem.FileSize, bubbletree.LabelFoldersOnly)
	inks := bubbletree.BuildInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := bubbletree.RenderToCanvas(&nodes, root, 800, 600, inks)

	out := filepath.Join(t.TempDir(), "bubble.svg")
	g.Expect(cv.Render(out)).To(Succeed())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	dec := xml.NewDecoder(bytes.NewReader(data))

	var rootElement string

	for {
		tok, xmlErr := dec.Token()
		if xmlErr != nil {
			break
		}

		if se, ok := tok.(xml.StartElement); ok {
			rootElement = se.Name.Local

			break
		}
	}

	g.Expect(rootElement).To(Equal("svg"))
}

func TestRenderBubbleToCanvas_JPG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := testRoot()
	nodes := bubbletree.Layout(root, 400, 300, filesystem.FileSize, bubbletree.LabelFoldersOnly)
	inks := bubbletree.BuildInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := bubbletree.RenderToCanvas(&nodes, root, 400, 300, inks)

	out := filepath.Join(t.TempDir(), "bubble.jpg")
	g.Expect(cv.Render(out)).To(Succeed())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("jpeg"))
}

func TestRenderBubbleToCanvas_EmptyDirectory(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Path: "empty"}
	nodes := bubbletree.Layout(root, 800, 800, filesystem.FileSize, bubbletree.LabelFoldersOnly)
	inks := bubbletree.BuildInks(root, "", "", "", "")

	cv := bubbletree.RenderToCanvas(&nodes, root, 800, 800, inks)

	out := filepath.Join(t.TempDir(), "empty.png")
	g.Expect(cv.Render(out)).To(Succeed())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info.Size()).To(BeNumerically(">", 0))
}

func TestRenderBubbleToCanvas_LabelsAll(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := testRoot()
	nodes := bubbletree.Layout(root, 1000, 800, filesystem.FileSize, bubbletree.LabelAll)
	inks := bubbletree.BuildInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := bubbletree.RenderToCanvas(&nodes, root, 1000, 800, inks)

	out := filepath.Join(t.TempDir(), "labels-all.svg")
	g.Expect(cv.Render(out)).To(Succeed())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	// LabelAll should emit <text> elements for file labels in the SVG.
	g.Expect(bytes.Contains(data, []byte("<text"))).To(BeTrue(),
		"expected SVG to contain at least one <text> element with LabelAll")
}

func TestRenderBubbleToCanvas_LabelsNone(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := testRoot()
	nodes := bubbletree.Layout(root, 1000, 800, filesystem.FileSize, bubbletree.LabelNone)
	inks := bubbletree.BuildInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := bubbletree.RenderToCanvas(&nodes, root, 1000, 800, inks)

	out := filepath.Join(t.TempDir(), "labels-none.svg")
	g.Expect(cv.Render(out)).To(Succeed())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	// LabelNone should emit no <text> or <textPath> elements.
	g.Expect(bytes.Contains(data, []byte("<text"))).To(BeFalse(),
		"expected SVG to contain no <text> elements with LabelNone")
}

func TestRenderBubbleToCanvas_CategoricalFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := testRoot()
	nodes := bubbletree.Layout(root, 1000, 800, filesystem.FileSize, bubbletree.LabelFoldersOnly)
	inks := bubbletree.BuildInks(
		root,
		filesystem.FileType, palette.Categorization,
		filesystem.FileSize, palette.Temperature,
	)
	cv := bubbletree.RenderToCanvas(&nodes, root, 1000, 800, inks)

	out := filepath.Join(t.TempDir(), "cat.png")
	g.Expect(cv.Render(out)).To(Succeed())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("png"))
}
```

- [ ] **Step 2: Run the new tests**

Run: `go test ./internal/bubbletree/...`
Expected: all tests pass.

- [ ] **Step 3: Lint via Explore subagent**

Dispatch `Explore` to run `task lint`. Fix issues.

- [ ] **Step 4: Commit**

```bash
git add internal/bubbletree/render_test.go
git commit -m "test(bubbletree): add PNG/SVG/JPG render coverage"
```

---

## Task 7: Add `internal/bubbletree/state.go`

The pipeline state struct + the methods needed by shared stages.

**Files:**
- Create: `internal/bubbletree/state.go`

- [ ] **Step 1: Create `internal/bubbletree/state.go`**

```go
package bubbletree

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// State is the pipeline state for the bubbletree visualization.
type State struct {
	stages.CommonState

	Config             *config.Bubbletree
	IncludeBinaryFiles bool

	// Resolved during the pipeline:
	Size          metric.Name
	FillMetric    metric.Name
	FillPalette   palette.PaletteName
	BorderMetric  metric.Name
	BorderPalette palette.PaletteName
	Labels        LabelMode
	Inks          Inks
	Nodes         BubbleNode
	LegendConfig  *canvas.LegendConfig
}

// Common exposes the embedded CommonState so shared stages can mutate it.
func (s *State) Common() *stages.CommonState { return &s.CommonState }

// IncludeBinary lets State satisfy stages.BinaryFilterToggler.
func (s *State) IncludeBinary() bool { return s.IncludeBinaryFiles }
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/bubbletree/...`
Expected: no errors.

- [ ] **Step 3: Lint via Explore subagent**

Dispatch `Explore` to run `task lint`. Fix issues.

- [ ] **Step 4: Commit**

```bash
git add internal/bubbletree/state.go
git commit -m "feat(bubbletree): add pipeline State"
```

---

## Task 8: Add `internal/bubbletree/stages.go`

The six viz-specific stages. `ResolveMetrics` absorbs the `resolveFillMetric` and `resolveLabels` helpers currently in `cmd/codeviz/bubbletree_cmd.go`.

**Files:**
- Create: `internal/bubbletree/stages.go`

- [ ] **Step 1: Create `internal/bubbletree/stages.go`**

```go
package bubbletree

import (
	"log/slog"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// ResolveMetrics resolves size, fill, border metrics + palettes plus the
// label mode, and populates Common().Requested with the metrics the
// scan/provider stages must collect.
func ResolveMetrics(s *State) error {
	cfg := s.Config

	s.Size = metric.Name(stages.PtrString(cfg.Size))
	s.FillMetric = resolveFillMetric(cfg, s.Size)
	s.FillPalette = stages.ResolveFillPalette(cfg.Fill, s.FillMetric)
	s.BorderMetric, s.BorderPalette = stages.ResolveBorderMetricAndPalette(cfg.Border)
	s.Labels = resolveLabels(cfg)

	s.Common().Requested = stages.CollectRequestedMetrics(s.Size, cfg.Fill, cfg.Border)

	return nil
}

func resolveFillMetric(cfg *config.Bubbletree, size metric.Name) metric.Name {
	if fill := cfg.Fill.MetricName(); fill != "" {
		return fill
	}

	return size
}

func resolveLabels(cfg *config.Bubbletree) LabelMode {
	if lbl := stages.PtrString(cfg.Labels); lbl != "" {
		return LabelMode(lbl)
	}

	return LabelFoldersOnly
}

// BuildInksStage builds the bubble inks. Also emits the "Rendering image"
// log line preserved from the legacy renderAndLog helper.
func BuildInksStage(s *State) error {
	c := s.Common()

	slog.Info("Rendering image", "output", c.Output, "width", c.Width, "height", c.Height)

	s.Inks = BuildInks(c.Root, s.FillMetric, s.FillPalette, s.BorderMetric, s.BorderPalette)

	return nil
}

// BuildLegendStage builds the legend config from inks. Bubbletree does not
// reserve canvas space for the legend; the legend overlays the bubbles.
func BuildLegendStage(s *State) error {
	pos, orient := legend.ResolveOptions(
		stages.PtrString(s.Config.Legend),
		stages.PtrString(s.Config.LegendOrientation),
	)
	s.LegendConfig = legend.Build(
		pos, orient,
		s.Inks.Fill, s.FillMetric,
		s.Inks.Border, s.BorderMetric,
		s.Size,
	)

	return nil
}

// LayoutStage runs the bubble layout algorithm.
func LayoutStage(s *State) error {
	c := s.Common()

	s.Nodes = Layout(c.Root, c.Width, c.Height, s.Size, s.Labels)

	return nil
}

// RenderStage renders the bubble tree to a canvas and attaches the legend.
func RenderStage(s *State) error {
	c := s.Common()

	cv := RenderToCanvas(&s.Nodes, c.Root, c.Width, c.Height, s.Inks)
	if s.LegendConfig != nil {
		cv.SetLegend(*s.LegendConfig)
	}

	c.Canvas = cv

	return nil
}

// LogResult logs the final summary.
func LogResult(s *State) error {
	c := s.Common()
	files, dirs := stages.CountAll(c.Root)

	slog.Info(
		"Rendered bubble tree",
		"files", files,
		"directories", dirs,
		"output", c.Output,
		"width", c.Width,
		"height", c.Height,
		"size_metric", string(s.Size),
		"fill_metric", string(s.FillMetric),
		"fill_palette", string(s.FillPalette),
		"border_metric", string(s.BorderMetric),
		"border_palette", string(s.BorderPalette),
	)

	return nil
}

// Compile-time checks.
var (
	_ pipeline.Stage[*State] = ResolveMetrics
	_ pipeline.Stage[*State] = BuildInksStage
	_ pipeline.Stage[*State] = BuildLegendStage
	_ pipeline.Stage[*State] = LayoutStage
	_ pipeline.Stage[*State] = RenderStage
	_ pipeline.Stage[*State] = LogResult
)
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/bubbletree/...`
Expected: no errors. (Nothing calls these stages yet; that happens in Task 9.)

- [ ] **Step 3: Lint via Explore subagent**

Dispatch `Explore` to run `task lint`. Fix issues.

- [ ] **Step 4: Commit**

```bash
git add internal/bubbletree/stages.go
git commit -m "feat(bubbletree): add pipeline stages"
```

---

## Task 9: Rewrite `BubbletreeCmd.Run` as a pipeline composition

Replace the open-coded `Run` method with `pipeline.Run`, and delete `renderAndLog`, `resolveFillMetric`, and `resolveLabels` from `cmd/codeviz/bubbletree_cmd.go`.

**Files:**
- Modify: `cmd/codeviz/bubbletree_cmd.go`

- [ ] **Step 1: Replace the imports**

The new `Run` needs `pipeline`, `bubbletree`, and `stages`. It no longer needs `bubbletree.LabelMode` from `cmd/codeviz` (that resolution moved into the package), `legend`, `export`, `filter`, `metric`, `model`, `palette`, `provider`, `scan`, or `slog`.

After the edit, the imports of `cmd/codeviz/bubbletree_cmd.go` should look like:

```go
import (
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/bubbletree"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)
```

(`filter`, `metric`, `provider`, `config` are still needed by `Validate` / `validateConfig` / `applyOverrides`.)

- [ ] **Step 2: Replace the `Run` method**

```go
func (c *BubbletreeCmd) Run(flags *Flags) error {
	if err := c.mergeConfigAndValidate(flags); err != nil {
		return err
	}

	state := &bubbletree.State{
		CommonState: stages.CommonState{
			TargetPath: c.TargetPath,
			Output:     c.Output,
			Flags:      toStagesFlags(flags),
			RootConfig: flags.Config,
			CLIFilters: c.Filter,
		},
		Config:             flags.Config.Bubbletree,
		IncludeBinaryFiles: c.IncludeBinaryFiles,
	}

	_, err := pipeline.Run[*bubbletree.State](
		state,
		stages.ValidatePaths,
		stages.ExportConfig,
		stages.BuildFilterRules,
		bubbletree.ResolveMetrics,
		stages.ScanFilesystem,
		stages.CheckGitRequirement,
		stages.RunProviders,
		stages.FilterBinaryFiles,
		stages.ExportData,
		stages.ResolveDimensions,
		bubbletree.BuildInksStage,
		bubbletree.BuildLegendStage,
		bubbletree.LayoutStage,
		bubbletree.RenderStage,
		stages.WriteCanvas,
		bubbletree.LogResult,
	)

	return eris.Wrap(err, "bubbletree pipeline failed")
}
```

- [ ] **Step 3: Delete now-dead helpers**

From `cmd/codeviz/bubbletree_cmd.go`, delete:

- The `renderAndLog` method.
- The `resolveFillMetric` method.
- The `resolveLabels` method.

What remains: `BubbletreeCmd` struct, `Validate`, `validateConfig`, `mergeConfigAndValidate`, `applyOverrides`, and the new `Run`.

- [ ] **Step 4: Run the full test suite**

Run: `go test ./...`
Expected: all tests pass — bubbletree validation tests in `cmd/codeviz/main_test.go`, treemap render tests, the new bubbletree render tests, the `internal/inks` tests, and everything else.

- [ ] **Step 5: Smoke test the binary**

Run:
```bash
task build
./bin/codeviz render bubbletree . -o /tmp/bubble.png -s file-size
```

Expected: exits 0, `/tmp/bubble.png` is a valid PNG (open it or `file /tmp/bubble.png`). Log output ends with a `Rendered bubble tree …` line whose keys match the pre-migration output.

- [ ] **Step 6: Lint via Explore subagent**

Dispatch `Explore` to run `task ci` (build + test + lint in one). Fix any issues.

- [ ] **Step 7: Commit**

```bash
git add cmd/codeviz/bubbletree_cmd.go
git commit -m "refactor(bubbletree): replace Run with pipeline composition"
```

- [ ] **Step 8: Push and open the PR**

```bash
git push -u origin feature/bubbletree-pipeline
gh pr create --fill --base main
```

---

## Self-Review

**Spec coverage:**

- Goal 1 (move bubble code into `internal/bubbletree`): Tasks 5, 7, 8 ✓
- Goal 2 (rewrite `Run` as `pipeline.Run`): Task 9 ✓
- Goal 3 (extract `internal/inks`, convert all four vizes): Tasks 1–4 ✓
- Goal 4 (comprehensive render tests): Task 6 ✓
- Non-goal "preserve legend overlay behavior": `LayoutStage` in Task 8 explicitly omits `legend.ReserveAndLayout` ✓
- Non-goal "no Goldie": render tests verify decodability + structural shape only ✓
- Success criterion "deleted files don't exist": Tasks 3 (`ink_builder.go`), 4 (`shape_inks.go`), 5 (`bubble_canvas*.go`) ✓
- Success criterion "`bubbletree_cmd.go` Run is short pipeline composition": Task 9 ✓

**Placeholder scan:** no TBDs, no "add error handling", no "similar to Task N". Every code step shows actual code.

**Type consistency:** `bubbletree.State`, `bubbletree.Inks`, `bubbletree.RenderToCanvas`, `bubbletree.BuildInks`, `bubbletree.LayoutStage` / `RenderStage` / etc. are used consistently across Tasks 5, 7, 8, 9. `pkginks` alias is used consistently in Tasks 2, 3, 5. `radialInks`, `bubbleInks`, `spiralInks` introduced in Task 4 don't conflict with anything else.

**Known soft spots:**

- Task 5 step 2 instructs renaming constants (e.g. `bubbleArcLabelInset` → `arcLabelInset`) as optional polish. If the implementer skips the rename, Task 6's render tests and Task 5's `render_internal_test.go` still work because they only reference `bubbleArcFontSize`, `bubbleMinArcFontSize`, `bubbleDefaultFontSize` from inside the same package — but the in-package test code needs to match whatever the implementer chose. The plan acknowledges this in Task 5 step 3.
- Task 2 introduces an import alias (`pkginks`). If the implementer prefers renaming the local `inks` variable to `tInks` (or similar) instead, that's an equivalent solution. The plan accepts either approach.

---

## Execution

After landing all nine tasks via the PR opened in Task 9 step 8, request the Squad to review and Ripley to handle any final integration.
