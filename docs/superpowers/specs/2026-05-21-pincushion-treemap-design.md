# Pincushion Treemap Effect

**Issue:** #263  
**Date:** 2026-05-21  
**Status:** Approved

## Summary

Add a "pincushion" visual effect to treemap file tiles — a radial gradient that darkens edges by 40%, creating a stuffed-cushion appearance. The gradient's focal point is offset toward the enclosing directory's centre, weighted by the file's metric contribution to its directory.

## Decisions

| Decision | Choice |
|----------|--------|
| Scope | File tiles only; directories stay flat |
| Intensity | 40% edge darkening |
| Default | Pincushion ON; `--flat` disables |
| Border interaction | Border drawn on top of cushion shading |
| Gradient centre | Weighted offset toward parent directory centre |
| Ink architecture | `Ink` becomes an interface with multiple implementations |
| Fill architecture | `model.Fill` interface with `SolidFill` and `RadialGradientFill` |
| Naming convention | Ink/Fill types correspond: `RadialGradientInk` → `RadialGradientFill`, leaving room for `LinearGradientInk` → `LinearGradientFill`, etc. |

## Gradient Focus Calculation

For each file tile, the gradient's brightest point is offset from the file rectangle's centre toward its parent directory rectangle's centre:

```
fileCentre  = centre(fileRect)
dirCentre   = centre(parentDirRect)
weight      = fileMetricValue / dirTotalMetricValue

focusAbs    = lerp(fileCentre, dirCentre, weight)

// Expressed as fraction of file rect (may exceed [0,1])
focus.X     = (focusAbs.X - fileRect.X) / fileRect.W
focus.Y     = (focusAbs.Y - fileRect.Y) / fileRect.H
```

**Effect:** Large files (high weight) have their highlight pulled strongly toward the directory centre, creating visual cohesion within directory groups. Small files keep their highlight near their own centre.

**Edge cases:**
- Single file in directory → weight = 1.0 → focus at directory centre
- Tiny file → weight ≈ 0 → focus at file centre (symmetric cushion)
- Focus may land outside file rectangle bounds (intentional)

## Architecture

### model.Fill Interface

New file: `internal/canvas/model/fill.go`

```go
package model

import "image/color"

type Fill interface {
    isFill() // sealed marker
}

type SolidFill struct {
    Color color.RGBA
}

type RadialGradientFill struct {
    Center color.RGBA   // brightest colour (at focus point)
    Edge   color.RGBA   // darkened colour (at rect edges)
    FocusX float64      // focus X as fraction of rect width
    FocusY float64      // focus Y as fraction of rect height
}

type Point struct {
    X, Y float64
}

func (SolidFill) isFill()            {}
func (RadialGradientFill) isFill()   {}
```

### Backend Signature Change

`DrawRectangle` in `model/backend.go` changes:

```go
// Before
DrawRectangle(pos Position, size Size, fill, border color.RGBA, borderWidth float64)

// After
DrawRectangle(pos Position, size Size, fill, border Fill, borderWidth float64)
```

### Ink Interface

`internal/canvas/ink.go` — `Ink` becomes an interface:

```go
type Ink interface {
    Dip(value MetricValue) color.RGBA
    Fill(value MetricValue, focus model.Point) model.Fill
    Info() InkInfo
}
```

Existing implementations (`fixedInk`, `numericInk`, `categoricalInk`) implement `Fill()` by returning `SolidFill{Color: ink.Dip(value)}`, ignoring the focus parameter.

Public constructors (`FixedInk()`, `NumericInk()`, `CategoricalInk()`) return `Ink` (the interface) — no API change for callers.

### RadialGradientInk

New file: `internal/canvas/radial_gradient_ink.go`

```go
type RadialGradientInk struct {
    Inner  Ink
    Darken float64 // fraction to darken edges (0.4 = 40%)
}

func NewRadialGradientInk(inner Ink) Ink {
    return &RadialGradientInk{Inner: inner, Darken: 0.4}
}

func (g *RadialGradientInk) Dip(value MetricValue) color.RGBA {
    return g.Inner.Dip(value) // delegate
}

func (g *RadialGradientInk) Fill(value MetricValue, focus model.Point) model.Fill {
    base := g.Inner.Dip(value)
    edge := darken(base, g.Darken)
    return model.RadialGradientFill{
        Center: base,
        Edge:   edge,
        FocusX: focus.X,
        FocusY: focus.Y,
    }
}

func (g *RadialGradientInk) Info() InkInfo {
    return g.Inner.Info() // delegate introspection
}
```

### Shape Layer

`Rectangle` in `internal/canvas/shape.go` gains a `Focus` field:

```go
type Rectangle struct {
    Spec  ShapeStyle
    Fill  MetricValue
    Focus model.Point
    Pos   model.Position
    Size  model.Size
}

func (r Rectangle) drawTo(b model.Backend) {
    fill := r.Spec.Fill.Fill(r.Fill, r.Focus)
    border := model.SolidFill{Color: r.Spec.Border.Dip(r.Fill)}
    b.DrawRectangle(r.Pos, r.Size, fill, border, r.Spec.BorderWidth)
}
```

Other shapes (circles, lines, text) continue using `Dip()` — pincushion applies to treemap file rectangles only.

### Raster Backend

For `RadialGradientFill`, draw concentric inset rectangles with colours interpolated from edge to centre, offset toward the focus point. The number of steps scales with rectangle size (diminishing returns beyond ~20 steps for small tiles).

### SVG Backend

For `RadialGradientFill`, emit a `<radialGradient>` element with `fx`/`fy` attributes set to the focus point, and two stops: edge colour at 100% radius, centre colour at 0%. Reference via `fill="url(#grad-N)"`.

### Treemap Renderer

In `internal/treemap/render.go`, the `addFileRectForFile` function computes focus and passes it to the canvas:

```go
// Compute focus for pincushion
fileCentre := centre(fileRect)
dirCentre  := centre(parentDirRect)
weight     := fileValue / dirTotalValue
focusAbs   := lerp(fileCentre, dirCentre, weight)
focus      := model.Point{
    X: (focusAbs.X - fileRect.X) / fileRect.W,
    Y: (focusAbs.Y - fileRect.Y) / fileRect.H,
}

canvas.AddRectangle(spec, metricValue, pos, size, focus)
```

### CLI

`cmd/codeviz/treemap_cmd.go`:

```go
type TreemapCmd struct {
    // ... existing fields
    Flat bool `kong:"help='Disable pincushion shading',default='false'"`
}
```

In `Run()`, conditionally wrap:

```go
fillInk := metricInk
if !cmd.Flat {
    fillInk = canvas.NewRadialGradientInk(metricInk)
}
```

## Files Changed

| File | Nature of Change |
|------|-----------------|
| `internal/canvas/model/fill.go` | New — Fill interface, SolidFill, RadialGradientFill |
| `internal/canvas/model/backend.go` | Modified — DrawRectangle signature |
| `internal/canvas/ink.go` | Major refactor — struct → interface + implementations |
| `internal/canvas/ink_introspection.go` | Modified — methods move to implementations |
| `internal/canvas/radial_gradient_ink.go` | New — RadialGradientInk wrapper |
| `internal/canvas/shape.go` | Modified — Rectangle.Focus field, drawTo uses Fill() |
| `internal/canvas/raster/backend.go` | Modified — gradient rendering path |
| `internal/canvas/svg/backend.go` | Modified — radialGradient SVG emission |
| `internal/treemap/render.go` | Modified — focus calculation, AddRectangle call |
| `cmd/codeviz/treemap_cmd.go` | Modified — --flat flag, conditional wrapping |

## Not In Scope

- Pincushion on spiral/bubbletree visualisations (future work)
- Linear gradient fills (future ink/fill type pair)
- Configurable darkening percentage (hardcoded 0.4 for now)
- Configurable focus offset formula (hardcoded lerp by weight)
