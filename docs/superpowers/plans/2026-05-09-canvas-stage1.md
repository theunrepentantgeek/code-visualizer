# Canvas Stage 1 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the `internal/canvas/` package — a retained-then-render drawing surface with Ink colour resolution, Spec style templates, layered z-ordering, and pluggable backends (raster, SVG, mock) — fully tested in isolation with zero changes to existing code.

**Architecture:** Shapes are added to a Canvas with a layer assignment, then rendered in batch. Ink resolves metric values to RGBA via palette + bucketing. Specs bundle Inks with visual properties (border width, label style). At render time, the Canvas sorts shapes by layer, resolves colours via `Ink.Dip()`, and dispatches drawing primitives to a Backend. Two production backends (raster via `fogleman/gg`, SVG via direct XML) and one mock backend for unit testing.

**Tech Stack:** Go 1.26+, `fogleman/gg` (raster), `golang.org/x/image` (fonts), `github.com/rotisserie/eris` (errors), Gomega (assertions), Goldie v2 (golden-file snapshots)

**Spec:** `docs/superpowers/specs/2026-05-08-canvas-design.md`

**Lint constraints:**
- `funlen`: 65 lines (ignore-comments), excluded in `_test.go`
- `revive` `max-public-structs`: 5 per file (enabled via `enable-all-rules`)
- `revive` `line-length-limit`: 120 chars
- `revive` `cognitive-complexity`: 10

---

## File Structure

All new files live under `internal/canvas/`. No existing files are modified.

| File | Responsibility | Public Structs |
|------|---------------|----------------|
| `internal/canvas/metric_value.go` | `MetricValue` struct, constructors | 1 (`MetricValue`) |
| `internal/canvas/ink.go` | `Ink` struct, constructors (`FixedInk`, `NumericInk`, `CategoricalInk`), `Dip()` | 1 (`Ink`) |
| `internal/canvas/ink_options.go` | `InkOption`, `MappingStrategy`, `WithMapping()`, `WithOpacity()` | 1 (`MappingStrategy`) |
| `internal/canvas/ink_introspection.go` | `Boundaries()`, `Palette()`, `Categories()` on Ink | 0 |
| `internal/canvas/spec.go` | `ShapeStyle`, `RectangleSpec`, `DiscSpec`, `LineSpec` | 4 (`ShapeStyle`, `RectangleSpec`, `DiscSpec`, `LineSpec`) |
| `internal/canvas/text_spec.go` | `TextSpec`, `TextAnchor`, `LabelStyle` | 1 (`TextSpec`) |
| `internal/canvas/shape.go` | `Rectangle`, `Disc`, `Text`, `Line`, `Path` | 5 (`Rectangle`, `Disc`, `Text`, `Line`, `Path`) |
| `internal/canvas/geometry.go` | `Position`, `Size` | 2 (`Position`, `Size`) |
| `internal/canvas/layer.go` | `Layer` type, layer constants | 0 (Layer is `type Layer int`) |
| `internal/canvas/backend.go` | `Backend` interface | 0 (interface, not struct) |
| `internal/canvas/canvas.go` | `Canvas` struct, `NewCanvas()`, `Add*()`, `Render()` | 1 (`Canvas`) |
| `internal/canvas/format.go` | `ImageFormat`, `FormatFromPath()` | 0 (ImageFormat is `type ImageFormat int`) |
| `internal/canvas/text_colour.go` | `TextColourFor()` | 0 |
| `internal/canvas/legend.go` | `LegendConfig`, `LegendEntry`, `LegendRole`, `LegendPosition`, `LegendOrientation` | 2 (`LegendConfig`, `LegendEntry`) |
| `internal/canvas/raster/backend.go` | `New()` → `canvas.Backend` (unexported `rasterBackend` struct) | 0 |
| `internal/canvas/svg/backend.go` | `New()` → `canvas.Backend` (unexported `svgBackend` struct) | 0 |
| **Test files** | |
| `internal/canvas/metric_value_test.go` | MetricValue tests |
| `internal/canvas/ink_test.go` | Ink tests (Dip, opacity, edge cases) |
| `internal/canvas/ink_introspection_test.go` | Introspection tests |
| `internal/canvas/format_test.go` | FormatFromPath tests |
| `internal/canvas/text_colour_test.go` | TextColourFor tests |
| `internal/canvas/canvas_test.go` | Canvas add/render tests with mock backend |
| `internal/canvas/mock_backend_test.go` | `mockBackend` for testing (test-only file) |
| `internal/canvas/layer_test.go` | Layer ordering tests |
| `internal/canvas/raster/backend_test.go` | Raster backend golden-file tests |
| `internal/canvas/svg/backend_test.go` | SVG backend golden-file tests |

---

### Task 1: MetricValue Type

**Files:**
- Create: `internal/canvas/metric_value.go`
- Create: `internal/canvas/metric_value_test.go`

- [ ] **Step 1: Create the canvas package directory**

```bash
mkdir -p internal/canvas
```

- [ ] **Step 2: Write the failing test**

Create `internal/canvas/metric_value_test.go`:

```go
package canvas

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

func TestMeasureValue_SetsMeasureKind(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mv := MeasureValue(3.14)
	g.Expect(mv.Kind).To(Equal(metric.Measure))
	g.Expect(mv.Measure).To(Equal(3.14))
}

func TestQuantityValue_SetsQuantityKind(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mv := QuantityValue(42)
	g.Expect(mv.Kind).To(Equal(metric.Quantity))
	g.Expect(mv.Quantity).To(Equal(42))
}

func TestCategoryValue_SetsCategoryKind(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	mv := CategoryValue("go")
	g.Expect(mv.Kind).To(Equal(metric.Classification))
	g.Expect(mv.Category).To(Equal("go"))
}

func TestZeroValue_HasZeroKind(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var mv MetricValue
	g.Expect(mv.Kind).To(Equal(metric.Quantity))
	g.Expect(mv.Measure).To(Equal(0.0))
	g.Expect(mv.Quantity).To(Equal(0))
	g.Expect(mv.Category).To(BeEmpty())
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd internal/canvas && go test ./... -run TestMeasureValue -v
```

Expected: FAIL — `MetricValue`, `MeasureValue`, `QuantityValue`, `CategoryValue` undefined.

- [ ] **Step 4: Write minimal implementation**

Create `internal/canvas/metric_value.go`:

```go
// Package canvas provides a retained-then-render drawing surface with
// layered z-ordering and pluggable backend implementations.
package canvas

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

- [ ] **Step 5: Run test to verify it passes**

```bash
cd internal/canvas && go test ./... -v
```

Expected: PASS — all 4 tests green.

- [ ] **Step 6: Commit**

```bash
git add internal/canvas/metric_value.go internal/canvas/metric_value_test.go
git commit -m "feat(canvas): add MetricValue type and constructors"
```

---

### Task 2: InkOption, MappingStrategy

**Files:**
- Create: `internal/canvas/ink_options.go`

This task has no separate test file — `InkOption` and `MappingStrategy` are consumed by `Ink` and tested in Task 3.

- [ ] **Step 1: Write the implementation**

Create `internal/canvas/ink_options.go`:

```go
package canvas

// MappingStrategy controls how numeric metric values are mapped to palette colours.
type MappingStrategy int

const (
	// Quantile uses equal-count buckets (current default behaviour).
	Quantile MappingStrategy = iota
	// Linear uses evenly spaced buckets across the min-max range.
	Linear
	// Logarithmic uses log-scale spacing.
	Logarithmic
)

// InkOption configures ink behaviour.
type InkOption func(*inkConfig)

type inkConfig struct {
	strategy MappingStrategy
	opacity  float64
}

func defaultInkConfig() inkConfig {
	return inkConfig{
		strategy: Quantile,
		opacity:  1.0,
	}
}

// WithMapping sets the mapping strategy for numeric inks.
func WithMapping(strategy MappingStrategy) InkOption {
	return func(c *inkConfig) {
		c.strategy = strategy
	}
}

// WithOpacity sets the opacity applied when Dip() resolves a colour.
// Default is 1.0 (fully opaque). The opacity is applied to the alpha channel
// of the resolved colour.
func WithOpacity(opacity float64) InkOption {
	return func(c *inkConfig) {
		c.opacity = opacity
	}
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd internal/canvas && go build ./...
```

Expected: BUILD OK — no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/canvas/ink_options.go
git commit -m "feat(canvas): add InkOption and MappingStrategy types"
```

---

### Task 3: Ink — Core (Dip + Constructors)

**Files:**
- Create: `internal/canvas/ink.go`
- Create: `internal/canvas/ink_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/canvas/ink_test.go`:

```go
package canvas

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

func TestFixedInk_Dip_ReturnsFixedColour(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	ink := FixedInk(red)

	result := ink.Dip(MeasureValue(99.9))
	g.Expect(result).To(Equal(red))
}

func TestFixedInk_Dip_IgnoresMetricValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	blue := color.RGBA{R: 0, G: 0, B: 255, A: 255}
	ink := FixedInk(blue)

	g.Expect(ink.Dip(QuantityValue(0))).To(Equal(blue))
	g.Expect(ink.Dip(CategoryValue("anything"))).To(Equal(blue))
	g.Expect(ink.Dip(MetricValue{})).To(Equal(blue))
}

func TestFixedInk_WithOpacity(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	ink := FixedInk(white, WithOpacity(0.5))

	result := ink.Dip(MetricValue{})
	g.Expect(result.R).To(Equal(uint8(255)))
	g.Expect(result.G).To(Equal(uint8(255)))
	g.Expect(result.B).To(Equal(uint8(255)))
	g.Expect(result.A).To(BeNumerically("~", 128, 2))
}

func TestNumericInk_Dip_MapsToColour(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	values := []float64{10, 20, 30, 40, 50}
	pal := palette.GetPalette(palette.Neutral)
	ink := NumericInk(values, pal)

	lowResult := ink.Dip(MeasureValue(10))
	highResult := ink.Dip(MeasureValue(50))

	g.Expect(lowResult.A).To(Equal(uint8(255)))
	g.Expect(highResult.A).To(Equal(uint8(255)))
	g.Expect(lowResult).NotTo(Equal(highResult))
}

func TestNumericInk_Dip_UsesQuantity(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	values := []float64{1, 2, 3, 4, 5}
	pal := palette.GetPalette(palette.Neutral)
	ink := NumericInk(values, pal)

	lowResult := ink.Dip(QuantityValue(1))
	highResult := ink.Dip(QuantityValue(5))

	g.Expect(lowResult).NotTo(Equal(highResult))
}

func TestCategoricalInk_Dip_MapsCategories(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	categories := []string{"go", "rs", "py"}
	pal := palette.GetPalette(palette.Categorization)
	ink := CategoricalInk(categories, pal)

	goCol := ink.Dip(CategoryValue("go"))
	rsCol := ink.Dip(CategoryValue("rs"))
	pyCol := ink.Dip(CategoryValue("py"))

	g.Expect(goCol.A).To(Equal(uint8(255)))
	g.Expect(rsCol.A).To(Equal(uint8(255)))
	g.Expect(pyCol.A).To(Equal(uint8(255)))

	colours := map[color.RGBA]bool{goCol: true, rsCol: true, pyCol: true}
	g.Expect(colours).To(HaveLen(3))
}

func TestCategoricalInk_Dip_UnknownCategory_ReturnsGrey(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	categories := []string{"go"}
	pal := palette.GetPalette(palette.Categorization)
	ink := CategoricalInk(categories, pal)

	result := ink.Dip(CategoryValue("unknown"))
	g.Expect(result).To(Equal(color.RGBA{R: 128, G: 128, B: 128, A: 255}))
}

func TestNumericInk_WithOpacity(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	values := []float64{10, 50}
	pal := palette.GetPalette(palette.Neutral)
	ink := NumericInk(values, pal, WithOpacity(0.18))

	result := ink.Dip(MeasureValue(30))
	g.Expect(result.A).To(BeNumerically("~", 46, 2))
}

func TestNumericInk_EmptyValues_ReturnsMiddleColour(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Neutral)
	ink := NumericInk(nil, pal)

	result := ink.Dip(MeasureValue(42))
	mid := len(pal.Colours) / 2
	g.Expect(result).To(Equal(pal.Colours[mid]))
}

func TestFixedInk_IsCopySafe(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	red := color.RGBA{R: 255, A: 255}
	ink1 := FixedInk(red)
	ink2 := ink1

	r1 := ink1.Dip(MetricValue{})
	r2 := ink2.Dip(MetricValue{})
	g.Expect(r1).To(Equal(r2))
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd internal/canvas && go test ./... -run TestFixedInk -v
```

Expected: FAIL — `Ink`, `FixedInk`, `NumericInk`, `CategoricalInk` undefined.

- [ ] **Step 3: Write minimal implementation**

Create `internal/canvas/ink.go`:

```go
package canvas

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

type inkKind int

const (
	inkFixed       inkKind = iota
	inkNumeric
	inkCategorical
)

// Ink resolves metric values to colours.
// Fixed inks ignore the metric value; metric inks resolve via palette + mapping strategy.
//
// Ink is safe to copy; internal state is shared via pointers.
type Ink struct {
	kind       inkKind
	color      color.RGBA
	boundaries *metric.BucketBoundaries
	catMapper  *palette.CategoricalMapper
	pal        palette.ColourPalette
	categories []string
	opacity    float64
}

// FixedInk always produces the same colour regardless of input.
func FixedInk(c color.RGBA, opts ...InkOption) Ink {
	cfg := defaultInkConfig()
	for _, o := range opts {
		o(&cfg)
	}

	return Ink{
		kind:    inkFixed,
		color:   c,
		opacity: cfg.opacity,
	}
}

// NumericInk maps numeric metric values to palette colours.
// Takes the full dataset of values (for bucketing), the palette,
// and optional configuration options.
func NumericInk(values []float64, pal palette.ColourPalette, opts ...InkOption) Ink {
	cfg := defaultInkConfig()
	for _, o := range opts {
		o(&cfg)
	}

	buckets := metric.ComputeBuckets(values, len(pal.Colours))

	return Ink{
		kind:       inkNumeric,
		boundaries: &buckets,
		pal:        pal,
		opacity:    cfg.opacity,
	}
}

// CategoricalInk maps string categories to palette colours.
func CategoricalInk(categories []string, pal palette.ColourPalette, opts ...InkOption) Ink {
	cfg := defaultInkConfig()
	for _, o := range opts {
		o(&cfg)
	}

	return Ink{
		kind:       inkCategorical,
		catMapper:  palette.NewCategoricalMapper(categories, pal),
		pal:        pal,
		categories: categories,
		opacity:    cfg.opacity,
	}
}

// Dip resolves a MetricValue to an RGBA colour.
func (ink Ink) Dip(value MetricValue) color.RGBA {
	var c color.RGBA

	switch ink.kind {
	case inkFixed:
		c = ink.color
	case inkNumeric:
		c = ink.dipNumeric(value)
	case inkCategorical:
		c = ink.catMapper.Map(value.Category)
	}

	return applyOpacity(c, ink.opacity)
}

func (ink Ink) dipNumeric(value MetricValue) color.RGBA {
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

- [ ] **Step 4: Run test to verify it passes**

```bash
cd internal/canvas && go test ./... -v
```

Expected: PASS — all Ink and MetricValue tests green.

- [ ] **Step 5: Commit**

```bash
git add internal/canvas/ink.go internal/canvas/ink_test.go
git commit -m "feat(canvas): add Ink type with Dip colour resolution"
```

---

### Task 4: Ink Introspection

**Files:**
- Create: `internal/canvas/ink_introspection.go`
- Create: `internal/canvas/ink_introspection_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/canvas/ink_introspection_test.go`:

```go
package canvas

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

func TestNumericInk_Boundaries_ReturnsBucketValues(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	values := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90}
	pal := palette.GetPalette(palette.Neutral)
	ink := NumericInk(values, pal)

	boundaries := ink.Boundaries()
	g.Expect(boundaries).NotTo(BeEmpty())
}

func TestFixedInk_Boundaries_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := FixedInk(white)
	g.Expect(ink.Boundaries()).To(BeNil())
}

func TestCategoricalInk_Boundaries_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := CategoricalInk([]string{"go"}, palette.GetPalette(palette.Categorization))
	g.Expect(ink.Boundaries()).To(BeNil())
}

func TestNumericInk_Palette_ReturnsPalette(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Temperature)
	ink := NumericInk([]float64{1, 2}, pal)

	g.Expect(ink.Palette().Name).To(Equal(palette.Temperature))
}

func TestFixedInk_Palette_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := FixedInk(white)
	g.Expect(ink.Palette().Colours).To(BeEmpty())
}

func TestCategoricalInk_Categories_ReturnsList(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cats := []string{"go", "rs", "py"}
	ink := CategoricalInk(cats, palette.GetPalette(palette.Categorization))

	g.Expect(ink.Categories()).To(Equal(cats))
}

func TestNumericInk_Categories_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := NumericInk([]float64{1, 2}, palette.GetPalette(palette.Neutral))
	g.Expect(ink.Categories()).To(BeNil())
}

func TestFixedInk_Categories_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := FixedInk(white)
	g.Expect(ink.Categories()).To(BeNil())
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd internal/canvas && go test ./... -run TestNumericInk_Boundaries -v
```

Expected: FAIL — `Boundaries`, `Palette`, `Categories` methods undefined on Ink. Also `white` is undefined.

- [ ] **Step 3: Write minimal implementation**

Create `internal/canvas/ink_introspection.go`:

```go
package canvas

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// Colours used by introspection tests and internal defaults.
var (
	white = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	black = color.RGBA{R: 0, G: 0, B: 0, A: 255}
)

// Boundaries returns the bucket boundary values for numeric inks.
// Returns nil for fixed or categorical inks.
func (ink Ink) Boundaries() []float64 {
	if ink.kind != inkNumeric || ink.boundaries == nil {
		return nil
	}

	return ink.boundaries.Boundaries
}

// Palette returns the colour palette used by this ink.
// Returns an empty palette for fixed inks.
func (ink Ink) Palette() palette.ColourPalette {
	if ink.kind == inkFixed {
		return palette.ColourPalette{}
	}

	return ink.pal
}

// Categories returns the category labels for categorical inks.
// Returns nil for fixed or numeric inks.
func (ink Ink) Categories() []string {
	if ink.kind != inkCategorical {
		return nil
	}

	return ink.categories
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd internal/canvas && go test ./... -v
```

Expected: PASS — all introspection tests green.

- [ ] **Step 5: Commit**

```bash
git add internal/canvas/ink_introspection.go internal/canvas/ink_introspection_test.go
git commit -m "feat(canvas): add Ink introspection methods (Boundaries, Palette, Categories)"
```

---

### Task 5: Geometry Helpers (Position, Size)

**Files:**
- Create: `internal/canvas/geometry.go`

No separate tests — these are trivial data structs used by shapes and backends.

- [ ] **Step 1: Write the implementation**

Create `internal/canvas/geometry.go`:

```go
package canvas

// Position represents a 2D coordinate.
type Position struct {
	X, Y float64
}

// Size represents a width and height.
type Size struct {
	Width, Height float64
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd internal/canvas && go build ./...
```

Expected: BUILD OK.

- [ ] **Step 3: Commit**

```bash
git add internal/canvas/geometry.go
git commit -m "feat(canvas): add Position and Size geometry helpers"
```

---

### Task 6: Layer Type and Constants

**Files:**
- Create: `internal/canvas/layer.go`
- Create: `internal/canvas/layer_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/canvas/layer_test.go`:

```go
package canvas

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestLayerOrdering_BackgroundFirst(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(LayerBackground).To(BeNumerically("<", LayerStructure))
	g.Expect(LayerStructure).To(BeNumerically("<", LayerContent))
	g.Expect(LayerContent).To(BeNumerically("<", LayerOverlay))
}

func TestLayerOrdering_GapsExist(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(LayerStructure - LayerBackground).To(Equal(Layer(10)))
	g.Expect(LayerContent - LayerStructure).To(Equal(Layer(10)))
	g.Expect(LayerOverlay - LayerContent).To(Equal(Layer(10)))
}

func TestLayer_CustomValue_BetweenStandard(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	custom := Layer(15)
	g.Expect(custom).To(BeNumerically(">", LayerStructure))
	g.Expect(custom).To(BeNumerically("<", LayerContent))
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd internal/canvas && go test ./... -run TestLayerOrdering -v
```

Expected: FAIL — `Layer`, `LayerBackground`, etc. undefined.

- [ ] **Step 3: Write minimal implementation**

Create `internal/canvas/layer.go`:

```go
package canvas

// Layer controls the z-ordering of shapes on the canvas.
// Lower values are drawn first (behind higher values).
// The 10-unit gaps between constants leave room for future intermediate layers.
type Layer int

const (
	// LayerBackground is for canvas background fills.
	LayerBackground Layer = 0
	// LayerStructure is for edges, guide tracks, and directory borders.
	LayerStructure Layer = 10
	// LayerContent is for file rectangles, file discs, and data shapes.
	LayerContent Layer = 20
	// LayerOverlay is for labels, legends, and annotations.
	LayerOverlay Layer = 30
)
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd internal/canvas && go test ./... -run TestLayer -v
```

Expected: PASS — all 3 layer tests green.

- [ ] **Step 5: Commit**

```bash
git add internal/canvas/layer.go internal/canvas/layer_test.go
git commit -m "feat(canvas): add Layer type with z-ordering constants"
```

---

### Task 7: Spec Types (ShapeStyle, RectangleSpec, DiscSpec, LineSpec)

**Files:**
- Create: `internal/canvas/spec.go`

No separate tests — these are data structs tested through Canvas integration tests.

- [ ] **Step 1: Write the implementation**

Create `internal/canvas/spec.go`:

```go
package canvas

// ShapeStyle bundles the visual properties shared by all closed-shape specs.
type ShapeStyle struct {
	Fill        Ink
	Border      Ink
	BorderWidth float64
	ShowLabel   bool
	LabelInk    Ink
	LabelStyle  LabelStyle
}

// RectangleSpec defines the visual template for rectangles.
type RectangleSpec struct {
	ShapeStyle
}

// DiscSpec defines the visual template for circles/discs.
type DiscSpec struct {
	ShapeStyle
}

// LineSpec defines the visual template for lines.
type LineSpec struct {
	Stroke      Ink
	StrokeWidth float64
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd internal/canvas && go build ./...
```

Expected: BUILD OK.

- [ ] **Step 3: Commit**

```bash
git add internal/canvas/spec.go
git commit -m "feat(canvas): add ShapeStyle, RectangleSpec, DiscSpec, LineSpec"
```

---

### Task 8: TextSpec, TextAnchor, LabelStyle

**Files:**
- Create: `internal/canvas/text_spec.go`

- [ ] **Step 1: Write the implementation**

Create `internal/canvas/text_spec.go`:

```go
package canvas

// LabelStyle controls how labels are rendered on shapes.
type LabelStyle int

const (
	// LabelCentered places text centered inside the shape.
	LabelCentered LabelStyle = iota
	// LabelArc curves text along a circle boundary (used by bubble tree directories).
	LabelArc
	// LabelRadial places text outside the shape, rotated outward (used by radial/spiral).
	LabelRadial
)

// TextAnchor controls horizontal text alignment.
type TextAnchor int

const (
	// AnchorStart aligns text to the left.
	AnchorStart TextAnchor = iota
	// AnchorMiddle centers text horizontally.
	AnchorMiddle
	// AnchorEnd aligns text to the right.
	AnchorEnd
)

// TextSpec defines the visual template for standalone text.
// Font family is intentionally fixed (sans-serif for SVG, goregular for raster)
// and is not exposed as a configurable field.
type TextSpec struct {
	Ink      Ink
	FontSize float64
	Anchor   TextAnchor
	Rotation float64 // radians
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd internal/canvas && go build ./...
```

Expected: BUILD OK.

- [ ] **Step 3: Commit**

```bash
git add internal/canvas/text_spec.go
git commit -m "feat(canvas): add TextSpec, TextAnchor, LabelStyle types"
```

---

### Task 9: Shape Types

**Files:**
- Create: `internal/canvas/shape.go`

- [ ] **Step 1: Write the implementation**

Create `internal/canvas/shape.go`:

```go
package canvas

// Rectangle carries geometry and metric values for rectangular shapes.
type Rectangle struct {
	Spec        *RectangleSpec
	X, Y, W, H float64
	Fill        MetricValue
	Border      MetricValue
	Label       string
}

// Disc carries geometry and metric values for circular shapes.
type Disc struct {
	Spec   *DiscSpec
	X, Y   float64
	Radius float64
	Angle  float64 // angular position; used for radial/external label orientation
	Fill   MetricValue
	Border MetricValue
	Label  string
}

// Text carries position and content for standalone text.
type Text struct {
	Spec    *TextSpec
	X, Y    float64
	Content string
}

// Line carries start and end positions for line segments.
type Line struct {
	Spec   *LineSpec
	X1, Y1 float64
	X2, Y2 float64
}

// Path carries a sequence of positions for multi-point paths.
type Path struct {
	Spec   *LineSpec
	Points []Position
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd internal/canvas && go build ./...
```

Expected: BUILD OK.

- [ ] **Step 3: Commit**

```bash
git add internal/canvas/shape.go
git commit -m "feat(canvas): add Rectangle, Disc, Text, Line, Path shape types"
```

---

### Task 10: Backend Interface

**Files:**
- Create: `internal/canvas/backend.go`

- [ ] **Step 1: Write the implementation**

Create `internal/canvas/backend.go`:

```go
package canvas

import (
	"image/color"
)

// Backend is the rendering interface implemented by output format adapters.
// Methods receive resolved RGBA colours and primitive geometry —
// no Inks, Specs, or MetricValues.
type Backend interface {
	DrawRectangle(pos Position, size Size, fill, border color.RGBA, borderWidth float64)
	DrawDisc(center Position, radius float64, fill, border color.RGBA, borderWidth float64)
	DrawLine(from, to Position, stroke color.RGBA, strokeWidth float64)
	DrawPath(points []Position, stroke color.RGBA, strokeWidth float64)
	DrawText(pos Position, text string, ink color.RGBA, fontSize float64, anchor TextAnchor, rotation float64)
	DrawArcText(center Position, radius float64, text string, ink color.RGBA, fontSize float64)
	Finish(outputPath string) error
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd internal/canvas && go build ./...
```

Expected: BUILD OK.

- [ ] **Step 3: Commit**

```bash
git add internal/canvas/backend.go
git commit -m "feat(canvas): add Backend interface"
```

---

### Task 11: FormatFromPath Utility

**Files:**
- Create: `internal/canvas/format.go`
- Create: `internal/canvas/format_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/canvas/format_test.go`:

```go
package canvas

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestFormatFromPath_SupportedFormats(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		path     string
		expected ImageFormat
	}{
		{"png lowercase", "output.png", FormatPNG},
		{"png uppercase", "output.PNG", FormatPNG},
		{"png mixed case", "output.Png", FormatPNG},
		{"jpg lowercase", "output.jpg", FormatJPG},
		{"jpg uppercase", "output.JPG", FormatJPG},
		{"jpeg lowercase", "output.jpeg", FormatJPG},
		{"jpeg uppercase", "output.JPEG", FormatJPG},
		{"jpeg mixed case", "output.JpEg", FormatJPG},
		{"svg lowercase", "output.svg", FormatSVG},
		{"svg uppercase", "output.SVG", FormatSVG},
		{"svg mixed case", "output.Svg", FormatSVG},
		{"path with dirs", "results/my-chart.png", FormatPNG},
		{"deep path jpg", "a/b/c/chart.jpeg", FormatJPG},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			fmt, err := FormatFromPath(tc.path)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(fmt).To(Equal(tc.expected))
		})
	}
}

func TestFormatFromPath_Errors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		path        string
		errContains string
	}{
		{"no extension", "output", "no file extension"},
		{"unsupported bmp", "output.bmp", "unsupported image format"},
		{"unsupported gif", "output.gif", "unsupported image format"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			_, err := FormatFromPath(tc.path)
			g.Expect(err).To(HaveOccurred())
			g.Expect(err.Error()).To(ContainSubstring(tc.errContains))
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd internal/canvas && go test ./... -run TestFormatFromPath -v
```

Expected: FAIL — `ImageFormat`, `FormatFromPath`, etc. undefined.

- [ ] **Step 3: Write minimal implementation**

Create `internal/canvas/format.go`:

```go
package canvas

import (
	"path/filepath"
	"strings"

	"github.com/rotisserie/eris"
)

// ImageFormat represents a supported output image format.
type ImageFormat int

const (
	// FormatPNG is the PNG raster format.
	FormatPNG ImageFormat = iota
	// FormatJPG is the JPEG raster format.
	FormatJPG
	// FormatSVG is the SVG vector format.
	FormatSVG
)

// FormatFromPath determines the image format from the file extension.
// The match is case-insensitive. Both ".jpg" and ".jpeg" map to FormatJPG.
func FormatFromPath(path string) (ImageFormat, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".png":
		return FormatPNG, nil
	case ".jpg", ".jpeg":
		return FormatJPG, nil
	case ".svg":
		return FormatSVG, nil
	case "":
		return 0, eris.New("output path has no file extension; supported formats: png, jpg, jpeg, svg")
	default:
		return 0, eris.Errorf("unsupported image format %q; supported formats: png, jpg, jpeg, svg", ext)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd internal/canvas && go test ./... -run TestFormatFromPath -v
```

Expected: PASS — all format tests green.

- [ ] **Step 5: Commit**

```bash
git add internal/canvas/format.go internal/canvas/format_test.go
git commit -m "feat(canvas): add FormatFromPath utility"
```

---

### Task 12: TextColourFor Utility

**Files:**
- Create: `internal/canvas/text_colour.go`
- Create: `internal/canvas/text_colour_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/canvas/text_colour_test.go`:

```go
package canvas

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"
)

func TestTextColourFor_DarkOnLightFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	lightFill := color.RGBA{R: 240, G: 240, B: 240, A: 255}
	textCol := TextColourFor(lightFill)

	g.Expect(textCol.R).To(BeNumerically("<", 100))
	g.Expect(textCol.G).To(BeNumerically("<", 100))
	g.Expect(textCol.B).To(BeNumerically("<", 100))
}

func TestTextColourFor_LightOnDarkFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	darkFill := color.RGBA{R: 20, G: 20, B: 20, A: 255}
	textCol := TextColourFor(darkFill)

	g.Expect(textCol.R).To(BeNumerically(">", 150))
	g.Expect(textCol.G).To(BeNumerically(">", 150))
	g.Expect(textCol.B).To(BeNumerically(">", 150))
}

func TestTextColourFor_MidGrey_ReturnsDark(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Luminance of (200,200,200) is ~0.58, above 0.5 → dark text.
	midFill := color.RGBA{R: 200, G: 200, B: 200, A: 255}
	textCol := TextColourFor(midFill)

	g.Expect(textCol).To(Equal(color.RGBA{R: 0, G: 0, B: 0, A: 255}))
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd internal/canvas && go test ./... -run TestTextColourFor -v
```

Expected: FAIL — `TextColourFor` undefined.

- [ ] **Step 3: Write minimal implementation**

Create `internal/canvas/text_colour.go`:

```go
package canvas

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// TextColourFor returns black or white text depending on fill luminance.
// Uses WCAG 2.0 relative luminance with a 0.5 threshold.
func TextColourFor(fill color.RGBA) color.RGBA {
	lum := palette.RelativeLuminance(fill)
	if lum > 0.5 {
		return black
	}

	return white
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd internal/canvas && go test ./... -run TestTextColourFor -v
```

Expected: PASS — all 3 tests green.

- [ ] **Step 5: Commit**

```bash
git add internal/canvas/text_colour.go internal/canvas/text_colour_test.go
git commit -m "feat(canvas): add TextColourFor contrast utility"
```

---

### Task 13: Legend Types

**Files:**
- Create: `internal/canvas/legend.go`

No separate tests — legend types are data structs tested through Canvas integration tests in Stage 2.

- [ ] **Step 1: Write the implementation**

Create `internal/canvas/legend.go`:

```go
package canvas

// LegendPosition specifies where the legend is placed on the canvas.
type LegendPosition string

const (
	LegendPositionNone         LegendPosition = "none"
	LegendPositionTopLeft      LegendPosition = "top-left"
	LegendPositionTopCenter    LegendPosition = "top-center"
	LegendPositionTopRight     LegendPosition = "top-right"
	LegendPositionCenterRight  LegendPosition = "center-right"
	LegendPositionBottomRight  LegendPosition = "bottom-right"
	LegendPositionBottomCenter LegendPosition = "bottom-center"
	LegendPositionBottomLeft   LegendPosition = "bottom-left"
	LegendPositionCenterLeft   LegendPosition = "center-left"
)

// LegendOrientation controls whether swatches are stacked vertically
// or laid out horizontally.
type LegendOrientation string

const (
	LegendOrientationVertical   LegendOrientation = "vertical"
	LegendOrientationHorizontal LegendOrientation = "horizontal"
)

// LegendRole identifies what visual property a legend entry describes.
type LegendRole string

const (
	LegendRoleFill   LegendRole = "Fill"
	LegendRoleBorder LegendRole = "Border"
	LegendRoleSize   LegendRole = "Size"
)

// LegendEntry describes one metric shown in the legend.
type LegendEntry struct {
	Role       LegendRole
	MetricName string
	Ink        Ink
}

// LegendConfig holds everything needed to render a legend.
type LegendConfig struct {
	Position    LegendPosition
	Orientation LegendOrientation
	Entries     []LegendEntry
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd internal/canvas && go build ./...
```

Expected: BUILD OK.

- [ ] **Step 3: Commit**

```bash
git add internal/canvas/legend.go
git commit -m "feat(canvas): add LegendConfig and LegendEntry types"
```

---

### Task 14: Mock Backend (Test Only)

**Files:**
- Create: `internal/canvas/mock_backend_test.go`

This is test-only infrastructure used by Canvas tests in Task 15.

- [ ] **Step 1: Write the mock backend**

Create `internal/canvas/mock_backend_test.go`:

```go
package canvas

import (
	"image/color"
)

// drawCall records a single drawing operation dispatched to the mock backend.
type drawCall struct {
	method string
	pos    Position
	size   Size
	fill   color.RGBA
	border color.RGBA
	text   string
}

// mockBackend records all drawing calls for test assertions.
type mockBackend struct {
	calls      []drawCall
	finishPath string
	finishErr  error
}

func newMockBackend() *mockBackend {
	return &mockBackend{}
}

func (m *mockBackend) DrawRectangle(pos Position, size Size, fill, border color.RGBA, borderWidth float64) {
	m.calls = append(m.calls, drawCall{
		method: "DrawRectangle",
		pos:    pos,
		size:   size,
		fill:   fill,
		border: border,
	})
}

func (m *mockBackend) DrawDisc(center Position, radius float64, fill, border color.RGBA, borderWidth float64) {
	m.calls = append(m.calls, drawCall{
		method: "DrawDisc",
		pos:    center,
		fill:   fill,
		border: border,
	})
}

func (m *mockBackend) DrawLine(from, to Position, stroke color.RGBA, strokeWidth float64) {
	m.calls = append(m.calls, drawCall{
		method: "DrawLine",
		pos:    from,
	})
}

func (m *mockBackend) DrawPath(points []Position, stroke color.RGBA, strokeWidth float64) {
	m.calls = append(m.calls, drawCall{
		method: "DrawPath",
	})
}

func (m *mockBackend) DrawText(pos Position, text string, ink color.RGBA, fontSize float64, anchor TextAnchor, rotation float64) {
	m.calls = append(m.calls, drawCall{
		method: "DrawText",
		pos:    pos,
		text:   text,
		fill:   ink,
	})
}

func (m *mockBackend) DrawArcText(center Position, radius float64, text string, ink color.RGBA, fontSize float64) {
	m.calls = append(m.calls, drawCall{
		method: "DrawArcText",
		pos:    center,
		text:   text,
		fill:   ink,
	})
}

func (m *mockBackend) Finish(outputPath string) error {
	m.finishPath = outputPath

	return m.finishErr
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd internal/canvas && go test -c -o /dev/null ./... 2>&1
```

Expected: BUILD OK (test binary compiles).

- [ ] **Step 3: Commit**

```bash
git add internal/canvas/mock_backend_test.go
git commit -m "test(canvas): add mock backend for canvas unit tests"
```

---

### Task 15: Canvas Core (Add + RenderTo)

**Files:**
- Create: `internal/canvas/canvas.go`
- Create: `internal/canvas/canvas_test.go`

The Canvas stores shapes, sorts them by layer at render time, resolves Ink colours, and dispatches to a Backend. `Render()` selects the backend from the file extension. `RenderTo()` takes an explicit backend for testing.

- [ ] **Step 1: Write the failing tests**

Create `internal/canvas/canvas_test.go`:

```go
package canvas

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

func TestCanvas_AddRectangle_DispatchesToBackend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	red := color.RGBA{R: 255, A: 255}
	spec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:        FixedInk(red),
			Border:      FixedInk(black),
			BorderWidth: 2.0,
		},
	}

	c.AddRectangle(LayerContent, Rectangle{
		Spec: spec,
		X: 10, Y: 20, W: 100, H: 50,
	})

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(HaveLen(1))
	g.Expect(mb.calls[0].method).To(Equal("DrawRectangle"))
	g.Expect(mb.calls[0].fill).To(Equal(red))
}

func TestCanvas_AddDisc_DispatchesToBackend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	blue := color.RGBA{B: 255, A: 255}
	spec := &DiscSpec{
		ShapeStyle: ShapeStyle{
			Fill:        FixedInk(blue),
			Border:      FixedInk(black),
			BorderWidth: 1.0,
		},
	}

	c.AddDisc(LayerContent, Disc{
		Spec:   spec,
		X:      400,
		Y:      300,
		Radius: 50,
	})

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(HaveLen(1))
	g.Expect(mb.calls[0].method).To(Equal("DrawDisc"))
	g.Expect(mb.calls[0].fill).To(Equal(blue))
}

func TestCanvas_AddText_DispatchesToBackend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	spec := &TextSpec{
		Ink:      FixedInk(black),
		FontSize: 14,
		Anchor:   AnchorMiddle,
	}

	c.AddText(LayerOverlay, Text{
		Spec:    spec,
		X:       100,
		Y:       200,
		Content: "hello",
	})

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(HaveLen(1))
	g.Expect(mb.calls[0].method).To(Equal("DrawText"))
	g.Expect(mb.calls[0].text).To(Equal("hello"))
}

func TestCanvas_AddLine_DispatchesToBackend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	spec := &LineSpec{
		Stroke:      FixedInk(black),
		StrokeWidth: 1.0,
	}

	c.AddLine(LayerStructure, Line{
		Spec: spec,
		X1: 0, Y1: 0, X2: 100, Y2: 100,
	})

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(HaveLen(1))
	g.Expect(mb.calls[0].method).To(Equal("DrawLine"))
}

func TestCanvas_AddPath_DispatchesToBackend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	spec := &LineSpec{
		Stroke:      FixedInk(black),
		StrokeWidth: 2.0,
	}

	c.AddPath(LayerStructure, Path{
		Spec: spec,
		Points: []Position{
			{X: 0, Y: 0},
			{X: 50, Y: 50},
			{X: 100, Y: 0},
		},
	})

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(HaveLen(1))
	g.Expect(mb.calls[0].method).To(Equal("DrawPath"))
}

func TestCanvas_LayerOrdering_BackgroundBeforeContent(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	bgSpec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:   FixedInk(white),
			Border: FixedInk(white),
		},
	}

	fgSpec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:   FixedInk(black),
			Border: FixedInk(black),
		},
	}

	// Add content first, then background — layer ordering should override insertion order.
	c.AddRectangle(LayerContent, Rectangle{
		Spec: fgSpec,
		X: 0, Y: 0, W: 100, H: 100,
	})
	c.AddRectangle(LayerBackground, Rectangle{
		Spec: bgSpec,
		X: 0, Y: 0, W: 800, H: 600,
	})

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(HaveLen(2))
	g.Expect(mb.calls[0].fill).To(Equal(white))
	g.Expect(mb.calls[1].fill).To(Equal(black))
}

func TestCanvas_InsertionOrder_WithinSameLayer(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	red := color.RGBA{R: 255, A: 255}
	green := color.RGBA{G: 255, A: 255}

	spec1 := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:   FixedInk(red),
			Border: FixedInk(red),
		},
	}

	spec2 := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:   FixedInk(green),
			Border: FixedInk(green),
		},
	}

	c.AddRectangle(LayerContent, Rectangle{Spec: spec1, W: 100, H: 100})
	c.AddRectangle(LayerContent, Rectangle{Spec: spec2, W: 50, H: 50})

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(HaveLen(2))
	g.Expect(mb.calls[0].fill).To(Equal(red))
	g.Expect(mb.calls[1].fill).To(Equal(green))
}

func TestCanvas_InkResolution_NumericInk(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(400, 400)
	pal := palette.GetPalette(palette.Neutral)
	ink := NumericInk([]float64{10, 50, 90}, pal)

	spec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:   ink,
			Border: FixedInk(black),
		},
	}

	c.AddRectangle(LayerContent, Rectangle{
		Spec: spec,
		W:    100,
		H:    100,
		Fill: MeasureValue(10),
	})

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(HaveLen(1))
	g.Expect(mb.calls[0].fill.A).To(Equal(uint8(255)))
}

func TestCanvas_MultipleShapeTypes_MixedLayers(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)
	rectSpec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:   FixedInk(white),
			Border: FixedInk(black),
		},
	}

	lineSpec := &LineSpec{
		Stroke:      FixedInk(black),
		StrokeWidth: 1.0,
	}

	textSpec := &TextSpec{
		Ink:      FixedInk(black),
		FontSize: 12,
	}

	c.AddText(LayerOverlay, Text{Spec: textSpec, Content: "label"})
	c.AddLine(LayerStructure, Line{Spec: lineSpec, X2: 100, Y2: 100})
	c.AddRectangle(LayerBackground, Rectangle{Spec: rectSpec, W: 800, H: 600})

	mb := newMockBackend()
	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(HaveLen(3))
	g.Expect(mb.calls[0].method).To(Equal("DrawRectangle"))
	g.Expect(mb.calls[1].method).To(Equal("DrawLine"))
	g.Expect(mb.calls[2].method).To(Equal("DrawText"))
}

func TestCanvas_Empty_NoErrors(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(100, 100)
	mb := newMockBackend()

	err := c.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.calls).To(BeEmpty())
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd internal/canvas && go test ./... -run TestCanvas -v
```

Expected: FAIL — `Canvas`, `NewCanvas`, `RenderTo` undefined.

- [ ] **Step 3: Write minimal implementation**

Create `internal/canvas/canvas.go`:

```go
package canvas

import (
	"slices"

	"github.com/rotisserie/eris"
)

// shapeKind tags the type of shape stored in a layered entry.
type shapeKind int

const (
	shapeRectangle shapeKind = iota
	shapeDisc
	shapeText
	shapeLine
	shapePath
)

// layeredShape holds a shape with its assigned layer and insertion order.
type layeredShape struct {
	layer Layer
	order int
	kind  shapeKind
	rect  *Rectangle
	disc  *Disc
	text  *Text
	line  *Line
	path  *Path
}

// Canvas is a retained-then-render drawing surface.
// Shapes are added with layer assignments, then rendered in batch.
type Canvas struct {
	width  int
	height int
	shapes []layeredShape
	legend *LegendConfig
}

// NewCanvas creates a canvas for the given dimensions.
func NewCanvas(width, height int) *Canvas {
	return &Canvas{
		width:  width,
		height: height,
	}
}

// AddRectangle records a rectangle on the given layer.
func (c *Canvas) AddRectangle(layer Layer, r Rectangle) {
	c.shapes = append(c.shapes, layeredShape{
		layer: layer,
		order: len(c.shapes),
		kind:  shapeRectangle,
		rect:  &r,
	})
}

// AddDisc records a disc on the given layer.
func (c *Canvas) AddDisc(layer Layer, d Disc) {
	c.shapes = append(c.shapes, layeredShape{
		layer: layer,
		order: len(c.shapes),
		kind:  shapeDisc,
		disc:  &d,
	})
}

// AddText records text on the given layer.
func (c *Canvas) AddText(layer Layer, t Text) {
	c.shapes = append(c.shapes, layeredShape{
		layer: layer,
		order: len(c.shapes),
		kind:  shapeText,
		text:  &t,
	})
}

// AddLine records a line on the given layer.
func (c *Canvas) AddLine(layer Layer, l Line) {
	c.shapes = append(c.shapes, layeredShape{
		layer: layer,
		order: len(c.shapes),
		kind:  shapeLine,
		line:  &l,
	})
}

// AddPath records a path on the given layer.
func (c *Canvas) AddPath(layer Layer, p Path) {
	c.shapes = append(c.shapes, layeredShape{
		layer: layer,
		order: len(c.shapes),
		kind:  shapePath,
		path:  &p,
	})
}

// SetLegend configures the legend for this canvas.
func (c *Canvas) SetLegend(config LegendConfig) {
	c.legend = &config
}

// Render resolves all inks, sorts shapes by layer, selects the backend
// from the file extension, and writes the output.
func (c *Canvas) Render(outputPath string) error {
	format, err := FormatFromPath(outputPath)
	if err != nil {
		return err
	}

	backend, err := c.createBackend(format)
	if err != nil {
		return err
	}

	if err := c.RenderTo(backend); err != nil {
		return err
	}

	return backend.Finish(outputPath)
}

// RenderTo dispatches all shapes to the given backend, sorted by layer.
// This method is the primary test seam — tests inject a mock backend.
func (c *Canvas) RenderTo(backend Backend) error {
	sorted := make([]layeredShape, len(c.shapes))
	copy(sorted, c.shapes)

	slices.SortStableFunc(sorted, func(a, b layeredShape) int {
		if a.layer != b.layer {
			return int(a.layer - b.layer)
		}

		return a.order - b.order
	})

	for _, s := range sorted {
		c.dispatchShape(backend, s)
	}

	return nil
}

func (c *Canvas) dispatchShape(backend Backend, s layeredShape) {
	switch s.kind {
	case shapeRectangle:
		c.drawRectangle(backend, s.rect)
	case shapeDisc:
		c.drawDisc(backend, s.disc)
	case shapeText:
		c.drawText(backend, s.text)
	case shapeLine:
		c.drawLine(backend, s.line)
	case shapePath:
		c.drawPath(backend, s.path)
	}
}

func (c *Canvas) drawRectangle(b Backend, r *Rectangle) {
	fill := r.Spec.Fill.Dip(r.Fill)
	border := r.Spec.Border.Dip(r.Border)

	b.DrawRectangle(
		Position{X: r.X, Y: r.Y},
		Size{Width: r.W, Height: r.H},
		fill, border,
		r.Spec.BorderWidth,
	)
}

func (c *Canvas) drawDisc(b Backend, d *Disc) {
	fill := d.Spec.Fill.Dip(d.Fill)
	border := d.Spec.Border.Dip(d.Border)

	b.DrawDisc(
		Position{X: d.X, Y: d.Y},
		d.Radius,
		fill, border,
		d.Spec.BorderWidth,
	)
}

func (c *Canvas) drawText(b Backend, t *Text) {
	ink := t.Spec.Ink.Dip(MetricValue{})

	b.DrawText(
		Position{X: t.X, Y: t.Y},
		t.Content, ink,
		t.Spec.FontSize,
		t.Spec.Anchor,
		t.Spec.Rotation,
	)
}

func (c *Canvas) drawLine(b Backend, l *Line) {
	stroke := l.Spec.Stroke.Dip(MetricValue{})

	b.DrawLine(
		Position{X: l.X1, Y: l.Y1},
		Position{X: l.X2, Y: l.Y2},
		stroke,
		l.Spec.StrokeWidth,
	)
}

func (c *Canvas) drawPath(b Backend, p *Path) {
	stroke := p.Spec.Stroke.Dip(MetricValue{})

	b.DrawPath(p.Points, stroke, p.Spec.StrokeWidth)
}

// createBackend creates the appropriate backend for the given format.
// Backend subpackages are imported and instantiated here.
func (c *Canvas) createBackend(format ImageFormat) (Backend, error) {
	switch format {
	case FormatPNG, FormatJPG:
		return nil, eris.New("raster backend not yet available")
	case FormatSVG:
		return nil, eris.New("SVG backend not yet available")
	default:
		return nil, eris.Errorf("unsupported format: %d", format)
	}
}
```

Note: `createBackend` returns stub errors for now. Tasks 17 and 18 implement the real backends and wire them in.

- [ ] **Step 4: Run test to verify it passes**

```bash
cd internal/canvas && go test ./... -v
```

Expected: PASS — all Canvas tests green (they use `RenderTo` with the mock backend, bypassing `createBackend`).

- [ ] **Step 5: Commit**

```bash
git add internal/canvas/canvas.go internal/canvas/canvas_test.go
git commit -m "feat(canvas): add Canvas with Add*, RenderTo, and layer sorting"
```

---

### Task 16: Raster Backend (internal/canvas/raster/)

**Files:**
- Create: `internal/canvas/raster/backend.go`
- Create: `internal/canvas/raster/backend_test.go`

- [ ] **Step 1: Create the raster subpackage directory**

```bash
mkdir -p internal/canvas/raster
```

- [ ] **Step 2: Write the failing tests**

Create `internal/canvas/raster/backend_test.go`:

```go
package raster

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
)

func TestRasterBackend_DrawRectangle_ProducesValidPNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 200)
	red := colRGBA(255, 0, 0, 255)
	black := colRGBA(0, 0, 0, 255)

	b.DrawRectangle(
		canvas.Position{X: 10, Y: 10},
		canvas.Size{Width: 80, Height: 60},
		red, black, 2.0,
	)

	out := filepath.Join(t.TempDir(), "rect.png")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	img := loadImage(t, out)
	g.Expect(img.Bounds().Dx()).To(Equal(200))
	g.Expect(img.Bounds().Dy()).To(Equal(200))
}

func TestRasterBackend_DrawDisc_ProducesValidPNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 200)
	blue := colRGBA(0, 0, 255, 255)
	black := colRGBA(0, 0, 0, 255)

	b.DrawDisc(
		canvas.Position{X: 100, Y: 100},
		50, blue, black, 1.0,
	)

	out := filepath.Join(t.TempDir(), "disc.png")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	img := loadImage(t, out)
	g.Expect(img.Bounds().Dx()).To(Equal(200))
}

func TestRasterBackend_DrawText_ProducesValidPNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 100)
	black := colRGBA(0, 0, 0, 255)

	b.DrawText(
		canvas.Position{X: 100, Y: 50},
		"hello", black, 14.0,
		canvas.AnchorMiddle, 0,
	)

	out := filepath.Join(t.TempDir(), "text.png")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info.Size()).To(BeNumerically(">", 0))
}

func TestRasterBackend_DrawLine_ProducesValidPNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 200)
	black := colRGBA(0, 0, 0, 255)

	b.DrawLine(
		canvas.Position{X: 0, Y: 0},
		canvas.Position{X: 200, Y: 200},
		black, 2.0,
	)

	out := filepath.Join(t.TempDir(), "line.png")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRasterBackend_DrawPath_ProducesValidPNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 200)
	black := colRGBA(0, 0, 0, 255)

	b.DrawPath(
		[]canvas.Position{
			{X: 10, Y: 10},
			{X: 100, Y: 50},
			{X: 190, Y: 10},
		},
		black, 1.0,
	)

	out := filepath.Join(t.TempDir(), "path.png")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRasterBackend_Finish_JPG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(100, 100)
	out := filepath.Join(t.TempDir(), "test.jpg")

	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info.Size()).To(BeNumerically(">", 0))
}

func TestRasterBackend_Finish_UnsupportedFormat(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(100, 100)
	out := filepath.Join(t.TempDir(), "test.bmp")

	err := b.Finish(out)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("unsupported"))
}

func TestRasterBackend_ImplementsBackendInterface(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var b canvas.Backend = New(100, 100)
	g.Expect(b).NotTo(BeNil())
}

// colRGBA is a helper to construct color.RGBA inline.
func colRGBA(r, g, b, a uint8) [4]uint8 {
	// Return is destructured at call site — not used directly.
	// Actually, let's just use image/color directly.
	panic("unused")
}

func loadImage(t *testing.T, path string) image.Image {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatal(err)
	}

	return img
}
```

Wait — the `colRGBA` helper is wrong. Let me fix the test file. The tests should use `image/color` directly.

Actually, let me rewrite the test file properly:

Create `internal/canvas/raster/backend_test.go`:

```go
package raster

import (
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
)

func TestRasterBackend_DrawRectangle_ProducesValidPNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 200)
	red := color.RGBA{R: 255, A: 255}
	blk := color.RGBA{A: 255}

	b.DrawRectangle(
		canvas.Position{X: 10, Y: 10},
		canvas.Size{Width: 80, Height: 60},
		red, blk, 2.0,
	)

	out := filepath.Join(t.TempDir(), "rect.png")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	img := loadImage(t, out)
	g.Expect(img.Bounds().Dx()).To(Equal(200))
	g.Expect(img.Bounds().Dy()).To(Equal(200))
}

func TestRasterBackend_DrawDisc_ProducesValidPNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 200)
	blue := color.RGBA{B: 255, A: 255}
	blk := color.RGBA{A: 255}

	b.DrawDisc(
		canvas.Position{X: 100, Y: 100},
		50, blue, blk, 1.0,
	)

	out := filepath.Join(t.TempDir(), "disc.png")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	img := loadImage(t, out)
	g.Expect(img.Bounds().Dx()).To(Equal(200))
}

func TestRasterBackend_DrawText_ProducesValidPNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 100)
	blk := color.RGBA{A: 255}

	b.DrawText(
		canvas.Position{X: 100, Y: 50},
		"hello", blk, 14.0,
		canvas.AnchorMiddle, 0,
	)

	out := filepath.Join(t.TempDir(), "text.png")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info.Size()).To(BeNumerically(">", 0))
}

func TestRasterBackend_DrawLine_ProducesValidPNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 200)
	blk := color.RGBA{A: 255}

	b.DrawLine(
		canvas.Position{X: 0, Y: 0},
		canvas.Position{X: 200, Y: 200},
		blk, 2.0,
	)

	out := filepath.Join(t.TempDir(), "line.png")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRasterBackend_DrawPath_ProducesValidPNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 200)
	blk := color.RGBA{A: 255}

	b.DrawPath(
		[]canvas.Position{
			{X: 10, Y: 10},
			{X: 100, Y: 50},
			{X: 190, Y: 10},
		},
		blk, 1.0,
	)

	out := filepath.Join(t.TempDir(), "path.png")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRasterBackend_Finish_JPG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(100, 100)
	out := filepath.Join(t.TempDir(), "test.jpg")

	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info.Size()).To(BeNumerically(">", 0))
}

func TestRasterBackend_Finish_UnsupportedFormat(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(100, 100)
	out := filepath.Join(t.TempDir(), "test.bmp")

	err := b.Finish(out)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("unsupported"))
}

func TestRasterBackend_ImplementsBackendInterface(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var b canvas.Backend = New(100, 100)
	g.Expect(b).NotTo(BeNil())
}

func loadImage(t *testing.T, path string) image.Image {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatal(err)
	}

	return img
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd internal/canvas/raster && go test ./... -v
```

Expected: FAIL — `raster` package and `New` function don't exist.

- [ ] **Step 4: Write minimal implementation**

Create `internal/canvas/raster/backend.go`:

```go
// Package raster implements the canvas.Backend interface for raster
// output formats (PNG, JPG) using the fogleman/gg graphics library.
package raster

import (
	"image/color"
	"image/jpeg"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/fogleman/gg"
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
)

const jpegQuality = 95

type rasterBackend struct {
	dc *gg.Context
}

// New creates a raster backend with the given dimensions.
func New(width, height int) canvas.Backend {
	dc := gg.NewContext(width, height)

	return &rasterBackend{dc: dc}
}

func (r *rasterBackend) DrawRectangle(pos canvas.Position, size canvas.Size, fill, border color.RGBA, borderWidth float64) {
	r.dc.SetColor(fill)
	r.dc.DrawRectangle(pos.X, pos.Y, size.Width, size.Height)
	r.dc.Fill()

	if borderWidth > 0 {
		r.dc.SetColor(border)
		r.dc.SetLineWidth(borderWidth)
		r.dc.DrawRectangle(pos.X, pos.Y, size.Width, size.Height)
		r.dc.Stroke()
	}
}

func (r *rasterBackend) DrawDisc(center canvas.Position, radius float64, fill, border color.RGBA, borderWidth float64) {
	r.dc.SetColor(fill)
	r.dc.DrawCircle(center.X, center.Y, radius)
	r.dc.Fill()

	if borderWidth > 0 {
		r.dc.SetColor(border)
		r.dc.SetLineWidth(borderWidth)
		r.dc.DrawCircle(center.X, center.Y, radius)
		r.dc.Stroke()
	}
}

func (r *rasterBackend) DrawLine(from, to canvas.Position, stroke color.RGBA, strokeWidth float64) {
	r.dc.SetColor(stroke)
	r.dc.SetLineWidth(strokeWidth)
	r.dc.DrawLine(from.X, from.Y, to.X, to.Y)
	r.dc.Stroke()
}

func (r *rasterBackend) DrawPath(points []canvas.Position, stroke color.RGBA, strokeWidth float64) {
	if len(points) < 2 {
		return
	}

	r.dc.SetColor(stroke)
	r.dc.SetLineWidth(strokeWidth)
	r.dc.MoveTo(points[0].X, points[0].Y)

	for _, p := range points[1:] {
		r.dc.LineTo(p.X, p.Y)
	}

	r.dc.Stroke()
}

func (r *rasterBackend) DrawText(
	pos canvas.Position,
	text string,
	ink color.RGBA,
	fontSize float64,
	anchor canvas.TextAnchor,
	rotation float64,
) {
	r.dc.SetColor(ink)

	ax := anchorX(anchor)

	if rotation != 0 {
		r.dc.RotateAbout(rotation, pos.X, pos.Y)
	}

	r.dc.DrawStringAnchored(text, pos.X, pos.Y, ax, 0.5)

	if rotation != 0 {
		r.dc.RotateAbout(-rotation, pos.X, pos.Y)
	}
}

func (r *rasterBackend) DrawArcText(
	center canvas.Position,
	radius float64,
	text string,
	ink color.RGBA,
	fontSize float64,
) {
	if text == "" || radius <= 0 {
		return
	}

	r.dc.SetColor(ink)
	arcRadius := radius - 14.0
	totalAngle := float64(len([]rune(text))) * fontSize * 0.6 / arcRadius
	startAngle := -math.Pi/2.0 - totalAngle/2.0
	charAngle := totalAngle / float64(len([]rune(text)))

	for i, ch := range text {
		angle := startAngle + float64(i)*charAngle + charAngle/2.0
		cx := center.X + arcRadius*math.Cos(angle)
		cy := center.Y + arcRadius*math.Sin(angle)

		r.dc.Push()
		r.dc.RotateAbout(angle+math.Pi/2.0, cx, cy)
		r.dc.DrawStringAnchored(string(ch), cx, cy, 0.5, 0.5)
		r.dc.Pop()
	}
}

func (r *rasterBackend) Finish(outputPath string) error {
	ext := strings.ToLower(filepath.Ext(outputPath))

	switch ext {
	case ".png":
		return eris.Wrap(r.dc.SavePNG(outputPath), "failed to save PNG")
	case ".jpg", ".jpeg":
		return r.saveJPG(outputPath)
	default:
		return eris.Errorf("unsupported raster format %q", ext)
	}
}

func (r *rasterBackend) saveJPG(path string) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return eris.Wrap(err, "failed to create JPEG file")
	}

	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = eris.Wrap(closeErr, "failed to close JPEG file")
		}
	}()

	if err := jpeg.Encode(f, r.dc.Image(), &jpeg.Options{Quality: jpegQuality}); err != nil {
		return eris.Wrap(err, "failed to encode JPEG")
	}

	return nil
}

func anchorX(a canvas.TextAnchor) float64 {
	switch a {
	case canvas.AnchorStart:
		return 0.0
	case canvas.AnchorMiddle:
		return 0.5
	case canvas.AnchorEnd:
		return 1.0
	default:
		return 0.0
	}
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd internal/canvas/raster && go test ./... -v
```

Expected: PASS — all raster backend tests green.

- [ ] **Step 6: Commit**

```bash
git add internal/canvas/raster/
git commit -m "feat(canvas): add raster backend (PNG/JPG via fogleman/gg)"
```

---

### Task 17: SVG Backend (internal/canvas/svg/)

**Files:**
- Create: `internal/canvas/svg/backend.go`
- Create: `internal/canvas/svg/backend_test.go`

- [ ] **Step 1: Create the SVG subpackage directory**

```bash
mkdir -p internal/canvas/svg
```

- [ ] **Step 2: Write the failing tests**

Create `internal/canvas/svg/backend_test.go`:

```go
package svg

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
)

func TestSVGBackend_DrawRectangle_ProducesValidSVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 200)
	red := color.RGBA{R: 255, A: 255}
	blk := color.RGBA{A: 255}

	b.DrawRectangle(
		canvas.Position{X: 10, Y: 10},
		canvas.Size{Width: 80, Height: 60},
		red, blk, 2.0,
	)

	out := filepath.Join(t.TempDir(), "rect.svg")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	content := readFile(t, out)
	g.Expect(content).To(ContainSubstring("<svg"))
	g.Expect(content).To(ContainSubstring("<rect"))
	g.Expect(content).To(ContainSubstring("</svg>"))
}

func TestSVGBackend_DrawDisc_ProducesValidSVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 200)
	blue := color.RGBA{B: 255, A: 255}
	blk := color.RGBA{A: 255}

	b.DrawDisc(
		canvas.Position{X: 100, Y: 100},
		50, blue, blk, 1.0,
	)

	out := filepath.Join(t.TempDir(), "disc.svg")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	content := readFile(t, out)
	g.Expect(content).To(ContainSubstring("<circle"))
}

func TestSVGBackend_DrawText_ProducesValidSVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 100)
	blk := color.RGBA{A: 255}

	b.DrawText(
		canvas.Position{X: 100, Y: 50},
		"hello", blk, 14.0,
		canvas.AnchorMiddle, 0,
	)

	out := filepath.Join(t.TempDir(), "text.svg")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	content := readFile(t, out)
	g.Expect(content).To(ContainSubstring("<text"))
	g.Expect(content).To(ContainSubstring("hello"))
}

func TestSVGBackend_DrawLine_ProducesValidSVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 200)
	blk := color.RGBA{A: 255}

	b.DrawLine(
		canvas.Position{X: 0, Y: 0},
		canvas.Position{X: 200, Y: 200},
		blk, 2.0,
	)

	out := filepath.Join(t.TempDir(), "line.svg")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	content := readFile(t, out)
	g.Expect(content).To(ContainSubstring("<line"))
}

func TestSVGBackend_DrawPath_ProducesValidSVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(200, 200)
	blk := color.RGBA{A: 255}

	b.DrawPath(
		[]canvas.Position{
			{X: 10, Y: 10},
			{X: 100, Y: 50},
			{X: 190, Y: 10},
		},
		blk, 1.0,
	)

	out := filepath.Join(t.TempDir(), "path.svg")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	content := readFile(t, out)
	g.Expect(content).To(ContainSubstring("<path"))
}

func TestSVGBackend_DrawArcText_ProducesValidSVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(400, 400)
	blk := color.RGBA{A: 255}

	b.DrawArcText(
		canvas.Position{X: 200, Y: 200},
		100, "hello", blk, 14.0,
	)

	out := filepath.Join(t.TempDir(), "arctext.svg")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	content := readFile(t, out)
	g.Expect(content).To(ContainSubstring("<textPath"))
}

func TestSVGBackend_ImplementsBackendInterface(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var b canvas.Backend = New(100, 100)
	g.Expect(b).NotTo(BeNil())
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd internal/canvas/svg && go test ./... -v
```

Expected: FAIL — `svg` package and `New` function don't exist.

- [ ] **Step 4: Write minimal implementation**

Create `internal/canvas/svg/backend.go`:

```go
// Package svg implements the canvas.Backend interface for SVG vector output
// using direct XML generation.
package svg

import (
	"bytes"
	"fmt"
	"html"
	"image/color"
	"math"
	"os"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
)

type svgBackend struct {
	width  int
	height int
	buf    bytes.Buffer
}

// New creates an SVG backend with the given dimensions.
func New(width, height int) canvas.Backend {
	b := &svgBackend{width: width, height: height}
	b.writeHeader()

	return b
}

func (s *svgBackend) writeHeader() {
	fmt.Fprintf(&s.buf,
		`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d">`+"\n",
		s.width, s.height,
	)
}

func (s *svgBackend) DrawRectangle(pos canvas.Position, size canvas.Size, fill, border color.RGBA, borderWidth float64) {
	fmt.Fprintf(&s.buf,
		`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s" stroke="%s" stroke-width="%.1f"/>`+"\n",
		pos.X, pos.Y, size.Width, size.Height,
		rgbaToCSS(fill), rgbaToCSS(border), borderWidth,
	)
}

func (s *svgBackend) DrawDisc(center canvas.Position, radius float64, fill, border color.RGBA, borderWidth float64) {
	fmt.Fprintf(&s.buf,
		`<circle cx="%.2f" cy="%.2f" r="%.2f" fill="%s" stroke="%s" stroke-width="%.1f"/>`+"\n",
		center.X, center.Y, radius,
		rgbaToCSS(fill), rgbaToCSS(border), borderWidth,
	)
}

func (s *svgBackend) DrawLine(from, to canvas.Position, stroke color.RGBA, strokeWidth float64) {
	fmt.Fprintf(&s.buf,
		`<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="%s" stroke-width="%.1f"/>`+"\n",
		from.X, from.Y, to.X, to.Y,
		rgbaToCSS(stroke), strokeWidth,
	)
}

func (s *svgBackend) DrawPath(points []canvas.Position, stroke color.RGBA, strokeWidth float64) {
	if len(points) < 2 {
		return
	}

	d := fmt.Sprintf("M%.2f %.2f", points[0].X, points[0].Y)
	for _, p := range points[1:] {
		d += fmt.Sprintf(" L%.2f %.2f", p.X, p.Y)
	}

	fmt.Fprintf(&s.buf,
		`<path d="%s" fill="none" stroke="%s" stroke-width="%.1f"/>`+"\n",
		d, rgbaToCSS(stroke), strokeWidth,
	)
}

func (s *svgBackend) DrawText(
	pos canvas.Position,
	text string,
	ink color.RGBA,
	fontSize float64,
	anchor canvas.TextAnchor,
	rotation float64,
) {
	anchorStr := svgAnchor(anchor)
	escaped := html.EscapeString(text)

	if rotation != 0 {
		deg := rotation * 180.0 / math.Pi

		fmt.Fprintf(&s.buf,
			`<text x="%.2f" y="%.2f" fill="%s" font-size="%.1f" font-family="sans-serif" `+
				`text-anchor="%s" dominant-baseline="central" `+
				`transform="rotate(%.2f %.2f %.2f)">%s</text>`+"\n",
			pos.X, pos.Y, rgbaToCSS(ink), fontSize,
			anchorStr, deg, pos.X, pos.Y, escaped,
		)

		return
	}

	fmt.Fprintf(&s.buf,
		`<text x="%.2f" y="%.2f" fill="%s" font-size="%.1f" font-family="sans-serif" `+
			`text-anchor="%s" dominant-baseline="central">%s</text>`+"\n",
		pos.X, pos.Y, rgbaToCSS(ink), fontSize, anchorStr, escaped,
	)
}

func (s *svgBackend) DrawArcText(
	center canvas.Position,
	radius float64,
	text string,
	ink color.RGBA,
	fontSize float64,
) {
	if text == "" || radius <= 0 {
		return
	}

	arcR := radius - 14.0
	pathID := fmt.Sprintf("arc-%d", s.buf.Len())

	fmt.Fprintf(&s.buf,
		`<defs><path id="%s" d="M%.2f,%.2f A%.2f,%.2f 0 1,1 %.2f,%.2f" fill="none"/></defs>`+"\n",
		pathID,
		center.X, center.Y-arcR,
		arcR, arcR,
		center.X-0.01, center.Y-arcR,
	)

	fmt.Fprintf(&s.buf,
		`<text fill="%s" font-size="%.1f" font-family="sans-serif">`+
			`<textPath href="#%s" startOffset="50%%" text-anchor="middle">%s</textPath></text>`+"\n",
		rgbaToCSS(ink), fontSize, pathID, html.EscapeString(text),
	)
}

func (s *svgBackend) Finish(outputPath string) (err error) {
	s.buf.WriteString("</svg>\n")

	f, err := os.Create(outputPath)
	if err != nil {
		return eris.Wrap(err, "failed to create SVG file")
	}

	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = eris.Wrap(closeErr, "failed to close SVG file")
		}
	}()

	if _, err := f.Write(s.buf.Bytes()); err != nil {
		return eris.Wrap(err, "failed to write SVG")
	}

	return nil
}

func rgbaToCSS(c color.RGBA) string {
	if c.A == 255 {
		return fmt.Sprintf("rgb(%d,%d,%d)", c.R, c.G, c.B)
	}

	return fmt.Sprintf("rgba(%d,%d,%d,%.3f)", c.R, c.G, c.B, float64(c.A)/255.0)
}

func svgAnchor(a canvas.TextAnchor) string {
	switch a {
	case canvas.AnchorStart:
		return "start"
	case canvas.AnchorMiddle:
		return "middle"
	case canvas.AnchorEnd:
		return "end"
	default:
		return "start"
	}
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd internal/canvas/svg && go test ./... -v
```

Expected: PASS — all SVG backend tests green.

- [ ] **Step 6: Commit**

```bash
git add internal/canvas/svg/
git commit -m "feat(canvas): add SVG backend"
```

---

### Task 18: Wire Backends into Canvas.Render()

**Files:**
- Modify: `internal/canvas/canvas.go` — update `createBackend()` to use real backends

- [ ] **Step 1: Write the failing test**

Add to `internal/canvas/canvas_test.go`:

```go
func TestCanvas_Render_PNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(200, 200)
	spec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:   FixedInk(white),
			Border: FixedInk(black),
		},
	}

	c.AddRectangle(LayerBackground, Rectangle{
		Spec: spec,
		W:    200,
		H:    200,
	})

	out := filepath.Join(t.TempDir(), "output.png")
	err := c.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	info, statErr := os.Stat(out)
	g.Expect(statErr).NotTo(HaveOccurred())
	g.Expect(info.Size()).To(BeNumerically(">", 0))
}

func TestCanvas_Render_SVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(200, 200)
	spec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:   FixedInk(white),
			Border: FixedInk(black),
		},
	}

	c.AddRectangle(LayerBackground, Rectangle{
		Spec: spec,
		W:    200,
		H:    200,
	})

	out := filepath.Join(t.TempDir(), "output.svg")
	err := c.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	data, readErr := os.ReadFile(out)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("<svg"))
}

func TestCanvas_Render_JPG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(200, 200)
	spec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:   FixedInk(white),
			Border: FixedInk(black),
		},
	}

	c.AddRectangle(LayerBackground, Rectangle{
		Spec: spec,
		W:    200,
		H:    200,
	})

	out := filepath.Join(t.TempDir(), "output.jpg")
	err := c.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	info, statErr := os.Stat(out)
	g.Expect(statErr).NotTo(HaveOccurred())
	g.Expect(info.Size()).To(BeNumerically(">", 0))
}

func TestCanvas_Render_UnsupportedFormat(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(100, 100)
	err := c.Render("output.bmp")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("unsupported"))
}
```

Update the import block at the top of `canvas_test.go` to add `os` and `path/filepath`:

```go
import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd internal/canvas && go test ./... -run TestCanvas_Render -v
```

Expected: FAIL — `createBackend` returns stub errors for PNG/SVG.

- [ ] **Step 3: Update createBackend to wire real backends**

In `internal/canvas/canvas.go`, update the import block to add the backend packages:

```go
import (
	"slices"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/raster"
	svgbackend "github.com/theunrepentantgeek/code-visualizer/internal/canvas/svg"
)
```

Replace the `createBackend` function body:

```go
func (c *Canvas) createBackend(format ImageFormat) (Backend, error) {
	switch format {
	case FormatPNG, FormatJPG:
		return raster.New(c.width, c.height), nil
	case FormatSVG:
		return svgbackend.New(c.width, c.height), nil
	default:
		return nil, eris.Errorf("unsupported format: %d", format)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd internal/canvas && go test ./... -v
```

Expected: PASS — all Canvas tests green including `Render_PNG`, `Render_SVG`, `Render_JPG`.

- [ ] **Step 5: Commit**

```bash
git add internal/canvas/canvas.go internal/canvas/canvas_test.go
git commit -m "feat(canvas): wire raster and SVG backends into Canvas.Render()"
```

---

### Task 19: Full Integration Test — End-to-End Render

**Files:**
- Modify: `internal/canvas/canvas_test.go` — add end-to-end test with all shape types

This test exercises the full pipeline: multiple shape types across multiple layers, rendered through a real backend.

- [ ] **Step 1: Write the integration test**

Add to `internal/canvas/canvas_test.go`:

```go
func TestCanvas_Integration_AllShapeTypes_PNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)

	bgSpec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:   FixedInk(white),
			Border: FixedInk(white),
		},
	}

	c.AddRectangle(LayerBackground, Rectangle{
		Spec: bgSpec,
		W:    800, H: 600,
	})

	lineSpec := &LineSpec{
		Stroke:      FixedInk(color.RGBA{R: 200, G: 200, B: 200, A: 255}),
		StrokeWidth: 1.0,
	}

	c.AddLine(LayerStructure, Line{
		Spec: lineSpec,
		X1: 0, Y1: 300, X2: 800, Y2: 300,
	})

	pal := palette.GetPalette(palette.Temperature)
	fillInk := NumericInk([]float64{10, 20, 30, 40, 50}, pal)

	rectSpec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:        fillInk,
			Border:      FixedInk(black),
			BorderWidth: 1.0,
		},
	}

	c.AddRectangle(LayerContent, Rectangle{
		Spec: rectSpec,
		X: 50, Y: 50, W: 200, H: 150,
		Fill: MeasureValue(10),
	})

	c.AddRectangle(LayerContent, Rectangle{
		Spec: rectSpec,
		X: 300, Y: 50, W: 200, H: 150,
		Fill: MeasureValue(50),
	})

	discSpec := &DiscSpec{
		ShapeStyle: ShapeStyle{
			Fill:        FixedInk(color.RGBA{R: 100, G: 200, B: 100, A: 255}),
			Border:      FixedInk(black),
			BorderWidth: 1.0,
		},
	}

	c.AddDisc(LayerContent, Disc{
		Spec: discSpec,
		X: 650, Y: 125, Radius: 60,
	})

	textSpec := &TextSpec{
		Ink:      FixedInk(black),
		FontSize: 14,
		Anchor:   AnchorMiddle,
	}

	c.AddText(LayerOverlay, Text{
		Spec:    textSpec,
		X:       400,
		Y:       500,
		Content: "Canvas Integration Test",
	})

	pathSpec := &LineSpec{
		Stroke:      FixedInk(color.RGBA{R: 255, G: 100, B: 100, A: 255}),
		StrokeWidth: 2.0,
	}

	c.AddPath(LayerStructure, Path{
		Spec: pathSpec,
		Points: []Position{
			{X: 50, Y: 400},
			{X: 200, Y: 350},
			{X: 400, Y: 450},
			{X: 600, Y: 380},
			{X: 750, Y: 420},
		},
	})

	out := filepath.Join(t.TempDir(), "integration.png")
	err := c.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	info, statErr := os.Stat(out)
	g.Expect(statErr).NotTo(HaveOccurred())
	g.Expect(info.Size()).To(BeNumerically(">", 1000))
}

func TestCanvas_Integration_AllShapeTypes_SVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(800, 600)

	bgSpec := &RectangleSpec{
		ShapeStyle: ShapeStyle{
			Fill:   FixedInk(white),
			Border: FixedInk(white),
		},
	}

	c.AddRectangle(LayerBackground, Rectangle{
		Spec: bgSpec,
		W:    800, H: 600,
	})

	discSpec := &DiscSpec{
		ShapeStyle: ShapeStyle{
			Fill:        FixedInk(color.RGBA{R: 100, B: 200, A: 255}),
			Border:      FixedInk(black),
			BorderWidth: 2.0,
		},
	}

	c.AddDisc(LayerContent, Disc{
		Spec: discSpec,
		X: 400, Y: 300, Radius: 100,
	})

	textSpec := &TextSpec{
		Ink:      FixedInk(black),
		FontSize: 16,
		Anchor:   AnchorMiddle,
	}

	c.AddText(LayerOverlay, Text{
		Spec: textSpec,
		X: 400, Y: 300, Content: "SVG Test",
	})

	out := filepath.Join(t.TempDir(), "integration.svg")
	err := c.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	data, readErr := os.ReadFile(out)
	g.Expect(readErr).NotTo(HaveOccurred())

	content := string(data)
	g.Expect(content).To(ContainSubstring("<svg"))
	g.Expect(content).To(ContainSubstring("<rect"))
	g.Expect(content).To(ContainSubstring("<circle"))
	g.Expect(content).To(ContainSubstring("<text"))
	g.Expect(content).To(ContainSubstring("SVG Test"))
	g.Expect(content).To(ContainSubstring("</svg>"))
}
```

- [ ] **Step 2: Run tests to verify they pass**

```bash
cd internal/canvas && go test ./... -v
```

Expected: PASS — all integration tests green.

- [ ] **Step 3: Run the full project test suite to verify no regressions**

```bash
go test ./... -count=1
```

Expected: PASS — all existing tests still green, plus all new canvas tests.

- [ ] **Step 4: Commit**

```bash
git add internal/canvas/canvas_test.go
git commit -m "test(canvas): add end-to-end integration tests for PNG and SVG"
```

---

### Task 20: Build Verification and Cleanup

Final verification that the complete canvas package compiles, passes all tests, and existing code is unmodified.

- [ ] **Step 1: Verify the full project builds**

```bash
go build ./...
```

Expected: BUILD OK — zero errors.

- [ ] **Step 2: Run the full test suite**

```bash
go test ./... -count=1
```

Expected: PASS — all packages green including `internal/canvas`, `internal/canvas/raster`, `internal/canvas/svg`.

- [ ] **Step 3: Run go vet**

```bash
go vet ./...
```

Expected: No issues.

- [ ] **Step 4: Verify no existing files were modified**

```bash
git diff --name-only HEAD -- internal/render/ internal/palette/ internal/metric/ cmd/codeviz/
```

Expected: No output — zero changes to existing code.

- [ ] **Step 5: Verify file structure**

```bash
find internal/canvas -type f | sort
```

Expected output:
```
internal/canvas/backend.go
internal/canvas/canvas.go
internal/canvas/canvas_test.go
internal/canvas/format.go
internal/canvas/format_test.go
internal/canvas/geometry.go
internal/canvas/ink.go
internal/canvas/ink_introspection.go
internal/canvas/ink_introspection_test.go
internal/canvas/ink_options.go
internal/canvas/ink_test.go
internal/canvas/layer.go
internal/canvas/layer_test.go
internal/canvas/legend.go
internal/canvas/metric_value.go
internal/canvas/metric_value_test.go
internal/canvas/mock_backend_test.go
internal/canvas/raster/backend.go
internal/canvas/raster/backend_test.go
internal/canvas/shape.go
internal/canvas/spec.go
internal/canvas/svg/backend.go
internal/canvas/svg/backend_test.go
internal/canvas/text_colour.go
internal/canvas/text_colour_test.go
internal/canvas/text_spec.go
```

- [ ] **Step 6: Final commit (if any fixups needed)**

```bash
git status
```

If clean, no commit needed. If there are any leftover changes, commit with:

```bash
git add -A
git commit -m "chore(canvas): final Stage 1 cleanup"
```
