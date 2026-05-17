# Spiral Canvas Migration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the spiral visualization from the old `internal/render/` pipeline to the new `internal/canvas/` abstraction, deleting the old spiral-specific render code and colour-application functions.

**Architecture:** The spiral uses a flat-list pipeline: time-bucket creation → layout → disc sizing → colour application → render. The Canvas migration replaces the last two stages. Colour application (the `applyFill`/`applyBorder` functions that aggregate metrics per time-bucket and stamp `FillColour`/`BorderColour` onto `SpiralNode`s) is replaced by creating Inks from the aggregated dataset and passing `MetricValue`s to Canvas shapes. Rendering (the `render.RenderSpiral()` call and its paired raster/SVG backends) is replaced by adding shapes to a Canvas and calling `canvas.Render()`. The spiral layout node (`SpiralNode`) has its `FillColour` and `BorderColour` fields stripped — it becomes geometry-only. The layout algorithm and time-bucket logic are unchanged.

**Tech Stack:** Go 1.26.1 + Canvas (`internal/canvas/`), Kong (CLI), fogleman/gg (raster backend), eris (errors), Gomega (test assertions)

**Branch:** Create `feature/canvas-spiral` from `main`

---

## Context for the implementer

### Canvas API summary

The Canvas API is in `internal/canvas/`. Key types:

- **`Canvas`** — retained-then-render surface. Call `NewCanvas(w, h)`, add shapes with `AddRectangle`, `AddDisc`, `AddText`, `AddLine`, `AddPath`, then call `Render(outputPath)`.
- **`Ink`** — resolves metric values to colours. Three constructors:
  - `FixedInk(c color.RGBA)` — always produces the same colour
  - `NumericInk(name metric.Name, values []float64, pal palette.ColourPalette)` — maps numeric metrics via quantile bucketing
  - `CategoricalInk(name metric.Name, categories []string, pal palette.ColourPalette)` — maps categories to colours
- **`Ink.Info()`** — returns `InkInfo{Kind InkKind, MetricName metric.Name}` for introspection
- **`InkKind`** — exported constants: `InkFixed`, `InkNumeric`, `InkCategorical`
- **`MetricValue`** — carries `Kind`, `Measure`, `Quantity`, `Category` fields. Constructors: `MeasureValue(v)`, `QuantityValue(v)`, `CategoryValue(v)`.
- **`Ink.Dip(MetricValue) color.RGBA`** — resolves a metric value to a colour
- **Shapes:**
  - `Disc{Spec *DiscSpec, X, Y, Radius, Angle float64, Fill MetricValue, Border MetricValue, Label string}`
  - `Path{Spec *LineSpec, Points []Position}`
  - `Text{Spec *TextSpec, X, Y float64, Content string}`
  - `Rectangle{Spec *RectangleSpec, X, Y, W, H float64, Fill MetricValue, Border MetricValue, Label string}`
- **Specs:**
  - `DiscSpec{ShapeStyle{Fill Ink, Border Ink, BorderWidth, ShowLabel, LabelInk, LabelStyle}}`
  - `LineSpec{Stroke Ink, StrokeWidth float64}`
  - `TextSpec{Ink, FontSize, Anchor TextAnchor, Rotation float64}`
- **Layers** — `LayerBackground=0`, `LayerStructure=10`, `LayerContent=20`, `LayerOverlay=30`
- **`TextColourFor(fill color.RGBA) color.RGBA`** — WCAG-contrast label colour selection
- **`LabelStyle`** — `LabelCentered`, `LabelArc`, `LabelRadial`

### Current spiral pipeline (what we're replacing)

1. `spiral.BuildTimeBuckets(resolution, start, end)` → `[]TimeBucket`
2. `assignFilesToBuckets(buckets, records)` → files distributed into buckets
3. `aggregateBucketMetrics(buckets, cfg)` → fills `SizeValue`, `FillValue`, `FillLabel`, `BorderValue`, `BorderLabel` on each bucket
4. `spiral.Layout(buckets, w, h, resolution, labels)` → `[]SpiralNode` (flat list, geometry + labels)
5. `spiral.MaxDiscRadius(…)` + `applySpiralDiscSizes(nodes, buckets, maxDisc)` → disc radii set
6. `applyFill(nodes, buckets, cfg)` → stamps `FillColour color.RGBA` on each node
7. `applyBorder(nodes, buckets, cfg)` → stamps `BorderColour *color.RGBA` on each node
8. `render.RenderSpiral(nodes, w, h, outputPath, legend)` → dispatches to raster or SVG backend

The render backend has three passes:
- **Pass 1 (track):** Draws a faint Archimedean spiral guide line through all nodes
- **Pass 2 (discs):** Draws filled circles with borders for each node
- **Pass 3 (labels):** Draws rotated labels tangent to the spiral

### New pipeline (what we're building)

1–5 unchanged (buckets, layout, disc sizing stay the same).

6. Create `fillInk` and `borderInk` from aggregated bucket values + palette
7. Walk the flat node+bucket list, add `canvas.Disc` shapes (with `MetricValue` instead of resolved colours)
8. Add the spiral track as a `canvas.Path` (guide line on `LayerStructure`)
9. Add labels as `canvas.Text` with `LabelRadial` style
10. `canvas.Render(outputPath)` — Canvas resolves inks, sorts by layer, dispatches to backend

### Key difference from treemap migration

The treemap operates on a **tree** of per-file data; inks are built from individual file metrics. The spiral operates on a **flat list** of time-bucket-aggregated values; inks are built from aggregated bucket-level values (sums/modes). The `buildMetricInk` helper from `treemap_canvas.go` won't work here — the spiral needs its own ink builders that take `[]TimeBucket` instead of `*model.Directory`.

### What gets deleted

- `internal/render/spiral.go` — spiral PNG rendering (all functions)
- `internal/render/svg_spiral.go` — spiral SVG rendering (all functions)
- `internal/render/spiral_test.go` — all spiral render tests (replaced with Canvas-based tests)
- `cmd/codeviz/spiral_cmd.go` — `applyFill`, `applyBorder`, `applySpiralNumericFill`, `applySpiralCategoricalFill`, `applySpiralNumericBorder`, `applySpiralCategoricalBorder`, `collectBucketCategories`

### What stays unchanged

- `internal/spiral/` package — layout, time buckets, resolution, node positioning
- `cmd/codeviz/spiral_cmd.go` — `Run()`, `scanAndRunProviders`, `buildTimeBuckets`, `aggregateBucketMetrics`, `aggregateBucket`, `aggregateColourMetric`, `sumNumericMetric`, `modeCategory`, `commitTimeRange`, `assignFilesToBuckets`, `applySpiralDiscSizes`, `resolveResolution`, `resolveLabels`, `resolveFillMetric`, `resolveFillPalette`, `validatePaths`, `buildFilterRules`, `checkGitRepo`, `collectSpiralMetrics`, `filterBinaryFiles`, `applyOverrides`, `Validate`, `validateConfig`, `mergeConfigAndValidate`, `logRendered`

### Render constants needed in the Canvas bridge

These constants are currently in `internal/render/spiral.go` and will be needed in the Canvas bridge:

```go
var (
    spiralDefaultFill   = color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
    spiralDefaultBorder = color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
    spiralTrackColour   = color.RGBA{R: 0xDD, G: 0xDD, B: 0xDD, A: 0xFF}
    spiralLabelColour   = color.RGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF}
)

const (
    spiralTrackWidth    = 1.0
    spiralLabelGap      = 4.0
    spiralTrackMinSteps = 500
)
```

Also the border width function:

```go
func spiralBorderWidth(discRadius float64) float64 {
    if discRadius < 8 {
        return 2.0
    }
    return 3.0
}
```

And the track interpolation helpers:

```go
type trackParams struct {
    cx, cy   float64
    a, b     float64
    maxTheta float64
}

func inferTrackParams(nodes []spiral.SpiralNode, width, height int) trackParams { ... }
func spiralGrowthRate(first, last spiral.SpiralNode) float64 { ... }
func spiralTrackSteps(nodeCount int) int { ... }
```

### Lint rules to watch

- **funlen: 65 lines** (ignore-comments)
- **revive: max-public-structs 5/file**, line-length 120, cognitive-complexity 10, no flag-parameter, no unused-parameter, early-return preferred
- **gci:** import order (stdlib, blank line, third-party)
- **godox:** no TODO/BUG/FIXME comments
- **nilaway:** nil guards before dereferencing
- **nlreturn:** blank line before return statements
- **unparam:** parameters must vary across call sites

### Build/test/lint commands

- `task build` — Build the codeviz binary
- `task test` — Run all tests
- `task lint` — Run golangci-lint
- `task ci` — Build, test, lint (full CI)

---

## Task 1: Strip colour fields from SpiralNode

**Files:**
- Modify: `internal/spiral/node.go`
- Modify: `internal/render/spiral.go` (will break — expected, fixed in Task 5)
- Modify: `internal/render/svg_spiral.go` (will break — expected, fixed in Task 5)
- Modify: `internal/render/spiral_test.go` (will break — expected, fixed in Task 5)

**Goal:** Remove `FillColour` and `BorderColour` from `SpiralNode` so it becomes geometry-only. This is the first step to decouple layout from colour.

- [ ] **Step 1: Remove colour fields from SpiralNode**

Edit `internal/spiral/node.go` to remove `FillColour` and `BorderColour` fields, and the `image/color` import:

```go
// Package spiral implements data types and layout algorithms for spiral timeline visualizations.
package spiral

import (
	"time"

	"github.com/theunrepentantgeek/code-visualizer/internal/viz"
)

// LabelMode is an alias for [viz.LabelMode].
type LabelMode = viz.LabelMode

const (
	LabelAll  = viz.LabelAll
	LabelLaps = viz.LabelLaps
	LabelNone = viz.LabelNone
)

// SpiralNode is a positioned visual element on the rendered spiral timeline.
// X and Y are absolute pixel coordinates on the canvas.
type SpiralNode struct {
	X, Y         float64   // pixel position on canvas
	DiscRadius   float64   // radius in pixels (from size metric)
	Angle        float64   // angle in radians (clockwise from 12-o'clock / north)
	SpiralRadius float64   // distance from canvas centre to this point
	TimeStart    time.Time // start of this time bucket (inclusive)
	TimeEnd      time.Time // end of this time bucket (exclusive)
	Label        string    // time label (e.g. "2pm", "Apr 29")
	ShowLabel    bool      // whether to render label
}
```

- [ ] **Step 2: Verify the render package breaks**

Run: `task build 2>&1 | head -20`
Expected: Compilation errors in `internal/render/spiral.go`, `internal/render/svg_spiral.go`, and `internal/render/spiral_test.go` referencing `FillColour` and `BorderColour`. This is expected — they will be fixed when the old render code is deleted in Task 5.

- [ ] **Step 3: Commit**

```bash
git add internal/spiral/node.go
git commit -m "refactor(spiral): strip colour fields from SpiralNode

SpiralNode becomes geometry-only. FillColour and BorderColour removed.
Intentionally breaks render/spiral.go — fixed when old render code is
deleted in a later task.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 2: Build spiral-to-canvas bridge

**Files:**
- Create: `cmd/codeviz/spiral_canvas.go`

**Goal:** Create the bridge that translates spiral layout nodes + time-bucket data into Canvas shapes. This is analogous to `treemap_canvas.go` but operates on a flat list of time-bucket-aggregated values instead of a per-file tree.

**Dependencies:** Task 1 (SpiralNode is now geometry-only)

- [ ] **Step 1: Create `cmd/codeviz/spiral_canvas.go`**

```go
package main

import (
	"image/color"
	"math"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/spiral"
)

var (
	spiralDefaultFill   = color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
	spiralDefaultBorder = color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
	spiralTrackColour   = color.RGBA{R: 0xDD, G: 0xDD, B: 0xDD, A: 0xFF}
	spiralLabelColour   = color.RGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF}
	spiralBgColour      = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
)

const (
	spiralTrackWidth    = 1.0
	spiralLabelGap      = 4.0
	spiralTrackMinSteps = 500
)

// spiralInks holds the Ink instances for a spiral render pass.
type spiralInks struct {
	fill   canvas.Ink
	border canvas.Ink
}

// buildSpiralInks creates fill and border inks from aggregated time-bucket data.
func buildSpiralInks(
	buckets []spiral.TimeBucket,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
	borderMetric metric.Name,
	borderPaletteName palette.PaletteName,
) spiralInks {
	inks := spiralInks{
		fill:   canvas.FixedInk(spiralDefaultFill),
		border: canvas.FixedInk(spiralDefaultBorder),
	}

	if fillMetric != "" {
		inks.fill = buildBucketInk(
			buckets, fillMetric, fillPaletteName,
			func(b *spiral.TimeBucket) float64 { return b.FillValue },
			func(b *spiral.TimeBucket) string { return b.FillLabel },
			spiralDefaultFill,
		)
	}

	if borderMetric != "" {
		inks.border = buildBucketInk(
			buckets, borderMetric, borderPaletteName,
			func(b *spiral.TimeBucket) float64 { return b.BorderValue },
			func(b *spiral.TimeBucket) string { return b.BorderLabel },
			spiralDefaultBorder,
		)
	}

	return inks
}

// buildBucketInk creates an Ink from time-bucket-aggregated metric values.
func buildBucketInk(
	buckets []spiral.TimeBucket,
	m metric.Name,
	palName palette.PaletteName,
	numericFn func(*spiral.TimeBucket) float64,
	categoryFn func(*spiral.TimeBucket) string,
	fallback color.RGBA,
) canvas.Ink {
	p, ok := provider.Get(m)
	if !ok {
		return canvas.FixedInk(fallback)
	}

	pal := palette.GetPalette(palName)

	if p.Kind() == metric.Quantity || p.Kind() == metric.Measure {
		values := make([]float64, len(buckets))
		for i := range buckets {
			values[i] = numericFn(&buckets[i])
		}

		return canvas.NumericInk(m, values, pal)
	}

	seen := map[string]bool{}

	var categories []string

	for i := range buckets {
		cat := categoryFn(&buckets[i])
		if cat != "" && !seen[cat] {
			seen[cat] = true
			categories = append(categories, cat)
		}
	}

	return canvas.CategoricalInk(m, categories, pal)
}

// renderSpiralToCanvas builds a Canvas from spiral nodes and time buckets.
func renderSpiralToCanvas(
	nodes []spiral.SpiralNode,
	buckets []spiral.TimeBucket,
	width, height int,
	inks spiralInks,
) *canvas.Canvas {
	cv := canvas.NewCanvas(width, height)

	addSpiralBackground(cv, width, height)
	addSpiralTrack(cv, nodes, width, height)
	addSpiralDiscs(cv, nodes, buckets, inks)
	addSpiralLabels(cv, nodes)

	return cv
}

// addSpiralBackground adds the white background rectangle.
func addSpiralBackground(cv *canvas.Canvas, width, height int) {
	bgSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        canvas.FixedInk(spiralBgColour),
			Border:      canvas.FixedInk(spiralBgColour),
			BorderWidth: 0,
		},
	}

	cv.AddRectangle(canvas.LayerBackground, canvas.Rectangle{
		Spec: bgSpec,
		W:    float64(width), H: float64(height),
	})
}

// addSpiralTrack adds the faint guide curve as a Path on the Structure layer.
func addSpiralTrack(cv *canvas.Canvas, nodes []spiral.SpiralNode, width, height int) {
	if len(nodes) < 2 {
		return
	}

	params := inferSpiralTrackParams(nodes, width, height)
	steps := spiralTrackSteps(len(nodes))
	points := make([]canvas.Position, steps)

	for i := range steps {
		t := float64(i) / float64(steps-1)
		theta := t * params.maxTheta
		r := params.a + params.b*theta
		points[i] = canvas.Position{
			X: params.cx + r*math.Sin(theta),
			Y: params.cy - r*math.Cos(theta),
		}
	}

	trackSpec := &canvas.LineSpec{
		Stroke:      canvas.FixedInk(spiralTrackColour),
		StrokeWidth: spiralTrackWidth,
	}

	cv.AddPath(canvas.LayerStructure, canvas.Path{
		Spec:   trackSpec,
		Points: points,
	})
}

// addSpiralDiscs adds filled circles with borders for each active node.
func addSpiralDiscs(
	cv *canvas.Canvas,
	nodes []spiral.SpiralNode,
	buckets []spiral.TimeBucket,
	inks spiralInks,
) {
	for i, n := range nodes {
		if n.DiscRadius <= 0 {
			continue
		}

		fillMV := spiralMetricValue(&buckets[i], inks.fill)
		borderMV := spiralMetricValue(&buckets[i], inks.border)

		discSpec := &canvas.DiscSpec{
			ShapeStyle: canvas.ShapeStyle{
				Fill:        inks.fill,
				Border:      inks.border,
				BorderWidth: spiralBorderWidth(n.DiscRadius),
			},
		}

		cv.AddDisc(canvas.LayerContent, canvas.Disc{
			Spec:   discSpec,
			X:      n.X,
			Y:      n.Y,
			Radius: n.DiscRadius,
			Angle:  n.Angle,
			Fill:   fillMV,
			Border: borderMV,
		})
	}
}

// addSpiralLabels adds rotated text labels tangent to the spiral.
func addSpiralLabels(cv *canvas.Canvas, nodes []spiral.SpiralNode) {
	for _, n := range nodes {
		if !n.ShowLabel || n.Label == "" {
			continue
		}

		addSpiralLabel(cv, n)
	}
}

// addSpiralLabel adds a single rotated label for a spiral node.
func addSpiralLabel(cv *canvas.Canvas, n spiral.SpiralNode) {
	labelR := n.DiscRadius + spiralLabelGap
	lx := n.X + labelR*math.Sin(n.Angle)
	ly := n.Y - labelR*math.Cos(n.Angle)

	norm := math.Mod(n.Angle, 2*math.Pi)
	if norm < 0 {
		norm += 2 * math.Pi
	}

	var anchor canvas.TextAnchor

	var rotation float64

	if norm <= math.Pi {
		anchor = canvas.AnchorStart
		rotation = n.Angle
	} else {
		anchor = canvas.AnchorEnd
		rotation = n.Angle + math.Pi
	}

	labelSpec := &canvas.TextSpec{
		Ink:      canvas.FixedInk(spiralLabelColour),
		FontSize: 0,
		Anchor:   anchor,
		Rotation: rotation,
	}

	cv.AddText(canvas.LayerOverlay, canvas.Text{
		Spec:    labelSpec,
		X:       lx,
		Y:       ly,
		Content: n.Label,
	})
}

// spiralMetricValue builds a MetricValue from a time bucket for the given ink.
func spiralMetricValue(bucket *spiral.TimeBucket, ink canvas.Ink) canvas.MetricValue {
	info := ink.Info()

	switch info.Kind {
	case canvas.InkNumeric:
		return canvas.MeasureValue(bucket.FillValue)
	case canvas.InkCategorical:
		return canvas.CategoryValue(bucket.FillLabel)
	default:
		return canvas.MetricValue{}
	}
}

// spiralBorderWidth returns the border stroke width for a spiral disc.
func spiralBorderWidth(discRadius float64) float64 {
	if discRadius < 8 {
		return 2.0
	}

	return 3.0
}

// spiralTrackSteps returns the number of interpolation steps for the track curve.
func spiralTrackSteps(nodeCount int) int {
	steps := 3 * nodeCount
	if steps < spiralTrackMinSteps {
		return spiralTrackMinSteps
	}

	return steps
}

// spiralTrackParams holds spiral geometry for drawing the guide track.
type spiralTrackParams struct {
	cx, cy   float64
	a, b     float64
	maxTheta float64
}

// inferSpiralTrackParams reconstructs Archimedean spiral parameters from nodes.
func inferSpiralTrackParams(
	nodes []spiral.SpiralNode,
	width, height int,
) spiralTrackParams {
	cx := float64(width) / 2
	cy := float64(height) / 2

	first := nodes[0]
	last := nodes[len(nodes)-1]

	var b float64
	if last.Angle > 0 {
		b = (last.SpiralRadius - first.SpiralRadius) / last.Angle
	}

	return spiralTrackParams{
		cx:       cx,
		cy:       cy,
		a:        first.SpiralRadius,
		b:        b,
		maxTheta: last.Angle,
	}
}
```

**Important:** The `spiralMetricValue` function needs to look at whether the ink is for fill or border and select the right bucket field accordingly. The above implementation always uses `FillValue`/`FillLabel` regardless of which ink is passed. This needs fixing — see the corrected version below:

Actually, the issue is that `spiralMetricValue` can't know which bucket field to use from the ink alone. Instead, pass the appropriate values directly. Replace `spiralMetricValue` with two helpers:

```go
// spiralFillMetricValue builds a fill MetricValue from a time bucket.
func spiralFillMetricValue(bucket *spiral.TimeBucket, ink canvas.Ink) canvas.MetricValue {
	info := ink.Info()

	switch info.Kind {
	case canvas.InkNumeric:
		return canvas.MeasureValue(bucket.FillValue)
	case canvas.InkCategorical:
		return canvas.CategoryValue(bucket.FillLabel)
	default:
		return canvas.MetricValue{}
	}
}

// spiralBorderMetricValue builds a border MetricValue from a time bucket.
func spiralBorderMetricValue(bucket *spiral.TimeBucket, ink canvas.Ink) canvas.MetricValue {
	info := ink.Info()

	switch info.Kind {
	case canvas.InkNumeric:
		return canvas.MeasureValue(bucket.BorderValue)
	case canvas.InkCategorical:
		return canvas.CategoryValue(bucket.BorderLabel)
	default:
		return canvas.MetricValue{}
	}
}
```

And update `addSpiralDiscs` to use them:

```go
		fillMV := spiralFillMetricValue(&buckets[i], inks.fill)
		borderMV := spiralBorderMetricValue(&buckets[i], inks.border)
```

Delete the single `spiralMetricValue` function and replace with these two.

- [ ] **Step 2: Verify the file compiles (ignoring render breakage)**

Run: `go build ./cmd/codeviz/ 2>&1 | grep -v 'render/spiral'`
Expected: No errors from `spiral_canvas.go`.

- [ ] **Step 3: Commit**

```bash
git add cmd/codeviz/spiral_canvas.go
git commit -m "feat(spiral): add spiral-to-canvas bridge

Creates spiralInks, buildSpiralInks, buildBucketInk, and
renderSpiralToCanvas functions. Uses Canvas Disc shapes for nodes,
Path for the guide track, and Text with radial rotation for labels.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 3: Wire spiral command to Canvas pipeline

**Files:**
- Modify: `cmd/codeviz/spiral_cmd.go`

**Goal:** Replace the old `layoutAndRender` method to use the Canvas pipeline instead of `render.RenderSpiral()`. Delete the old colour application functions (`applyFill`, `applyBorder`, `applySpiralNumericFill`, `applySpiralCategoricalFill`, `applySpiralNumericBorder`, `applySpiralCategoricalBorder`, `collectBucketCategories`).

**Dependencies:** Task 2 (bridge functions exist)

- [ ] **Step 1: Rewrite `layoutAndRender` to use Canvas pipeline**

Replace the `layoutAndRender` method in `cmd/codeviz/spiral_cmd.go` with:

```go
func (c *SpiralCmd) layoutAndRender(
	flags *Flags,
	cfg *config.Spiral,
	root *model.Directory,
	buckets []spiral.TimeBucket,
) error {
	width := ptrInt(flags.Config.Width, 1920)
	height := ptrInt(flags.Config.Height, 1920)
	resolution := c.resolveResolution(cfg)
	labels := c.resolveLabels(cfg)

	nodes := spiral.Layout(buckets, width, height, resolution, labels)
	maxDisc := spiral.MaxDiscRadius(len(buckets), width, height, resolution)
	applySpiralDiscSizes(nodes, buckets, maxDisc)

	fillMetric := c.resolveFillMetric(cfg)
	fillPaletteName := c.resolveFillPalette(cfg, fillMetric)
	borderMetric, borderPaletteName := c.resolveBorderMetricAndPalette(cfg)

	inks := buildSpiralInks(buckets, fillMetric, fillPaletteName, borderMetric, borderPaletteName)

	legendPos, legendOrient := resolveLegendOptions(ptrString(cfg.Legend), ptrString(cfg.LegendOrientation))
	sizeMetric := metric.Name(ptrString(cfg.Size))

	if legendPos != render.LegendNone {
		slog.Warn("Legend rendering not yet supported in Canvas pipeline; legend will be omitted")
	}

	_ = legendOrient
	_ = sizeMetric

	slog.Info("Rendering image", "output", c.Output, "width", width, "height", height)

	cv := renderSpiralToCanvas(nodes, buckets, width, height, inks)

	if err := cv.Render(c.Output); err != nil {
		return eris.Wrap(err, "render failed")
	}

	c.logRendered(root, width, height, sizeMetric, fillMetric, fillPaletteName, borderMetric, borderPaletteName)

	return nil
}
```

- [ ] **Step 2: Extract `resolveBorderMetricAndPalette` helper**

The old `applyBorder` returned `(metric.Name, palette.PaletteName)` while also stamping colours. Extract just the metric/palette resolution into a clean helper:

```go
// resolveBorderMetricAndPalette resolves the border metric name and palette.
func (*SpiralCmd) resolveBorderMetricAndPalette(
	cfg *config.Spiral,
) (metric.Name, palette.PaletteName) {
	border := specMetric(cfg.Border)
	if border == "" {
		return "", ""
	}

	borderPaletteName := specPalette(cfg.Border)
	if borderPaletteName == "" {
		if p, ok := provider.Get(border); ok {
			borderPaletteName = p.DefaultPalette()
		} else {
			borderPaletteName = palette.Neutral
		}
	}

	return border, borderPaletteName
}
```

- [ ] **Step 3: Delete the old colour application functions**

Remove these functions from `cmd/codeviz/spiral_cmd.go`:
- `applyFill` (lines 571–596)
- `applySpiralNumericFill` (lines 598–614)
- `applySpiralCategoricalFill` (lines 616–629)
- `applyBorder` (lines 632–665)
- `applySpiralNumericBorder` (lines 667–684)
- `applySpiralCategoricalBorder` (lines 686–700)
- `collectBucketCategories` (lines 703–722)

- [ ] **Step 4: Remove unused imports**

After deleting the colour functions, remove these imports from `spiral_cmd.go` if they are no longer used:
- `"github.com/theunrepentantgeek/code-visualizer/internal/render"` — keep only if `render.LegendNone`, `render.FormatFromPath`, or `resolveLegendOptions` still references it
- `"github.com/theunrepentantgeek/code-visualizer/internal/palette"` — may still be needed for `palette.PaletteName`, `palette.Neutral`, `palette.GetPalette`

Check carefully: `render.FormatFromPath` is called in `validatePaths()` — if it's still there, keep the render import. If it can be replaced with `canvas.FormatFromPath`, do so.

- [ ] **Step 5: Verify build**

Run: `go build ./cmd/codeviz/ 2>&1 | grep -v 'render/spiral'`
Expected: Only errors from `internal/render/spiral.go` (the old render code still references `FillColour`/`BorderColour`). The command itself should compile.

- [ ] **Step 6: Commit**

```bash
git add cmd/codeviz/spiral_cmd.go
git commit -m "feat(spiral): wire spiral command to Canvas pipeline

Rewrote layoutAndRender to use buildSpiralInks + renderSpiralToCanvas +
cv.Render(). Deleted 7 old colour application functions (~150 lines).
Added resolveBorderMetricAndPalette helper. Legend temporarily omitted
with slog.Warn.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 4: Remove spiral code from render package

**Files:**
- Delete: `internal/render/spiral.go`
- Delete: `internal/render/svg_spiral.go`
- Delete: `internal/render/spiral_test.go`

**Goal:** Remove all spiral-specific rendering code from the old render package. After this task, the build should be clean.

**Dependencies:** Task 3 (spiral command no longer references `render.RenderSpiral`)

- [ ] **Step 1: Delete the spiral render files**

```bash
rm internal/render/spiral.go
rm internal/render/svg_spiral.go
rm internal/render/spiral_test.go
```

- [ ] **Step 2: Check for remaining references**

Run: `grep -rn 'RenderSpiral\|renderSpiralImage\|renderSpiralSVG\|drawSpiralTrack\|drawSpiralDiscs\|drawSpiralLabels\|drawSingleSpot\|drawSpiralLabel\|writeSpiralSVG\|sampleSpiralNodes' internal/render/ cmd/codeviz/`
Expected: No results.

- [ ] **Step 3: Verify full build and test**

Run: `task build && task test`
Expected: Build succeeds. All tests pass (the deleted spiral_test.go tests are replaced by Canvas-based tests in Task 5).

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "refactor(render): remove spiral-specific render code

Deleted spiral.go (PNG rendering), svg_spiral.go (SVG rendering),
and spiral_test.go (6 tests). Spiral now renders via the Canvas pipeline.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 5: Add Canvas-based spiral rendering tests

**Files:**
- Create: `cmd/codeviz/spiral_canvas_test.go`

**Goal:** Add tests for the spiral Canvas bridge functions (ink building, disc creation, track generation, label rendering, end-to-end output).

**Dependencies:** Task 4 (clean build)

- [ ] **Step 1: Create test file**

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
	"time"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/spiral"
)

func makeSpiralTestFile(name, ext string, size int64) *model.File {
	f := &model.File{Name: name, Extension: ext}
	f.SetQuantity(filesystem.FileSize, size)
	f.SetClassification(filesystem.FileType, ext)

	return f
}

func sampleTimeBuckets() []spiral.TimeBucket {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	return []spiral.TimeBucket{
		{
			Start: t0, End: t0.Add(time.Hour),
			Files: []*model.File{
				makeSpiralTestFile("a.go", "go", 100),
				makeSpiralTestFile("b.go", "go", 200),
			},
			SizeValue: 300, FillValue: 300, FillLabel: "go",
		},
		{
			Start: t0.Add(time.Hour), End: t0.Add(2 * time.Hour),
			Files: []*model.File{
				makeSpiralTestFile("c.py", "py", 50),
			},
			SizeValue: 50, FillValue: 50, FillLabel: "py",
		},
		{
			Start: t0.Add(2 * time.Hour), End: t0.Add(3 * time.Hour),
			Files:     []*model.File{},
			SizeValue: 0,
		},
	}
}

func TestSpiralBorderWidth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(spiralBorderWidth(5)).To(Equal(2.0))
	g.Expect(spiralBorderWidth(7)).To(Equal(2.0))
	g.Expect(spiralBorderWidth(8)).To(Equal(3.0))
	g.Expect(spiralBorderWidth(20)).To(Equal(3.0))
}

func TestBuildSpiralInks_Numeric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := sampleTimeBuckets()
	inks := buildSpiralInks(
		buckets,
		filesystem.FileSize, palette.Foliage,
		"", "",
	)

	g.Expect(inks.fill.Info().Kind).To(Equal(canvas.InkNumeric))
	g.Expect(inks.fill.Info().MetricName).To(Equal(metric.Name(filesystem.FileSize)))
	g.Expect(inks.border.Info().Kind).To(Equal(canvas.InkFixed))
}

func TestBuildSpiralInks_Categorical(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := sampleTimeBuckets()
	inks := buildSpiralInks(
		buckets,
		filesystem.FileType, palette.Categorization,
		"", "",
	)

	g.Expect(inks.fill.Info().Kind).To(Equal(canvas.InkCategorical))
	g.Expect(inks.border.Info().Kind).To(Equal(canvas.InkFixed))
}

func TestRenderSpiralToCanvas_PNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := sampleTimeBuckets()
	nodes := spiral.Layout(buckets, 800, 800, spiral.Hourly, spiral.LabelLaps)
	inks := buildSpiralInks(buckets, "", "", "", "")

	cv := renderSpiralToCanvas(nodes, buckets, 800, 800, inks)
	out := filepath.Join(t.TempDir(), "spiral.png")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	f, openErr := os.Open(out)
	g.Expect(openErr).NotTo(HaveOccurred())

	defer f.Close()

	_, imgFmt, decErr := image.Decode(f)
	g.Expect(decErr).NotTo(HaveOccurred())
	g.Expect(imgFmt).To(Equal("png"))
}

func TestRenderSpiralToCanvas_SVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := sampleTimeBuckets()
	nodes := spiral.Layout(buckets, 800, 800, spiral.Daily, spiral.LabelAll)
	inks := buildSpiralInks(buckets, "", "", "", "")

	cv := renderSpiralToCanvas(nodes, buckets, 800, 800, inks)
	out := filepath.Join(t.TempDir(), "spiral.svg")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	data, readErr := os.ReadFile(out)
	g.Expect(readErr).NotTo(HaveOccurred())

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

func TestRenderSpiralToCanvas_JPG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := sampleTimeBuckets()
	nodes := spiral.Layout(buckets, 800, 800, spiral.Hourly, spiral.LabelNone)
	inks := buildSpiralInks(buckets, "", "", "", "")

	cv := renderSpiralToCanvas(nodes, buckets, 800, 800, inks)
	out := filepath.Join(t.TempDir(), "spiral.jpg")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	info, statErr := os.Stat(out)
	g.Expect(statErr).NotTo(HaveOccurred())
	g.Expect(info.Size()).To(BeNumerically(">", 0))
}
```

- [ ] **Step 2: Run tests**

Run: `task test`
Expected: All tests pass, including the 6 new spiral canvas tests.

- [ ] **Step 3: Commit**

```bash
git add cmd/codeviz/spiral_canvas_test.go
git commit -m "test(spiral): add Canvas-based spiral rendering tests

6 tests: border width, numeric inks, categorical inks, end-to-end
PNG/SVG/JPG output.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 6: Accept temporary legend regression

**Files:**
- Modify: `cmd/codeviz/spiral_cmd.go` (if not already done in Task 3)

**Goal:** Ensure the legend regression is properly logged. The Canvas pipeline doesn't render legends yet — this is a known temporary regression that will be addressed when legends are migrated to Canvas as a separate feature.

**Dependencies:** Task 3

- [ ] **Step 1: Verify legend warning is in place**

Check that `layoutAndRender` already contains:

```go
if legendPos != render.LegendNone {
    slog.Warn("Legend rendering not yet supported in Canvas pipeline; legend will be omitted")
}
```

If not, add it.

- [ ] **Step 2: Verify the render import can be replaced**

Check if `render.FormatFromPath` in `validatePaths()` can be replaced with `canvas.FormatFromPath`. If the Canvas package has its own `FormatFromPath`, switch to it. This may allow removing the `internal/render` import entirely from `spiral_cmd.go`.

Check if `render.LegendNone` can be replaced with a direct string check on `ptrString(cfg.Legend)` (e.g., `ptrString(cfg.Legend) != "" && ptrString(cfg.Legend) != "none"`). This would fully decouple `spiral_cmd.go` from the render package.

- [ ] **Step 3: Commit if changes were made**

```bash
git add cmd/codeviz/spiral_cmd.go
git commit -m "feat(spiral): accept temporary legend regression

Legend rendering not yet supported in Canvas pipeline. Warning logged
when legend position is configured.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 7: CI verification

**Files:** None (verification only)

**Goal:** Run the full CI pipeline to verify the migration is clean.

**Dependencies:** Tasks 1–6

- [ ] **Step 1: Run full CI**

Run: `task ci`
Expected: Build passes, all tests pass, 0 lint issues.

- [ ] **Step 2: Fix any lint issues**

Common issues to watch for:
- `gci` import ordering
- `nlreturn` blank line before returns
- `revive` early-return, unused-parameter, flag-parameter
- `funlen` if any function exceeds 65 lines
- `unparam` if any parameter is always the same across call sites

Fix issues and re-run `task ci` until clean.

- [ ] **Step 3: Verify no changes to non-spiral viz types**

Run: `git diff --name-only main..HEAD`
Expected: Only files related to spiral migration. No changes to:
- `cmd/codeviz/treemap_*.go`
- `cmd/codeviz/radialtree_cmd.go`
- `cmd/codeviz/bubbletree_cmd.go`
- `internal/render/radialtree.go`, `internal/render/bubbletree.go`
- `internal/render/svg_radial.go`, `internal/render/svg_bubble.go`

- [ ] **Step 4: Commit any lint fixes**

```bash
git add -A
git commit -m "fix(spiral): address CI lint issues

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 8: Create pull request

**Files:** None (git operations only)

**Dependencies:** Task 7 (clean CI)

- [ ] **Step 1: Push branch**

```bash
git push -u origin feature/canvas-spiral
```

- [ ] **Step 2: Create PR**

```bash
gh pr create \
  --title "feat: migrate spiral visualization to Canvas pipeline" \
  --body "## Summary

Migrates the spiral visualization from the old internal/render/ pipeline to the new Canvas abstraction (Stage 2, second of four viz types).

## Changes

### Spiral migration
- Stripped FillColour/BorderColour fields from SpiralNode (now geometry-only)
- Created spiral_canvas.go bridge: flat node+bucket iteration, Canvas Disc/Path/Text shapes with MetricValues
- Rewired layoutAndRender to use buildSpiralInks → renderSpiralToCanvas → cv.Render()
- Deleted 7 old colour application functions from spiral_cmd.go

### Render package cleanup
- Deleted spiral.go (PNG rendering), svg_spiral.go (SVG rendering)
- Deleted spiral_test.go (6 old tests, replaced with Canvas-based tests)

### Testing
- 6 new Canvas-based spiral tests (ink construction, border width, end-to-end PNG/SVG/JPG)

## Known regression
Legend rendering temporarily unavailable for spiral. Warning logged when legend position is configured.

## Stats
{run git diff --stat main..HEAD and paste here}" \
  --base main
```
