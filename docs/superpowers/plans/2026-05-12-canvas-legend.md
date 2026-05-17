# Canvas Legend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restore legend rendering through the Canvas pipeline for all four visualization types (treemap, spiral, radial tree, bubbletree).

**Architecture:** Add `DrawLegend` to the Backend interface. Shared measurement/positioning lives in a new `legendlayout` package. Each backend (raster, SVG) implements format-specific legend rendering. Canvas converts its user-facing `LegendConfig` to backend-facing `LegendData` using Ink's internal swatch data, then delegates to the backend after shape dispatch. The CLI layer passes the same Ink objects used for visualization shapes into the legend, ensuring colour consistency.

**Tech Stack:** Go 1.26+ · fogleman/gg (text measurement + raster drawing) · Gomega (assertions) · Goldie v2 (golden-file snapshots)

**Spec:** `docs/superpowers/specs/2026-05-12-canvas-legend-design.md`

**Commands:** `task build` · `task test` · `task lint` · `task ci`

---

## File Structure

### New files
| File | Purpose |
|------|---------|
| `internal/canvas/model/legend.go` | Backend-facing legend types: `LegendData`, `LegendEntryData`, `LegendSwatch`, `LegendEntryKind` + rendering constants |
| `internal/canvas/legendlayout/layout.go` | Shared measurement (`MeasureLegend`), positioning (`LegendOrigin`), space reservation (`ReserveSpace`) |
| `internal/canvas/legendlayout/layout_test.go` | Tests for measurement, positioning, space reservation, `FormatBreakpoint` |
| `internal/canvas/raster/legend.go` | Raster legend drawing via gg (adapted from `render/legend_png.go`) |
| `internal/canvas/raster/legend_test.go` | Tests for raster legend rendering |
| `internal/canvas/svg/legend.go` | SVG legend writing as XML (adapted from `render/legend_svg.go`) |
| `internal/canvas/svg/legend_test.go` | Tests for SVG legend rendering |
| `internal/canvas/legend_test.go` | Tests for `toLegendData`, `ReserveSpace`, `DefaultOrientation` |

### Modified files
| File | Changes |
|------|---------|
| `internal/canvas/model/backend.go` | Add `DrawLegend(data LegendData, canvasW, canvasH int)` to `Backend` interface |
| `internal/canvas/ink.go` | Add unexported `legendSwatches()` and `legendEntryKind()` methods |
| `internal/canvas/legend.go` | Add `DefaultOrientation`, `ReserveSpace()`, `toLegendData()` |
| `internal/canvas/canvas.go` | Add legend dispatch in `RenderTo()` after shape loop |
| `internal/canvas/mock_backend_test.go` | Add `DrawLegend` to mock |
| `internal/canvas/raster/backend.go` | Add `DrawLegend` method (delegates to legend.go) |
| `internal/canvas/svg/backend.go` | Add `DrawLegend` method (delegates to legend.go) |
| `cmd/codeviz/legend_builder.go` | Rewrite to produce `canvas.LegendConfig` using Ink objects |
| `cmd/codeviz/legend_builder_test.go` | Update tests for canvas types |
| `cmd/codeviz/treemap_cmd.go` | Reorder ink/legend flow, wire `SetLegend`, remove warning, switch `render.FormatFromPath` → `canvas.FormatFromPath` |
| `cmd/codeviz/spiral_cmd.go` | Wire `SetLegend`, remove warning |
| `cmd/codeviz/radialtree_cmd.go` | Wire `SetLegend`, remove warning |
| `cmd/codeviz/bubbletree_cmd.go` | Wire `SetLegend` |

### Deleted files
| File | Reason |
|------|--------|
| `internal/render/legend.go` | Types/measurement migrated to `model` + `legendlayout` |
| `internal/render/legend_png.go` | Drawing code migrated to `raster/legend.go` |
| `internal/render/legend_svg.go` | Writing code migrated to `svg/legend.go` |
| `internal/render/legend_test.go` | Tests migrated to `legendlayout` + backend tests |
| `internal/render/svg_helpers.go` | `colourToHex` replaced by `rgbaToCSS` in svg backend |
| `internal/render/save.go` | `saveContextPNG` only used by deleted tests |

---

### Task 1: Backend-facing legend types and interface

**Files:**
- Create: `internal/canvas/model/legend.go`
- Modify: `internal/canvas/model/backend.go`
- Modify: `internal/canvas/mock_backend_test.go`
- Modify: `internal/canvas/raster/backend.go`
- Modify: `internal/canvas/svg/backend.go`

- [ ] **Step 1: Create `model/legend.go` with types and constants**

```go
package model

import (
	"image/color"
)

// LegendEntryKind distinguishes numeric (continuous gradient) from
// categorical (discrete label) legend entries.
type LegendEntryKind int

const (
	// LegendEntryNumeric is for Quantity/Measure metrics with colour gradients.
	LegendEntryNumeric LegendEntryKind = iota
	// LegendEntryCategorical is for Classification metrics with labelled swatches.
	LegendEntryCategorical
)

// LegendData holds fully resolved rendering data for a legend overlay.
// Position and Orientation use the same string values as canvas.LegendPosition
// and canvas.LegendOrientation (e.g., "bottom-right", "vertical").
type LegendData struct {
	Position    string
	Orientation string
	Entries     []LegendEntryData
}

// LegendEntryData describes one metric section within the legend.
type LegendEntryData struct {
	Title    string // e.g., "Fill: file-size"
	Kind     LegendEntryKind
	Swatches []LegendSwatch
}

// LegendSwatch pairs a colour with an optional label.
// For numeric entries the label is the breakpoint value at the divider
// (empty string on the last swatch). For categorical entries every
// swatch has a label.
type LegendSwatch struct {
	Colour color.RGBA
	Label  string
}

// Legend rendering constants — shared by all backends.
const (
	LegendPadding    = 12.0
	LegendMargin     = 16.0
	SwatchSize       = 28.0
	SwatchGap        = 4.0
	LabelGap         = 6.0
	EntryGap         = 14.0
	LegendFontSize   = 12.0
	TitleFontSize    = 13.0
	LegendLineHeight = 16.0
)
```

- [ ] **Step 2: Add `DrawLegend` to the `Backend` interface**

In `internal/canvas/model/backend.go`, add the `DrawLegend` method to the `Backend` interface, between `DrawArcText` and `Finish`:

```go
DrawLegend(data LegendData, canvasW, canvasH int)
```

The full interface becomes:

```go
type Backend interface {
	DrawRectangle(pos Position, size Size, fill, border color.RGBA, borderWidth float64)
	DrawDisc(center Position, radius float64, fill, border color.RGBA, borderWidth float64)
	DrawLine(from, to Position, stroke color.RGBA, strokeWidth float64)
	DrawPath(points []Position, stroke color.RGBA, strokeWidth float64)
	DrawText(pos Position, text string, ink color.RGBA, fontSize float64, anchor TextAnchor, rotation float64)
	DrawArcText(center Position, radius float64, text string, ink color.RGBA, fontSize float64)
	DrawLegend(data LegendData, canvasW, canvasH int)
	Finish(outputPath string) error
}
```

- [ ] **Step 3: Add `DrawLegend` stub to `mockBackend` in `mock_backend_test.go`**

Add a `legendData` field and the method:

```go
// Add field to mockBackend struct:
legendData *model.LegendData

// Add method:
func (m *mockBackend) DrawLegend(data model.LegendData, canvasW, canvasH int) {
	m.legendData = &data
}
```

Note: `mock_backend_test.go` is in package `canvas`, so use the full import `model` for the type. The existing file already imports `"image/color"` — add `"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"` to the imports.

- [ ] **Step 4: Add `DrawLegend` stub to raster backend**

In `internal/canvas/raster/backend.go`, add:

```go
func (r *rasterBackend) DrawLegend(_ model.LegendData, _, _ int) {
	// Legend rendering implemented in legend.go (Task 5)
}
```

- [ ] **Step 5: Add `DrawLegend` stub to SVG backend**

In `internal/canvas/svg/backend.go`, add:

```go
func (s *svgBackend) DrawLegend(_ model.LegendData, _, _ int) {
	// Legend rendering implemented in legend.go (Task 6)
}
```

- [ ] **Step 6: Verify build passes**

Run: `task build`
Expected: BUILD SUCCESS — all types compile, interface satisfied

- [ ] **Step 7: Commit**

```bash
git add internal/canvas/model/legend.go internal/canvas/model/backend.go \
       internal/canvas/mock_backend_test.go \
       internal/canvas/raster/backend.go internal/canvas/svg/backend.go
git commit -m "feat(canvas): add legend types and DrawLegend to Backend interface

Add LegendData, LegendEntryData, LegendSwatch types to model package.
Add DrawLegend method to Backend interface with stubs in all backends.
Add rendering constants (padding, margins, swatch sizes)."
```

---

### Task 2: Shared measurement and positioning (legendlayout)

**Files:**
- Create: `internal/canvas/legendlayout/layout.go`
- Create: `internal/canvas/legendlayout/layout_test.go`

This package adapts the measurement and positioning logic from `internal/render/legend.go` and the measurement functions from `internal/render/legend_png.go` (lines 271–421). It uses `gg.NewContext(1, 1)` for text measurement — the same approach the SVG legend already uses.

- [ ] **Step 1: Write layout_test.go with measurement, positioning, and format tests**

```go
package legendlayout

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
)

func TestFormatBreakpoint_IntegerValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(FormatBreakpoint(42)).To(Equal("42"))
	g.Expect(FormatBreakpoint(0)).To(Equal("0"))
	g.Expect(FormatBreakpoint(1000)).To(Equal("1000"))
}

func TestFormatBreakpoint_FloatValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(FormatBreakpoint(3.14)).To(Equal("3.1"))
	g.Expect(FormatBreakpoint(0.5)).To(Equal("0.5"))
}

func TestLegendOrigin_AllPositions_InBounds(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	canvasW, canvasH := 800.0, 600.0
	legendW, legendH := 100.0, 50.0

	positions := []string{
		"top-left", "top-center", "top-right",
		"center-right", "bottom-right", "bottom-center",
		"bottom-left", "center-left",
	}

	for _, pos := range positions {
		ox, oy := LegendOrigin(pos, canvasW, canvasH, legendW, legendH)
		g.Expect(ox).To(BeNumerically(">=", 0), "x out of bounds for %s", pos)
		g.Expect(oy).To(BeNumerically(">=", 0), "y out of bounds for %s", pos)
		g.Expect(ox+legendW).To(BeNumerically("<=", canvasW),
			"right edge out of bounds for %s", pos)
		g.Expect(oy+legendH).To(BeNumerically("<=", canvasH),
			"bottom edge out of bounds for %s", pos)
	}
}

func TestLegendOrigin_TopLeft_IsNearOrigin(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ox, oy := LegendOrigin("top-left", 800, 600, 100, 50)
	g.Expect(ox).To(Equal(model.LegendMargin))
	g.Expect(oy).To(Equal(model.LegendMargin))
}

func TestLegendOrigin_BottomRight_IsNearCorner(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ox, oy := LegendOrigin("bottom-right", 800, 600, 100, 50)
	g.Expect(ox).To(Equal(800.0 - 100.0 - model.LegendMargin))
	g.Expect(oy).To(Equal(600.0 - 50.0 - model.LegendMargin))
}

func TestMeasureLegend_EmptyEntries_ReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data := &model.LegendData{Orientation: "vertical"}
	w, h := MeasureLegend(data)
	g.Expect(w).To(BeZero())
	g.Expect(h).To(BeZero())
}

func TestMeasureLegend_Nil_ReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	w, h := MeasureLegend(nil)
	g.Expect(w).To(BeZero())
	g.Expect(h).To(BeZero())
}

func TestMeasureLegend_Vertical_NonZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data := makeSampleLegendData("vertical")
	w, h := MeasureLegend(data)
	g.Expect(w).To(BeNumerically(">", 0))
	g.Expect(h).To(BeNumerically(">", 0))
}

func TestMeasureLegend_Horizontal_WiderThanVertical(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dataH := makeSampleLegendData("horizontal")
	dataV := makeSampleLegendData("vertical")
	wH, _ := MeasureLegend(dataH)
	wV, _ := MeasureLegend(dataV)
	g.Expect(wH).To(BeNumerically(">", wV),
		"horizontal legend should be wider than vertical")
}

func TestMeasureLegend_Horizontal_ShorterThanVertical(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dataH := makeSampleLegendData("horizontal")
	dataV := makeSampleLegendData("vertical")
	_, hH := MeasureLegend(dataH)
	_, hV := MeasureLegend(dataV)
	g.Expect(hH).To(BeNumerically("<", hV),
		"horizontal legend should be shorter than vertical")
}

func TestReserveSpace_NilData_ReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	wReduce, hReduce := ReserveSpace(nil)
	g.Expect(wReduce).To(BeZero())
	g.Expect(hReduce).To(BeZero())
}

func TestReserveSpace_NonePosition_ReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data := &model.LegendData{Position: "none"}
	wReduce, hReduce := ReserveSpace(data)
	g.Expect(wReduce).To(BeZero())
	g.Expect(hReduce).To(BeZero())
}

func TestReserveSpace_CenterRight_ReducesWidth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data := makeSampleLegendData("vertical")
	data.Position = "center-right"
	wReduce, hReduce := ReserveSpace(data)
	g.Expect(wReduce).To(BeNumerically(">", 0))
	g.Expect(hReduce).To(BeZero())
}

func TestReserveSpace_BottomCenter_ReducesHeight(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data := makeSampleLegendData("vertical")
	data.Position = "bottom-center"
	wReduce, hReduce := ReserveSpace(data)
	g.Expect(wReduce).To(BeZero())
	g.Expect(hReduce).To(BeNumerically(">", 0))
}

func TestReserveSpace_CornerVertical_ReducesWidth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data := makeSampleLegendData("vertical")
	data.Position = "bottom-right"
	wReduce, hReduce := ReserveSpace(data)
	g.Expect(wReduce).To(BeNumerically(">", 0))
	g.Expect(hReduce).To(BeZero())
}

func TestReserveSpace_CornerHorizontal_ReducesHeight(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data := makeSampleLegendData("horizontal")
	data.Position = "bottom-right"
	wReduce, hReduce := ReserveSpace(data)
	g.Expect(wReduce).To(BeZero())
	g.Expect(hReduce).To(BeNumerically(">", 0))
}

// makeSampleLegendData creates test legend data with both numeric and
// categorical entries.
func makeSampleLegendData(orientation string) *model.LegendData {
	return &model.LegendData{
		Position:    "bottom-right",
		Orientation: orientation,
		Entries: []model.LegendEntryData{
			{
				Title: "Fill: file-size",
				Kind:  model.LegendEntryNumeric,
				Swatches: []model.LegendSwatch{
					{Colour: color.RGBA{R: 50, G: 50, B: 200, A: 255}, Label: "100"},
					{Colour: color.RGBA{R: 100, G: 100, B: 200, A: 255}, Label: "500"},
					{Colour: color.RGBA{R: 150, G: 150, B: 200, A: 255}, Label: "1000"},
					{Colour: color.RGBA{R: 200, G: 200, B: 200, A: 255}, Label: "5000"},
					{Colour: color.RGBA{R: 250, G: 250, B: 200, A: 255}, Label: ""},
				},
			},
			{
				Title: "Border: file-type",
				Kind:  model.LegendEntryCategorical,
				Swatches: []model.LegendSwatch{
					{Colour: color.RGBA{R: 0, G: 173, B: 216, A: 255}, Label: "go"},
					{Colour: color.RGBA{R: 222, G: 165, B: 132, A: 255}, Label: "rs"},
					{Colour: color.RGBA{R: 53, G: 114, B: 165, A: 255}, Label: "py"},
				},
			},
		},
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `task test`
Expected: FAIL — `legendlayout` package doesn't exist yet

- [ ] **Step 3: Create `layout.go` with measurement, positioning, and space reservation**

Create `internal/canvas/legendlayout/layout.go`. Adapt the measurement functions from `render/legend_png.go:271-421`, positioning from `render/legend.go:153-200`, and `formatBreakpoint` from `render/legend_png.go:433-439`.

Key differences from the old code:
- Functions accept `*model.LegendData` instead of `*LegendInfo`
- Entry iteration uses `model.LegendEntryData` with `Kind`/`Swatches` fields instead of accessor methods
- `MeasureLegend` creates its own `gg.NewContext(1,1)` internally
- All constants referenced via `model.LegendPadding` etc.

```go
// Package legendlayout provides shared legend measurement and positioning
// used by the Canvas layer and both backends.
package legendlayout

import (
	"fmt"
	"strconv"

	"github.com/fogleman/gg"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
)

// FormatBreakpoint formats a numeric breakpoint for display.
func FormatBreakpoint(v float64) string {
	if v == float64(int(v)) {
		return strconv.Itoa(int(v))
	}

	return fmt.Sprintf("%.1f", v)
}

// MeasureLegend computes the total width and height of the legend box
// including padding. Returns (0, 0) if data is nil or has no entries.
func MeasureLegend(data *model.LegendData) (width, height float64) {
	if data == nil || len(data.Entries) == 0 {
		return 0, 0
	}

	dc := gg.NewContext(1, 1)

	if data.Orientation == "horizontal" {
		return measureLegendH(dc, data)
	}

	return measureLegendV(dc, data)
}

// LegendOrigin computes the top-left (x, y) for the legend box.
func LegendOrigin(
	position string,
	canvasW, canvasH float64,
	legendW, legendH float64,
) (ox, oy float64) {
	m := model.LegendMargin

	switch position {
	case "top-left":
		return m, m
	case "top-center":
		return (canvasW - legendW) / 2, m
	case "top-right":
		return canvasW - legendW - m, m
	case "center-right":
		return canvasW - legendW - m, (canvasH - legendH) / 2
	case "bottom-right":
		return canvasW - legendW - m, canvasH - legendH - m
	case "bottom-center":
		return (canvasW - legendW) / 2, canvasH - legendH - m
	case "center-left":
		return m, (canvasH - legendH) / 2
	default:
		return m, canvasH - legendH - m
	}
}

// ReserveSpace computes the width and height reductions needed to reserve
// space for the legend. Returns zeros if data is nil, position is "none",
// or there are no entries.
func ReserveSpace(data *model.LegendData) (widthReduction, heightReduction float64) {
	if data == nil || data.Position == "none" || len(data.Entries) == 0 {
		return 0, 0
	}

	w, h := MeasureLegend(data)
	m := model.LegendMargin

	switch data.Position {
	case "center-left", "center-right":
		return w + 2*m, 0
	case "top-center", "bottom-center":
		return 0, h + 2*m
	default:
		if data.Orientation == "vertical" {
			return w + 2*m, 0
		}

		return 0, h + 2*m
	}
}

func measureLegendV(dc *gg.Context, data *model.LegendData) (width, height float64) {
	var totalH float64

	maxW := 0.0

	for i, entry := range data.Entries {
		if i > 0 {
			totalH += model.EntryGap
		}

		tw, _ := dc.MeasureString(entry.Title)
		totalH += model.TitleFontSize + model.LabelGap

		if tw > maxW {
			maxW = tw
		}

		entryW, entryH := measureEntryV(dc, entry)
		totalH += entryH

		if entryW > maxW {
			maxW = entryW
		}
	}

	return maxW + 2*model.LegendPadding, totalH + 2*model.LegendPadding
}

func measureLegendH(dc *gg.Context, data *model.LegendData) (width, height float64) {
	var totalW float64

	maxH := 0.0

	for i, entry := range data.Entries {
		if i > 0 {
			totalW += model.EntryGap
		}

		entryW, entryH := measureSingleEntryH(dc, entry)
		totalW += entryW

		if entryH > maxH {
			maxH = entryH
		}
	}

	return totalW + 2*model.LegendPadding, maxH + 2*model.LegendPadding
}

func measureSingleEntryH(dc *gg.Context, entry model.LegendEntryData) (width, height float64) {
	tw, _ := dc.MeasureString(entry.Title)
	titleH := model.TitleFontSize + model.LabelGap

	entryW, entryH := measureEntryH(dc, entry)

	w := max(tw, entryW)
	h := titleH + entryH

	return w, h
}

func measureEntryV(dc *gg.Context, entry model.LegendEntryData) (width, height float64) {
	if entry.Kind == model.LegendEntryCategorical {
		return measureCategoryV(dc, entry)
	}

	return measureNumericV(dc, entry)
}

func measureEntryH(dc *gg.Context, entry model.LegendEntryData) (width, height float64) {
	if entry.Kind == model.LegendEntryCategorical {
		return measureCategoryH(dc, entry)
	}

	return measureNumericH(entry)
}

func measureNumericV(dc *gg.Context, entry model.LegendEntryData) (width, height float64) {
	n := len(entry.Swatches)
	h := float64(n) * model.SwatchSize
	w := model.SwatchSize

	for _, sw := range entry.Swatches {
		if sw.Label != "" {
			tw, _ := dc.MeasureString(sw.Label)

			if bw := model.SwatchSize + model.LabelGap + tw; bw > w {
				w = bw
			}
		}
	}

	return w, h
}

func measureNumericH(entry model.LegendEntryData) (width, height float64) {
	n := len(entry.Swatches)
	w := float64(n) * model.SwatchSize
	h := model.SwatchSize + model.LegendLineHeight + model.LabelGap

	return w, h
}

func measureCategoryV(dc *gg.Context, entry model.LegendEntryData) (width, height float64) {
	n := len(entry.Swatches)

	w := model.SwatchSize
	h := float64(n) * (model.SwatchSize + model.SwatchGap)

	for _, sw := range entry.Swatches {
		tw, _ := dc.MeasureString(sw.Label)

		if cw := model.SwatchSize + model.LabelGap + tw; cw > w {
			w = cw
		}
	}

	return w, h
}

func measureCategoryH(dc *gg.Context, entry model.LegendEntryData) (width, height float64) {
	w := 0.0

	for _, sw := range entry.Swatches {
		tw, _ := dc.MeasureString(sw.Label)
		w += max(model.SwatchSize, tw) + model.SwatchGap + model.LabelGap
	}

	h := model.SwatchSize + model.LegendLineHeight + model.LabelGap

	return w, h
}
```

- [ ] **Step 4: Run tests**

Run: `task test`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/canvas/legendlayout/
git commit -m "feat(canvas): add legendlayout package for shared measurement

Measurement, positioning, and space reservation functions adapted from
internal/render/legend.go. Uses gg.NewContext(1,1) for text measurement."
```

---

### Task 3: Ink swatch extraction

**Files:**
- Modify: `internal/canvas/ink.go`
- Create: `internal/canvas/ink_legend_test.go`

Add methods to Ink that extract the resolved swatch data needed for legend rendering. The Ink already carries bucket boundaries, palette, categories, and the categorical mapper — these methods expose that data as `model.LegendSwatch` slices.

- [ ] **Step 1: Write `ink_legend_test.go`**

```go
package canvas

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

func TestLegendSwatches_FixedInk_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := FixedInk(white)
	g.Expect(ink.legendSwatches()).To(BeNil())
}

func TestLegendSwatches_NumericInk_ReturnsBucketColours(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Temperature)
	ink := NumericInk("file-size", []float64{10, 50, 100, 500, 1000}, pal)

	swatches := ink.legendSwatches()
	g.Expect(swatches).NotTo(BeEmpty())

	// Each swatch should have a non-zero colour
	for _, sw := range swatches {
		g.Expect(sw.Colour.A).To(Equal(uint8(255)))
	}

	// Last swatch should have empty label (no boundary after last bucket)
	g.Expect(swatches[len(swatches)-1].Label).To(BeEmpty())
}

func TestLegendSwatches_CategoricalInk_ReturnsCategoryLabels(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Categorization)
	ink := CategoricalInk("file-type", []string{"go", "py", "rs"}, pal)

	swatches := ink.legendSwatches()
	g.Expect(swatches).To(HaveLen(3))

	// Labels should be sorted and present
	g.Expect(swatches[0].Label).To(Equal("go"))
	g.Expect(swatches[1].Label).To(Equal("py"))
	g.Expect(swatches[2].Label).To(Equal("rs"))
}

func TestLegendEntryKind_FixedInk_ReturnsNumeric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := FixedInk(white)
	g.Expect(ink.legendEntryKind()).To(Equal(model.LegendEntryNumeric))
}

func TestLegendEntryKind_CategoricalInk_ReturnsCategorical(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Categorization)
	ink := CategoricalInk("file-type", []string{"go"}, pal)
	g.Expect(ink.legendEntryKind()).To(Equal(model.LegendEntryCategorical))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `task test`
Expected: FAIL — `legendSwatches` and `legendEntryKind` don't exist

- [ ] **Step 3: Add `legendSwatches()` and `legendEntryKind()` to `ink.go`**

Add at the bottom of `internal/canvas/ink.go`:

```go
// legendEntryKind returns the LegendEntryKind for this ink.
func (ink Ink) legendEntryKind() model.LegendEntryKind {
	if ink.kind == inkCategorical {
		return model.LegendEntryCategorical
	}

	return model.LegendEntryNumeric
}

// legendSwatches extracts resolved swatch data for legend rendering.
// Returns nil for fixed inks (no meaningful swatch data).
func (ink Ink) legendSwatches() []model.LegendSwatch {
	switch ink.kind {
	case inkNumeric:
		return ink.numericLegendSwatches()
	case inkCategorical:
		return ink.categoricalLegendSwatches()
	default:
		return nil
	}
}

func (ink Ink) numericLegendSwatches() []model.LegendSwatch {
	if ink.boundaries == nil {
		return nil
	}

	n := ink.boundaries.NumBuckets()
	if n <= 0 || len(ink.pal.Colours) == 0 {
		return nil
	}

	swatches := make([]model.LegendSwatch, n)

	for i := range n {
		colour := palette.MapNumericToColour(i, n, ink.pal)

		var label string
		if i < len(ink.boundaries.Boundaries) {
			label = legendlayout.FormatBreakpoint(ink.boundaries.Boundaries[i])
		}

		swatches[i] = model.LegendSwatch{
			Colour: colour,
			Label:  label,
		}
	}

	return swatches
}

func (ink Ink) categoricalLegendSwatches() []model.LegendSwatch {
	if ink.catMapper == nil || len(ink.categories) == 0 {
		return nil
	}

	sorted := make([]string, len(ink.categories))
	copy(sorted, ink.categories)
	slices.Sort(sorted)

	swatches := make([]model.LegendSwatch, len(sorted))

	for i, cat := range sorted {
		swatches[i] = model.LegendSwatch{
			Colour: ink.catMapper.Map(cat),
			Label:  cat,
		}
	}

	return swatches
}
```

Add to the imports at the top of `ink.go`:
- `"slices"`
- `"github.com/theunrepentantgeek/code-visualizer/internal/canvas/legendlayout"`
- `"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"`

Note: `ink.go` already imports `palette` and `metric`.

- [ ] **Step 4: Run tests**

Run: `task test`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/canvas/ink.go internal/canvas/ink_legend_test.go
git commit -m "feat(canvas): add legend swatch extraction to Ink

Add legendSwatches() and legendEntryKind() methods to Ink.
Numeric inks produce bucket colour + breakpoint label swatches.
Categorical inks produce sorted category label + colour swatches.
Fixed inks return nil (used for size-only legend entries)."
```

---

### Task 4: Canvas legend methods

**Files:**
- Modify: `internal/canvas/legend.go`
- Create: `internal/canvas/legend_test.go`

Update the existing `LegendConfig` type with `DefaultOrientation`, `ReserveSpace`, and `toLegendData` methods.

- [ ] **Step 1: Write `legend_test.go`**

```go
package canvas

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

func TestDefaultOrientation_CenterPositions_ReturnsHorizontal(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(DefaultOrientation(LegendPositionTopCenter)).To(Equal(LegendOrientationHorizontal))
	g.Expect(DefaultOrientation(LegendPositionBottomCenter)).To(Equal(LegendOrientationHorizontal))
}

func TestDefaultOrientation_SidePositions_ReturnsVertical(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sides := []LegendPosition{
		LegendPositionTopLeft,
		LegendPositionTopRight,
		LegendPositionCenterRight,
		LegendPositionBottomRight,
		LegendPositionBottomLeft,
		LegendPositionCenterLeft,
	}

	for _, pos := range sides {
		g.Expect(DefaultOrientation(pos)).To(Equal(LegendOrientationVertical),
			"expected vertical for %s", pos)
	}
}

func TestToLegendData_NilEntries_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	lc := &LegendConfig{
		Position:    LegendPositionNone,
		Orientation: LegendOrientationVertical,
	}

	g.Expect(lc.toLegendData()).To(BeNil())
}

func TestToLegendData_NumericEntry_ProducesSwatches(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Temperature)
	fillInk := NumericInk("file-size", []float64{10, 50, 100, 500, 1000}, pal)

	lc := &LegendConfig{
		Position:    LegendPositionBottomRight,
		Orientation: LegendOrientationVertical,
		Entries: []LegendEntry{
			{Role: LegendRoleFill, MetricName: "file-size", Ink: fillInk},
		},
	}

	data := lc.toLegendData()
	g.Expect(data).NotTo(BeNil())
	g.Expect(data.Position).To(Equal("bottom-right"))
	g.Expect(data.Orientation).To(Equal("vertical"))
	g.Expect(data.Entries).To(HaveLen(1))
	g.Expect(data.Entries[0].Title).To(Equal("Fill: file-size"))
	g.Expect(data.Entries[0].Kind).To(Equal(model.LegendEntryNumeric))
	g.Expect(data.Entries[0].Swatches).NotTo(BeEmpty())
}

func TestToLegendData_CategoricalEntry_ProducesSwatches(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Categorization)
	borderInk := CategoricalInk("file-type", []string{"go", "py", "rs"}, pal)

	lc := &LegendConfig{
		Position:    LegendPositionTopLeft,
		Orientation: LegendOrientationHorizontal,
		Entries: []LegendEntry{
			{Role: LegendRoleBorder, MetricName: "file-type", Ink: borderInk},
		},
	}

	data := lc.toLegendData()
	g.Expect(data).NotTo(BeNil())
	g.Expect(data.Entries[0].Kind).To(Equal(model.LegendEntryCategorical))
	g.Expect(data.Entries[0].Swatches).To(HaveLen(3))
}

func TestToLegendData_FixedInkEntry_EmptySwatches(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	lc := &LegendConfig{
		Position:    LegendPositionBottomRight,
		Orientation: LegendOrientationVertical,
		Entries: []LegendEntry{
			{Role: LegendRoleSize, MetricName: "file-lines", Ink: FixedInk(white)},
		},
	}

	data := lc.toLegendData()
	g.Expect(data).NotTo(BeNil())
	g.Expect(data.Entries[0].Kind).To(Equal(model.LegendEntryNumeric))
	g.Expect(data.Entries[0].Swatches).To(BeNil())
}

func TestReserveSpace_NonePosition_ReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	lc := &LegendConfig{Position: LegendPositionNone}
	wReduce, hReduce := lc.ReserveSpace()
	g.Expect(wReduce).To(BeZero())
	g.Expect(hReduce).To(BeZero())
}

func TestReserveSpace_WithEntries_NonZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Temperature)
	fillInk := NumericInk("file-size", []float64{10, 50, 100}, pal)

	lc := &LegendConfig{
		Position:    LegendPositionCenterRight,
		Orientation: LegendOrientationVertical,
		Entries: []LegendEntry{
			{Role: LegendRoleFill, MetricName: "file-size", Ink: fillInk},
		},
	}

	wReduce, hReduce := lc.ReserveSpace()
	g.Expect(wReduce).To(BeNumerically(">", 0))
	g.Expect(hReduce).To(BeZero())
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `task test`
Expected: FAIL — `DefaultOrientation`, `toLegendData`, `ReserveSpace` don't exist

- [ ] **Step 3: Add methods to `legend.go`**

Add to the bottom of `internal/canvas/legend.go`:

```go
// DefaultOrientation returns the default orientation for a given position.
// Center positions default to horizontal; all others to vertical.
func DefaultOrientation(pos LegendPosition) LegendOrientation {
	switch pos {
	case LegendPositionTopCenter, LegendPositionBottomCenter:
		return LegendOrientationHorizontal
	default:
		return LegendOrientationVertical
	}
}

// ReserveSpace computes the width and height reductions needed to reserve
// space for the legend within the canvas. Returns zeros if the legend is
// disabled or has no entries.
func (lc *LegendConfig) ReserveSpace() (widthReduction, heightReduction float64) {
	data := lc.toLegendData()

	return legendlayout.ReserveSpace(data)
}

// toLegendData converts the canvas-facing LegendConfig to the backend-facing
// LegendData. Returns nil if the legend is disabled or has no entries.
func (lc *LegendConfig) toLegendData() *model.LegendData {
	if lc == nil || lc.Position == LegendPositionNone || len(lc.Entries) == 0 {
		return nil
	}

	entries := make([]model.LegendEntryData, len(lc.Entries))

	for i, e := range lc.Entries {
		entries[i] = model.LegendEntryData{
			Title:    string(e.Role) + ": " + e.MetricName,
			Kind:     e.Ink.legendEntryKind(),
			Swatches: e.Ink.legendSwatches(),
		}
	}

	return &model.LegendData{
		Position:    string(lc.Position),
		Orientation: string(lc.Orientation),
		Entries:     entries,
	}
}
```

Add imports at the top of `legend.go`:
```go
import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/legendlayout"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
)
```

- [ ] **Step 4: Run tests**

Run: `task test`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/canvas/legend.go internal/canvas/legend_test.go
git commit -m "feat(canvas): add DefaultOrientation, ReserveSpace, and toLegendData

LegendConfig can now convert to model.LegendData using Ink swatch
extraction and compute space reservation for treemap layout."
```

---

### Task 5: Canvas RenderTo legend dispatch + raster backend rendering

**Files:**
- Modify: `internal/canvas/canvas.go`
- Modify: `internal/canvas/canvas_test.go`
- Modify: `internal/canvas/raster/backend.go`
- Create: `internal/canvas/raster/legend.go`
- Create: `internal/canvas/raster/legend_test.go`

- [ ] **Step 1: Add legend dispatch test in `canvas_test.go`**

Add at the end of `canvas_test.go`:

```go
func TestCanvas_SetLegend_DispatchesToBackend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)

	pal := palette.GetPalette(palette.Temperature)
	fillInk := NumericInk("file-size", []float64{10, 50, 100}, pal)

	c.SetLegend(LegendConfig{
		Position:    LegendPositionBottomRight,
		Orientation: LegendOrientationVertical,
		Entries: []LegendEntry{
			{Role: LegendRoleFill, MetricName: "file-size", Ink: fillInk},
		},
	})

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.legendData).NotTo(BeNil())
	g.Expect(mb.legendData.Position).To(Equal("bottom-right"))
	g.Expect(mb.legendData.Entries).To(HaveLen(1))
}

func TestCanvas_NoLegend_DoesNotCallDrawLegend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	mb := newMockBackend()

	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.legendData).To(BeNil())
}
```

- [ ] **Step 2: Add legend dispatch to `RenderTo` in `canvas.go`**

In `internal/canvas/canvas.go`, update `RenderTo()`. After the shape dispatch loop and before `return nil`, add:

```go
	if c.legend != nil {
		data := c.legend.toLegendData()
		if data != nil {
			backend.DrawLegend(*data, c.width, c.height)
		}
	}
```

Also update the `SetLegend` comment — remove the "Note: legend rendering is not yet implemented" line:

```go
// SetLegend configures the legend overlay for this canvas.
func (c *Canvas) SetLegend(config LegendConfig) {
	c.legend = &config
}
```

- [ ] **Step 3: Run canvas tests**

Run: `task test`
Expected: ALL PASS (canvas tests + legend dispatch tests)

- [ ] **Step 4: Write `raster/legend_test.go`**

```go
package raster

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
)

func TestDrawLegend_EmptyData_DoesNotPanic(t *testing.T) {
	t.Parallel()

	b := New(800, 600)
	b.DrawLegend(model.LegendData{Position: "none"}, 800, 600)
}

func TestDrawLegend_Vertical_ProducesImage(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(800, 600)
	b.DrawLegend(*makeSampleData("vertical"), 800, 600)

	out := filepath.Join(t.TempDir(), "legend.png")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	fi, statErr := os.Stat(out)
	g.Expect(statErr).NotTo(HaveOccurred())

	if fi != nil {
		g.Expect(fi.Size()).To(BeNumerically(">", 0))
	}
}

func TestDrawLegend_Horizontal_ProducesImage(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(800, 600)
	b.DrawLegend(*makeSampleData("horizontal"), 800, 600)

	out := filepath.Join(t.TempDir(), "legend.png")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	fi, statErr := os.Stat(out)
	g.Expect(statErr).NotTo(HaveOccurred())

	if fi != nil {
		g.Expect(fi.Size()).To(BeNumerically(">", 0))
	}
}

func makeSampleData(orientation string) *model.LegendData {
	return &model.LegendData{
		Position:    "bottom-right",
		Orientation: orientation,
		Entries: []model.LegendEntryData{
			{
				Title: "Fill: file-size",
				Kind:  model.LegendEntryNumeric,
				Swatches: []model.LegendSwatch{
					{Colour: color.RGBA{R: 50, G: 50, B: 200, A: 255}, Label: "100"},
					{Colour: color.RGBA{R: 100, G: 100, B: 200, A: 255}, Label: "500"},
					{Colour: color.RGBA{R: 200, G: 200, B: 200, A: 255}, Label: ""},
				},
			},
			{
				Title: "Border: file-type",
				Kind:  model.LegendEntryCategorical,
				Swatches: []model.LegendSwatch{
					{Colour: color.RGBA{R: 0, G: 173, B: 216, A: 255}, Label: "go"},
					{Colour: color.RGBA{R: 222, G: 165, B: 132, A: 255}, Label: "rs"},
				},
			},
		},
	}
}
```

- [ ] **Step 5: Create `raster/legend.go`**

Adapt from `render/legend_png.go`. The raster backend's `DrawLegend` method calls into this file. Key differences from old code:
- Uses `model.LegendData` / `model.LegendEntryData` / `model.LegendSwatch` instead of `render.LegendEntry`
- Uses `legendlayout.MeasureLegend()` and `legendlayout.LegendOrigin()` for sizing
- Accesses `.Kind` and `.Swatches` fields directly instead of via accessor methods
- Uses `model.` prefixed constants

The raster backend's `DrawLegend` in `backend.go` should be updated to call the actual implementation:

In `internal/canvas/raster/backend.go`, replace the stub:

```go
func (r *rasterBackend) DrawLegend(data model.LegendData, canvasW, canvasH int) {
	drawLegend(r.dc, data, canvasW, canvasH)
}
```

Create `internal/canvas/raster/legend.go` — adapt from `internal/render/legend_png.go:16-439`. Use `model.LegendData` types throughout. Reference `legendlayout.MeasureLegend` and `legendlayout.LegendOrigin` for measurement and positioning. Split functions to stay under the 65-line funlen limit.

- [ ] **Step 6: Run tests**

Run: `task test`
Expected: ALL PASS

- [ ] **Step 7: Commit**

```bash
git add internal/canvas/canvas.go internal/canvas/canvas_test.go \
       internal/canvas/raster/backend.go internal/canvas/raster/legend.go \
       internal/canvas/raster/legend_test.go
git commit -m "feat(canvas): legend dispatch in RenderTo + raster backend rendering

Canvas.RenderTo now calls backend.DrawLegend after shape dispatch.
Raster backend renders legends with gg (adapted from render/legend_png.go)."
```

---

### Task 6: SVG backend legend rendering

**Files:**
- Modify: `internal/canvas/svg/backend.go`
- Create: `internal/canvas/svg/legend.go`
- Create: `internal/canvas/svg/legend_test.go`

- [ ] **Step 1: Write `svg/legend_test.go`**

```go
package svg

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
)

func TestDrawLegend_EmptyData_DoesNotPanic(t *testing.T) {
	t.Parallel()

	b := New(800, 600)
	b.DrawLegend(model.LegendData{Position: "none"}, 800, 600)
}

func TestDrawLegend_Vertical_ProducesOutput(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(800, 600)
	b.DrawLegend(*makeSampleData("vertical"), 800, 600)

	out := filepath.Join(t.TempDir(), "legend.svg")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	content, readErr := os.ReadFile(out)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("<g transform="))
	g.Expect(string(content)).To(ContainSubstring("file-size"))
	g.Expect(string(content)).To(ContainSubstring("file-type"))
}

func TestDrawLegend_Horizontal_ProducesOutput(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(800, 600)
	b.DrawLegend(*makeSampleData("horizontal"), 800, 600)

	out := filepath.Join(t.TempDir(), "legend.svg")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	content, readErr := os.ReadFile(out)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("<g transform="))
	g.Expect(string(content)).To(ContainSubstring("fill-opacity"))
}

func makeSampleData(orientation string) *model.LegendData {
	return &model.LegendData{
		Position:    "bottom-right",
		Orientation: orientation,
		Entries: []model.LegendEntryData{
			{
				Title: "Fill: file-size",
				Kind:  model.LegendEntryNumeric,
				Swatches: []model.LegendSwatch{
					{Colour: color.RGBA{R: 50, G: 50, B: 200, A: 255}, Label: "100"},
					{Colour: color.RGBA{R: 100, G: 100, B: 200, A: 255}, Label: "500"},
					{Colour: color.RGBA{R: 200, G: 200, B: 200, A: 255}, Label: ""},
				},
			},
			{
				Title: "Border: file-type",
				Kind:  model.LegendEntryCategorical,
				Swatches: []model.LegendSwatch{
					{Colour: color.RGBA{R: 0, G: 173, B: 216, A: 255}, Label: "go"},
					{Colour: color.RGBA{R: 222, G: 165, B: 132, A: 255}, Label: "rs"},
				},
			},
		},
	}
}
```

- [ ] **Step 2: Create `svg/legend.go`**

Adapt from `render/legend_svg.go`. Key differences:
- Writes to `*bytes.Buffer` (the svgBackend's buf) instead of `*os.File`
- Uses `model.LegendData` types and `legendlayout.MeasureLegend` / `legendlayout.LegendOrigin`
- Uses `rgbaToCSS` (already in svg/backend.go) instead of `colourToHex`
- Accesses `.Kind` and `.Swatches` fields directly

Update `svg/backend.go` to call the actual implementation:

```go
func (s *svgBackend) DrawLegend(data model.LegendData, canvasW, canvasH int) {
	writeSVGLegend(&s.buf, data, canvasW, canvasH)
}
```

Create `internal/canvas/svg/legend.go` — adapt from `internal/render/legend_svg.go`. Write to `*bytes.Buffer` instead of `*os.File`. Use `legendlayout.MeasureLegend` for text measurement. Use `rgbaToCSS` for colour formatting. Use `legendlayout.LegendOrigin` for positioning. Split functions to stay under the 65-line funlen limit.

- [ ] **Step 3: Run tests**

Run: `task test`
Expected: ALL PASS

- [ ] **Step 4: Commit**

```bash
git add internal/canvas/svg/backend.go internal/canvas/svg/legend.go \
       internal/canvas/svg/legend_test.go
git commit -m "feat(canvas): SVG backend legend rendering

SVG backend renders legends as XML elements (adapted from
render/legend_svg.go). Uses legendlayout for shared measurement."
```

---

### Task 7: CLI integration — legend builder and all viz commands

**Files:**
- Modify: `cmd/codeviz/legend_builder.go`
- Modify: `cmd/codeviz/legend_builder_test.go`
- Modify: `cmd/codeviz/treemap_cmd.go`
- Modify: `cmd/codeviz/spiral_cmd.go`
- Modify: `cmd/codeviz/radialtree_cmd.go`
- Modify: `cmd/codeviz/bubbletree_cmd.go`

- [ ] **Step 1: Rewrite `legend_builder.go` to use canvas types**

Replace the contents of `cmd/codeviz/legend_builder.go`. The new version:
- Returns `canvas.LegendConfig` instead of `render.LegendInfo`
- Accepts Ink objects instead of independently computing buckets
- Uses `canvas.LegendPosition`, `canvas.LegendOrientation`, `canvas.LegendEntry`
- Removes `buildLegendEntry`, `buildCategorySwatches` (no longer needed — Ink carries data)
- Removes the `render` import

```go
package main

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

// resolveLegendOptions resolves the legend position and orientation from config.
// Empty position defaults to "bottom-right"; empty orientation is resolved from position.
func resolveLegendOptions(posStr, orientStr string) (canvas.LegendPosition, canvas.LegendOrientation) {
	pos := canvas.LegendPosition(posStr)
	if pos == "" {
		pos = canvas.LegendPositionBottomRight
	}

	orient := canvas.LegendOrientation(orientStr)
	if orient == "" {
		orient = canvas.DefaultOrientation(pos)
	}

	return pos, orient
}

// buildLegendConfig constructs a LegendConfig from resolved options and
// the pre-built Ink objects used for rendering. Returns nil if the legend
// is disabled (position "none") or no entries are produced.
func buildLegendConfig(
	position canvas.LegendPosition,
	orientation canvas.LegendOrientation,
	fillInk canvas.Ink,
	fillMetric metric.Name,
	borderInk canvas.Ink,
	borderMetric metric.Name,
	sizeMetric metric.Name,
) *canvas.LegendConfig {
	if position == canvas.LegendPositionNone {
		return nil
	}

	if orientation == "" {
		orientation = canvas.DefaultOrientation(position)
	}

	var entries []canvas.LegendEntry

	if fillMetric != "" {
		entries = append(entries, canvas.LegendEntry{
			Role:       canvas.LegendRoleFill,
			MetricName: string(fillMetric),
			Ink:        fillInk,
		})
	}

	if borderMetric != "" {
		entries = append(entries, canvas.LegendEntry{
			Role:       canvas.LegendRoleBorder,
			MetricName: string(borderMetric),
			Ink:        borderInk,
		})
	}

	if sizeMetric != "" && sizeMetric != fillMetric {
		entries = append(entries, canvas.LegendEntry{
			Role:       canvas.LegendRoleSize,
			MetricName: string(sizeMetric),
			Ink:        canvas.FixedInk(white),
		})
	}

	if len(entries) == 0 {
		return nil
	}

	return &canvas.LegendConfig{
		Position:    position,
		Orientation: orientation,
		Entries:     entries,
	}
}
```

Note: `white` is already defined in `ink_introspection.go` as a package-level variable. If not accessible from `cmd/codeviz`, use `color.RGBA{R: 255, G: 255, B: 255, A: 255}` and add the `"image/color"` import.

- [ ] **Step 2: Rewrite `legend_builder_test.go` to use canvas types**

Replace the contents of `cmd/codeviz/legend_builder_test.go`:

```go
package main

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

func TestResolveLegendOptions_EmptyDefaults(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pos, orient := resolveLegendOptions("", "")
	g.Expect(pos).To(Equal(canvas.LegendPositionBottomRight))
	g.Expect(orient).To(Equal(canvas.LegendOrientationVertical))
}

func TestResolveLegendOptions_ExplicitValues(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pos, orient := resolveLegendOptions("top-left", "horizontal")
	g.Expect(pos).To(Equal(canvas.LegendPositionTopLeft))
	g.Expect(orient).To(Equal(canvas.LegendOrientationHorizontal))
}

func TestResolveLegendOptions_PositionOnly_DerivesOrientation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pos, orient := resolveLegendOptions("top-center", "")
	g.Expect(pos).To(Equal(canvas.LegendPositionTopCenter))
	g.Expect(orient).To(Equal(canvas.LegendOrientationHorizontal))
}

func TestResolveLegendOptions_None_DisablesLegend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pos, _ := resolveLegendOptions("none", "")
	g.Expect(pos).To(Equal(canvas.LegendPositionNone))
}

func TestBuildLegendConfig_NonePosition_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := buildLegendConfig(
		canvas.LegendPositionNone, canvas.LegendOrientationVertical,
		canvas.FixedInk(color.RGBA{A: 255}), "file-size",
		canvas.FixedInk(color.RGBA{A: 255}), "",
		"file-lines",
	)

	g.Expect(cfg).To(BeNil())
}

func TestBuildLegendConfig_FillOnly_SingleEntry(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := makeLegendTestRoot()
	pal := palette.GetPalette(palette.Temperature)
	values := collectNumericValues(root, "file-size")
	fillInk := canvas.NumericInk("file-size", values, pal)

	cfg := buildLegendConfig(
		canvas.LegendPositionBottomRight, canvas.LegendOrientationVertical,
		fillInk, "file-size",
		canvas.FixedInk(color.RGBA{A: 255}), "",
		"file-size",
	)

	if cfg == nil {
		t.Fatal("expected non-nil LegendConfig")
	} else {
		g.Expect(cfg.Entries).To(HaveLen(1))
		g.Expect(cfg.Entries[0].Role).To(Equal(canvas.LegendRoleFill))
		g.Expect(cfg.Entries[0].MetricName).To(Equal("file-size"))
	}
}

func TestBuildLegendConfig_FillAndBorder_TwoEntries(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := makeLegendTestRoot()
	pal := palette.GetPalette(palette.Temperature)
	values := collectNumericValues(root, "file-size")
	fillInk := canvas.NumericInk("file-size", values, pal)

	types := collectDistinctTypes(root, "file-type")
	catPal := palette.GetPalette(palette.Categorization)
	borderInk := canvas.CategoricalInk("file-type", types, catPal)

	cfg := buildLegendConfig(
		canvas.LegendPositionBottomRight, canvas.LegendOrientationVertical,
		fillInk, "file-size",
		borderInk, "file-type",
		"file-size",
	)

	if cfg == nil {
		t.Fatal("expected non-nil LegendConfig")
	} else {
		g.Expect(cfg.Entries).To(HaveLen(2))
		g.Expect(cfg.Entries[0].Role).To(Equal(canvas.LegendRoleFill))
		g.Expect(cfg.Entries[1].Role).To(Equal(canvas.LegendRoleBorder))
	}
}

func TestBuildLegendConfig_DifferentSizeMetric_AddsEntry(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := makeLegendTestRoot()
	pal := palette.GetPalette(palette.Temperature)
	values := collectNumericValues(root, "file-size")
	fillInk := canvas.NumericInk("file-size", values, pal)

	cfg := buildLegendConfig(
		canvas.LegendPositionBottomRight, canvas.LegendOrientationVertical,
		fillInk, "file-size",
		canvas.FixedInk(color.RGBA{A: 255}), "",
		"file-lines",
	)

	if cfg == nil {
		t.Fatal("expected non-nil LegendConfig")
	} else {
		g.Expect(cfg.Entries).To(HaveLen(2))
		g.Expect(cfg.Entries[1].Role).To(Equal(canvas.LegendRoleSize))
		g.Expect(cfg.Entries[1].MetricName).To(Equal("file-lines"))
	}
}

func TestBuildLegendConfig_SameSizeAsFill_NoSizeEntry(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := makeLegendTestRoot()
	pal := palette.GetPalette(palette.Temperature)
	values := collectNumericValues(root, "file-size")
	fillInk := canvas.NumericInk("file-size", values, pal)

	cfg := buildLegendConfig(
		canvas.LegendPositionBottomRight, canvas.LegendOrientationVertical,
		fillInk, "file-size",
		canvas.FixedInk(color.RGBA{A: 255}), "",
		"file-size",
	)

	if cfg == nil {
		t.Fatal("expected non-nil LegendConfig")
	} else {
		g.Expect(cfg.Entries).To(HaveLen(1))
	}
}

func TestBuildLegendConfig_NoMetrics_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := buildLegendConfig(
		canvas.LegendPositionBottomRight, canvas.LegendOrientationVertical,
		canvas.FixedInk(color.RGBA{A: 255}), "",
		canvas.FixedInk(color.RGBA{A: 255}), "",
		"",
	)

	g.Expect(cfg).To(BeNil())
}

func makeLegendTestRoot() *model.Directory {
	f1 := &model.File{Name: "main.go", Extension: "go"}
	f1.SetQuantity(filesystem.FileSize, 500)
	f1.SetQuantity(filesystem.FileLines, 50)
	f1.SetClassification(filesystem.FileType, "go")

	f2 := &model.File{Name: "lib.rs", Extension: "rs"}
	f2.SetQuantity(filesystem.FileSize, 1000)
	f2.SetQuantity(filesystem.FileLines, 100)
	f2.SetClassification(filesystem.FileType, "rs")

	f3 := &model.File{Name: "app.py", Extension: "py"}
	f3.SetQuantity(filesystem.FileSize, 200)
	f3.SetQuantity(filesystem.FileLines, 20)
	f3.SetClassification(filesystem.FileType, "py")

	return &model.Directory{
		Name:  "root",
		Files: []*model.File{f1, f2, f3},
	}
}
```

- [ ] **Step 3: Wire treemap command**

In `cmd/codeviz/treemap_cmd.go`, update `renderAndLog()`. The key changes:
1. Move Ink creation before legend building
2. Build `canvas.LegendConfig` using `buildLegendConfig` with the Inks
3. Use `legendConfig.ReserveSpace()` instead of `render.ReserveLegendSpace()`
4. Call `cv.SetLegend()` before rendering
5. Remove the "Legend rendering not yet available" warning
6. Remove `render` import; update `reserveAndLayout` and `legendLayoutOffset` to use canvas types
7. Switch `render.FormatFromPath` → `canvas.FormatFromPath` in `validatePaths()`

The updated `renderAndLog` flow:

```go
func (c *TreemapCmd) renderAndLog(...) error {
	size := metric.Name(ptrString(cfg.Size))
	files, dirs := countAll(root)

	slog.Info("Rendering image", "output", c.Output, "width", width, "height", height)

	// Build inks first — legend uses the same Ink objects
	borderName, borderPaletteName := resolveBorderPaletteName(cfg)
	inks := buildTreemapInks(root, fillMetric, fillPaletteName, borderName, borderPaletteName)

	// Build legend config from the Inks
	legendPos, legendOrient := resolveLegendOptions(ptrString(cfg.Legend), ptrString(cfg.LegendOrientation))
	legendConfig := buildLegendConfig(
		legendPos, legendOrient,
		inks.fill, fillMetric,
		inks.border, borderName,
		size,
	)

	// Reserve space and layout
	layoutW, layoutH := width, height
	if legendConfig != nil {
		wReduce, hReduce := legendConfig.ReserveSpace()
		lw := width - int(wReduce)
		lh := height - int(hReduce)
		if lw >= minReservableSize && lh >= minReservableSize {
			layoutW, layoutH = lw, lh
		}
	}

	rects := treemap.Layout(root, layoutW, layoutH, size)

	if layoutW < width || layoutH < height {
		if legendConfig != nil {
			wReduce, hReduce := legendConfig.ReserveSpace()
			dx, dy := legendLayoutOffset(legendConfig, wReduce, hReduce)
			treemap.OffsetRects(&rects, dx, dy)
		}
	}

	cv := renderTreemapToCanvas(rects, root, width, height, inks)

	if legendConfig != nil {
		cv.SetLegend(*legendConfig)
	}

	slog.Debug("rendering", "width", width, "height", height, "output", c.Output)

	if err := cv.Render(c.Output); err != nil {
		return eris.Wrap(err, "render failed")
	}

	// ... slog.Info("Rendered treemap", ...) unchanged ...
}
```

Update `reserveAndLayout` to accept `*canvas.LegendConfig` (remove it — the logic is now inline in renderAndLog).

Update `legendLayoutOffset` and `cornerLegendOffset` to use `canvas.LegendPosition` / `canvas.LegendOrientation` instead of `render.LegendPosition` / `render.LegendOrientation`.

Switch `validatePaths()` from `render.FormatFromPath` to `canvas.FormatFromPath`.

Remove the `render` import from `treemap_cmd.go`.

- [ ] **Step 4: Wire spiral command**

In `cmd/codeviz/spiral_cmd.go`, in the `renderAndLog` method, after `cv := renderSpiralToCanvas(...)` and before `cv.Render(...)`:

```go
	legendPos, legendOrient := resolveLegendOptions(ptrString(cfg.Legend), ptrString(cfg.LegendOrientation))
	legendConfig := buildLegendConfig(
		legendPos, legendOrient,
		inks.fill, fillMetric,
		inks.border, borderMetric,
		metric.Name(ptrString(cfg.Size)),
	)

	if legendConfig != nil {
		cv.SetLegend(*legendConfig)
	}
```

Remove the `legendStr` warning block (lines 235-238).

- [ ] **Step 5: Wire radial tree command**

In `cmd/codeviz/radialtree_cmd.go`, in `renderAndLog`, after `cv := renderRadialToCanvas(...)`:

```go
	legendPos, legendOrient := resolveLegendOptions(ptrString(cfg.Legend), ptrString(cfg.LegendOrientation))
	legendConfig := buildLegendConfig(
		legendPos, legendOrient,
		inks.fill, fillMetric,
		inks.border, borderMetric,
		metric.Name(ptrString(cfg.DiscSize)),
	)

	if legendConfig != nil {
		cv.SetLegend(*legendConfig)
	}
```

Remove the `legendStr` warning block (lines 182-185).

- [ ] **Step 6: Wire bubbletree command**

In `cmd/codeviz/bubbletree_cmd.go`, in `renderAndLog`, after `cv := renderBubbleToCanvas(...)`:

```go
	legendPos, legendOrient := resolveLegendOptions(ptrString(cfg.Legend), ptrString(cfg.LegendOrientation))
	legendConfig := buildLegendConfig(
		legendPos, legendOrient,
		inks.fill, fillMetric,
		inks.border, borderMetric,
		metric.Name(ptrString(cfg.Size)),
	)

	if legendConfig != nil {
		cv.SetLegend(*legendConfig)
	}
```

Add the `canvas` import if not already present (bubbletree_cmd.go should already have it).

- [ ] **Step 7: Run tests**

Run: `task test`
Expected: ALL PASS

- [ ] **Step 8: Commit**

```bash
git add cmd/codeviz/legend_builder.go cmd/codeviz/legend_builder_test.go \
       cmd/codeviz/treemap_cmd.go cmd/codeviz/spiral_cmd.go \
       cmd/codeviz/radialtree_cmd.go cmd/codeviz/bubbletree_cmd.go
git commit -m "feat: wire legend rendering for all four visualization types

Rewrite legend_builder.go to use canvas types and Ink objects.
Wire SetLegend in treemap, spiral, radial tree, and bubbletree commands.
Remove 'legend omitted' warnings. Switch treemap to canvas.FormatFromPath."
```

---

### Task 8: Delete old render legend files and CI check

**Files:**
- Delete: `internal/render/legend.go`
- Delete: `internal/render/legend_png.go`
- Delete: `internal/render/legend_svg.go`
- Delete: `internal/render/legend_test.go`
- Delete: `internal/render/svg_helpers.go`
- Delete: `internal/render/save.go`

- [ ] **Step 1: Delete old render legend files**

```bash
rm internal/render/legend.go \
   internal/render/legend_png.go \
   internal/render/legend_svg.go \
   internal/render/legend_test.go \
   internal/render/svg_helpers.go \
   internal/render/save.go
```

- [ ] **Step 2: Run build to check for broken references**

Run: `task build`
Expected: BUILD SUCCESS — no remaining references to deleted code

If the build fails due to remaining references to `render.LegendPosition` etc., check `label.go` and `format.go` for any indirect dependencies and fix.

- [ ] **Step 3: Run full CI**

Run: `task ci`
Expected: BUILD + TEST + LINT all pass with 0 issues

Fix any lint issues (common: unused imports, max-public-structs, funlen violations).

- [ ] **Step 4: Commit**

```bash
git add -u internal/render/
git commit -m "chore: delete old render legend files

Legend rendering has been fully migrated to the canvas pipeline.
Deleted: legend.go, legend_png.go, legend_svg.go, legend_test.go,
svg_helpers.go, save.go from internal/render/."
```

- [ ] **Step 5: Final CI verification**

Run: `task ci`
Expected: 0 issues, all tests pass

---

## Dependency graph

```
Task 1 (types + interface)
  ├── Task 2 (measurement) ──────┐
  ├── Task 3 (ink swatches) ─────┤
  │                              ├── Task 4 (canvas legend methods)
  │                              │     └── Task 5 (RenderTo + raster backend)
  │                              │     └── Task 6 (SVG backend)
  │                              └── Task 7 (CLI integration) ← depends on Tasks 4-6
  └── Task 8 (cleanup) ← depends on Task 7
```

Tasks 2 and 3 can run in parallel after Task 1.
Tasks 5 and 6 can run in parallel after Task 4.
Task 7 requires Tasks 4-6 complete.
Task 8 is the final cleanup after everything is wired.
