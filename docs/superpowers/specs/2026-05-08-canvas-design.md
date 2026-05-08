# Canvas Abstraction Design

A drawing abstraction for `code-visualizer` that replaces duplicated per-visualization rendering code with a shared Canvas API.

## Problem

The current rendering layer has four visualization types (treemap, radial tree, bubble tree, spiral), each with paired raster (PNG/JPG via `fogleman/gg`) and SVG (direct XML) renderers.
This produces ~4600 lines in `internal/render/` with near-identical patterns for color resolution, shape drawing, and format dispatch.
The `cmd/codeviz/` layer adds another ~800 lines of duplicated `applyFillColours`/`applyBorderColours` logic.
Layout node types (`TreemapRectangle`, `RadialNode`, `BubbleNode`, `SpiralNode`) each carry `FillColour color.RGBA` and `BorderColour *color.RGBA` fields, embedding resolved color data into geometry.

## Approach

Introduce a new `internal/canvas/` package with:

- **Ink** — a smart colour resolver that maps metric values to RGBA via palette + mapping strategy.
- **Specs** — CSS-like templates that bundle inks with visual properties (line width, opacity, label style).
- **Canvas** — a retained-then-render drawing surface with layered z-ordering and two backend implementations (raster, SVG).

Layout node types become geometry-only (no color fields).
Visualizations set up Inks and Specs, add shapes to the Canvas with metric values, and call `Render()`.

## Package Location

`internal/canvas/`

## Ink

Ink encapsulates the full metric-to-colour pipeline: raw metric value → bucketing/mapping → palette lookup → RGBA.
Visualizations create Inks once (with the full dataset for distribution analysis) and then `Dip()` them with per-shape metric values.

### Types

```go
// Ink resolves metric values to colours.
// Fixed inks ignore the metric value; metric inks resolve via palette + mapping strategy.
type Ink struct {
    // Unexported fields:
    // kind       inkKind
    // color      color.RGBA           (for fixed inks)
    // boundaries *metric.BucketBoundaries (for numeric inks)
    // catMapper  *palette.CategoricalMapper (for categorical inks)
    // pal        palette.ColourPalette
    // strategy   MappingStrategy
}
```

### Constructors

```go
// FixedInk always produces the same colour regardless of input.
func FixedInk(c color.RGBA) Ink

// NumericInk maps numeric metric values to palette colours.
// Takes the full dataset of values (for bucketing), the palette, and optional mapping options.
// Default mapping strategy is quantile-based (current behavior).
func NumericInk(values []float64, pal palette.ColourPalette, opts ...InkOption) Ink

// CategoricalInk maps string categories to palette colours.
func CategoricalInk(categories []string, pal palette.ColourPalette) Ink
```

### Resolution

```go
// Dip resolves a numeric metric value to an RGBA colour.
// For fixed inks, the value is ignored.
func (ink Ink) Dip(value float64) color.RGBA

// DipCategory resolves a categorical value to an RGBA colour.
func (ink Ink) DipCategory(value string) color.RGBA
```

### Mapping Strategies

```go
type MappingStrategy int

const (
    Quantile    MappingStrategy = iota // equal-count buckets (current default)
    Linear                              // evenly spaced across min-max range
    Logarithmic                         // log-scale spacing
)

// InkOption configures ink behavior.
func WithMapping(strategy MappingStrategy) InkOption
```

The Ink owns the full pipeline.
Visualizations call `NumericInk(allValues, palette)` once, then `ink.Dip(fileValue)` per shape.
This eliminates all the duplicated `applyFillColours`/`applyBorderColours` code from `cmd/codeviz/`.

## Specs (Style Templates)

Specs define the complete visual template for a shape type — inks, line widths, opacity, label settings.
Like a CSS class: define once, apply to many shapes.
Shapes can override specific inks per-instance (the CSS-class-with-inline-override pattern).

### Label Styles

```go
type LabelStyle int

const (
    LabelCentered LabelStyle = iota // text centered inside shape
    LabelArc                         // text curved along circle boundary (bubble dirs)
    LabelRadial                      // text outside shape, rotated outward (radial/spiral)
)
```

### Spec Types

```go
// RectangleSpec defines the visual template for rectangles.
type RectangleSpec struct {
    Fill        Ink
    Border      Ink
    BorderWidth float64
    Opacity     float64    // 1.0 = fully opaque (default); applies to fill only, border is always opaque
    ShowLabel   bool
    LabelInk    Ink
    LabelStyle  LabelStyle
}

// DiscSpec defines the visual template for circles/discs.
type DiscSpec struct {
    Fill        Ink
    Border      Ink
    BorderWidth float64
    Opacity     float64   // fill opacity only; border is always opaque
    ShowLabel   bool
    LabelInk    Ink
    LabelStyle  LabelStyle
}

// LineSpec defines the visual template for lines.
type LineSpec struct {
    Stroke      Ink
    StrokeWidth float64
}

// TextAnchor controls horizontal text alignment.
type TextAnchor int
const (
    AnchorStart  TextAnchor = iota // left-aligned
    AnchorMiddle                    // centered
    AnchorEnd                       // right-aligned
)

// TextSpec defines the visual template for standalone text.
type TextSpec struct {
    Ink       Ink
    FontSize  float64
    Anchor    TextAnchor
    Rotation  float64    // radians
}
```

### Example Usage

```go
// Treemap file rectangles:
fileSpec := canvas.RectangleSpec{
    Fill:        canvas.NumericInk(fileSizeValues, temperaturePalette),
    Border:      canvas.FixedInk(structuralBorderColor),
    BorderWidth: 2.0,
    ShowLabel:   true,
    LabelInk:    canvas.FixedInk(darkTextColor),
    LabelStyle:  canvas.LabelCentered,
}

// Bubble tree directory circles:
dirDiscSpec := canvas.DiscSpec{
    Fill:        canvas.NumericInk(churnValues, foliagePalette),
    Border:      canvas.FixedInk(structuralBorderColor),
    BorderWidth: 0.5,
    Opacity:     0.18,
    ShowLabel:   true,
    LabelInk:    canvas.FixedInk(labelColor),
    LabelStyle:  canvas.LabelArc,
}
```

## Canvas

A retained-then-render drawing surface.
Callers add shapes, then call `Render()` which handles z-ordering and produces output.

### Creation

```go
// NewCanvas creates a canvas for the given dimensions and output path.
// The format (PNG, JPG, SVG) is inferred from the file extension.
func NewCanvas(width, height int, outputPath string) (*Canvas, error)
```

### Shape Types

Shapes carry geometry and raw metric values.
Colour resolution happens at render time via `spec.Fill.Dip(shape.FillValue)`.

```go
type Rectangle struct {
    Spec           *RectangleSpec
    X, Y, W, H    float64
    FillValue      float64 // metric value for spec.Fill.Dip() (ignored if Fill is fixed)
    BorderValue    float64 // metric value for spec.Border.Dip()
    FillCategory   string  // for categorical fill inks
    BorderCategory string  // for categorical border inks
    Label          string
}

type Disc struct {
    Spec           *DiscSpec
    X, Y           float64
    Radius         float64
    Angle          float64 // for radial/external label orientation
    FillValue      float64
    BorderValue    float64
    FillCategory   string
    BorderCategory string
    Label          string
}

type Line struct {
    Spec   *LineSpec
    X1, Y1 float64
    X2, Y2 float64
}

type Path struct {
    Spec   *LineSpec
    Points []Point
}

type Point struct {
    X, Y float64
}
```

### Drawing Methods

```go
func (c *Canvas) AddRectangle(layer Layer, r Rectangle)
func (c *Canvas) AddDisc(layer Layer, d Disc)
func (c *Canvas) AddLine(layer Layer, l Line)
func (c *Canvas) AddPath(layer Layer, p Path)
func (c *Canvas) Render() error
```

## Layers and Z-Ordering

Shapes are assigned to layers when added.
At render time, the Canvas draws layers in order (lowest value first).
Within a layer, shapes are drawn in insertion order — callers control intra-layer ordering.

```go
type Layer int

const (
    LayerBackground Layer = 0
    LayerStructure  Layer = 10  // edges, guide tracks, directory borders
    LayerContent    Layer = 20  // file rectangles, file discs
    LayerOverlay    Layer = 30  // labels, legends
)
```

### Layer Mapping

| Current multi-pass              | Canvas layer     |
|---------------------------------|------------------|
| Bubble: dir circles (large→small) | LayerStructure |
| Bubble: file circles            | LayerContent     |
| Radial: edge lines              | LayerStructure   |
| Radial/Spiral: node discs       | LayerContent     |
| All: labels                     | LayerOverlay     |
| Spiral: guide track             | LayerStructure   |
| Treemap: directory headers      | LayerStructure   |
| Treemap: file rectangles        | LayerContent     |

## Legend

The Canvas owns legend rendering.
Inks carry palette metadata, so the legend can be generated from the Inks used in the visualization.

```go
func (c *Canvas) SetLegend(config LegendConfig)

type LegendConfig struct {
    Position    LegendPosition
    Orientation LegendOrientation
    Entries     []LegendEntry
}

type LegendEntry struct {
    Role string // "Fill", "Border", "Size"
    Ink  Ink    // palette/mapping metadata for legend rendering
}
```

The existing `ReserveLegendSpace()` pattern carries over — the Canvas knows the legend dimensions and can carve out space before the visualization fills the remaining area.

The legend is rendered as part of `Render()` on `LayerOverlay`.

## Backend Implementations

The Canvas uses an internal backend interface.
Two implementations: raster (PNG/JPG via `fogleman/gg`) and SVG (direct XML).
The backend is chosen at Canvas creation time from the output path extension.

```go
type backend interface {
    drawRectangle(x, y, w, h float64, fill, border color.RGBA, borderWidth, opacity float64)
    drawDisc(x, y, r float64, fill, border color.RGBA, borderWidth, opacity float64)
    drawLine(x1, y1, x2, y2 float64, stroke color.RGBA, strokeWidth float64)
    drawPath(points []Point, stroke color.RGBA, strokeWidth float64)
    drawText(x, y float64, text string, ink color.RGBA, fontSize float64, anchor TextAnchor, rotation float64)
    drawArcText(x, y, radius float64, text string, ink color.RGBA, fontSize float64)
    finish(outputPath string) error
}
```

### rasterBackend

Wraps `*gg.Context` from `fogleman/gg`.
Maps each `draw*` method to the corresponding `gg` calls (`DrawRectangle`/`Fill`/`Stroke`, `DrawCircle`, `DrawStringAnchored`, etc.).
`finish()` saves as PNG or JPG.

### svgBackend

Wraps `*os.File`.
Maps each `draw*` method to SVG XML elements (`<rect>`, `<circle>`, `<line>`, `<path>`, `<text>`, `<textPath>`).
`finish()` writes the closing `</svg>` tag and closes the file.

## Migration Path

### Stage 1: Build the Canvas Package

Create `internal/canvas/` with all types and implementations.
Fully testable in isolation using golden-file snapshot tests (Goldie v2).
No changes to existing code.

### Stage 2: Migrate Visualizations

Migrate one visualization at a time. For each:

1. **Strip colour fields** from the layout node type (`FillColour`, `BorderColour`).
2. **Replace render functions** — delete the old `internal/render/` files for that viz type and replace with Canvas-based drawing code.
3. **Update the command** — replace `applyFillColours`/`applyBorderColours` with Ink creation and Canvas shape-adding.
4. **Run golden-file tests** — verify visual output matches (or update goldens where the output intentionally improves).

Suggested migration order:

1. **Treemap** — simplest (rectangles only, no multi-pass, no rotated labels beyond centered).
2. **Spiral** — flat node list (no tree walk), uses Line/Path + Discs.
3. **Radial tree** — tree walk, edges + discs + rotated labels.
4. **Bubble tree** — most complex (arc text, opacity, multi-pass z-ordering).

### What Gets Deleted

- `internal/render/renderer.go` (treemap PNG)
- `internal/render/radialtree.go` (radial PNG)
- `internal/render/bubbletree.go` (bubble PNG)
- `internal/render/spiral.go` (spiral PNG)
- `internal/render/svg_treemap.go` (treemap SVG)
- `internal/render/svg_radial.go` (radial SVG)
- `internal/render/svg_bubble.go` (bubble SVG)
- `internal/render/svg_spiral.go` (spiral SVG)
- `internal/render/svg_helpers.go` (absorbed into SVG backend)
- All `applyFillColours`/`applyBorderColours` functions in `cmd/codeviz/`

### What Stays

- Layout packages (`internal/treemap/`, `internal/radialtree/`, `internal/bubbletree/`, `internal/spiral/`) — geometry-only after stripping colour fields.
- `internal/palette/` — used by Ink.
- `internal/metric/` — used by Ink for bucketing.
- `internal/render/label.go`, `internal/render/format.go`, `internal/render/save.go` — utility code that may move into the canvas package or stay as helpers.
