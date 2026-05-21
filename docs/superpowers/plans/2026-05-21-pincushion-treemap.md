# Pincushion Treemap Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add radial gradient "pincushion" shading to treemap file tiles, with weighted focus offset toward the parent directory centre.

**Architecture:** Introduce a `model.Fill` interface (replacing `color.RGBA` in backend signatures) with `SolidFill` and `RadialGradientFill` implementations. Convert `canvas.Ink` from a struct to an interface. Add `RadialGradientInk` wrapper that produces `RadialGradientFill` values. The treemap renderer computes a per-tile focus point and passes it through the new `Fill()` method.

**Tech Stack:** Go 1.26+, fogleman/gg (raster rendering), Gomega (test assertions), Kong (CLI parsing)

---

## File Structure

| File | Responsibility |
|------|---------------|
| `internal/canvas/model/fill.go` | New — `Fill` interface, `SolidFill`, `RadialGradientFill`, `Point` types |
| `internal/canvas/model/backend.go` | Modified — `DrawRectangle` and `DrawDisc` fill/border params become `Fill` |
| `internal/canvas/ink.go` | Major refactor — struct → interface + concrete implementations |
| `internal/canvas/ink_introspection.go` | Modified — methods move onto concrete types |
| `internal/canvas/ink_options.go` | Minor — unchanged (options configure concrete ink types) |
| `internal/canvas/radial_gradient_ink.go` | New — `RadialGradientInk` wrapper |
| `internal/canvas/shape.go` | Modified — `Rectangle` gains `Focus` field; `drawTo` uses `Fill()` |
| `internal/canvas/mock_backend_test.go` | Modified — signature update |
| `internal/canvas/raster/backend.go` | Modified — gradient rendering for `RadialGradientFill` |
| `internal/canvas/svg/backend.go` | Modified — `<radialGradient>` SVG emission |
| `internal/treemap/render.go` | Modified — focus calculation per-tile |
| `internal/treemap/inks.go` | Modified — `Inks` struct uses `canvas.Ink` interface |
| `cmd/codeviz/treemap_cmd.go` | Modified — `--flat` flag |
| `internal/treemap/state.go` | Modified — `Flat` field on State |
| `internal/treemap/stages.go` | Modified — pass Flat through BuildInksStage |

---

### Task 1: model.Fill Interface

**Files:**
- Create: `internal/canvas/model/fill.go`
- Test: `internal/canvas/model/fill_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/canvas/model/fill_test.go`:

```go
package model_test

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
)

func TestSolidFill_ImplementsFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var fill model.Fill = model.SolidFill{Color: color.RGBA{R: 255, A: 255}}
	g.Expect(fill).NotTo(BeNil())
}

func TestRadialGradientFill_ImplementsFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var fill model.Fill = model.RadialGradientFill{
		Center: color.RGBA{R: 255, G: 255, B: 255, A: 255},
		Edge:   color.RGBA{R: 100, G: 100, B: 100, A: 255},
		Focus:  model.Point{X: 0.5, Y: 0.5},
	}
	g.Expect(fill).NotTo(BeNil())
}

func TestPoint_Zero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	p := model.Point{}
	g.Expect(p.X).To(Equal(0.0))
	g.Expect(p.Y).To(Equal(0.0))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/bevan/github/code-visualizer && go test ./internal/canvas/model/ -run TestSolidFill -v`
Expected: FAIL — types not defined.

- [ ] **Step 3: Write the implementation**

Create `internal/canvas/model/fill.go`:

```go
package model

import "image/color"

// Point represents a 2D coordinate as fractions (may exceed [0,1]).
type Point struct {
	X, Y float64
}

// Fill is a sealed interface describing how a shape's interior is painted.
type Fill interface {
	isFill()
}

// SolidFill paints a uniform colour.
type SolidFill struct {
	Color color.RGBA
}

// RadialGradientFill paints a radial gradient from a centre colour
// (at the focus point) to an edge colour (at the shape boundary).
type RadialGradientFill struct {
	Center color.RGBA
	Edge   color.RGBA
	Focus  Point
}

func (SolidFill) isFill()            {}
func (RadialGradientFill) isFill()   {}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/bevan/github/code-visualizer && go test ./internal/canvas/model/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/canvas/model/fill.go internal/canvas/model/fill_test.go
git commit -m "feat(model): add Fill interface with SolidFill and RadialGradientFill

Part of #263 — pincushion treemap effect."
```

---

### Task 2: Update Backend Interface Signature

**Files:**
- Modify: `internal/canvas/model/backend.go`
- Modify: `internal/canvas/raster/backend.go`
- Modify: `internal/canvas/svg/backend.go`
- Modify: `internal/canvas/mock_backend_test.go`
- Modify: `internal/canvas/shape.go`

- [ ] **Step 1: Update the Backend interface**

In `internal/canvas/model/backend.go`, change `DrawRectangle` and `DrawDisc` signatures from `fill, border color.RGBA` to `fill, border Fill`:

```go
type Backend interface {
	DrawRectangle(pos Position, size Size, fill, border Fill, borderWidth float64)
	DrawDisc(center Position, radius float64, fill, border Fill, borderWidth float64)
	DrawLine(from, to Position, stroke color.RGBA, strokeWidth float64)
	DrawPath(points []Position, stroke color.RGBA, strokeWidth float64)
	DrawText(pos Position, text string, ink color.RGBA, fontSize float64, anchor TextAnchor, rotation float64)
	DrawArcText(center Position, radius float64, text string, ink color.RGBA, fontSize float64)
	Finish(outputPath string) error
}
```

Remove the `"image/color"` import if it becomes unused (it won't — still needed by DrawLine, DrawText, DrawArcText).

- [ ] **Step 2: Update the raster backend**

In `internal/canvas/raster/backend.go`, change `DrawRectangle` and `DrawDisc` to accept `model.Fill`. For now, type-assert to `model.SolidFill` to keep existing behaviour working while we build out gradient rendering later:

```go
func (r *rasterBackend) DrawRectangle(
	pos model.Position, size model.Size, fill, border model.Fill, borderWidth float64,
) {
	fillColour := solidColor(fill)
	borderColour := solidColor(border)

	r.dc.SetColor(nrgba(fillColour))
	r.dc.DrawRectangle(pos.X, pos.Y, size.Width, size.Height)
	r.dc.Fill()

	if borderWidth > 0 {
		r.dc.SetColor(nrgba(borderColour))
		r.dc.SetLineWidth(borderWidth)
		r.dc.DrawRectangle(pos.X, pos.Y, size.Width, size.Height)
		r.dc.Stroke()
	}
}

func (r *rasterBackend) DrawDisc(
	center model.Position, radius float64, fill, border model.Fill, borderWidth float64,
) {
	fillColour := solidColor(fill)
	borderColour := solidColor(border)

	r.dc.SetColor(nrgba(fillColour))
	r.dc.DrawCircle(center.X, center.Y, radius)
	r.dc.Fill()

	if borderWidth > 0 {
		r.dc.SetColor(nrgba(borderColour))
		r.dc.SetLineWidth(borderWidth)
		r.dc.DrawCircle(center.X, center.Y, radius)
		r.dc.Stroke()
	}
}

// solidColor extracts the colour from a Fill, falling back to opaque black.
func solidColor(f model.Fill) color.RGBA {
	switch v := f.(type) {
	case model.SolidFill:
		return v.Color
	case model.RadialGradientFill:
		return v.Center // temporary: use center colour until gradient rendering is implemented
	default:
		return color.RGBA{A: 255}
	}
}
```

- [ ] **Step 3: Update the SVG backend**

In `internal/canvas/svg/backend.go`, apply the same pattern — extract solid colour for now:

```go
func (s *svgBackend) DrawRectangle(
	pos model.Position, size model.Size, fill, border model.Fill, borderWidth float64,
) {
	fillColour := solidColor(fill)
	borderColour := solidColor(border)

	fmt.Fprintf(
		&s.buf,
		`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s" stroke="%s" stroke-width="%.1f"/>`+"\n",
		pos.X, pos.Y, size.Width, size.Height,
		rgbaToCSS(fillColour), rgbaToCSS(borderColour), borderWidth,
	)
}

func (s *svgBackend) DrawDisc(
	center model.Position, radius float64, fill, border model.Fill, borderWidth float64,
) {
	fillColour := solidColor(fill)
	borderColour := solidColor(border)

	fmt.Fprintf(
		&s.buf,
		`<circle cx="%.2f" cy="%.2f" r="%.2f" fill="%s" stroke="%s" stroke-width="%.1f"/>`+"\n",
		center.X, center.Y, radius,
		rgbaToCSS(fillColour), rgbaToCSS(borderColour), borderWidth,
	)
}

func solidColor(f model.Fill) color.RGBA {
	switch v := f.(type) {
	case model.SolidFill:
		return v.Color
	case model.RadialGradientFill:
		return v.Center
	default:
		return color.RGBA{A: 255}
	}
}
```

- [ ] **Step 4: Update the mock backend**

In `internal/canvas/mock_backend_test.go`:

```go
func (m *mockBackend) DrawRectangle(pos Position, size Size, fill, border model.Fill, _ float64) {
	m.calls = append(m.calls, drawCall{
		method: "DrawRectangle",
		pos:    pos,
		size:   size,
		fill:   solidColorTest(fill),
		border: solidColorTest(border),
	})
}

func (m *mockBackend) DrawDisc(center Position, _ float64, fill, border model.Fill, _ float64) {
	m.calls = append(m.calls, drawCall{
		method: "DrawDisc",
		pos:    center,
		fill:   solidColorTest(fill),
		border: solidColorTest(border),
	})
}

func solidColorTest(f model.Fill) color.RGBA {
	switch v := f.(type) {
	case model.SolidFill:
		return v.Color
	case model.RadialGradientFill:
		return v.Center
	default:
		return color.RGBA{A: 255}
	}
}
```

Add the `model` import: `"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"`

- [ ] **Step 5: Update shape.go drawTo methods**

In `internal/canvas/shape.go`, change `Rectangle.drawTo` and `Disc.drawTo` to wrap their resolved colours in `model.SolidFill`:

```go
func (r *Rectangle) drawTo(b Backend) {
	fill := model.SolidFill{Color: r.Spec.Fill.Dip(r.Fill)}
	border := model.SolidFill{Color: r.Spec.Border.Dip(r.Border)}

	b.DrawRectangle(
		Position{X: r.X, Y: r.Y},
		Size{Width: r.W, Height: r.H},
		fill, border,
		r.Spec.BorderWidth,
	)
}

func (d *Disc) drawTo(b Backend) {
	fill := model.SolidFill{Color: d.Spec.Fill.Dip(d.Fill)}
	border := model.SolidFill{Color: d.Spec.Border.Dip(d.Border)}

	b.DrawDisc(
		Position{X: d.X, Y: d.Y},
		d.Radius,
		fill, border,
		d.Spec.BorderWidth,
	)
}
```

Add the `model` import to `shape.go`:
```go
"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
```

- [ ] **Step 6: Run all tests to verify no regression**

Run: `cd /home/bevan/github/code-visualizer && go test ./internal/canvas/... ./internal/treemap/... -v`
Expected: all existing tests PASS (behaviour unchanged — all fills are solid)

- [ ] **Step 7: Commit**

```bash
git add internal/canvas/model/backend.go internal/canvas/raster/backend.go \
  internal/canvas/svg/backend.go internal/canvas/mock_backend_test.go \
  internal/canvas/shape.go
git commit -m "refactor(backend): change DrawRectangle/DrawDisc to accept model.Fill

Backend methods now take the Fill interface instead of color.RGBA for
fill and border parameters. All callers pass SolidFill for now.

Part of #263 — pincushion treemap effect."
```

---

### Task 3: Convert Ink From Struct to Interface

**Files:**
- Modify: `internal/canvas/ink.go`
- Modify: `internal/canvas/ink_introspection.go`
- Modify: `internal/canvas/ink_options.go`
- Modify: `internal/canvas/spec.go`
- Modify: `internal/canvas/shape.go`

- [ ] **Step 1: Define the Ink interface and rename the struct**

In `internal/canvas/ink.go`, rename the existing `Ink` struct to `baseInk` (unexported) and define an `Ink` interface:

```go
// Ink resolves metric values to colours and fill specifications.
type Ink interface {
	Dip(value MetricValue) color.RGBA
	Fill(value MetricValue, focus model.Point) model.Fill
	Info() InkInfo
	legendEntryKind() model.LegendEntryKind
	legendSwatches() []model.LegendSwatch
}
```

Rename:
- `type Ink struct` → `type baseInk struct`
- All methods `func (ink Ink)` → `func (ink *baseInk)`
- All constructors return `Ink` (the interface) but construct `*baseInk` internally

- [ ] **Step 2: Add Fill() method to baseInk**

```go
func (ink *baseInk) Fill(value MetricValue, _ model.Point) model.Fill {
	return model.SolidFill{Color: ink.Dip(value)}
}
```

The baseInk always returns a SolidFill, ignoring the focus point.

- [ ] **Step 3: Update constructors to return Ink interface**

```go
func FixedInk(c color.RGBA, opts ...InkOption) Ink {
	cfg := defaultInkConfig()
	for _, o := range opts {
		o(&cfg)
	}

	return &baseInk{
		kind:    inkFixed,
		color:   c,
		opacity: cfg.opacity,
	}
}

func NumericInk(name metric.Name, values []float64, pal palette.ColourPalette, opts ...InkOption) Ink {
	cfg := defaultInkConfig()
	for _, o := range opts {
		o(&cfg)
	}

	buckets := metric.ComputeBuckets(values, len(pal.Colours))

	return &baseInk{
		kind:       inkNumeric,
		metricName: name,
		boundaries: &buckets,
		pal:        pal,
		opacity:    cfg.opacity,
	}
}

func CategoricalInk(name metric.Name, categories []string, pal palette.ColourPalette, opts ...InkOption) Ink {
	cfg := defaultInkConfig()
	for _, o := range opts {
		o(&cfg)
	}

	return &baseInk{
		kind:       inkCategorical,
		metricName: name,
		catMapper:  palette.NewCategoricalMapper(categories, pal),
		pal:        pal,
		categories: categories,
		opacity:    cfg.opacity,
	}
}
```

- [ ] **Step 4: Update ink_introspection.go**

Change all methods from `func (ink Ink)` to `func (ink *baseInk)`:

```go
func (ink *baseInk) Info() InkInfo {
	return InkInfo{
		Kind:       InkKind(ink.kind),
		MetricName: ink.metricName,
	}
}

func (ink *baseInk) Boundaries() []float64 {
	if ink.kind != inkNumeric || ink.boundaries == nil {
		return nil
	}

	return ink.boundaries.Boundaries
}

func (ink *baseInk) Palette() palette.ColourPalette {
	if ink.kind == inkFixed {
		return palette.ColourPalette{}
	}

	return ink.pal
}

func (ink *baseInk) Categories() []string {
	if ink.kind != inkCategorical {
		return nil
	}

	return ink.categories
}
```

Note: `Boundaries()`, `Palette()`, `Categories()` are NOT on the `Ink` interface — they are specific to `baseInk`. If callers need them, they type-assert. Check if the legend code needs access via a separate interface or direct assertion.

- [ ] **Step 5: Update shape.go to use Ink.Fill()**

Change `Rectangle.drawTo` to call `Fill()`:

```go
func (r *Rectangle) drawTo(b Backend) {
	fill := r.Spec.Fill.Fill(r.Fill, r.Focus)
	border := model.SolidFill{Color: r.Spec.Border.Dip(r.Border)}

	b.DrawRectangle(
		Position{X: r.X, Y: r.Y},
		Size{Width: r.W, Height: r.H},
		fill, border,
		r.Spec.BorderWidth,
	)
}
```

Add a `Focus model.Point` field to `Rectangle`:

```go
type Rectangle struct {
	Spec        *RectangleSpec
	X, Y, W, H float64
	Fill        MetricValue
	Border      MetricValue
	Focus       model.Point
}
```

- [ ] **Step 6: Fix compilation errors across the codebase**

Search for all uses of the old `Ink` struct fields and method calls. Key places:
- `internal/canvas/legend.go` / `legend_render.go` — uses `ink.Info()`, `ink.legendSwatches()` — these now go through the interface.
- `internal/treemap/inks.go` — `Inks` struct has `Fill canvas.Ink` and `Border canvas.Ink` — already uses the type name; it just becomes an interface.
- `internal/inks/` package — builds Ink instances; constructors already return the right type.
- Any test file doing `ink.Boundaries()` etc — will need type assertion to `*baseInk` or we add a `BoundaryInk` sub-interface.

If `Boundaries()`, `Palette()`, `Categories()` are called from legend building, add an `IntrospectableInk` interface:

```go
// IntrospectableInk provides detailed ink metadata for legend rendering.
type IntrospectableInk interface {
	Ink
	Boundaries() []float64
	Palette() palette.ColourPalette
	Categories() []string
}
```

- [ ] **Step 7: Run all tests**

Run: `cd /home/bevan/github/code-visualizer && go test ./... 2>&1 | head -50`
Expected: All tests PASS. Fix any remaining compilation errors.

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "refactor(canvas): convert Ink from struct to interface

Ink is now an interface with Dip(), Fill(), Info() and legend methods.
Existing ink kinds (fixed, numeric, categorical) implement it via baseInk.
Rectangle gains a Focus field for gradient rendering.

Part of #263 — pincushion treemap effect."
```

---

### Task 4: RadialGradientInk

**Files:**
- Create: `internal/canvas/radial_gradient_ink.go`
- Create: `internal/canvas/radial_gradient_ink_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/canvas/radial_gradient_ink_test.go`:

```go
package canvas

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
)

func TestRadialGradientInk_Dip_DelegatesToInner(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	red := color.RGBA{R: 255, A: 255}
	inner := FixedInk(red)
	gradient := NewRadialGradientInk(inner)

	result := gradient.Dip(MetricValue{})
	g.Expect(result).To(Equal(red))
}

func TestRadialGradientInk_Fill_ReturnsRadialGradientFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	inner := FixedInk(white)
	gradient := NewRadialGradientInk(inner)

	focus := model.Point{X: 0.35, Y: 0.35}
	fill := gradient.Fill(MetricValue{}, focus)

	rgf, ok := fill.(model.RadialGradientFill)
	g.Expect(ok).To(BeTrue())
	g.Expect(rgf.Center).To(Equal(white))
	g.Expect(rgf.Focus).To(Equal(focus))
	// Edge should be darker than centre
	g.Expect(rgf.Edge.R).To(BeNumerically("<", rgf.Center.R))
	g.Expect(rgf.Edge.G).To(BeNumerically("<", rgf.Center.G))
	g.Expect(rgf.Edge.B).To(BeNumerically("<", rgf.Center.B))
	g.Expect(rgf.Edge.A).To(Equal(uint8(255)))
}

func TestRadialGradientInk_Fill_DarkensBy40Percent(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	base := color.RGBA{R: 200, G: 100, B: 50, A: 255}
	inner := FixedInk(base)
	gradient := NewRadialGradientInk(inner)

	fill := gradient.Fill(MetricValue{}, model.Point{X: 0.5, Y: 0.5})
	rgf := fill.(model.RadialGradientFill)

	// 40% darker: channel * 0.6
	g.Expect(rgf.Edge.R).To(Equal(uint8(120))) // 200 * 0.6 = 120
	g.Expect(rgf.Edge.G).To(Equal(uint8(60)))  // 100 * 0.6 = 60
	g.Expect(rgf.Edge.B).To(Equal(uint8(30)))  // 50 * 0.6 = 30
}

func TestRadialGradientInk_Info_DelegatesToInner(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	inner := FixedInk(color.RGBA{A: 255})
	gradient := NewRadialGradientInk(inner)

	info := gradient.Info()
	g.Expect(info.Kind).To(Equal(InkFixed))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/bevan/github/code-visualizer && go test ./internal/canvas/ -run TestRadialGradientInk -v`
Expected: FAIL — `NewRadialGradientInk` not defined.

- [ ] **Step 3: Write the implementation**

Create `internal/canvas/radial_gradient_ink.go`:

```go
package canvas

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
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

func (g *RadialGradientInk) Info() InkInfo {
	return g.inner.Info()
}

func (g *RadialGradientInk) legendEntryKind() model.LegendEntryKind {
	return g.inner.legendEntryKind()
}

func (g *RadialGradientInk) legendSwatches() []model.LegendSwatch {
	return g.inner.legendSwatches()
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

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/bevan/github/code-visualizer && go test ./internal/canvas/ -run TestRadialGradientInk -v`
Expected: PASS

- [ ] **Step 5: Run all canvas tests**

Run: `cd /home/bevan/github/code-visualizer && go test ./internal/canvas/... -v`
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add internal/canvas/radial_gradient_ink.go internal/canvas/radial_gradient_ink_test.go
git commit -m "feat(canvas): add RadialGradientInk wrapper

Wraps any Ink to produce RadialGradientFill values with 40% edge
darkening and a configurable focus point.

Part of #263 — pincushion treemap effect."
```

---

### Task 5: Raster Backend Gradient Rendering

**Files:**
- Modify: `internal/canvas/raster/backend.go`
- Create: `internal/canvas/raster/backend_test.go` (if not exists, or extend)

- [ ] **Step 1: Write the failing test**

Create or extend `internal/canvas/raster/gradient_test.go`:

```go
package raster_test

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/raster"
)

func TestRasterBackend_DrawRectangle_WithRadialGradientFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	backend := raster.New(200, 200)

	fill := model.RadialGradientFill{
		Center: color.RGBA{R: 255, G: 255, B: 255, A: 255},
		Edge:   color.RGBA{R: 100, G: 100, B: 100, A: 255},
		Focus:  model.Point{X: 0.5, Y: 0.5},
	}
	border := model.SolidFill{Color: color.RGBA{A: 255}}

	backend.DrawRectangle(
		model.Position{X: 10, Y: 10},
		model.Size{Width: 180, Height: 180},
		fill, border, 1.0,
	)

	tmp := filepath.Join(t.TempDir(), "gradient.png")
	err := backend.Finish(tmp)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(tmp)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info.Size()).To(BeNumerically(">", 0))
}
```

- [ ] **Step 2: Run test to verify it passes with placeholder (centre colour only)**

Run: `cd /home/bevan/github/code-visualizer && go test ./internal/canvas/raster/ -run TestRasterBackend_DrawRectangle_WithRadialGradientFill -v`
Expected: PASS (but no actual gradient — just solid centre colour from the placeholder code in Task 2)

- [ ] **Step 3: Implement gradient rendering in raster backend**

In `internal/canvas/raster/backend.go`, replace the placeholder `solidColor` fallback in `DrawRectangle` with actual gradient rendering:

```go
func (r *rasterBackend) DrawRectangle(
	pos model.Position, size model.Size, fill, border model.Fill, borderWidth float64,
) {
	switch f := fill.(type) {
	case model.SolidFill:
		r.dc.SetColor(nrgba(f.Color))
		r.dc.DrawRectangle(pos.X, pos.Y, size.Width, size.Height)
		r.dc.Fill()
	case model.RadialGradientFill:
		r.drawRadialGradientRect(pos, size, f)
	default:
		r.dc.SetColor(nrgba(color.RGBA{A: 255}))
		r.dc.DrawRectangle(pos.X, pos.Y, size.Width, size.Height)
		r.dc.Fill()
	}

	if borderWidth > 0 {
		borderColour := solidColor(border)
		r.dc.SetColor(nrgba(borderColour))
		r.dc.SetLineWidth(borderWidth)
		r.dc.DrawRectangle(pos.X, pos.Y, size.Width, size.Height)
		r.dc.Stroke()
	}
}

func (r *rasterBackend) drawRadialGradientRect(
	pos model.Position, size model.Size, grad model.RadialGradientFill,
) {
	// Focus in absolute coordinates
	fx := pos.X + grad.Focus.X*size.Width
	fy := pos.Y + grad.Focus.Y*size.Height

	// Maximum distance from focus to any corner determines the gradient radius
	maxDist := maxCornerDist(fx, fy, pos.X, pos.Y, size.Width, size.Height)

	// Number of steps scales with size (diminishing returns beyond ~30)
	steps := int(min(max(size.Width, size.Height)/4, 30))
	if steps < 4 {
		steps = 4
	}

	// Draw from outermost to innermost (painter's algorithm)
	for i := range steps {
		t := float64(steps-1-i) / float64(steps-1) // 1.0 (outer) → 0.0 (inner)
		dist := maxDist * t

		colour := lerpColour(grad.Center, grad.Edge, t)
		r.dc.SetColor(nrgba(colour))

		// Inset rectangle clipped to the original bounds
		inset := dist * 0.7 // scale factor to approximate radial falloff in a rect
		x := max(pos.X, fx-inset)
		y := max(pos.Y, fy-inset)
		x2 := min(pos.X+size.Width, fx+inset)
		y2 := min(pos.Y+size.Height, fy+inset)

		if x2 > x && y2 > y {
			r.dc.DrawRectangle(x, y, x2-x, y2-y)
			r.dc.Fill()
		}
	}
}

func maxCornerDist(fx, fy, rx, ry, w, h float64) float64 {
	corners := [4][2]float64{
		{rx, ry}, {rx + w, ry}, {rx, ry + h}, {rx + w, ry + h},
	}

	maxD := 0.0
	for _, c := range corners {
		dx := c[0] - fx
		dy := c[1] - fy
		d := math.Sqrt(dx*dx + dy*dy)
		if d > maxD {
			maxD = d
		}
	}

	return maxD
}

func lerpColour(a, b color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		A: uint8(float64(a.A) + (float64(b.A)-float64(a.A))*t),
	}
}
```

Add `"math"` import if not already present.

- [ ] **Step 4: Run the gradient test again**

Run: `cd /home/bevan/github/code-visualizer && go test ./internal/canvas/raster/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/canvas/raster/
git commit -m "feat(raster): implement radial gradient rendering for DrawRectangle

Draws concentric inset rectangles fading from edge colour to centre
colour, offset toward the configured focus point.

Part of #263 — pincushion treemap effect."
```

---

### Task 6: SVG Backend Gradient Rendering

**Files:**
- Modify: `internal/canvas/svg/backend.go`
- Create: `internal/canvas/svg/backend_test.go` (or extend)

- [ ] **Step 1: Write the failing test**

Create `internal/canvas/svg/gradient_test.go`:

```go
package svg_test

import (
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	svgbackend "github.com/theunrepentantgeek/code-visualizer/internal/canvas/svg"
)

func TestSVGBackend_DrawRectangle_WithRadialGradientFill_EmitsGradient(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	backend := svgbackend.New(200, 200)

	fill := model.RadialGradientFill{
		Center: color.RGBA{R: 255, G: 255, B: 255, A: 255},
		Edge:   color.RGBA{R: 100, G: 100, B: 100, A: 255},
		Focus:  model.Point{X: 0.35, Y: 0.35},
	}
	border := model.SolidFill{Color: color.RGBA{A: 255}}

	backend.DrawRectangle(
		model.Position{X: 10, Y: 10},
		model.Size{Width: 180, Height: 180},
		fill, border, 1.0,
	)

	tmp := filepath.Join(t.TempDir(), "gradient.svg")
	err := backend.Finish(tmp)
	g.Expect(err).NotTo(HaveOccurred())

	data, err := os.ReadFile(tmp)
	g.Expect(err).NotTo(HaveOccurred())

	svg := string(data)
	g.Expect(svg).To(ContainSubstring("<radialGradient"))
	g.Expect(svg).To(ContainSubstring("fx="))
	g.Expect(svg).To(ContainSubstring("fy="))
	g.Expect(svg).To(ContainSubstring(`fill="url(#`))
	// Verify stops exist
	g.Expect(strings.Count(svg, "<stop")).To(BeNumerically(">=", 2))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/bevan/github/code-visualizer && go test ./internal/canvas/svg/ -run TestSVGBackend_DrawRectangle_WithRadialGradientFill -v`
Expected: FAIL — no `<radialGradient>` in output (placeholder returns solid).

- [ ] **Step 3: Implement gradient rendering in SVG backend**

In `internal/canvas/svg/backend.go`, add a gradient counter and update `DrawRectangle`:

Add a field to `svgBackend`:
```go
type svgBackend struct {
	width    int
	height   int
	buf      bytes.Buffer
	gradID   int
}
```

Update `DrawRectangle`:

```go
func (s *svgBackend) DrawRectangle(
	pos model.Position, size model.Size, fill, border model.Fill, borderWidth float64,
) {
	var fillAttr string

	switch f := fill.(type) {
	case model.SolidFill:
		fillAttr = rgbaToCSS(f.Color)
	case model.RadialGradientFill:
		id := s.emitRadialGradient(f)
		fillAttr = fmt.Sprintf("url(#%s)", id)
	default:
		fillAttr = "rgb(0,0,0)"
	}

	borderColour := solidColor(border)

	fmt.Fprintf(
		&s.buf,
		`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s" stroke="%s" stroke-width="%.1f"/>`+"\n",
		pos.X, pos.Y, size.Width, size.Height,
		fillAttr, rgbaToCSS(borderColour), borderWidth,
	)
}

func (s *svgBackend) emitRadialGradient(grad model.RadialGradientFill) string {
	s.gradID++
	id := fmt.Sprintf("rg%d", s.gradID)

	fmt.Fprintf(
		&s.buf,
		`<defs><radialGradient id="%s" cx="50%%" cy="50%%" r="70%%" fx="%.1f%%" fy="%.1f%%">`+
			`<stop offset="0%%" stop-color="%s"/>`+
			`<stop offset="100%%" stop-color="%s"/>`+
			`</radialGradient></defs>`+"\n",
		id,
		grad.Focus.X*100, grad.Focus.Y*100,
		rgbaToCSS(grad.Center), rgbaToCSS(grad.Edge),
	)

	return id
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/bevan/github/code-visualizer && go test ./internal/canvas/svg/ -run TestSVGBackend_DrawRectangle_WithRadialGradientFill -v`
Expected: PASS

- [ ] **Step 5: Run all SVG tests**

Run: `cd /home/bevan/github/code-visualizer && go test ./internal/canvas/svg/ -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/canvas/svg/
git commit -m "feat(svg): implement radial gradient rendering for DrawRectangle

Emits <radialGradient> elements with fx/fy attributes for gradient
focus offset. Each gradient gets a unique ID referenced via fill url().

Part of #263 — pincushion treemap effect."
```

---

### Task 7: Treemap Integration — Focus Calculation and --flat Flag

**Files:**
- Modify: `internal/treemap/render.go`
- Modify: `internal/treemap/inks.go`
- Modify: `internal/treemap/state.go`
- Modify: `internal/treemap/stages.go`
- Modify: `cmd/codeviz/treemap_cmd.go`
- Create: `internal/treemap/render_test.go` (extend existing)

- [ ] **Step 1: Add --flat flag to TreemapCmd**

In `cmd/codeviz/treemap_cmd.go`, add to the `TreemapCmd` struct:

```go
Flat bool `help:"Disable pincushion shading (flat solid fills)." default:"false"`
```

- [ ] **Step 2: Add Flat to treemap State and pass through pipeline**

In `internal/treemap/state.go`, add to `State`:

```go
Flat bool
```

In `cmd/codeviz/treemap_cmd.go` `Run()`, after creating `state`, set:

```go
state.Flat = c.Flat
```

(Or pass via an override on the Config. Keep it simple — put it directly on State.)

- [ ] **Step 3: Wrap fill ink with RadialGradientInk in BuildInksStage**

In `internal/treemap/stages.go`, modify `BuildInksStage`:

```go
func BuildInksStage(s *State) error {
	c := s.Common()

	slog.Info("Rendering image", "output", c.Output, "width", c.Width, "height", c.Height)

	s.Inks = BuildInks(c.Root, s.FillMetric, s.FillPalette, s.BorderMetric, s.BorderPalette)

	if !s.Flat {
		s.Inks.Fill = canvas.NewRadialGradientInk(s.Inks.Fill)
	}

	return nil
}
```

- [ ] **Step 4: Compute focus point in addFileRectForFile**

In `internal/treemap/render.go`, modify `addRect` to pass the parent directory rect, and update `addFileRectForFile` to compute focus:

Change `addRect` signature to receive and pass the directory's own rectangle:

```go
func addRect(
	cv *canvas.Canvas,
	rect TreemapRectangle,
	node *model.Directory,
	inks Inks,
	sizeMetric metric.Name,
) {
	if !rect.IsDirectory {
		addFileRectForFile(cv, rect, nil, inks, rect, 0)
		return
	}

	addDirectoryShapes(cv, rect)

	dirTotal := directoryTotalWeight(node, sizeMetric)

	fileIdx := 0
	dirIdx := 0

	for i := range rect.Children {
		child := rect.Children[i]
		if child.IsDirectory && dirIdx < len(node.Dirs) {
			addRect(cv, child, node.Dirs[dirIdx], inks, sizeMetric)
			dirIdx++
		} else if !child.IsDirectory && fileIdx < len(node.Files) {
			fileWeight := fileMetricWeight(node.Files[fileIdx], sizeMetric)
			addFileRectForFile(cv, child, node.Files[fileIdx], inks, rect, fileWeight/dirTotal)
			fileIdx++
		}
	}
}
```

Update `addFileRectForFile` to compute focus from the weight and parent rect:

```go
func addFileRectForFile(
	cv *canvas.Canvas,
	rect TreemapRectangle,
	file *model.File,
	inks Inks,
	parentDir TreemapRectangle,
	weightFraction float64,
) {
	if rect.W <= 0 || rect.H <= 0 {
		return
	}

	focus := computeFocus(rect, parentDir, weightFraction)

	hasBorder := inks.Border.Info().Kind
	fillMV := pkginks.MetricValueForFile(file, inks.Fill)
	borderMV := pkginks.MetricValueForFile(file, inks.Border)

	spec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        inks.Fill,
			Border:      inks.Border,
			BorderWidth: DynBorderWidth(rect.W, rect.H, hasBorder),
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
		Focus:  focus,
	})

	if rect.Label != "" && rect.W >= 40 && rect.H >= 16 {
		fillColour := inks.Fill.Dip(fillMV)
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
}
```

Add the focus computation helper:

```go
func computeFocus(
	fileRect, dirRect TreemapRectangle,
	weightFraction float64,
) model.Point {
	if fileRect.W <= 0 || fileRect.H <= 0 {
		return model.Point{X: 0.5, Y: 0.5}
	}

	fileCX := fileRect.X + fileRect.W/2
	fileCY := fileRect.Y + fileRect.H/2
	dirCX := dirRect.X + dirRect.W/2
	dirCY := dirRect.Y + dirRect.H/2

	// Lerp from file centre toward directory centre by weight fraction
	focusX := fileCX + (dirCX-fileCX)*weightFraction
	focusY := fileCY + (dirCY-fileCY)*weightFraction

	// Express as fraction of file rect (may exceed [0,1])
	return model.Point{
		X: (focusX - fileRect.X) / fileRect.W,
		Y: (focusY - fileRect.Y) / fileRect.H,
	}
}
```

Add weight helpers:

```go
func directoryTotalWeight(dir *model.Directory, sizeMetric metric.Name) float64 {
	total := 0.0
	for _, f := range dir.Files {
		total += fileMetricWeight(f, sizeMetric)
	}

	if total <= 0 {
		total = float64(len(dir.Files))
	}

	return total
}

func fileMetricWeight(file *model.File, sizeMetric metric.Name) float64 {
	if file == nil {
		return 1.0
	}

	if sizeMetric == "" {
		return 1.0
	}

	v, ok := file.Metrics[sizeMetric]
	if !ok {
		return 1.0
	}

	switch val := v.(type) {
	case int:
		return float64(val)
	case float64:
		return val
	default:
		return 1.0
	}
}
```

Note: Import `"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"` at the top of `render.go`. Also add `metric` to the imports and pass `sizeMetric` through `RenderToCanvas`.

- [ ] **Step 5: Update RenderToCanvas signature to accept sizeMetric**

```go
func RenderToCanvas(
	rects TreemapRectangle,
	root *model.Directory,
	width, height int,
	inks Inks,
	sizeMetric metric.Name,
) *canvas.Canvas {
	cv := canvas.NewCanvas(width, height)

	bgSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        canvas.FixedInk(bgColour),
			Border:      canvas.FixedInk(bgColour),
			BorderWidth: 0,
		},
	}
	cv.AddRectangle(canvas.LayerBackground, canvas.Rectangle{
		Spec: bgSpec,
		X:    0, Y: 0,
		W: float64(width), H: float64(height),
	})

	addRect(cv, rects, root, inks, sizeMetric)

	return cv
}
```

Update `RenderStage` in `stages.go` to pass `s.Size`:

```go
func RenderStage(s *State) error {
	c := s.Common()

	cv := RenderToCanvas(s.Root, c.Root, c.Width, c.Height, s.Inks, s.Size)
	if s.LegendConfig != nil {
		cv.SetLegend(*s.LegendConfig)
	}

	slog.Debug("rendering", "width", c.Width, "height", c.Height, "output", c.Output)

	c.Canvas = cv

	return nil
}
```

- [ ] **Step 6: Fix compilation — update render_test.go callers**

Search for calls to `RenderToCanvas` in test files and add the `sizeMetric` parameter (pass `""` for tests that don't care about focus).

- [ ] **Step 7: Run all tests**

Run: `cd /home/bevan/github/code-visualizer && go test ./... 2>&1 | tail -20`
Expected: All PASS

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "feat(treemap): add pincushion effect with --flat flag

File tiles use radial gradient shading with focus offset toward the
parent directory centre, weighted by the file's size contribution.
Use --flat to disable.

Closes #263."
```

---

### Task 8: Integration Test — Golden File

**Files:**
- Extend: `internal/treemap/render_test.go`

- [ ] **Step 1: Add a golden-file integration test**

Add a test that renders a small treemap with pincushion enabled and compares against a golden file:

```go
func TestRenderToCanvas_PincushionEffect_GoldenFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Build a simple tree with a directory containing 3 files of different sizes
	dir := &model.Directory{
		Name: "root",
		Files: []*model.File{
			{Name: "big.go", Metrics: map[metric.Name]any{"file-lines": 100}},
			{Name: "medium.go", Metrics: map[metric.Name]any{"file-lines": 50}},
			{Name: "small.go", Metrics: map[metric.Name]any{"file-lines": 10}},
		},
	}

	rects := Layout(dir, 400, 300, "file-lines")
	fillInk := canvas.NewRadialGradientInk(
		canvas.NumericInk("file-lines", []float64{10, 50, 100}, palette.GetPalette(palette.Foliage)),
	)
	inks := Inks{
		Fill:   fillInk,
		Border: canvas.FixedInk(color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}),
	}

	cv := RenderToCanvas(rects, dir, 400, 300, inks, "file-lines")

	// Render to PNG in temp dir and compare golden
	tmp := filepath.Join(t.TempDir(), "pincushion.png")
	err := cv.Render(tmp)
	g.Expect(err).NotTo(HaveOccurred())

	goldie.New(t).AssertWithTemplate(t.Name(), nil, readFile(t, tmp))
}
```

(Adjust based on actual goldie usage patterns in the codebase — check existing golden file tests for the exact API.)

- [ ] **Step 2: Run test to generate golden file**

Run: `cd /home/bevan/github/code-visualizer && go test ./internal/treemap/ -run TestRenderToCanvas_PincushionEffect -update -v`
Expected: generates golden file for first run

- [ ] **Step 3: Run test again without -update to verify it matches**

Run: `cd /home/bevan/github/code-visualizer && go test ./internal/treemap/ -run TestRenderToCanvas_PincushionEffect -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "test(treemap): add golden-file test for pincushion rendering

Verifies the gradient effect renders consistently."
```

---

### Task 9: Full Pipeline Test and Lint

- [ ] **Step 1: Run the full test suite**

Run: `cd /home/bevan/github/code-visualizer && task test`
Expected: All tests PASS

- [ ] **Step 2: Run the linter**

Run: `cd /home/bevan/github/code-visualizer && task lint`
Expected: No lint errors (or only pre-existing ones)

- [ ] **Step 3: Run the full CI check**

Run: `cd /home/bevan/github/code-visualizer && task ci`
Expected: Build, test, and lint all pass

- [ ] **Step 4: Fix any issues found**

Address any compilation errors, lint warnings, or test failures.

- [ ] **Step 5: Final commit if fixes were needed**

```bash
git add -A
git commit -m "fix: address lint and test issues from pincushion implementation"
```

---

### Task 10: Manual Verification

- [ ] **Step 1: Build the binary**

Run: `cd /home/bevan/github/code-visualizer && task build`

- [ ] **Step 2: Test with pincushion (default)**

Run against a real directory:
```bash
./bin/codeviz treemap ./internal -o /tmp/pincushion-test.png -s file-lines -f file-lines,foliage
```

Verify the output PNG shows gradient-shaded file tiles.

- [ ] **Step 3: Test with --flat**

```bash
./bin/codeviz treemap ./internal -o /tmp/flat-test.png -s file-lines -f file-lines,foliage --flat
```

Verify the output PNG shows solid-colour tiles (no gradient).

- [ ] **Step 4: Test SVG output**

```bash
./bin/codeviz treemap ./internal -o /tmp/pincushion-test.svg -s file-lines -f file-lines,foliage
```

Verify the SVG contains `<radialGradient>` elements.

- [ ] **Step 5: Commit any final golden file updates**

```bash
git add -A
git commit -m "chore: update golden files after pincushion implementation"
```
