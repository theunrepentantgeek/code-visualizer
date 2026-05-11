# Treemap Canvas Migration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the treemap visualization from the old `internal/render/` pipeline to the new `internal/canvas/` abstraction, deleting the old treemap-specific render code and colour-application functions.

**Architecture:** The treemap currently uses a three-stage pipeline: layout → colour application → render. The Canvas migration replaces the last two stages. Colour application (the `applyFillColours`/`applyBorderColours` functions that walk the model tree and stamp `color.RGBA` onto layout nodes) is replaced by creating Inks from the metric dataset and passing `MetricValue`s to Canvas shapes. Rendering (the `render.Render()` call and its paired raster/SVG backends) is replaced by adding shapes to a Canvas and calling `canvas.Render()`. The treemap layout node (`TreemapRectangle`) has its `FillColour` and `BorderColour` fields stripped — it becomes geometry-only. The treemap layout algorithm is unchanged.

**Tech Stack:** Go 1.26.1 + Canvas (`internal/canvas/`), Kong (CLI), fogleman/gg (raster backend), eris (errors), Gomega (test assertions), Goldie v2 (golden-file snapshots)

**Branch:** Create `feature/canvas-treemap` from `main`

---

## Context for the implementer

### Canvas API summary

The Canvas API is in `internal/canvas/`. Key types:

- **`Canvas`** — retained-then-render surface. Call `NewCanvas(w, h)`, add shapes with `AddRectangle`, `AddDisc`, `AddText`, `AddLine`, `AddPath`, then call `Render(outputPath)`.
- **`Ink`** — resolves metric values to colours. Three constructors:
  - `FixedInk(c color.RGBA)` — always produces the same colour
  - `NumericInk(values []float64, pal palette.ColourPalette)` — maps numeric metrics via quantile bucketing
  - `CategoricalInk(categories []string, pal palette.ColourPalette)` — maps categories to colours
- **`MetricValue`** — carries `Kind`, `Measure`, `Quantity`, `Category` fields
- **`Ink.Dip(MetricValue) color.RGBA`** — resolves a metric value to a colour
- **Shapes** — `Rectangle{Spec, X, Y, W, H, Fill MetricValue, Border MetricValue, Label}`, etc.
- **Specs** — `RectangleSpec{ShapeStyle{Fill Ink, Border Ink, BorderWidth, ShowLabel, LabelInk, LabelStyle}}`, `TextSpec{Ink, FontSize, Anchor, Rotation}`
- **Layers** — `LayerBackground=0`, `LayerStructure=10`, `LayerContent=20`, `LayerOverlay=30`
- **`TextColourFor(fill color.RGBA) color.RGBA`** — WCAG-contrast label colour selection (in `internal/canvas/text_colour.go`)

### Current treemap pipeline (what we're replacing)

1. `treemap.Layout(root, w, h, sizeMetric)` → `TreemapRectangle` tree (geometry + empty colour fields)
2. `applyFillColours(&rects, root, fillMetric, fillPalette)` — walks model tree, stamps `FillColour color.RGBA` on each leaf
3. `applyBorderColours(&rects, root, cfg)` — walks model tree, stamps `BorderColour *color.RGBA` on each leaf
4. `render.Render(rects, w, h, outputPath, legend)` — dispatches to raster or SVG backend

### New pipeline (what we're building)

1. `treemap.Layout(root, w, h, sizeMetric)` → `TreemapRectangle` tree (geometry-only, no colour fields)
2. Create `fillInk` and `borderInk` from metric data + palette
3. Walk the layout tree + model tree in parallel, add `canvas.Rectangle` shapes (with `MetricValue` instead of resolved colours)
4. `canvas.Render(outputPath)` — Canvas resolves inks, sorts by layer, dispatches to backend

### What gets deleted

- `internal/render/renderer.go` — treemap-specific functions (`Render`, `renderTreemapImage`, `drawRect`, `drawDirectoryHeader`, `drawFileRect`, `treemapBorderWidth`)
- `internal/render/svg_treemap.go` — entire file
- `internal/render/renderer_test.go` — treemap-specific tests (will be replaced with Canvas-based tests)
- `cmd/codeviz/treemap_cmd.go` — `applyFillColours`, `applyBorderColours`, `applyNumericFillColours`, `applyCategoricalFillColours`, `applyNumericBorderColours`, `applyCategoricalBorderColours`

### What stays in `internal/render/` (shared, used by other viz types)

- `legend.go`, `legend_png.go`, `legend_svg.go`, `legend_test.go` — legend rendering (shared across all viz types)
- `format.go` — `FormatFromPath` (Canvas has its own copy; render's copy stays for other viz types)
- `save.go` — `saveContextPNG`/`saveContextJPG` (used by other viz renderers)
- `label.go` — `ShouldShowLabel`, `TextColourFor` (used by other viz renderers)
- `svg_helpers.go` — `writeSVGText`, `writeSVGTextRotated` (used by other viz SVG renderers)
- All other viz render files (spiral, radial, bubble)

### Lint rules to watch

- **funlen: 65 lines** (ignore-comments). The treemap `renderAndLog` function is at the limit.
- **revive: max-public-structs 5/file**, line-length 120, cognitive-complexity 10
- **gci:** import order (stdlib, blank line, third-party)
- **godox:** no TODO/BUG/FIXME comments
- **nilaway:** nil guards before dereferencing
- **wsl_v5:** blank lines before certain statements

### Build/test/lint commands

- `task build` — Build the codeviz binary
- `task test` — Run all tests
- `task lint` — Run golangci-lint
- `task ci` — Build, test, lint (full CI)

---

## Task 1: Strip colour fields from TreemapRectangle

Remove the visual fields from the layout node, making it geometry-only.

**Files:**
- Modify: `internal/treemap/node.go`

- [ ] **Step 1: Remove colour fields**

In `internal/treemap/node.go`, remove the `FillColour` and `BorderColour` fields and the `image/color` import:

```go
package treemap

// TreemapRectangle is a positioned visual element in the rendered treemap.
type TreemapRectangle struct {
	X           float64
	Y           float64
	W           float64
	H           float64
	Label       string
	ShowLabel   bool
	IsDirectory bool
	Children    []TreemapRectangle
}
```

- [ ] **Step 2: Run build to find all broken references**

Run: `go build ./... 2>&1 | head -50`

This will produce compile errors in `internal/render/renderer.go`, `internal/render/svg_treemap.go`, `internal/render/renderer_test.go`, and `cmd/codeviz/treemap_cmd.go`. These are expected — they will be fixed in subsequent tasks. Do **not** fix them yet. Just confirm the errors are all in the expected files.

- [ ] **Step 3: Commit**

```bash
git add internal/treemap/node.go
git commit -m "refactor(treemap): strip colour fields from TreemapRectangle

TreemapRectangle is now geometry-only. Colour resolution moves to the
Canvas pipeline in subsequent commits. Build is intentionally broken at
this commit.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 2: Add Ink introspection API

The bridge code needs to know the kind of an Ink (fixed, numeric, categorical) and, for metric inks, the metric name — so it can extract the right `MetricValue` from model files. Currently `inkKind` and its constants are unexported and there's no `Info()` method. Add these.

**Files:**
- Modify: `internal/canvas/ink.go` (add `metricName` field, accept metric name in constructors)
- Modify: `internal/canvas/ink_introspection.go` (add `InkKind` type, `Info()` method, `InkInfo` struct)
- Modify: `internal/canvas/ink_introspection_test.go` (add tests for new API)

- [ ] **Step 1: Add exported InkKind constants and InkInfo struct**

Add to `internal/canvas/ink_introspection.go`:

```go
// InkKind identifies the type of ink for introspection.
type InkKind int

const (
	InkFixed       InkKind = InkKind(inkFixed)
	InkNumeric     InkKind = InkKind(inkNumeric)
	InkCategorical InkKind = InkKind(inkCategorical)
)

// InkInfo carries introspection data about an Ink.
type InkInfo struct {
	Kind       InkKind
	MetricName metric.Name
}

// Info returns introspection data about the ink's kind and metric.
func (ink Ink) Info() InkInfo {
	return InkInfo{
		Kind:       InkKind(ink.kind),
		MetricName: ink.metricName,
	}
}
```

- [ ] **Step 2: Add metricName field to Ink and update constructors**

In `internal/canvas/ink.go`, add a `metricName metric.Name` field to the `Ink` struct. Update `NumericInk` and `CategoricalInk` to accept the metric name as their first parameter:

```go
// NumericInk maps numeric metric values to palette colours.
func NumericInk(name metric.Name, values []float64, pal palette.ColourPalette, opts ...InkOption) Ink {
	// ...existing code...
	return Ink{
		kind:       inkNumeric,
		metricName: name,
		// ...rest unchanged
	}
}

// CategoricalInk maps string categories to palette colours.
func CategoricalInk(name metric.Name, categories []string, pal palette.ColourPalette, opts ...InkOption) Ink {
	// ...existing code...
	return Ink{
		kind:       inkCategorical,
		metricName: name,
		// ...rest unchanged
	}
}
```

- [ ] **Step 3: Fix all existing call sites**

All existing callers of `NumericInk` and `CategoricalInk` (in `ink_test.go`, `ink_introspection_test.go`, `canvas_test.go`) need the new metric name parameter. Pass a test metric name like `metric.Name("test-metric")` in tests.

- [ ] **Step 4: Add introspection tests**

Add tests to `ink_introspection_test.go`:

```go
func TestInkInfo_Fixed(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := FixedInk(color.RGBA{R: 255, A: 255})
	info := ink.Info()

	g.Expect(info.Kind).To(Equal(InkFixed))
	g.Expect(info.MetricName).To(Equal(metric.Name("")))
}

func TestInkInfo_Numeric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := NumericInk("file-size", []float64{1, 2, 3}, testPalette())
	info := ink.Info()

	g.Expect(info.Kind).To(Equal(InkNumeric))
	g.Expect(info.MetricName).To(Equal(metric.Name("file-size")))
}

func TestInkInfo_Categorical(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := CategoricalInk("file-type", []string{"go", "rs"}, testPalette())
	info := ink.Info()

	g.Expect(info.Kind).To(Equal(InkCategorical))
	g.Expect(info.MetricName).To(Equal(metric.Name("file-type")))
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/canvas/... -count=1 -v 2>&1 | tail -20`

Expected: All canvas tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/canvas/
git commit -m "feat(canvas): add Ink introspection API

Exported InkKind constants (InkFixed, InkNumeric, InkCategorical),
InkInfo struct, and Info() method. NumericInk and CategoricalInk
now accept a metric.Name parameter for introspection by viz bridges.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 3: Build the treemap-to-canvas bridge

Create a new file that walks the layout tree + model tree in parallel and adds Canvas shapes. This replaces `applyFillColours`/`applyBorderColours` + `render.Render()`.

**Files:**
- Create: `cmd/codeviz/treemap_canvas.go`

- [ ] **Step 1: Create the bridge file**

Create `cmd/codeviz/treemap_canvas.go`:

```go
package main

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/treemap"
)

const (
	treemapHeaderHeight = treemap.HeaderHeight
	treemapMinBorderDim = 20.0
	treemapMidBorderDim = 100.0
)

var (
	treemapStructuralBorder = color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
	treemapHeaderFill       = color.RGBA{R: 0x44, G: 0x44, B: 0x44, A: 0xFF}
	treemapDefaultFill      = color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
	treemapBgColour         = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	treemapWhiteText        = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
)

// treemapInks holds the Ink instances for a treemap render pass.
type treemapInks struct {
	fill   canvas.Ink
	border canvas.Ink
}

// buildTreemapInks creates fill and border inks from metric configuration.
func buildTreemapInks(
	root *model.Directory,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
	borderMetric metric.Name,
	borderPaletteName palette.PaletteName,
) treemapInks {
	inks := treemapInks{
		fill:   canvas.FixedInk(treemapDefaultFill),
		border: canvas.FixedInk(treemapStructuralBorder),
	}

	inks.fill = buildMetricInk(root, fillMetric, fillPaletteName, treemapDefaultFill)
	if borderMetric != "" {
		inks.border = buildMetricInk(root, borderMetric, borderPaletteName, treemapStructuralBorder)
	}

	return inks
}

// buildMetricInk creates an Ink for a given metric, using the appropriate
// constructor based on the metric kind (numeric vs categorical).
func buildMetricInk(
	root *model.Directory,
	m metric.Name,
	palName palette.PaletteName,
	fallback color.RGBA,
) canvas.Ink {
	p, ok := provider.Get(m)
	if !ok {
		return canvas.FixedInk(fallback)
	}

	pal := palette.GetPalette(palName)

	if p.Kind() == metric.Quantity || p.Kind() == metric.Measure {
		values := collectNumericValues(root, m)
		if len(values) == 0 {
			return canvas.FixedInk(fallback)
		}

		return canvas.NumericInk(m, values, pal)
	}

	types := collectDistinctTypes(root, m)

	return canvas.CategoricalInk(m, types, pal)
}

// renderTreemapToCanvas walks the layout tree and model tree in parallel,
// adding shapes to the canvas. Returns the populated canvas.
func renderTreemapToCanvas(
	rects treemap.TreemapRectangle,
	root *model.Directory,
	width, height int,
	inks treemapInks,
) *canvas.Canvas {
	cv := canvas.NewCanvas(width, height)

	// Background
	bgSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        canvas.FixedInk(treemapBgColour),
			Border:      canvas.FixedInk(treemapBgColour),
			BorderWidth: 0,
		},
	}
	cv.AddRectangle(canvas.LayerBackground, canvas.Rectangle{
		Spec: bgSpec,
		X:    0, Y: 0,
		W: float64(width), H: float64(height),
	})

	addTreemapRect(cv, rects, root, inks)

	return cv
}

// addTreemapRect recursively adds shapes for a single treemap node.
func addTreemapRect(
	cv *canvas.Canvas,
	rect treemap.TreemapRectangle,
	node *model.Directory,
	inks treemapInks,
) {
	if rect.IsDirectory {
		addDirectoryShapes(cv, rect, inks)
	} else {
		addFileRectForFile(cv, rect, nil, inks)
		return
	}

	fileIdx := 0
	dirIdx := 0

	for i := range rect.Children {
		child := rect.Children[i]
		if child.IsDirectory && dirIdx < len(node.Dirs) {
			addTreemapRect(cv, child, node.Dirs[dirIdx], inks)
			dirIdx++
		} else if !child.IsDirectory && fileIdx < len(node.Files) {
			addFileRectForFile(cv, child, node.Files[fileIdx], inks)
			fileIdx++
		}
	}
}

func addDirectoryShapes(
	cv *canvas.Canvas,
	rect treemap.TreemapRectangle,
	inks treemapInks,
) {
	// Header bar fill
	headerSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        canvas.FixedInk(treemapHeaderFill),
			Border:      canvas.FixedInk(treemapHeaderFill),
			BorderWidth: 0,
		},
	}
	cv.AddRectangle(canvas.LayerStructure, canvas.Rectangle{
		Spec: headerSpec,
		X:    rect.X, Y: rect.Y,
		W: rect.W, H: treemapHeaderHeight,
	})

	// Header label
	if rect.Label != "" {
		labelSpec := &canvas.TextSpec{
			Ink:      canvas.FixedInk(treemapWhiteText),
			FontSize: 0,
			Anchor:   canvas.AnchorStart,
		}
		cv.AddText(canvas.LayerOverlay, canvas.Text{
			Spec:    labelSpec,
			X:       rect.X + 4,
			Y:       rect.Y + treemapHeaderHeight/2,
			Content: rect.Label,
		})
	}

	// Directory border
	borderSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        canvas.FixedInk(color.RGBA{A: 0}),
			Border:      canvas.FixedInk(treemapStructuralBorder),
			BorderWidth: treemapDynBorderWidth(rect.W, rect.H, true),
		},
	}
	cv.AddRectangle(canvas.LayerStructure, canvas.Rectangle{
		Spec: borderSpec,
		X:    rect.X, Y: rect.Y,
		W: rect.W, H: rect.H,
	})
}

func addFileRectForFile(
	cv *canvas.Canvas,
	rect treemap.TreemapRectangle,
	file *model.File,
	inks treemapInks,
) {
	if rect.W <= 0 || rect.H <= 0 {
		return
	}

	hasBorder := inks.border.Info().Kind != canvas.InkFixed

	fillMV := metricValueForFile(file, inks.fill)
	borderMV := metricValueForFile(file, inks.border)

	spec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        inks.fill,
			Border:      inks.border,
			BorderWidth: treemapDynBorderWidth(rect.W, rect.H, hasBorder),
		},
	}

	cv.AddRectangle(canvas.LayerContent, canvas.Rectangle{
		Spec:   spec,
		X:      rect.X,
		Y:      rect.Y,
		W:      rect.W,
		H:      rect.H,
		Fill:   fillMV,
		Border: borderMV,
		Label:  rect.Label,
	})
}

// metricValueForFile builds a MetricValue from a file's data for the given ink.
func metricValueForFile(file *model.File, ink canvas.Ink) canvas.MetricValue {
	if file == nil {
		return canvas.MetricValue{}
	}

	info := ink.Info()

	switch info.Kind {
	case canvas.InkFixed:
		return canvas.MetricValue{}
	case canvas.InkNumeric:
		m := info.MetricName
		if v, ok := file.Quantity(m); ok {
			return canvas.MetricValue{Kind: metric.Quantity, Quantity: v}
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

// treemapDynBorderWidth returns a dynamic border width based on rectangle
// size and whether a border metric is configured.
func treemapDynBorderWidth(w, h float64, hasBorderMetric bool) float64 {
	if !hasBorderMetric {
		return 0.5
	}

	minDim := min(w, h)

	switch {
	case minDim < treemapMinBorderDim:
		return 1.0
	case minDim >= treemapMidBorderDim:
		return 3.0
	default:
		return 2.0
	}
}
```

**Note:** This file uses `ink.Info()`, `canvas.InkFixed`, `canvas.InkNumeric`, `canvas.InkCategorical` from Task 2. Also uses `collectNumericValues`, `collectDistinctTypes` (existing functions in the `main` package from `treemap_cmd.go`), and `provider.Get` (explicit import).

- [ ] **Step 2: Verify the bridge file compiles**

Run: `go vet ./cmd/codeviz/... 2>&1 | head -20`

There will still be errors from `treemap_cmd.go` (which still references old colour fields). That's expected. Confirm no errors originate from `treemap_canvas.go`.

- [ ] **Step 3: Commit**

```bash
git add cmd/codeviz/treemap_canvas.go
git commit -m "feat(treemap): add treemap-to-canvas bridge

Walks layout tree + model tree in parallel, adding Canvas shapes with
MetricValues instead of pre-resolved colours. Replaces the old
applyFillColours/applyBorderColours + render.Render pipeline.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 4: Wire the treemap command to the Canvas pipeline

Replace the old rendering pipeline in `treemap_cmd.go` with the Canvas bridge.

**Files:**
- Modify: `cmd/codeviz/treemap_cmd.go`

- [ ] **Step 1: Replace renderAndLog**

Rewrite `renderAndLog` to use the new Canvas pipeline instead of the old `applyFillColours`/`applyBorderColours` + `render.Render`:

```go
func (c *TreemapCmd) renderAndLog(
	root *model.Directory,
	cfg *config.Treemap,
	width, height int,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
) error {
	size := metric.Name(ptrString(cfg.Size))
	files, dirs := countAll(root)

	slog.Info("Rendering image", "output", c.Output, "width", width, "height", height)

	// Build legend info before layout so we can reserve space for it.
	legendPos, legendOrient := resolveLegendOptions(ptrString(cfg.Legend), ptrString(cfg.LegendOrientation))
	borderName, borderPaletteName := resolveBorderPaletteName(cfg)
	legend := buildLegendInfo(
		legendPos, legendOrient, fillMetric, fillPaletteName,
		borderName, borderPaletteName, size, root,
	)

	layoutW, layoutH := reserveAndLayout(legend, width, height)

	rects := treemap.Layout(root, layoutW, layoutH, size)

	if layoutW < width || layoutH < height {
		wReduce, hReduce := render.ReserveLegendSpace(legend)
		dx, dy := legendLayoutOffset(legend, wReduce, hReduce)
		treemap.OffsetRects(&rects, dx, dy)
	}

	// Build inks from metric data
	inks := buildTreemapInks(root, fillMetric, fillPaletteName, borderName, borderPaletteName)

	// Populate canvas
	cv := renderTreemapToCanvas(rects, root, width, height, inks)

	slog.Debug("rendering", "width", width, "height", height, "output", c.Output)

	if err := cv.Render(c.Output); err != nil {
		return eris.Wrap(err, "render failed")
	}

	slog.Info("Rendered treemap",
		"files", files,
		"directories", dirs,
		"output", c.Output,
		"width", width,
		"height", height,
		"size_metric", string(size),
		"fill_metric", string(fillMetric),
		"fill_palette", string(fillPaletteName),
		"border_metric", string(borderName),
		"border_palette", string(borderPaletteName),
	)

	return nil
}
```

- [ ] **Step 2: Delete old colour application functions**

Remove these functions from `treemap_cmd.go`:
- `applyFillColours` (lines 496–520)
- `applyBorderColours` (lines 522–562 — the method on `*TreemapCmd`)
- `applyNumericFillColours` (lines 605–627)
- `applyCategoricalFillColours` (lines 629–651)
- `applyNumericBorderColours` (lines 653–676)
- `applyCategoricalBorderColours` (lines 678–701)

Keep `collectNumericValues`, `collectDistinctTypes`, `extractNumeric` — they're used by the new bridge code in `treemap_canvas.go`.

- [ ] **Step 3: Remove unused render import**

Remove `"github.com/theunrepentantgeek/code-visualizer/internal/render"` from the import list **only if** no remaining code in `treemap_cmd.go` references the `render` package. The legend functions (`render.ReserveLegendSpace`, `render.LegendInfo`, `render.LegendPosition*`, etc.) still use it, so the import likely stays.

- [ ] **Step 4: Run build**

Run: `go build ./cmd/codeviz/... 2>&1 | head -30`

Expected: Compiles successfully (the canvas pipeline replaces the old render calls).

- [ ] **Step 5: Commit**

```bash
git add cmd/codeviz/treemap_cmd.go
git commit -m "feat(treemap): wire treemap command to Canvas pipeline

renderAndLog now uses Canvas inks + bridge instead of the old
applyFillColours/applyBorderColours + render.Render pipeline.
Old colour application functions deleted.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 5: Update or split the render package for treemap removal

Remove treemap-specific code from `internal/render/`, keeping shared code for other viz types.

**Files:**
- Modify: `internal/render/renderer.go` — remove treemap-specific functions, keep package declaration and shared constants
- Delete: `internal/render/svg_treemap.go`
- Modify: `internal/render/renderer_test.go` — remove treemap-specific tests

- [ ] **Step 1: Remove treemap functions from renderer.go**

Remove these from `internal/render/renderer.go`:
- `Render` function (lines 25–46)
- `renderTreemapImage` function (lines 48–58)
- `drawRect` function (lines 60–70)
- `drawDirectoryHeader` function (lines 72–87)
- `drawFileRect` function (lines 89–121)
- `treemapBorderWidth` function (lines 125–140)

Keep:
- Package declaration and doc comment
- The colour constants (`structuralBorder`, `headerFill`, `defaultFill`, `bgColour`) — check if they're used by other viz renderers. If used only by treemap, remove them. If shared, keep them.
- Imports — clean up unused imports after removing functions.

**Check which constants are used elsewhere:**

```bash
grep -rn 'structuralBorder\|headerFill\|defaultFill\|bgColour' internal/render/ --include='*.go' | grep -v renderer.go | grep -v _test.go
```

If constants are only used in `renderer.go` and `svg_treemap.go`, remove them. If used in other files like `radialtree.go` or `bubbletree.go`, keep them.

- [ ] **Step 2: Delete svg_treemap.go**

```bash
git rm internal/render/svg_treemap.go
```

- [ ] **Step 3: Remove treemap-specific tests from renderer_test.go**

Remove these tests from `internal/render/renderer_test.go`:
- `TestRenderFlatDir`
- `TestRenderNestedDir`
- `TestRenderWithBorderColour`
- `TestRenderNoBorderWhenNil`
- `paletteTreemap` helper
- `TestGoldenFile_NeutralPalette`
- `TestGoldenFile_CategorizationPalette`
- `TestGoldenFile_TemperaturePalette`
- `TestGoldenFile_GoodBadPalette`
- `goldenPaletteTest` helper
- `BenchmarkScanAndRender`
- `createBenchFixture`
- `TestRender_JPG`
- `TestRender_JPEG`
- `TestRender_SVG`
- `TestRender_SVG_EscapesLabels`
- `TestRender_UnsupportedFormat`
- `TestRender_PNG_DecodesAsPNG`

These ALL test treemap rendering through `render.Render()` which no longer exists. If `renderer_test.go` becomes empty, delete it. If `makeFile` helper is used by other test files, keep it.

Check: `grep -rn 'makeFile' internal/render/ --include='*_test.go'` — if only in `renderer_test.go`, remove.

- [ ] **Step 4: Clean up renderer.go**

If `renderer.go` is now empty except for the package declaration and shared constants, check if it has any remaining exports used by other files in the package. If the file is effectively empty, consider deleting it and moving any remaining shared constants to a `shared.go` or similar. But if other viz renderers (radial, spiral, bubble) import these constants, leave it as-is for now.

- [ ] **Step 5: Run build and tests**

```bash
go build ./... 2>&1 | head -30
go test ./internal/render/... -count=1 -v 2>&1 | tail -30
```

Expected: Build succeeds. Remaining render tests (legend, other viz types) still pass.

- [ ] **Step 6: Commit**

```bash
git add internal/render/
git commit -m "refactor(render): remove treemap-specific render code

Treemap rendering now uses the Canvas pipeline. Removed treemap-specific
functions from renderer.go, deleted svg_treemap.go, and removed treemap
tests from renderer_test.go. Shared legend and utility code remains for
other visualization types.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 6: Add Canvas-based treemap rendering tests

Replace the deleted treemap render tests with new tests that exercise the Canvas pipeline.

**Files:**
- Create: `cmd/codeviz/treemap_canvas_test.go`

- [ ] **Step 1: Write unit tests for the bridge**

Create `cmd/codeviz/treemap_canvas_test.go` with tests that verify:

1. **`buildTreemapInks` returns correct ink kinds** — given numeric/categorical/no metric, the returned inks have the expected kind.
2. **`renderTreemapToCanvas` produces shapes** — given a simple flat layout + model, the canvas has the expected number of shapes (background + directory header + header text + directory border + N file rects).
3. **`treemapDynBorderWidth` returns correct widths** — test the three size thresholds.
4. **End-to-end PNG render** — `renderTreemapToCanvas` + `canvas.Render(tempFile.png)` produces a non-empty PNG file.
5. **End-to-end SVG render** — same as above but for SVG; verify valid XML output.
6. **End-to-end JPG render** — same as above but for JPG; verify decodable JPEG.

```go
package main

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

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/treemap"
)

func makeTestFile(name, ext string, size int64) *model.File {
	f := &model.File{Name: name, Extension: ext}
	f.SetQuantity(filesystem.FileSize, size)
	f.SetClassification(filesystem.FileType, ext)

	return f
}

func TestTreemapDynBorderWidth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(treemapDynBorderWidth(10, 10, false)).To(Equal(0.5))
	g.Expect(treemapDynBorderWidth(10, 10, true)).To(Equal(1.0))
	g.Expect(treemapDynBorderWidth(50, 50, true)).To(Equal(2.0))
	g.Expect(treemapDynBorderWidth(200, 200, true)).To(Equal(3.0))
}

func TestBuildTreemapInks_Numeric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeTestFile("a.go", "go", 100),
			makeTestFile("b.go", "go", 200),
		},
	}

	inks := buildTreemapInks(root, filesystem.FileSize, palette.Temperature, "", "")

	g.Expect(inks.fill.Info().Kind).To(Equal(canvas.InkNumeric))
	g.Expect(inks.border.Info().Kind).To(Equal(canvas.InkFixed))
}

func TestBuildTreemapInks_Categorical(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeTestFile("a.go", "go", 100),
			makeTestFile("b.rs", "rs", 200),
		},
	}

	inks := buildTreemapInks(root, filesystem.FileType, palette.Categorization, "", "")

	g.Expect(inks.fill.Info().Kind).To(Equal(canvas.InkCategorical))
}

func TestRenderTreemapToCanvas_PNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "flat",
		Files: []*model.File{
			makeTestFile("small.txt", "txt", 5),
			makeTestFile("medium.go", "go", 100),
			makeTestFile("large.rs", "rs", 1000),
		},
	}

	rects := treemap.Layout(root, 800, 600, filesystem.FileSize)
	inks := buildTreemapInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := renderTreemapToCanvas(rects, root, 800, 600, inks)

	out := filepath.Join(t.TempDir(), "treemap.png")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("png"))
}

func TestRenderTreemapToCanvas_SVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "flat",
		Files: []*model.File{
			makeTestFile("a.go", "go", 100),
			makeTestFile("b.go", "go", 200),
		},
	}

	rects := treemap.Layout(root, 400, 300, filesystem.FileSize)
	inks := buildTreemapInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := renderTreemapToCanvas(rects, root, 400, 300, inks)

	out := filepath.Join(t.TempDir(), "treemap.svg")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	decoder := xml.NewDecoder(bytes.NewReader(data))

	var rootElement string

	for {
		tok, xmlErr := decoder.Token()
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

func TestRenderTreemapToCanvas_JPG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "flat",
		Files: []*model.File{
			makeTestFile("a.go", "go", 100),
		},
	}

	rects := treemap.Layout(root, 400, 300, filesystem.FileSize)
	inks := buildTreemapInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := renderTreemapToCanvas(rects, root, 400, 300, inks)

	out := filepath.Join(t.TempDir(), "treemap.jpg")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("jpeg"))
}
```

- [ ] **Step 2: Run the new tests**

Run: `go test ./cmd/codeviz/... -count=1 -run TestTreemap -v 2>&1 | tail -30`

Expected: All tests pass.

- [ ] **Step 3: Commit**

```bash
git add cmd/codeviz/treemap_canvas_test.go
git commit -m "test(treemap): add Canvas-based treemap rendering tests

Tests cover ink construction (numeric, categorical), border width logic,
and end-to-end PNG/SVG/JPG rendering through the Canvas pipeline.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 7: Handle label rendering in the Canvas pipeline

The old treemap renderer had inline label logic (`ShouldShowLabel` + `TextColourFor`). The Canvas needs to handle this. Check if the Canvas's `ShapeStyle.ShowLabel` + `LabelInk` fields are wired to produce labels. If not, add label shapes explicitly in the bridge.

**Files:**
- Modify: `cmd/codeviz/treemap_canvas.go` (add label logic to `addFileRectForFile`)

- [ ] **Step 1: Check if Canvas auto-renders labels**

Read `internal/canvas/canvas.go` — specifically `drawRectangle`. Does it check `ShapeStyle.ShowLabel` and render a text shape for the label? Check what happens with `Rectangle.Label`.

If the Canvas does NOT auto-render labels (likely — Stage 1 focused on core shape dispatch), add explicit label handling in the bridge.

- [ ] **Step 2: Add label shapes in the bridge**

In `addFileRectForFile` in `treemap_canvas.go`, after adding the file rectangle, check if the rect is large enough for a label and add an explicit Text shape:

```go
// After adding the rectangle shape...
if rect.Label != "" && rect.W >= 40 && rect.H >= 16 {
	fillColour := inks.fill.Dip(fillMV)
	labelColour := canvas.TextColourFor(fillColour)

	labelSpec := &canvas.TextSpec{
		Ink:      canvas.FixedInk(labelColour),
		FontSize: 0,
		Anchor:   canvas.AnchorMiddle,
	}
	cv.AddText(canvas.LayerOverlay, canvas.Text{
		Spec:    labelSpec,
		X:       rect.X + rect.W/2,
		Y:       rect.Y + rect.H/2,
		Content: rect.Label,
	})
}
```

Note: The old renderer used `gg.MeasureString` to check if text fits. For now, use the simpler dimension check (40px × 16px minimums). This is a slight behavior change but acceptable — the precise text measurement requires a gg context which is a raster-only concept.

- [ ] **Step 3: Run tests**

Run: `go test ./cmd/codeviz/... -count=1 -run TestTreemap -v 2>&1 | tail -20`

Expected: All tests still pass.

- [ ] **Step 4: Commit**

```bash
git add cmd/codeviz/treemap_canvas.go
git commit -m "feat(treemap): add file label rendering to Canvas bridge

File labels are rendered as explicit Text shapes on the overlay layer.
Uses dimension-based visibility check (40×16px minimums) and
WCAG-contrast text colour from the resolved fill.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 8: Handle legend rendering

The legend is still rendered by the old `render` package (shared across all viz types). For the treemap migration, we keep using the existing legend system — the Canvas doesn't render legends yet (SetLegend is stub-only). The legend is drawn by calling back into the old render package's legend code.

**Important architectural decision:** Legend migration is NOT part of individual viz migrations. The legend will be migrated to Canvas separately after all four viz types are on Canvas. For now, we need a bridge that draws the legend into the Canvas output.

**Files:**
- Modify: `cmd/codeviz/treemap_canvas.go` (add legend rendering)

- [ ] **Step 1: Add post-render legend overlay**

The approach: after `canvas.Render()` writes the output, for raster formats, reopen the file, draw the legend using the existing `render` package's legend code, and save again. For SVG, append legend elements.

**Alternative simpler approach:** Since the legend is complex (~680 lines of rendering code) and will be migrated to Canvas later as a dedicated feature, the cleanest approach for now is to keep the legend rendering as a post-processing step using the existing render package code.

Add a `drawTreemapLegend` function to `treemap_canvas.go`:

```go
// drawTreemapLegend draws the legend onto the already-rendered output file.
// This is a temporary bridge — legend rendering will migrate to Canvas later.
func drawTreemapLegend(outputPath string, legend *render.LegendInfo, width, height int) error {
	if legend == nil || legend.Position == render.LegendPositionNone || len(legend.Entries) == 0 {
		return nil
	}

	format, err := canvas.FormatFromPath(outputPath)
	if err != nil {
		return err
	}

	switch format {
	case canvas.FormatSVG:
		return appendSVGLegend(outputPath, legend, width, height)
	case canvas.FormatPNG, canvas.FormatJPG:
		return overlayRasterLegend(outputPath, legend, width, height, format)
	default:
		return nil
	}
}
```

**However**, this requires reopening and modifying already-written files, which is fragile for raster formats (need to decode PNG/JPG, draw on top, re-encode). A better approach is to integrate the legend draw calls into the Canvas backend directly — but that's a larger change.

**Simplest viable approach:** For this PR, accept that legend rendering is temporarily done via the old render pipeline by having `renderAndLog` call the legend functions directly on the raster context or SVG file. But since we're now using Canvas (which owns the backend), we can't access the gg.Context or SVG file handle.

**Pragmatic decision:** The treemap's `renderAndLog` already builds `legend` before rendering. We can:
1. If legend is needed: fall back to the old render pipeline for the complete render (legend included)
2. If no legend: use the Canvas pipeline

**This is not ideal.** Let's instead accept a known limitation for this PR:

**For this PR: Legend will not be rendered.** The legend rendering code stays in the render package for the other 3 viz types that still use it. The treemap will temporarily lose legend support. We document this as a known regression that will be fixed when legend rendering is added to the Canvas package. This matches the design spec's migration path — legend moves to Canvas as a separate feature.

- [ ] **Step 2: Remove legend call from renderAndLog**

In `renderAndLog`, remove the legend-related code that calls `render.ReserveLegendSpace()` and `buildLegendInfo()`. The layout reservation can stay (it's geometry, not rendering), but the legend rendering call is removed since Canvas doesn't support it yet.

Actually — keep the legend space reservation logic so that when legend support is added, the layout geometry is already correct. Just don't render the legend.

Update `renderAndLog` to keep legend space reservation but skip legend rendering:

```go
// In renderAndLog, after canvas.Render:
// Legend rendering is not yet supported by the Canvas pipeline.
// When canvas legend support is implemented, add: cv.SetLegend(...)
```

- [ ] **Step 3: Log the regression**

Add a `slog.Warn` in `renderAndLog` when a legend position is configured:

```go
if legend != nil && legend.Position != render.LegendPositionNone {
	slog.Warn("Legend rendering not yet available with Canvas pipeline; legend omitted")
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./cmd/codeviz/... -count=1 -v 2>&1 | tail -30
```

- [ ] **Step 5: Commit**

```bash
git add cmd/codeviz/treemap_cmd.go cmd/codeviz/treemap_canvas.go
git commit -m "feat(treemap): accept temporary legend regression

Legend rendering is not yet supported by the Canvas pipeline. When a
legend position is configured, a warning is logged. Legend support will
be added to Canvas as a separate feature.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 9: Full CI verification

Run the complete CI pipeline to catch any lint, build, or test issues.

**Files:** None (verification only)

- [ ] **Step 1: Run full CI**

```bash
task ci
```

Expected: Build succeeds, all tests pass, 0 lint issues.

- [ ] **Step 2: Fix any issues**

Address any lint errors (gci imports, funlen, nilaway, etc.) identified by `task ci`.

- [ ] **Step 3: Verify no changes to non-treemap viz types**

```bash
git diff --name-only HEAD~$(git rev-list --count HEAD ^main) -- internal/render/radialtree.go internal/render/bubbletree.go internal/render/spiral.go internal/render/svg_radial.go internal/render/svg_bubble.go internal/render/svg_spiral.go cmd/codeviz/radialtree_cmd.go cmd/codeviz/bubbletree_cmd.go cmd/codeviz/spiral_cmd.go
```

Expected: Empty output — no changes to other visualization types.

- [ ] **Step 4: Verify the old render tests for other viz types still pass**

```bash
go test ./internal/render/... -count=1 -v 2>&1 | tail -30
```

- [ ] **Step 5: Commit any fixes**

```bash
git add -A
git commit -m "fix(treemap): address CI lint issues

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 10: Create PR

Push the branch and create a Pull Request.

**Files:** None

- [ ] **Step 1: Push branch**

```bash
git push -u origin feature/canvas-treemap
```

- [ ] **Step 2: Create PR**

```bash
gh pr create --title "feat: migrate treemap visualization to Canvas pipeline" --body "$(cat <<'EOF'
## Summary

Migrates the treemap visualization from the old `internal/render/` pipeline to the new `internal/canvas/` abstraction.

### Changes

- **TreemapRectangle** is now geometry-only (no `FillColour`/`BorderColour` fields)
- **New bridge** (`treemap_canvas.go`): walks layout+model trees, creates Canvas Inks and adds shapes with MetricValues
- **Old render code removed**: `render.Render()`, `renderTreemapImage`, `drawRect`, treemap SVG functions — all deleted
- **Old colour functions removed**: `applyFillColours`, `applyBorderColours` and their numeric/categorical variants
- **New tests**: Canvas-based tests replacing old render tests (PNG/SVG/JPG output, ink construction, border width logic)

### Known Regression

- **Legend not rendered**: The Canvas pipeline doesn't support legend rendering yet. A warning is logged when a legend position is configured. Legend support will be added to Canvas as a dedicated feature (benefits all viz types at once).

### What's NOT changed

- Layout algorithm (`internal/treemap/`) — unchanged except `TreemapRectangle` field removal
- Other viz types (spiral, radial, bubble) — completely untouched
- Shared render utilities (legend, format, save, label, SVG helpers) — preserved for other viz types
- CLI flags and configuration — unchanged
- The `internal/canvas/` package itself — used as-is from Stage 1

### Test Plan

- [ ] `task ci` passes (build + test + lint)
- [ ] PNG output renders correctly for treemap
- [ ] SVG output renders correctly for treemap
- [ ] JPG output renders correctly for treemap
- [ ] Other viz types still work (spiral, radial, bubble unmodified)
EOF
)"
```

---

## Design decisions

1. **Legend regression is acceptable.** The design spec says legend migration is a separate feature that benefits all viz types. Duplicating 680 lines of legend code into the Canvas pipeline for one viz type would be worse than a temporary regression.

2. **Label rendering uses dimension checks, not text measurement.** The old renderer used `gg.MeasureString` (raster-only). The Canvas bridge uses 40×16px minimums. This is a minor behavior change — some labels that previously fit by a few pixels may now be hidden, and vice versa. Acceptable for now; the Canvas could add text measurement support later.

3. **Bridge code lives in `cmd/codeviz/`, not `internal/canvas/`.** The bridge is visualization-specific (knows about treemap layout, model walking, metric extraction). It doesn't belong in the generic canvas package.

4. **Shared colour constants move.** The treemap-specific constants (headerFill, structuralBorder, etc.) move from `internal/render/renderer.go` to `cmd/codeviz/treemap_canvas.go`. If they were shared with other viz types, copies remain in `renderer.go`.
