# Inks Encapsulation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move the `Ink` interface, all concrete inks, `MetricValue`, and the legend rendering machinery out of `internal/canvas` so that `internal/inks` owns colour resolution and `internal/legend` owns legend authoring + rendering.

**Architecture:** Staged migration in six tasks. Each task is one commit. Tasks 2 and 4 introduce transient compatibility aliases in `canvas` so existing call sites keep compiling while internals migrate; Task 6 deletes them.

**Tech Stack:** Go 1.26, Gomega assertions, Goldie golden-file snapshots, golangci-lint v2.

**Spec:** [docs/superpowers/specs/2026-06-20-inks-encapsulation-design.md](../specs/2026-06-20-inks-encapsulation-design.md)

---

## File-Level Plan

### New files

| Path | Responsibility |
|---|---|
| `internal/inks/ink.go` | `Ink` interface, `baseInk`, `FixedInk`/`NumericInk`/`CategoricalInk`, opacity helpers |
| `internal/inks/options.go` | `Option` type, `WithOpacity` |
| `internal/inks/introspection.go` | `Info`, `Kind` + constants, `Boundaries()`/`Palette()`/`Categories()` accessors |
| `internal/inks/radial_gradient.go` | `RadialGradientInk` decorator, `NewRadialGradientInk`, `darken` |
| `internal/inks/metric_value.go` | `MetricValue`, `MeasureValue`, `QuantityValue`, `CategoryValue` |
| `internal/inks/legend_data.go` | `LegendData(Ink) (model.LegendEntryKind, []model.LegendSwatch)` |
| `internal/inks/ink_test.go` | Dip/Fill/option tests (moved from canvas) |
| `internal/inks/introspection_test.go` | Boundaries/Palette/Categories/Info tests (moved from canvas) |
| `internal/inks/radial_gradient_test.go` | RadialGradientInk tests (moved from canvas) |
| `internal/inks/metric_value_test.go` | MetricValue constructor tests (moved from canvas) |
| `internal/inks/legend_data_test.go` | LegendData function tests (replaces ink_legend_test.go) |
| `internal/legend/config.go` | `Config`/`Entry`/`Role`/`RoleFill`/`RoleBorder`/`RoleSize`, `DefaultOrientation`, `toLegendData`, `ReserveSpace` method |
| `internal/legend/render.go` | `RenderInto`, `legendBuilder`, `legendOrigin`, all swatch/text helpers |
| `internal/legend/render_test.go` | Tests for `RenderInto` (replaces canvas's TestCanvas_SetLegend_* tests) |
| `internal/canvas/aliases.go` | **Transient** — type/function aliases re-exporting `inks` & `legend` names. Deleted in Task 6. |

### Deleted files (across Tasks 2–4)

`internal/canvas/ink.go`, `ink_options.go`, `ink_introspection.go`, `radial_gradient_ink.go`, `metric_value.go`, `ink_test.go`, `ink_introspection_test.go`, `radial_gradient_ink_test.go`, `metric_value_test.go`, `ink_legend_test.go`, `legend.go`, `legend_render.go`.

### Modified files

| Path | Modification |
|---|---|
| `internal/canvas/canvas.go` | Add `DrawingMinY()`/`DrawingMaxY()` getters; remove `c.legend` field, `SetLegend`, `decomposeLegend` call in `RenderTo` (final state) |
| `internal/canvas/spec.go`, `shape.go`, `text_spec.go` | Internal references switch from `Ink`/`MetricValue` to `inks.Ink`/`inks.MetricValue` (Task 3) |
| `internal/canvas/canvas_test.go` | `fillAwareInk` mock implements new `inks.Ink` interface (no legend methods); `TestCanvas_SetLegend_*` tests removed (moved to legend package) |
| `internal/legend/legend.go` | Return `*legend.Config` instead of `*canvas.LegendConfig`; rename `canvas.LegendRole*` → `legend.Role*`; rename `canvas.LegendEntry` → `legend.Entry`; replace `canvas.FixedInk(white)` with `inks.FixedInk(white)` |
| `internal/legend/legend_test.go` | Adjust to `legend.Config`/`legend.Entry`/`legend.Role*` and `inks.*` types |
| `internal/legend/reserve.go` | Accept `*legend.Config` instead of `*canvas.LegendConfig` |
| `internal/legend/reserve_test.go` | Use `legend.Config` |
| `internal/treemap/inks.go`, `render.go`, `stages.go`, `labels.go`, `*_test.go` | Switch `canvas.*Ink`/`canvas.MetricValue`/`canvas.InkKind`/`canvas.RadialGradientInk` to `inks.*`; switch `cv.SetLegend(*cfg)` to `legend.RenderInto(cv, cfg)` |
| `internal/bubbletree/inks.go`, `render.go`, `stages.go`, `*_test.go` | Same kind of migration |
| `internal/spiral/inks.go`, `render.go`, `*_test.go` | Same |
| `internal/scatter/inks.go`, `render.go`, `*_test.go` | Same |
| `internal/radialtree/inks.go`, `render.go`, `*_test.go` | Same |
| `cmd/codeviz/*.go` | Same migration where applicable |

---

## Verification Commands

Throughout the plan, these commands are referenced by name:

- **Build:** `task build` — ensures the whole module compiles.
- **Test:** `task test` — runs `go test ./...`. All golden snapshots must match.
- **Lint:** `task lint` — runs golangci-lint with verbose output. Per repo convention, dispatch this via an `Explore` subagent that reports only exit status, failing linters, and offending file:line entries.
- **CI:** `task ci` — build + test + lint. Use the same Explore-dispatch convention for noisy output.
- **Focused tests:** `go test ./internal/<pkg>/... -count=1` for fast feedback on a single package.

The success gate for each task is **`task ci` passes**. If a task is in mid-state (e.g. aliases still present), CI still must pass.

---

## Task 1: Expose drawing-bound getters on Canvas

**Files:**
- Modify: `internal/canvas/canvas.go`
- Modify: `internal/canvas/canvas_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/canvas/canvas_test.go`:

```go
func TestCanvas_DrawingBounds_Getters_ReturnZerosByDefault(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	g.Expect(c.DrawingMinY()).To(Equal(0))
	g.Expect(c.DrawingMaxY()).To(Equal(600))
}

func TestCanvas_DrawingBounds_Getters_ReturnSetValues(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	c.SetDrawingBounds(40, 560)
	g.Expect(c.DrawingMinY()).To(Equal(40))
	g.Expect(c.DrawingMaxY()).To(Equal(560))
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/canvas/ -run TestCanvas_DrawingBounds_Getters -count=1`
Expected: build failure — `DrawingMinY` undefined.

- [ ] **Step 3: Add the getters**

In `internal/canvas/canvas.go`, immediately after `SetDrawingBounds`:

```go
// DrawingMinY returns the topmost Y pixel available for non-title content.
func (c *Canvas) DrawingMinY() int { return c.drawingMinY }

// DrawingMaxY returns the bottommost Y pixel (exclusive) available for
// non-footer content.
func (c *Canvas) DrawingMaxY() int { return c.drawingMaxY }
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/canvas/ -count=1`
Expected: PASS.

- [ ] **Step 5: Run CI**

Dispatch an `Explore` subagent: "Run `task ci`. Report exit status, count and identity of failing linters/tests, and offending file:line entries. One-line note if everything passes."
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/canvas/canvas.go internal/canvas/canvas_test.go
git commit -m "feat(canvas): expose DrawingMinY/DrawingMaxY getters

Prepares for moving legend rendering out of canvas; the new
legend.RenderInto entry point will need read access to the drawing
bounds reserved by title and footer."
```

---

## Task 2: Build the `inks` package and re-export from `canvas` via aliases

This task **moves** all ink-related types out of `canvas` and into `inks`, then adds aliases in `canvas` so external callers (visualization packages, `internal/legend`, etc.) continue to compile.

**Files:**
- Create: `internal/inks/ink.go`, `options.go`, `introspection.go`, `radial_gradient.go`, `metric_value.go`, `legend_data.go`
- Create: `internal/inks/ink_test.go`, `introspection_test.go`, `radial_gradient_test.go`, `metric_value_test.go`, `legend_data_test.go`
- Create: `internal/canvas/aliases.go` (transient)
- Delete: `internal/canvas/ink.go`, `ink_options.go`, `ink_introspection.go`, `radial_gradient_ink.go`, `metric_value.go`, `ink_test.go`, `ink_introspection_test.go`, `radial_gradient_ink_test.go`, `metric_value_test.go`, `ink_legend_test.go`
- Modify: `internal/canvas/canvas_test.go` (drop `legendEntryKind`/`legendSwatches` methods on `fillAwareInk`)
- Modify: `internal/inks/inks.go` (existing helpers stop using `canvas.` prefix; rename `canvas.InkKind`/`canvas.InkNumeric`/`canvas.InkCategorical` references to local `Kind`/`KindNumeric`/`KindCategorical`)

The existing `internal/inks/inks_test.go` continues to work because the aliases in `canvas/aliases.go` keep `canvas.InkNumeric`/`canvas.MetricValue` etc. valid (the file uses them).

- [ ] **Step 1: Create `internal/inks/options.go`**

```go
package inks

// Option configures ink behaviour.
type Option func(*config)

type config struct {
	opacity float64
}

func defaultConfig() config {
	return config{
		opacity: 1.0,
	}
}

// WithOpacity sets the opacity applied when Dip() resolves a colour.
// Default is 1.0 (fully opaque). The opacity is applied to the alpha channel
// of the resolved colour.
func WithOpacity(opacity float64) Option {
	return func(c *config) {
		c.opacity = opacity
	}
}
```

- [ ] **Step 2: Create `internal/inks/metric_value.go`**

```go
package inks

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

// MetricValue carries the metric data needed to resolve a colour.
// The Kind field determines which of the remaining fields is used.
type MetricValue struct {
	Kind     metric.Kind
	Measure  float64
	Quantity int
	Category string
}

// MeasureValue creates a MetricValue for a float64 measure.
func MeasureValue(v float64) MetricValue {
	return MetricValue{Kind: metric.Measure, Measure: v}
}

// QuantityValue creates a MetricValue for an integer quantity.
func QuantityValue(v int) MetricValue {
	return MetricValue{Kind: metric.Quantity, Quantity: v}
}

// CategoryValue creates a MetricValue for a string classification.
func CategoryValue(v string) MetricValue {
	return MetricValue{Kind: metric.Classification, Category: v}
}
```

- [ ] **Step 3: Create `internal/inks/ink.go`**

The body is the existing `internal/canvas/ink.go` with these substitutions:

- Package: `canvas` → `inks`
- Type `inkKind` → `Kind`, constants `inkFixed`/`inkNumeric`/`inkCategorical` → `KindFixed`/`KindNumeric`/`KindCategorical` (note: also exposed; same identifiers as introspection.go expects)
- `InkOption` → `Option`, `inkConfig` → `config`, `defaultInkConfig()` → `defaultConfig()` (options live in `options.go`; remove the duplicate from `ink.go`)
- The `Ink` interface gains `Boundaries()`/`Palette()`/`Categories()` methods (currently public on `*baseInk` but not part of the interface)
- The unexported `legendEntryKind`/`legendSwatches` methods are **removed** from the interface and from `*baseInk` — their logic moves to `internal/inks/legend_data.go` in Step 6.
- `InkInfo` → `Info`

Full file:

```go
// Package inks owns the Ink interface and concrete ink implementations.
// Inks resolve metric values to colours and fill specifications; they are
// consumed by canvas shape specs (canvas.RectangleSpec, canvas.TextSpec, ...).
package inks

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// Kind identifies the type of an Ink for introspection purposes.
type Kind int

const (
	// KindFixed is a fixed-colour ink that ignores its input.
	KindFixed Kind = iota
	// KindNumeric is a numeric-bucket ink mapping float values to palette colours.
	KindNumeric
	// KindCategorical is a categorical ink mapping strings to palette colours.
	KindCategorical
)

// Ink resolves metric values to colours and fill specifications.
type Ink interface {
	Dip(value MetricValue) color.RGBA
	Fill(value MetricValue, focus model.Point) model.Fill
	Info() Info

	// Introspection accessors used by legend extraction and tests.
	// FixedInk values return nil/empty for all three.
	Boundaries() []float64
	Palette() palette.ColourPalette
	Categories() []string
}

type baseInk struct {
	kind       Kind
	metricName metric.Name
	color      color.RGBA
	boundaries *metric.BucketBoundaries
	catMapper  *palette.CategoricalMapper
	pal        palette.ColourPalette
	categories []string
	opacity    float64
}

// FixedInk always produces the same colour regardless of input.
func FixedInk(c color.RGBA, opts ...Option) Ink {
	cfg := defaultConfig()
	for _, o := range opts {
		o(&cfg)
	}

	return &baseInk{
		kind:    KindFixed,
		color:   c,
		opacity: cfg.opacity,
	}
}

// NumericInk maps numeric metric values to palette colours.
// Takes the full dataset of values (for bucketing), the palette,
// and optional configuration options.
func NumericInk(name metric.Name, values []float64, pal palette.ColourPalette, opts ...Option) Ink {
	cfg := defaultConfig()
	for _, o := range opts {
		o(&cfg)
	}

	buckets := metric.ComputeBuckets(values, len(pal.Colours))

	return &baseInk{
		kind:       KindNumeric,
		metricName: name,
		boundaries: &buckets,
		pal:        pal,
		opacity:    cfg.opacity,
	}
}

// CategoricalInk maps string categories to palette colours.
func CategoricalInk(name metric.Name, categories []string, pal palette.ColourPalette, opts ...Option) Ink {
	cfg := defaultConfig()
	for _, o := range opts {
		o(&cfg)
	}

	return &baseInk{
		kind:       KindCategorical,
		metricName: name,
		catMapper:  palette.NewCategoricalMapper(categories, pal),
		pal:        pal,
		categories: categories,
		opacity:    cfg.opacity,
	}
}

// Dip resolves a MetricValue to an RGBA colour.
func (ink *baseInk) Dip(value MetricValue) color.RGBA {
	var c color.RGBA

	switch ink.kind {
	case KindFixed:
		c = ink.color
	case KindNumeric:
		c = ink.dipNumeric(value)
	case KindCategorical:
		c = ink.catMapper.Map(value.Category)
	default:
		c = color.RGBA{A: 255}
	}

	return applyOpacity(c, ink.opacity)
}

func (ink *baseInk) Fill(value MetricValue, _ model.Point) model.Fill {
	return model.SolidFill{Color: ink.Dip(value)}
}

func (ink *baseInk) dipNumeric(value MetricValue) color.RGBA {
	var numericVal float64

	switch value.Kind {
	case metric.Quantity:
		numericVal = float64(value.Quantity)
	default:
		numericVal = value.Measure
	}

	idx := ink.boundaries.BucketIndex(numericVal)

	return palette.MapNumericToColour(idx, ink.boundaries.NumBuckets(), ink.pal)
}

func applyOpacity(c color.RGBA, opacity float64) color.RGBA {
	if opacity >= 1.0 {
		return c
	}

	c.A = uint8(float64(c.A) * clamp01(opacity))

	return c
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}

	if v > 1 {
		return 1
	}

	return v
}
```

- [ ] **Step 4: Create `internal/inks/introspection.go`**

```go
package inks

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// Info carries introspection data about an Ink.
type Info struct {
	Kind       Kind
	MetricName metric.Name
}

// Info returns introspection data about the ink's kind and metric.
func (ink *baseInk) Info() Info {
	return Info{
		Kind:       ink.kind,
		MetricName: ink.metricName,
	}
}

// Colours used by internal defaults.
var (
	white = color.RGBA{R: 255, G: 255, B: 255, A: 255} //nolint:gochecknoglobals
	black = color.RGBA{R: 0, G: 0, B: 0, A: 255}       //nolint:gochecknoglobals
)

// Boundaries returns the bucket boundary values for numeric inks.
// Returns nil for fixed or categorical inks.
func (ink *baseInk) Boundaries() []float64 {
	if ink.kind != KindNumeric || ink.boundaries == nil {
		return nil
	}

	return ink.boundaries.Boundaries
}

// Palette returns the colour palette used by this ink.
// Returns an empty palette for fixed inks.
func (ink *baseInk) Palette() palette.ColourPalette {
	if ink.kind == KindFixed {
		return palette.ColourPalette{}
	}

	return ink.pal
}

// Categories returns the category labels for categorical inks.
// Returns nil for fixed or numeric inks.
func (ink *baseInk) Categories() []string {
	if ink.kind != KindCategorical {
		return nil
	}

	return ink.categories
}
```

- [ ] **Step 5: Create `internal/inks/radial_gradient.go`**

```go
package inks

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

const defaultDarken = 0.4

// RadialGradientInk wraps another Ink to produce radial gradient fills.
// The inner ink provides the centre colour; edges are darkened by the
// configured fraction.
type RadialGradientInk struct {
	inner  Ink
	darken float64
}

// NewRadialGradientInk creates a RadialGradientInk that darkens edges by 40%.
func NewRadialGradientInk(inner Ink) Ink {
	return &RadialGradientInk{inner: inner, darken: defaultDarken}
}

func (g *RadialGradientInk) Dip(value MetricValue) color.RGBA {
	return g.inner.Dip(value)
}

func (g *RadialGradientInk) Fill(value MetricValue, focus model.Point) model.Fill {
	base := g.inner.Dip(value)

	return model.RadialGradientFill{
		Center: base,
		Edge:   darken(base, g.darken),
		Focus:  focus,
	}
}

func (g *RadialGradientInk) Info() Info {
	return g.inner.Info()
}

func (g *RadialGradientInk) Boundaries() []float64 {
	return g.inner.Boundaries()
}

func (g *RadialGradientInk) Palette() palette.ColourPalette {
	return g.inner.Palette()
}

func (g *RadialGradientInk) Categories() []string {
	return g.inner.Categories()
}

// darken reduces each RGB channel by the given fraction (0.4 = 40% darker).
func darken(c color.RGBA, fraction float64) color.RGBA {
	scale := 1.0 - fraction

	return color.RGBA{
		R: uint8(float64(c.R) * scale),
		G: uint8(float64(c.G) * scale),
		B: uint8(float64(c.B) * scale),
		A: c.A,
	}
}
```

- [ ] **Step 6: Create `internal/inks/legend_data.go`**

```go
package inks

import (
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/legendlayout"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// LegendData extracts the legend entry kind and swatch list from an Ink.
// Returns LegendEntryNumeric / nil for fixed inks (no meaningful swatch data).
func LegendData(ink Ink) (model.LegendEntryKind, []model.LegendSwatch) {
	switch ink.Info().Kind {
	case KindNumeric:
		return model.LegendEntryNumeric, numericSwatches(ink)
	case KindCategorical:
		return model.LegendEntryCategorical, categoricalSwatches(ink)
	default:
		return model.LegendEntryNumeric, nil
	}
}

func numericSwatches(ink Ink) []model.LegendSwatch {
	boundaries := ink.Boundaries()
	pal := ink.Palette()

	if len(boundaries) == 0 || len(pal.Colours) == 0 {
		return nil
	}

	n := len(boundaries)
	if n <= 0 {
		return nil
	}

	swatches := make([]model.LegendSwatch, n)

	for i := range n {
		colour := palette.MapNumericToColour(i, n, pal)

		var label string
		if i < len(boundaries) {
			label = legendlayout.FormatBreakpoint(boundaries[i])
		}

		swatches[i] = model.LegendSwatch{
			Colour: colour,
			Label:  label,
		}
	}

	return swatches
}

func categoricalSwatches(ink Ink) []model.LegendSwatch {
	categories := ink.Categories()
	pal := ink.Palette()

	if len(categories) == 0 || len(pal.Colours) == 0 {
		return nil
	}

	sorted := make([]string, len(categories))
	copy(sorted, categories)
	slices.Sort(sorted)

	mapper := palette.NewCategoricalMapper(sorted, pal)
	swatches := make([]model.LegendSwatch, len(sorted))

	for i, cat := range sorted {
		swatches[i] = model.LegendSwatch{
			Colour: mapper.Map(cat),
			Label:  cat,
		}
	}

	return swatches
}
```

> **Why a fresh `CategoricalMapper` here?** The existing `categoricalLegendSwatches` on `*baseInk` calls `ink.catMapper.Map(cat)` directly. Inside `inks`, callers of `LegendData` only have the `Ink` interface plus its introspection accessors — they can't reach `baseInk.catMapper`. Rebuilding the mapper from `Categories()` + `Palette()` reproduces the same mapping (the constructor is deterministic). If snapshot tests show any drift after Task 2, the alternative is to add a `categoricalColour(category) color.RGBA` accessor to `Ink` and forward it from `RadialGradientInk`.

- [ ] **Step 7: Move the ink test files**

For each pair below, copy the file content from canvas, change package to `inks`, and rename references (`InkKind` → `Kind`, `InkFixed`/`InkNumeric`/`InkCategorical` → `KindFixed`/`KindNumeric`/`KindCategorical`, `InkInfo` → `Info`, `InkOption` → `Option`). Delete the original.

| Move | Adjustments |
|---|---|
| `internal/canvas/ink_test.go` → `internal/inks/ink_test.go` | Package change only |
| `internal/canvas/ink_introspection_test.go` → `internal/inks/introspection_test.go` | Package change; `InkFixed`/`InkNumeric`/`InkCategorical` → `KindFixed`/etc.; `InkKind`/`InkInfo` → `Kind`/`Info` |
| `internal/canvas/radial_gradient_ink_test.go` → `internal/inks/radial_gradient_test.go` | Package change; the `LegendMethods_DelegateToInner` sub-test calling `legendEntryKind`/`legendSwatches` is **rewritten** to call `LegendData(gradient)` and compare to `LegendData(inner)` |
| `internal/canvas/metric_value_test.go` → `internal/inks/metric_value_test.go` | Package change only |
| `internal/canvas/ink_legend_test.go` → `internal/inks/legend_data_test.go` | Package change; replace `ink.legendEntryKind()` / `ink.legendSwatches()` with `kind, swatches := LegendData(ink)`; assertions stay the same in spirit |

The rewritten `RadialGradientInk_LegendMethods_DelegateToInner` test:

```go
func TestRadialGradientInk_LegendData_DelegatesToInner(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	inner := CategoricalInk(
		"language",
		[]string{"go", "rs"},
		palette.GetPalette(palette.Categorization),
	)
	gradient := NewRadialGradientInk(inner)

	innerKind, innerSwatches := LegendData(inner)
	gradientKind, gradientSwatches := LegendData(gradient)

	g.Expect(gradientKind).To(Equal(innerKind))
	g.Expect(gradientSwatches).To(Equal(innerSwatches))
}
```

The rewritten `legend_data_test.go` (replacing `ink_legend_test.go`):

```go
package inks

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

func TestLegendData_FixedInk_ReturnsNilSwatches(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := FixedInk(white)
	kind, swatches := LegendData(ink)
	g.Expect(kind).To(Equal(model.LegendEntryNumeric))
	g.Expect(swatches).To(BeNil())
}

func TestLegendData_NumericInk_ReturnsBucketColours(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Temperature)
	ink := NumericInk("file-size", []float64{10, 50, 100, 500, 1000}, pal)

	kind, swatches := LegendData(ink)
	g.Expect(kind).To(Equal(model.LegendEntryNumeric))
	g.Expect(swatches).NotTo(BeNil())
	g.Expect(len(swatches)).To(BeNumerically(">", 0))

	for _, sw := range swatches {
		g.Expect(sw.Colour.A).To(Equal(uint8(255)))
	}

	last := swatches[len(swatches)-1]
	g.Expect(last.Label).To(BeEmpty())
}

func TestLegendData_CategoricalInk_ReturnsCategoryLabels(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Categorization)
	ink := CategoricalInk("file-type", []string{"go", "py", "rs"}, pal)

	kind, swatches := LegendData(ink)
	g.Expect(kind).To(Equal(model.LegendEntryCategorical))
	g.Expect(swatches).To(HaveLen(3))

	g.Expect(swatches[0].Label).To(Equal("go"))
	g.Expect(swatches[1].Label).To(Equal("py"))
	g.Expect(swatches[2].Label).To(Equal("rs"))
}
```

- [ ] **Step 8: Update `internal/inks/inks.go`**

This file already lives in `inks` but imports `canvas`. After the move it no longer needs that import — all the types it referenced now live locally.

Open `internal/inks/inks.go` and apply these replacements throughout:

| From | To |
|---|---|
| `"github.com/theunrepentantgeek/code-visualizer/internal/canvas"` import | Remove |
| `canvas.Ink` | `Ink` |
| `canvas.FixedInk` | `FixedInk` |
| `canvas.NumericInk` | `NumericInk` |
| `canvas.CategoricalInk` | `CategoricalInk` |
| `canvas.MetricValue` | `MetricValue` |
| `canvas.InkNumeric` | `KindNumeric` |
| `canvas.InkCategorical` | `KindCategorical` |

The function `MetricValueForFile`'s switch becomes:

```go
switch info.Kind {
case KindNumeric:
    // ...
case KindCategorical:
    // ...
}
```

- [ ] **Step 9: Update `internal/inks/inks_test.go`**

This file uses `package inks_test` (external test package). It currently imports `canvas` for `canvas.MetricValue`, `canvas.InkNumeric`, etc. Switch those to `inks.MetricValue`, `inks.KindNumeric`, etc. The `inks` import (`"github.com/theunrepentantgeek/code-visualizer/internal/inks"`) is already there. Adjust calls:

| From | To |
|---|---|
| `canvas.InkNumeric` | `inks.KindNumeric` |
| `canvas.InkCategorical` | `inks.KindCategorical` |
| `canvas.InkFixed` | `inks.KindFixed` |
| `canvas.FixedInk` | `inks.FixedInk` |
| `canvas.MetricValue` | `inks.MetricValue` |

Remove the `canvas` import if it becomes unused. (Inspect carefully — some assertions may also reference `canvas.NumericInk` etc.)

- [ ] **Step 10: Create `internal/canvas/aliases.go` (transient)**

```go
package canvas

// This file is a transient compatibility shim introduced when the ink
// machinery moved to the inks package. It will be deleted in Task 6 of the
// inks-encapsulation plan, after every call site has been updated.

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
)

// Type aliases re-export ink types from inks for backward compatibility.
type (
	Ink         = inks.Ink //nolint:revive // transient alias
	InkInfo     = inks.Info
	InkKind     = inks.Kind
	InkOption   = inks.Option
	MetricValue = inks.MetricValue

	RadialGradientInk = inks.RadialGradientInk //nolint:revive // transient alias
)

// Kind constants.
const (
	InkFixed       = inks.KindFixed
	InkNumeric     = inks.KindNumeric
	InkCategorical = inks.KindCategorical
)

// Function re-exports.
var (
	FixedInk             = inks.FixedInk             //nolint:gochecknoglobals // transient alias
	NumericInk           = inks.NumericInk           //nolint:gochecknoglobals
	CategoricalInk       = inks.CategoricalInk       //nolint:gochecknoglobals
	NewRadialGradientInk = inks.NewRadialGradientInk //nolint:gochecknoglobals
	WithOpacity          = inks.WithOpacity          //nolint:gochecknoglobals
	MeasureValue         = inks.MeasureValue         //nolint:gochecknoglobals
	QuantityValue        = inks.QuantityValue        //nolint:gochecknoglobals
	CategoryValue        = inks.CategoryValue        //nolint:gochecknoglobals
)
```

> **Lint suppressions:** the aliases trip `revive` (stuttering names) and `gochecknoglobals` deliberately, because they exist precisely to preserve the old names. They are scoped to `aliases.go` and will disappear with the file in Task 6.

- [ ] **Step 11: Delete the old canvas ink files**

```bash
rm internal/canvas/ink.go
rm internal/canvas/ink_options.go
rm internal/canvas/ink_introspection.go
rm internal/canvas/radial_gradient_ink.go
rm internal/canvas/metric_value.go
rm internal/canvas/ink_test.go
rm internal/canvas/ink_introspection_test.go
rm internal/canvas/radial_gradient_ink_test.go
rm internal/canvas/metric_value_test.go
rm internal/canvas/ink_legend_test.go
```

- [ ] **Step 12: Fix the `fillAwareInk` mock and canvas's internal use of unexported legend methods**

In `internal/canvas/canvas_test.go`, the `fillAwareInk` type currently implements two unexported methods (`legendEntryKind`, `legendSwatches`) and returns `InkInfo`. After the move, `Ink` no longer requires those methods — it requires `Boundaries`, `Palette`, `Categories` instead. Replace the mock methods:

```go
func (*fillAwareInk) legendEntryKind() model.LegendEntryKind { ... }   // DELETE
func (*fillAwareInk) legendSwatches() []model.LegendSwatch { ... }      // DELETE

// Add:
func (*fillAwareInk) Boundaries() []float64           { return nil }
func (*fillAwareInk) Palette() palette.ColourPalette  { return palette.ColourPalette{} }
func (*fillAwareInk) Categories() []string            { return nil }
```

The existing import of `palette` may need adding to `canvas_test.go`. Verify with `goimports`.

`canvas/legend.go` currently calls `e.Ink.legendEntryKind()` and `e.Ink.legendSwatches()`. Switch both to:

```go
kind, swatches := inks.LegendData(e.Ink)
// ...
Kind:     kind,
Swatches: swatches,
```

Add the `inks` import to `legend.go`. (This file still lives in `canvas` for the duration of Task 2; it moves to `internal/legend` in Task 4.)

- [ ] **Step 13: Build & test**

Run: `go build ./...`
Expected: clean.

Run: `go test ./internal/inks/... ./internal/canvas/... -count=1`
Expected: PASS for both packages.

- [ ] **Step 14: Full CI gate**

Dispatch an `Explore` subagent: "Run `task ci`. Report exit status, count and identity of failing linters/tests, and offending file:line entries. One-line note if everything passes."
Expected: PASS. Golden snapshots unchanged.

- [ ] **Step 15: Commit**

```bash
git add internal/inks internal/canvas
git rm -- (the deleted files are already gone; git add handles the rest)
git status   # verify the diff
git commit -m "refactor(inks): move Ink types out of canvas

Move Ink interface, baseInk + factories, RadialGradientInk, MetricValue,
options, introspection, and legend-swatch extraction into the inks
package. canvas/aliases.go re-exports the old names so existing callers
keep compiling. Aliases will be removed in a later commit once call
sites migrate."
```

---

## Task 3: Migrate canvas internals to use `inks` directly

This task removes the alias indirection inside `canvas` itself. External callers still use the aliases; internal canvas code switches to `inks.…` names. The result is a smaller surface area to deal with in Task 6.

**Files:**
- Modify: `internal/canvas/spec.go`, `shape.go`, `text_spec.go`, `canvas.go`, `canvas_test.go`, `legend.go`, `legend_render.go`

- [ ] **Step 1: Update `internal/canvas/spec.go`**

```go
package canvas

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
)

// ShapeStyle bundles the visual properties shared by all closed-shape specs.
type ShapeStyle struct {
	Fill        inks.Ink
	Border      inks.Ink
	BorderWidth float64
}

type RectangleSpec struct {
	ShapeStyle
}

type DiscSpec struct {
	ShapeStyle
}

type LineSpec struct {
	Stroke      inks.Ink
	StrokeWidth float64
}
```

- [ ] **Step 2: Update `internal/canvas/text_spec.go`**

Replace `Ink Ink` field types with `Ink inks.Ink` in both `TextSpec` and `ArcTextSpec` (or whatever the existing field names are). Replace `MetricValue{}` with `inks.MetricValue{}` and `Spec.Ink.Dip(MetricValue{})` accordingly. Add `inks` import.

- [ ] **Step 3: Update `internal/canvas/shape.go`**

Replace `MetricValue{}` literals with `inks.MetricValue{}`. The `Spec.Fill.Fill(...)` and `.Dip(...)` calls don't change. Add `inks` import.

- [ ] **Step 4: Update `internal/canvas/legend.go` and `legend_render.go`**

Within these files, switch every internal use of `Ink`, `MetricValue`, `FixedInk`, etc. to the `inks.…` form. The exported types `LegendConfig`/`LegendEntry`/`LegendRole` stay defined in `canvas` for now — they move out in Task 4. `inks.LegendData(e.Ink)` was already wired in Task 2 Step 12.

- [ ] **Step 5: Update `internal/canvas/canvas_test.go`**

Inside `package canvas` tests, references to `FixedInk`, `NumericInk`, `MetricValue`, `MeasureValue`, etc. still resolve via the aliases — but per the goal of this task, switch them to `inks.…` form. Add an `inks` import (`. "github.com/theunrepentantgeek/code-visualizer/internal/inks"`-style dot import is **not** allowed by the repo conventions — use the regular form).

Also update the `fillAwareInk` mock's return type: `Info() inks.Info` instead of `Info() InkInfo`.

- [ ] **Step 6: Build & test**

Run: `go build ./...` then `go test ./internal/canvas/... ./internal/inks/... -count=1`.
Expected: PASS.

- [ ] **Step 7: Full CI gate**

Dispatch an `Explore` subagent: "Run `task ci`. Report exit status, failing linters/tests, and offending file:line entries."
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/canvas
git commit -m "refactor(canvas): use inks.* names directly internally

Removes alias indirection inside canvas; canvas/aliases.go still
re-exports for external callers (visualization packages,
internal/legend, cmd/codeviz) until their migration is complete."
```

---

## Task 4: Move legend types and rendering into `internal/legend`

**Files:**
- Create: `internal/legend/config.go`, `internal/legend/render.go`, `internal/legend/render_test.go`
- Delete: `internal/canvas/legend.go`, `internal/canvas/legend_render.go`
- Modify: `internal/canvas/canvas.go` (add transient `pendingLegend` field + thin `SetLegend` delegating to `legend.RenderInto`; remove `decomposeLegend` call from `RenderTo`)
- Modify: `internal/canvas/aliases.go` (add transient legend aliases)
- Modify: `internal/canvas/canvas_test.go` (remove the two `TestCanvas_SetLegend_*` tests; they move)
- Modify: `internal/legend/legend.go` (return `*Config` instead of `*canvas.LegendConfig`)
- Modify: `internal/legend/legend_test.go`, `internal/legend/reserve.go`, `internal/legend/reserve_test.go`

- [ ] **Step 1: Create `internal/legend/config.go`**

Take the existing `internal/canvas/legend.go` and apply these substitutions:

- Package: `canvas` → `legend`
- `LegendRole` → `Role`; `LegendRoleFill` → `RoleFill`; `LegendRoleBorder` → `RoleBorder`; `LegendRoleSize` → `RoleSize`
- `LegendConfig` → `Config`
- `LegendEntry` → `Entry`
- `LegendEntry.Ink` field is now typed `inks.Ink` (was `canvas.Ink`/`Ink`)
- `e.Ink.legendEntryKind()` / `e.Ink.legendSwatches()` (already replaced with `inks.LegendData` in Task 2) — keep the `inks.LegendData(e.Ink)` call

The resulting file:

```go
package legend

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/legendlayout"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
)

// Role identifies what visual property a legend entry describes.
type Role string

const (
	RoleFill   Role = "Fill"
	RoleBorder Role = "Border"
	RoleSize   Role = "Size"
)

// Entry describes one metric shown in the legend.
type Entry struct {
	Role       Role
	MetricName string
	Ink        inks.Ink
}

// Config holds everything needed to render a legend.
type Config struct {
	Position    model.LegendPosition
	Orientation model.LegendOrientation
	LabelSample []string
	Entries     []Entry
}

// DefaultOrientation returns the default orientation for a given position.
// Top-center and bottom-center default to horizontal; all others to vertical.
func DefaultOrientation(pos model.LegendPosition) model.LegendOrientation {
	switch pos {
	case model.LegendPositionTopCenter, model.LegendPositionBottomCenter:
		return model.LegendOrientationHorizontal
	default:
		return model.LegendOrientationVertical
	}
}

// ReserveSpace computes the width and height reductions needed to reserve
// space for the legend within the canvas. Returns zeros if the legend is
// disabled or has no entries.
func (cfg *Config) ReserveSpace() (widthReduction, heightReduction float64) {
	data := cfg.toLegendData()

	return legendlayout.ReserveSpace(data, legendlayout.NewBasicMeasurer())
}

func (cfg *Config) toLegendData() *model.LegendData {
	if cfg == nil || cfg.Position == model.LegendPositionNone || len(cfg.Entries) == 0 {
		return nil
	}

	entries := make([]model.LegendEntryData, len(cfg.Entries))

	for i, e := range cfg.Entries {
		kind, swatches := inks.LegendData(e.Ink)
		entries[i] = model.LegendEntryData{
			Label:    string(e.Role),
			Metric:   e.MetricName,
			Kind:     kind,
			Swatches: swatches,
			IsBorder: e.Role == RoleBorder,
		}
	}

	orient := cfg.Orientation
	if orient == "" {
		orient = DefaultOrientation(cfg.Position)
	}

	return &model.LegendData{
		Position:    cfg.Position,
		Orientation: orient,
		LabelSample: labelSampleData(cfg.LabelSample),
		Entries:     entries,
	}
}

func labelSampleData(lines []string) *model.LegendLabelSample {
	if len(lines) == 0 {
		return nil
	}

	sample := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}

		sample = append(sample, line)
	}

	if len(sample) == 0 {
		return nil
	}

	return &model.LegendLabelSample{Lines: sample}
}
```

- [ ] **Step 2: Create `internal/legend/render.go`**

Take the existing `internal/canvas/legend_render.go` and apply these substitutions:

- Package: `canvas` → `legend`
- Receiver methods on `*Canvas` (`c.decomposeLegend`, `c.legendOrigin`) become package functions taking `*canvas.Canvas` and any needed dimensions/bounds
- Type references: `RectangleSpec` → `canvas.RectangleSpec`, `TextSpec` → `canvas.TextSpec`, `Rectangle` → `canvas.Rectangle`, `Text` → `canvas.Text`, `LayerOverlay` → `canvas.LayerOverlay`, `AnchorMiddle`/`AnchorStart` → `canvas.AnchorMiddle`/`canvas.AnchorStart`
- `FixedInk` → `inks.FixedInk`
- Replace `lb.shapes = append(...)` with direct `cv.AddRectangle(canvas.LayerOverlay, ...)` / `cv.AddText(canvas.LayerOverlay, ...)`
- Use `cv.DrawingMinY()` / `cv.DrawingMaxY()` (added in Task 1) where the previous code used `c.drawingMinY`/`drawingMaxY`. Use the canvas width and height through new helper getters if needed — see below.

Required new public getters on `*canvas.Canvas` for this to work without reaching into private fields:

In `internal/canvas/canvas.go` (add):

```go
// Width returns the canvas width in pixels.
func (c *Canvas) Width() int { return c.width }

// Height returns the canvas height in pixels.
func (c *Canvas) Height() int { return c.height }
```

(If `Width()`/`Height()` already exist on `*Canvas`, skip the additions; verify with `grep_search "func (c \*Canvas) (Width|Height)" internal/canvas/canvas.go`.)

The new `render.go` structure (key entry point):

```go
package legend

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/legendlayout"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
)

// RenderInto adds the legend overlay shapes to cv at LayerOverlay.
// Does nothing when cfg is nil, has no entries, or is positioned None.
func RenderInto(cv *canvas.Canvas, cfg *Config) {
	if cfg == nil {
		return
	}

	data := cfg.toLegendData()
	if data == nil || data.Position == model.LegendPositionNone || len(data.Entries) == 0 {
		return
	}

	w, h := legendlayout.MeasureLegend(data, legendlayout.NewBasicMeasurer())
	ox, oy := legendOrigin(cv, data.Position, w, h)

	lb := newLegendBuilder(cv)
	lb.addBackground(ox, oy, w, h)

	px := ox + model.LegendPadding
	py := oy + model.LegendPadding

	if data.Orientation == model.LegendOrientationHorizontal {
		lb.addEntriesH(data, px, py)
	} else {
		contentAreaW := w - 2*model.LegendPadding
		lb.addEntriesV(data, px, py, contentAreaW)
	}
}

func legendOrigin(
	cv *canvas.Canvas, position model.LegendPosition, legendW, legendH float64,
) (ox, oy float64) {
	m := model.LegendMargin
	cw := float64(cv.Width())
	ch := float64(cv.Height())

	switch position {
	case model.LegendPositionTopCenter:
		return (cw - legendW) / 2, float64(cv.DrawingMinY()) + m
	case model.LegendPositionBottomCenter:
		return (cw - legendW) / 2, float64(cv.DrawingMaxY()) - legendH - m
	default:
		return legendlayout.LegendOrigin(position, cw, ch, legendW, legendH)
	}
}

type legendBuilder struct {
	cv       *canvas.Canvas
	bgFill   color.RGBA
	bgBorder color.RGBA
	swBorder color.RGBA
	titleInk color.RGBA
	labelInk color.RGBA
}

func newLegendBuilder(cv *canvas.Canvas) *legendBuilder {
	return &legendBuilder{
		cv:       cv,
		bgFill:   color.RGBA{R: 255, G: 255, B: 255, A: 230},
		bgBorder: color.RGBA{R: 153, G: 153, B: 153, A: 204},
		swBorder: color.RGBA{R: 102, G: 102, B: 102, A: 255},
		titleInk: color.RGBA{R: 38, G: 38, B: 38, A: 255},
		labelInk: color.RGBA{R: 51, G: 51, B: 51, A: 255},
	}
}

func (lb *legendBuilder) addRect(
	x, y, w, h float64, fill, border color.RGBA, borderWidth float64,
) {
	spec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        inks.FixedInk(fill),
			Border:      inks.FixedInk(border),
			BorderWidth: borderWidth,
		},
	}

	lb.cv.AddRectangle(canvas.LayerOverlay, canvas.Rectangle{
		Spec: spec, X: x, Y: y, W: w, H: h, Focus: model.Point{X: 0.5, Y: 0.5},
	})
}

func (lb *legendBuilder) addTextShape(
	x, y float64, content string, ink color.RGBA,
	fontSize float64, anchor canvas.TextAnchor,
) {
	spec := &canvas.TextSpec{
		Ink:      inks.FixedInk(ink),
		FontSize: fontSize,
		Anchor:   anchor,
	}

	lb.cv.AddText(canvas.LayerOverlay, canvas.Text{
		Spec: spec, X: x, Y: y, Content: content,
	})
}
```

Then port the remaining helpers (`addBackground`, `addEntriesV`, `addEntriesH`, `addEntry`, `addNumericSwatches{V,H}`, `addCategorySwatches{V,H}`, `addLabelSample`, `addSwatch`, `addOutlineSwatch`) byte-for-byte from `internal/canvas/legend_render.go`, but with the same `canvas.…` qualifications applied and reading from `lb.cv` instead of `lb.shapes/order`.

> **Insertion-order preservation:** the old code maintained its own `order` counter starting at `baseOrder := len(c.shapes)` and assigned per-shape ordinals. In the new flow, `lb.cv.AddRectangle` / `lb.cv.AddText` themselves increment a shape ordinal (`order: len(c.shapes)` in the existing canvas implementation), so calls in the same order produce the same final layer-then-order sort. Snapshot tests must confirm.

- [ ] **Step 3: Create `internal/legend/render_test.go`**

Move the two existing canvas tests `TestCanvas_SetLegend_DecomposesToPrimitives` and `TestCanvas_SetLegend_WithLabelSample_RendersSampleBeforeEntries` here, rewriting them against `legend.RenderInto`:

```go
package legend_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

func TestRenderInto_DecomposesToPrimitives(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cv := canvas.NewCanvas(800, 600)

	pal := palette.GetPalette(palette.Temperature)
	fillInk := inks.NumericInk("file-size", []float64{10, 50, 100}, pal)

	cfg := &legend.Config{
		Position:    model.LegendPositionBottomRight,
		Orientation: model.LegendOrientationVertical,
		Entries: []legend.Entry{
			{Role: legend.RoleFill, MetricName: "file-size", Ink: fillInk},
		},
	}

	legend.RenderInto(cv, cfg)

	mb := canvas.NewMockBackendForTest() // see Step 4
	err := cv.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.Calls()).NotTo(BeEmpty())
	g.Expect(mb.Calls()[0].Method).To(Equal("DrawRectangle"))

	hasLabel := false
	hasMetric := false
	for _, call := range mb.Calls() {
		if call.Method == "DrawText" && call.Text == "Fill" {
			hasLabel = true
		}
		if call.Method == "DrawText" && call.Text == "file-size" {
			hasMetric = true
		}
	}
	g.Expect(hasLabel).To(BeTrue())
	g.Expect(hasMetric).To(BeTrue())
}

// TestRenderInto_LabelSample_RendersBeforeEntries: similar structure, mirroring
// TestCanvas_SetLegend_WithLabelSample_RendersSampleBeforeEntries.
```

- [ ] **Step 4: Expose a mock-backend helper from `canvas` for cross-package tests**

The existing `internal/canvas/mock_backend_test.go` uses lowercase types (`mockBackend`, `mockCall`) only available to in-package tests. Cross-package tests in `legend_test` need access.

Two options — choose **Option A** unless there's a reason not to:

**Option A (recommended):** Add a tiny `internal/canvas/testsupport/mock.go` package exposing a public `MockBackend` with the same shape. The existing canvas-internal tests are unaffected. The legend tests import it.

**Option B:** Move `mock_backend_test.go` content to a public file `mock_backend.go` (not `_test.go`) so it's visible to other packages. Risk: production code accidentally references it. Mitigate with `//go:build testsupport` constraint or naming convention.

For this plan, use Option A. Create `internal/canvas/testsupport/mock.go`:

```go
// Package testsupport provides canvas-related test doubles for use by
// other packages' tests. It is intentionally separated from the canvas
// package so that production code cannot import the mock by mistake.
package testsupport

// Copy the existing mockBackend implementation from
// internal/canvas/mock_backend_test.go here, renaming:
//   mockBackend  -> MockBackend
//   mockCall     -> Call
//   newMockBackend -> NewMockBackend
//   .calls       -> .Calls (with a method or field rename)
// Match the model.Backend interface.
```

Verify with `go vet ./internal/canvas/testsupport/...`. Update the in-package canvas tests to keep using their existing lowercase mock (no change needed there).

- [ ] **Step 5: Update `internal/legend/legend.go`**

Replace `*canvas.LegendConfig` with `*Config`, `canvas.LegendEntry` with `Entry`, `canvas.LegendRoleFill` etc. with `RoleFill`, `canvas.FixedInk(white)` with `inks.FixedInk(white)`, `canvas.Ink` with `inks.Ink`.

```go
package legend

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

var white = color.RGBA{R: 255, G: 255, B: 255, A: 255} //nolint:gochecknoglobals

func ResolveOptions(posStr, orientStr string) (model.LegendPosition, model.LegendOrientation) {
	pos := model.LegendPosition(posStr)
	if pos == "" {
		pos = model.LegendPositionBottomRight
	}

	orient := model.LegendOrientation(orientStr)
	if orient == "" {
		orient = DefaultOrientation(pos)
	}

	return pos, orient
}

func Build(
	position model.LegendPosition,
	orientation model.LegendOrientation,
	fillInk inks.Ink,
	fillMetric metric.Name,
	borderInk inks.Ink,
	borderMetric metric.Name,
	sizeMetric metric.Name,
) *Config {
	if position == model.LegendPositionNone {
		return nil
	}

	if orientation == "" {
		orientation = DefaultOrientation(position)
	}

	var entries []Entry

	if fillMetric != "" {
		entries = append(entries, Entry{
			Role:       RoleFill,
			MetricName: string(fillMetric),
			Ink:        fillInk,
		})
	}

	if borderMetric != "" {
		entries = append(entries, Entry{
			Role:       RoleBorder,
			MetricName: string(borderMetric),
			Ink:        borderInk,
		})
	}

	if sizeMetric != "" && sizeMetric != fillMetric {
		entries = append(entries, Entry{
			Role:       RoleSize,
			MetricName: string(sizeMetric),
			Ink:        inks.FixedInk(white),
		})
	}

	if len(entries) == 0 {
		return nil
	}

	return &Config{
		Position:    position,
		Orientation: orientation,
		Entries:     entries,
	}
}
```

- [ ] **Step 6: Update `internal/legend/legend_test.go` and `internal/legend/reserve_test.go`**

Substitute `canvas.LegendConfig` → `*legend.Config` (or just `*Config` since this is now `package legend_test`), `canvas.LegendEntry` → `legend.Entry`, `canvas.LegendRoleFill` → `legend.RoleFill`, etc. Drop the `canvas` import if no longer needed; ensure tests build.

`internal/legend/reserve.go` switches its `*canvas.LegendConfig` parameter to `*Config`:

```go
package legend

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
)

const MinReservableSize = 100

func ReserveAndLayout(cfg *Config, width, height int) (layoutW, layoutH int) {
	if cfg == nil {
		return width, height
	}
	// ... rest unchanged ...
}

func LayoutOffset(cfg *Config, wReduce, hReduce float64) (dx, dy float64) {
	// ... unchanged body but parameter type is *Config ...
}

func cornerOffset(cfg *Config, wReduce, hReduce float64) (dx, dy float64) {
	// ... unchanged body but parameter type is *Config ...
}
```

- [ ] **Step 7: Update `internal/canvas/aliases.go` — add legend re-exports**

Merge the new imports into the file's existing `import (...)` block (do not create a second one) and append the new aliases at the end of the file:

Imports to add inside the existing block:

```go
"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
"github.com/theunrepentantgeek/code-visualizer/internal/legend"
```

New alias declarations to append:

```go
// Legend type aliases.
type (
	LegendConfig = legend.Config //nolint:revive // transient alias
	LegendEntry  = legend.Entry
	LegendRole   = legend.Role
)

const (
	LegendRoleFill   = legend.RoleFill
	LegendRoleBorder = legend.RoleBorder
	LegendRoleSize   = legend.RoleSize
)

// DefaultOrientation is preserved as a free function for backward
// compatibility (callers wrote canvas.DefaultOrientation(pos)).
func DefaultOrientation(p model.LegendPosition) model.LegendOrientation {
	return legend.DefaultOrientation(p)
}
```

- [ ] **Step 8: Replace `canvas.Canvas.SetLegend` with a transient wrapper**

In `internal/canvas/canvas.go`:

- The `legend *LegendConfig` field stays (renamed to `pendingLegend *legend.Config`).
- Add a new import for `internal/legend`.
- `SetLegend` becomes a thin wrapper that stashes the config:

```go
// SetLegend is a transient compatibility shim. New callers should use
// legend.RenderInto(cv, cfg) directly between adding data shapes and
// calling Render(). This wrapper is removed in Task 6.
func (c *Canvas) SetLegend(cfg LegendConfig) {
    c.pendingLegend = &cfg
}
```

> Taking `&cfg` of a value parameter is safe: escape analysis heap-allocates the parameter copy because its address escapes via `c.pendingLegend`. Each `SetLegend` call gets its own backing storage.

- In `RenderTo`, replace the `if c.legend != nil { allShapes = append(...) }` block with a call that flushes any pending legend through the new package function **before** sorting:

```go
func (c *Canvas) RenderTo(backend Backend) error {
    if c.pendingLegend != nil {
        legend.RenderInto(c, c.pendingLegend)
        c.pendingLegend = nil
    }

    // ... existing sort + dispatch loop unchanged, but no decomposeLegend call ...
}
```

> **Why nil out after flushing?** So that a caller invoking `RenderTo` twice doesn't double-add legend shapes. Production code calls `Render` (which calls `RenderTo`) once; tests that re-invoke would need to call `SetLegend` again, which matches the old semantics.

- [ ] **Step 9: Delete the old canvas legend files**

```bash
rm internal/canvas/legend.go
rm internal/canvas/legend_render.go
```

- [ ] **Step 10: Remove the canvas-internal SetLegend tests**

In `internal/canvas/canvas_test.go`, delete:
- `TestCanvas_SetLegend_DecomposesToPrimitives`
- `TestCanvas_SetLegend_WithLabelSample_RendersSampleBeforeEntries`

These have been re-homed in `internal/legend/render_test.go` in Step 3. Keep `TestCanvas_NoLegend_NoPrimitives` (a smoke test for the `pendingLegend == nil` path); confirm it still passes.

- [ ] **Step 11: Build & test**

Run: `go build ./...`
Expected: clean.

Run: `go test ./internal/legend/... ./internal/canvas/... ./internal/inks/... -count=1`
Expected: PASS in all three.

- [ ] **Step 12: Full CI gate**

Dispatch an `Explore` subagent: "Run `task ci`. Report exit status, failing linters/tests, and offending file:line entries."
Expected: PASS. **Critical:** all golden snapshots must match — this is the highest-risk task for snapshot drift.

If snapshots drift, the most likely cause is the `categoricalSwatches` rebuilding the `CategoricalMapper` from sorted categories (see Step 6 of Task 2). Compare the old code path: `*baseInk` stored `catMapper` constructed with the **unsorted** `categories` list, but `categoricalLegendSwatches` then sorted a **copy** before calling `ink.catMapper.Map(cat)` — so colours were assigned by the **original** order, with sorted labels. The new `categoricalSwatches` instead constructs a new mapper from sorted categories, which assigns colours by **sorted** order. This is a behaviour change.

If snapshots drift here, fix by building the mapper from the **original** category order:

```go
func categoricalSwatches(ink Ink) []model.LegendSwatch {
	categories := ink.Categories()
	pal := ink.Palette()

	if len(categories) == 0 || len(pal.Colours) == 0 {
		return nil
	}

	// Build mapper from original order (matches palette assignment in CategoricalInk).
	mapper := palette.NewCategoricalMapper(categories, pal)

	// Iterate in sorted order for label display.
	sorted := make([]string, len(categories))
	copy(sorted, categories)
	slices.Sort(sorted)

	swatches := make([]model.LegendSwatch, len(sorted))
	for i, cat := range sorted {
		swatches[i] = model.LegendSwatch{
			Colour: mapper.Map(cat),
			Label:  cat,
		}
	}

	return swatches
}
```

Re-run `task test`. Expected: PASS, snapshots unchanged.

- [ ] **Step 13: Commit**

```bash
git add internal/legend internal/canvas internal/inks
git commit -m "refactor(legend): move legend types and rendering out of canvas

LegendConfig/Entry/Role and the rendering machinery now live in
internal/legend. canvas exposes a transient SetLegend wrapper that
delegates to legend.RenderInto so existing visualization call sites
keep working. canvas/aliases.go re-exports the legend names for
backward compatibility. Aliases removed in a later commit."
```

---

## Task 5: Migrate visualization packages

Each sub-task is a separate commit so failures are bisectable per package. Apply the same mechanical substitutions in each:

**Substitution table for all viz packages:**

| From | To |
|---|---|
| `canvas.Ink` | `inks.Ink` |
| `canvas.MetricValue` | `inks.MetricValue` |
| `canvas.MeasureValue` / `QuantityValue` / `CategoryValue` | `inks.MeasureValue` / `QuantityValue` / `CategoryValue` |
| `canvas.FixedInk` / `NumericInk` / `CategoricalInk` | `inks.FixedInk` / `NumericInk` / `CategoricalInk` |
| `canvas.NewRadialGradientInk` | `inks.NewRadialGradientInk` |
| `canvas.RadialGradientInk` | `inks.RadialGradientInk` |
| `canvas.InkKind` | `inks.Kind` |
| `canvas.InkFixed` / `InkNumeric` / `InkCategorical` | `inks.KindFixed` / `KindNumeric` / `KindCategorical` |
| `canvas.WithOpacity` | `inks.WithOpacity` |
| `canvas.LegendConfig` | `*legend.Config` (always pointer in this codebase) |
| `canvas.LegendEntry` | `legend.Entry` |
| `canvas.LegendRoleFill` / `LegendRoleBorder` / `LegendRoleSize` | `legend.RoleFill` / `RoleBorder` / `RoleSize` |
| `canvas.DefaultOrientation(pos)` | `legend.DefaultOrientation(pos)` |
| `cv.SetLegend(*cfg)` line | Delete; insert `legend.RenderInto(cv, cfg)` immediately before `cv.Render(...)` |

Add `inks` and/or `legend` imports as needed; remove unused ones. The existing `pkginks "…/internal/inks"` alias in each viz package is now redundant (`inks` no longer collides with a local variable named `inks` because the type itself moved out of canvas) — but **leave the alias in place** for this task; alias cleanup is optional and out of scope.

> **Important:** `cv.SetLegend(*cfg)` takes a `LegendConfig` by value; `legend.RenderInto(cv, cfg)` takes a `*Config`. If the call site builds `cfg` as `*legend.Config` (which `legend.Build` returns), pass it directly. If `cfg` is nil, `RenderInto` is a no-op; do **not** wrap in `if cfg != nil { ... }`.

### Task 5a: treemap

**Files:**
- Modify: `internal/treemap/inks.go`, `render.go`, `stages.go`, `labels.go`
- Modify: `internal/treemap/inks_test.go`, `render_test.go`, `stages_test.go`, `render_focus_test.go`, `labels_test.go`

- [ ] **Step 1: Apply the substitution table to every file in `internal/treemap/`**

Use a series of `multi_replace_string_in_file` operations grouped by file. For each file, the sequence is:
1. Read the file.
2. Apply substitutions from the table above.
3. Update imports: add `internal/inks` and/or `internal/legend`, remove unused `canvas` aliases.

Specific known sites in treemap:
- `internal/treemap/inks.go:30-44`: `Fill canvas.Ink` → `Fill inks.Ink`; `Border canvas.Ink` → `Border inks.Ink`; `canvas.FixedInk(structuralBorder)` → `inks.FixedInk(structuralBorder)`
- `internal/treemap/stages.go:41`: `canvas.NewRadialGradientInk(...)` → `inks.NewRadialGradientInk(...)`
- `internal/treemap/render.go:27,28,86,87,103,118,119,120,219,220`: every `canvas.FixedInk` → `inks.FixedInk`; `canvas.InkNumeric` → `inks.KindNumeric`; `canvas.InkFixed` → `inks.KindFixed`; `canvas.TextColourFor` stays as `canvas.TextColourFor` (it remains in `canvas` per spec)
- `internal/treemap/labels.go`: `canvas.Ink` → `inks.Ink`; `canvas.TextColourFor` stays
- `internal/treemap/render.go::DynBorderWidth`: parameter `borderKind canvas.InkKind` → `borderKind inks.Kind`; compare with `inks.KindFixed`
- `internal/treemap/stages_test.go:80`: type assertion `viz.Inks.Fill.(*canvas.RadialGradientInk)` → `viz.Inks.Fill.(*inks.RadialGradientInk)`
- `internal/treemap/*_test.go`: any `canvas.InkNumeric`/`InkCategorical`/`InkFixed` → `inks.KindNumeric`/etc.

- [ ] **Step 2: Locate the `SetLegend` call site**

Run: `grep -n "SetLegend" internal/treemap/`
Expected: 0 or 1 hit per renderer entry point. If found, replace `cv.SetLegend(*cfg)` with `legend.RenderInto(cv, cfg)` placed just before the call to `cv.Render(...)` (or the equivalent renderer-finalising call).

- [ ] **Step 3: Build & test**

Run: `go test ./internal/treemap/... -count=1`
Expected: PASS. Golden snapshots unchanged.

- [ ] **Step 4: Lint check via Explore subagent**

Dispatch an `Explore` subagent: "Run `task lint`. Report failing linters and offending file:line entries. One-line note if no issues."
Expected: no new issues.

- [ ] **Step 5: Commit**

```bash
git add internal/treemap
git commit -m "refactor(treemap): use inks and legend packages directly"
```

### Task 5b: bubbletree

Same structure as Task 5a, but for `internal/bubbletree/`. Known substitution sites:

- `internal/bubbletree/inks.go:24-40`: `canvas.Ink` → `inks.Ink`; `canvas.FixedInk` → `inks.FixedInk`
- `internal/bubbletree/stages.go:49`: `canvas.NewRadialGradientInk(...)` → `inks.NewRadialGradientInk(...)`
- `internal/bubbletree/render.go:49,50,110,111,208,220,256`: all `canvas.FixedInk` → `inks.FixedInk`; `canvas.Ink` → `inks.Ink`; `canvas.WithOpacity` → `inks.WithOpacity`
- `internal/bubbletree/layout_stage_test.go:82`: `canvas.NumericInk(...)` → `inks.NumericInk(...)`
- `internal/bubbletree/inks_test.go:40-82`: `canvas.InkFixed`/`InkNumeric`/`InkCategorical` → `inks.KindFixed`/etc.

Steps 1–5 identical pattern. Commit message: `refactor(bubbletree): use inks and legend packages directly`.

### Task 5c: spiral

Same pattern for `internal/spiral/`. Known sites:

- `internal/spiral/inks.go:19-98`: `canvas.Ink`/`FixedInk`/`NumericInk`/`CategoricalInk` → `inks.…`
- `internal/spiral/render.go:44,45,78,140,152,190-199`: all `canvas.*Ink`/`MetricValue`/`MeasureValue`/`CategoryValue` → `inks.…`; `canvas.InkNumeric`/`InkCategorical` → `inks.KindNumeric`/`KindCategorical`
- `internal/spiral/stages_test.go:194`, `internal/spiral/inks_test.go:71-100`: `canvas.Ink*` → `inks.Kind*`

Commit: `refactor(spiral): use inks and legend packages directly`.

### Task 5d: scatter

Same pattern for `internal/scatter/`. Known sites:

- `internal/scatter/inks.go:25-110`: full migration
- `internal/scatter/render.go:34,35,57,58,74,139,148,160,179,221,223`: every `canvas.FixedInk` → `inks.FixedInk`; `canvas.TextColourFor` stays
- `internal/scatter/stages_test.go:126`: `canvas.InkNumeric` → `inks.KindNumeric`

Commit: `refactor(scatter): use inks and legend packages directly`.

### Task 5e: radialtree

Same pattern for `internal/radialtree/`. Known sites:

- `internal/radialtree/inks.go:25-40`: full migration
- `internal/radialtree/render.go:49,50,66,180,181,219,238,255,303,329,361,362`: full migration
- `internal/radialtree/inks_test.go:35-116`, `render_internal_test.go:49-89,332,361,362`: `canvas.Ink*` → `inks.Kind*`/`inks.MetricValue`

Commit: `refactor(radialtree): use inks and legend packages directly`.

### Task 5f: cmd/codeviz

**Files:**
- Modify: any file under `cmd/codeviz/` that imports `canvas` for ink or legend types

- [ ] **Step 1: Locate canvas-ink/legend references**

```bash
grep -rn "canvas\.\(Ink\|FixedInk\|NumericInk\|CategoricalInk\|MetricValue\|MeasureValue\|QuantityValue\|CategoryValue\|InkOption\|WithOpacity\|RadialGradientInk\|NewRadialGradientInk\|LegendConfig\|LegendEntry\|LegendRole\|DefaultOrientation\|InkKind\|InkFixed\|InkNumeric\|InkCategorical\|SetLegend\)" cmd/codeviz/
```

- [ ] **Step 2: Apply substitution table**

Update each match per the table at the top of Task 5.

- [ ] **Step 3: Build & test**

Run: `go test ./cmd/codeviz/... -count=1`
Expected: PASS.

- [ ] **Step 4: Lint check via Explore subagent**

Dispatch `Explore`: "Run `task lint`."
Expected: no new issues.

- [ ] **Step 5: Commit**

```bash
git add cmd/codeviz
git commit -m "refactor(cmd): use inks and legend packages directly"
```

---

## Task 6: Remove canvas compatibility aliases

After Task 5 completes, no caller outside `canvas` references the alias names. Time to delete them.

**Files:**
- Delete: `internal/canvas/aliases.go`
- Modify: `internal/canvas/canvas.go` (remove transient `SetLegend` wrapper, `pendingLegend` field, and the `legend.RenderInto(c, c.pendingLegend)` flush in `RenderTo`)

- [ ] **Step 1: Verify no external references to alias names**

```bash
grep -rn "canvas\.\(Ink\|FixedInk\|NumericInk\|CategoricalInk\|MetricValue\|MeasureValue\|QuantityValue\|CategoryValue\|InkOption\|WithOpacity\|RadialGradientInk\|NewRadialGradientInk\|LegendConfig\|LegendEntry\|LegendRole\|DefaultOrientation\|InkKind\|InkFixed\|InkNumeric\|InkCategorical\)" --include='*.go' .
```

Expected: zero hits outside `internal/canvas/aliases.go`.

If any hit appears in a viz package or `cmd/codeviz`, return to Task 5 for that package and fix.

- [ ] **Step 2: Verify no external references to `SetLegend`**

```bash
grep -rn "\.SetLegend(" --include='*.go' .
```

Expected: zero hits.

If any hit appears, replace it with `legend.RenderInto(cv, cfg)` and recommit under Task 5.

- [ ] **Step 3: Delete `internal/canvas/aliases.go`**

```bash
rm internal/canvas/aliases.go
```

- [ ] **Step 4: Remove `SetLegend` and the `pendingLegend` plumbing from `canvas.go`**

Delete:

```go
func (c *Canvas) SetLegend(cfg LegendConfig) { ... }
```

In the `Canvas` struct, delete the `pendingLegend *legend.Config` field. In `RenderTo`, delete the lines:

```go
if c.pendingLegend != nil {
    legend.RenderInto(c, c.pendingLegend)
    c.pendingLegend = nil
}
```

Remove the `internal/legend` import from `canvas.go` (it will now have no users in `canvas`).

- [ ] **Step 5: Build & test**

Run: `go build ./...`
Expected: clean. If a viz package fails because it still calls `cv.SetLegend(...)` or references an alias, return to Task 5 to finish that package.

Run: `go test ./... -count=1`
Expected: PASS. Golden snapshots unchanged.

- [ ] **Step 6: Full CI gate**

Dispatch an `Explore` subagent: "Run `task ci`. Report exit status, failing linters/tests, and offending file:line entries."
Expected: PASS.

- [ ] **Step 7: Verify the dependency invariant**

```bash
go list -deps ./internal/inks | grep -E "^github.com/theunrepentantgeek/code-visualizer/internal/canvas($|/)" || echo "inks does not depend on canvas"
```

Expected output: `inks does not depend on canvas` (because `inks` only imports `canvas/model` and `canvas/legendlayout`, not the root `canvas` package).

Also verify:

```bash
go list -deps ./internal/canvas | grep -E "^github.com/theunrepentantgeek/code-visualizer/internal/legend$" || echo "canvas does not depend on legend"
```

Expected: `canvas does not depend on legend`.

- [ ] **Step 8: Commit**

```bash
git add internal/canvas
git commit -m "refactor(canvas): remove transient ink/legend compatibility aliases

Completes the inks-encapsulation migration:
- canvas/aliases.go is gone
- Canvas.SetLegend is gone; callers use legend.RenderInto(cv, cfg)
- canvas no longer depends on legend
- inks no longer depends on canvas (only canvas/model and
  canvas/legendlayout, which are leaf subpackages)

See docs/superpowers/specs/2026-06-20-inks-encapsulation-design.md"
```

---

## Self-Review Checklist

Before considering the plan complete, the implementing agent should verify:

1. **Spec coverage.** Each numbered scope item in the spec maps to at least one task:
   - Spec §1 (move ink types) → Task 2.
   - Spec §2 (move legend) → Task 4.
   - Spec §3 (drawing-bound getters) → Task 1.
   - Spec §4 (drop stutter) → applied throughout Tasks 2, 4, 5.
   - Spec §5 (update viz callers + cmd/codeviz) → Task 5.

2. **Naming consistency.** Verify type names match across tasks:
   - `inks.Ink`, `inks.Kind`, `inks.KindFixed`/`KindNumeric`/`KindCategorical`, `inks.Info`, `inks.Option`, `inks.MetricValue`, `inks.RadialGradientInk`.
   - `legend.Config`, `legend.Entry`, `legend.Role`, `legend.RoleFill`/`RoleBorder`/`RoleSize`, `legend.RenderInto`, `legend.DefaultOrientation`.
   - `canvas.LayerOverlay`, `canvas.RectangleSpec`, `canvas.TextSpec` (unchanged).

3. **No placeholders.** Scan for `TBD` / `TODO` / "implement later" / "similar to". None present.

4. **Lint exception comments.** The `//nolint:revive` and `//nolint:gochecknoglobals` on alias declarations are deliberate; the alias file is transient. golangci-lint v2 in this repo respects file-scoped directives.

5. **Snapshot risk.** Task 4 Step 12 explicitly handles the one known behaviour-change vector (`CategoricalMapper` reconstruction order). All other moves are pure code relocation.

---

## Execution Notes for Agentic Workers

- **Per-task commit hygiene.** Each task corresponds to exactly one commit unless explicitly subdivided (Task 5 has six sub-commits, one per package). Do not squash.
- **Lint output volume.** `task lint` runs with `--verbose`; per the repo's agent-workflow rules in `.github/copilot-instructions.md`, dispatch lint runs via an `Explore` subagent that filters output to exit status, failing linters, and offending file:line.
- **Continuous execution.** Per the repo's "continuous execution" rule, never end a turn with only a status recap. After each commit, the next action is the next task's first step.
