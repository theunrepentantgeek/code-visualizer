# Canvas Abstraction Design

A drawing abstraction for `code-visualizer` that replaces duplicated per-visualization rendering code with a shared Canvas API.

## Problem

The current rendering layer has four visualization types (treemap, radial tree, bubble tree, spiral), each with paired raster (PNG/JPG via `fogleman/gg`) and SVG (direct XML) renderers.
This produces ~4600 lines in `internal/render/` with near-identical patterns for color resolution, shape drawing, and format dispatch.
The `cmd/codeviz/` layer adds another ~800 lines of duplicated `applyFillColours`/`applyBorderColours` logic — twenty `nolint:dupl` annotations in the cmd files confirm this pattern has been flagged by the linter for a while.
Layout node types (`TreemapRectangle`, `RadialNode`, `BubbleNode`, `SpiralNode`) each carry `FillColour color.RGBA` and `BorderColour *color.RGBA` fields, embedding resolved color data into geometry.

This is not a premature abstraction — it is a pattern proven across four implementations.

## Approach

Introduce a new `internal/canvas/` package with:

- **Ink** — a smart colour resolver that maps metric values to RGBA via palette + mapping strategy.
- **Specs** — CSS-like visual templates that bundle inks with line widths, label styles, and other visual properties.
- **Canvas** — a retained-then-render drawing surface with layered z-ordering and two backend implementations (raster, SVG).

Each layer has a clear single job: Ink resolves colours, Spec bundles visual style, Canvas handles drawing and z-order.
Layout node types become geometry-only (no color fields).
Visualizations set up Inks and Specs, add shapes to the Canvas with metric values, and call `Render()`.
Adding a new visualization becomes "write layout, create inks, fill canvas, call `Render()`" — no new renderer code needed.

## Package Location

`internal/canvas/`

## Ink

Ink encapsulates the full metric-to-colour pipeline: raw metric value → bucketing/mapping → palette lookup → RGBA.
This consolidates a pipeline currently scattered across three packages (`metric.ComputeBuckets` → `metric.BucketBoundaries.BucketIndex` → `palette.MapNumericToColour` for numeric; `palette.NewCategoricalMapper` → `CategoricalMapper.Map` for categorical) behind a single method call.
Visualizations create Inks once (with the full dataset for distribution analysis) and then `Dip()` them with per-shape metric values.

### MetricValue

`MetricValue` wraps everything needed to resolve a colour from a metric, unifying numeric and categorical metric data into a single type.
This eliminates the `FillValue`/`FillCategory`/`BorderValue`/`BorderCategory` field proliferation on shapes and allows Ink to own dispatch internally.

```go
// MetricValue carries the metric data needed to resolve a colour.
// The Kind field determines which of the remaining fields is used.
type MetricValue struct {
    Kind     metric.Kind
    Measure  float64
    Quantity int
    Category string
}
```

### Types

```go
// Ink resolves metric values to colours.
// Fixed inks ignore the metric value; metric inks resolve via palette + mapping strategy.
//
// Ink is safe to copy; internal state is shared via pointers.
type Ink struct {
    // Unexported fields:
    // kind       inkKind
    // color      color.RGBA           (for fixed inks)
    // boundaries *metric.BucketBoundaries (for numeric inks)
    // catMapper  *palette.CategoricalMapper (for categorical inks)
    // pal        palette.ColourPalette
    // strategy   MappingStrategy
    // opacity    float64              (default 1.0; applied to alpha channel on Dip)
}
```

### Constructors

All constructors accept optional `InkOption` parameters for future extensibility.

```go
// FixedInk always produces the same colour regardless of input.
func FixedInk(c color.RGBA, opts ...InkOption) Ink

// NumericInk maps numeric metric values to palette colours.
// Takes the full dataset of values (for bucketing), the palette, and optional mapping options.
// Default mapping strategy is quantile-based (current behavior).
func NumericInk(values []float64, pal palette.ColourPalette, opts ...InkOption) Ink

// CategoricalInk maps string categories to palette colours.
func CategoricalInk(categories []string, pal palette.ColourPalette, opts ...InkOption) Ink
```

### Resolution

```go
// Dip resolves a MetricValue to an RGBA colour.
// The Ink dispatches internally based on MetricValue.Kind:
// numeric inks use Measure/Quantity, categorical inks use Category,
// and fixed inks ignore the value entirely.
// Opacity is applied to the alpha channel of the resolved colour.
func (ink Ink) Dip(value MetricValue) color.RGBA
```

### Introspection

The legend renderer needs to iterate bucket boundaries, palette colours, and category lists
to draw individual swatches and format breakpoint labels.
Ink exposes query methods so it remains the single source of truth for palette and bucketing metadata.

```go
// Boundaries returns the bucket boundary values for numeric inks.
// Returns nil for fixed or categorical inks.
func (ink Ink) Boundaries() []float64

// Palette returns the colour palette used by this ink.
// Returns an empty palette for fixed inks.
func (ink Ink) Palette() palette.ColourPalette

// Categories returns the category labels for categorical inks.
// Returns nil for fixed or numeric inks.
func (ink Ink) Categories() []string
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

// WithOpacity sets the opacity applied when Dip() resolves a colour.
// Default is 1.0 (fully opaque). The opacity is applied to the alpha channel
// of the resolved colour, so a FixedInk with WithOpacity(0.18) produces
// a semi-transparent version of its fixed colour.
func WithOpacity(opacity float64) InkOption
```

The Ink owns the full pipeline.
Visualizations call `NumericInk(allValues, palette)` once, then `ink.Dip(metricValue)` per shape.
This eliminates all the duplicated `applyFillColours`/`applyBorderColours` code from `cmd/codeviz/`.

The data flow is unchanged from the current code — `NumericInk` takes `[]float64` which means the caller collects all values upfront for distribution analysis, just as `collectNumericValues` does today. It moves behind a cleaner API.

## Specs (Style Templates)

Specs define the complete visual template for a shape type — inks, line widths, label settings.
Like a CSS class: define once, apply to many shapes (the Flyweight pattern).

### ShapeStyle

`RectangleSpec` and `DiscSpec` share all their visual fields.
The common fields are extracted into an embedded `ShapeStyle` struct so that new visual properties (dash patterns, corner radius, etc.) only need to be added in one place.
Disc-specific or rectangle-specific fields can be added to their respective specs without losing the shared contract.

```go
// ShapeStyle bundles the visual properties shared by all closed-shape specs.
type ShapeStyle struct {
    Fill        Ink
    Border      Ink
    BorderWidth float64
    ShowLabel   bool
    LabelInk    Ink
    LabelStyle  LabelStyle
}
```

Opacity is a property of how an ink renders, not of the shape template.
It lives on the Ink via `WithOpacity()` — for example, the bubble tree's translucent directory circles
(currently `bubbleDirAlpha = 0x30`) become `NumericInk(churnValues, foliagePalette, WithOpacity(0.18))`.

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

// TextAnchor controls horizontal text alignment.
type TextAnchor int
const (
    AnchorStart  TextAnchor = iota // left-aligned
    AnchorMiddle                    // centered
    AnchorEnd                       // right-aligned
)

// TextSpec defines the visual template for standalone text.
// Font family is intentionally fixed (sans-serif for SVG, goregular for raster)
// and is not exposed as a configurable field.
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
    ShapeStyle: canvas.ShapeStyle{
        Fill:        canvas.NumericInk(fileSizeValues, temperaturePalette),
        Border:      canvas.FixedInk(structuralBorderColor),
        BorderWidth: 2.0,
        ShowLabel:   true,
        LabelInk:    canvas.FixedInk(darkTextColor),
        LabelStyle:  canvas.LabelCentered,
    },
}

// Bubble tree directory circles (translucent fill via WithOpacity):
dirDiscSpec := canvas.DiscSpec{
    ShapeStyle: canvas.ShapeStyle{
        Fill:        canvas.NumericInk(churnValues, foliagePalette, canvas.WithOpacity(0.18)),
        Border:      canvas.FixedInk(structuralBorderColor),
        BorderWidth: 0.5,
        ShowLabel:   true,
        LabelInk:    canvas.FixedInk(labelColor),
        LabelStyle:  canvas.LabelArc,
    },
}
```

## Canvas

A retained-then-render drawing surface.
Callers add shapes, then call `Render()` which handles z-ordering and produces output.
The Canvas does not hold a backend at construction time — backend selection is deferred to `Render()`, which infers the output format from the file extension and creates the appropriate backend.
This means the same Canvas can be rendered to multiple formats (PNG + SVG) without reconstruction.

### Creation

```go
// NewCanvas creates a canvas for the given dimensions.
// No output path or format is needed at creation time.
func NewCanvas(width, height int) *Canvas
```

### Shape Types

Shapes carry geometry and raw metric values.
Colour resolution happens at render time via `spec.Fill.Dip(shape.Fill)`.

```go
type Rectangle struct {
    Spec     *RectangleSpec
    X, Y, W, H float64
    Fill     MetricValue  // for spec.Fill.Dip()
    Border   MetricValue  // for spec.Border.Dip()
    Label    string
}

type Disc struct {
    Spec     *DiscSpec
    X, Y     float64
    Radius   float64
    Angle    float64      // angular position; used for radial/external label orientation
    Fill     MetricValue
    Border   MetricValue
    Label    string
}

type Text struct {
    Spec    *TextSpec
    X, Y    float64
    Content string
}

type Line struct {
    Spec   *LineSpec
    X1, Y1 float64
    X2, Y2 float64
}

type Path struct {
    Spec   *LineSpec
    Points []Position
}
```

### Drawing Methods

```go
func (c *Canvas) AddRectangle(layer Layer, r Rectangle)
func (c *Canvas) AddDisc(layer Layer, d Disc)
func (c *Canvas) AddText(layer Layer, t Text)
func (c *Canvas) AddLine(layer Layer, l Line)
func (c *Canvas) AddPath(layer Layer, p Path)

// Render resolves all inks, sorts shapes by layer, and writes the output.
// The format (PNG, JPG, SVG) is inferred from the file extension.
func (c *Canvas) Render(outputPath string) error
```

Shapes are recorded as data (the Command pattern) and executed in batch at render time.
This enables z-ordering without requiring callers to draw in layer order — a direct fix for the current multi-pass rendering in `bubbletree.go` (dirs → files → labels) and `radialtree.go` (edges → discs → labels).

Memory is not a concern: each `Rectangle` is ~80 bytes (6 float64s + MetricValues + pointer). For a 100k-file repo that's ~8MB of shapes — well within CLI tool bounds. The Canvas lives for one render call and gets garbage collected.

## Layers and Z-Ordering

Shapes are assigned to layers when added.
At render time, the Canvas draws layers in order (lowest value first).
Within a layer, shapes are drawn in insertion order — callers control intra-layer ordering (this is how the bubble tree's large-to-small directory circle ordering is preserved).

```go
type Layer int

const (
    LayerBackground Layer = 0
    LayerStructure  Layer = 10  // edges, guide tracks, directory borders
    LayerContent    Layer = 20  // file rectangles, file discs
    LayerOverlay    Layer = 30  // labels, legends
)
```

The 10-unit gaps between layer constants are intentional — they leave room for future layers (e.g., `Layer(15)` for a background gradient behind file shapes) without renumbering existing ones.

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
The legend renderer uses Ink's introspection methods (`Boundaries()`, `Palette()`, `Categories()`) to render gradient swatches and format breakpoint labels — the Ink is the single source of truth for all palette and bucketing metadata.

```go
func (c *Canvas) SetLegend(config LegendConfig)

type LegendConfig struct {
    Position    LegendPosition
    Orientation LegendOrientation
    Entries     []LegendEntry
}

type LegendRole string
const (
    LegendRoleFill   LegendRole = "Fill"
    LegendRoleBorder LegendRole = "Border"
    LegendRoleSize   LegendRole = "Size"
)

type LegendEntry struct {
    Role       LegendRole
    MetricName string     // display name of the metric being visualized
    Ink        Ink        // palette/mapping metadata for legend rendering (introspected via Boundaries/Palette/Categories)
}
```

The existing `ReserveLegendSpace()` pattern carries over — the Canvas knows the legend dimensions and can carve out space before the visualization fills the remaining area.

The legend is rendered as part of `Render()` on `LayerOverlay`.

## Backend Implementations

The Canvas uses a `Backend` interface exported from the `canvas` package.
Two implementations live in subpackages: raster (PNG/JPG via `fogleman/gg`) in `internal/canvas/raster/` and SVG (direct XML) in `internal/canvas/svg/`.
This is the Ports & Adapters pattern: the Canvas defines the port, each subpackage is an adapter.
Subpackages enforce that the backend interface is the *only* contract between the Canvas and its implementations — compiler-enforced isolation that prevents cross-contamination between backends.

Backend selection is deferred to `Render()` time.
The Canvas does not hold a backend at construction — `Render(outputPath)` infers the format from the file extension and creates the appropriate backend.

### Helper Structs

```go
// Position represents a 2D coordinate.
// Note: Point is unified with Position — they are the same concept.
type Position struct {
    X, Y float64
}

// Size represents a width and height.
type Size struct {
    Width, Height float64
}
```

These reduce parameter count on backend methods and prevent coordinate/dimension swap errors.

### Backend Interface

The Backend interface deals in resolved RGBA colours and primitive geometry, not in Inks or Specs.
The Canvas translates from the domain model (Ink, Spec, MetricValue) down to these rendering primitives.
Since opacity is resolved by Ink (applied to the alpha channel during `Dip()`), backend methods receive pre-resolved RGBA with alpha baked in — no separate opacity parameter is needed.

Each subpackage exports a constructor: `raster.New(width, height int) canvas.Backend` and `svg.New(width, height int) canvas.Backend`.

```go
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

`DrawArcText` is a composite operation rather than a pure geometric primitive.
It exists because curved text along a circle boundary is required by the bubble tree visualization and cannot be cleanly decomposed — the raster backend uses per-character glyph positioning (~210 lines from `bubble_font.go`) while the SVG backend uses the native `<textPath>` element.
This method carries the most implementation weight of any backend method.

### rasterBackend

Lives in `internal/canvas/raster/`.
Wraps `*gg.Context` from `fogleman/gg`.
Maps each `Draw*` method to the corresponding `gg` calls (`DrawRectangle`/`Fill`/`Stroke`, `DrawCircle`, `DrawStringAnchored`, etc.).
`Finish()` saves as PNG or JPG.

### svgBackend

Lives in `internal/canvas/svg/`.
Wraps `*os.File`.
Maps each `Draw*` method to SVG XML elements (`<rect>`, `<circle>`, `<line>`, `<path>`, `<text>`, `<textPath>`).
`Finish()` writes the closing `</svg>` tag and closes the file.

## Migration Path

### Stage 1: Build the Canvas Package

Create `internal/canvas/` with all types and implementations.
Fully testable in isolation using golden-file snapshot tests (Goldie v2).
No changes to existing code.
A test/mock backend should be built alongside the real backends for unit testing Canvas layer ordering and shape dispatch without rendering actual images.

### Stage 2: Migrate Visualizations

Migrate one visualization at a time. For each:

1. **Strip colour fields** from the layout node type (`FillColour`, `BorderColour`).
2. **Replace render functions** — delete the old `internal/render/` files for that viz type and replace with Canvas-based drawing code.
3. **Update the command** — replace `applyFillColours`/`applyBorderColours` with Ink creation and Canvas shape-adding.
4. **Run golden-file tests** — verify visual output matches (or update goldens where the output intentionally improves).

Each viz migration should be a single PR — the colour field stripping and render replacement must be atomic because the compiler will not allow a half-migrated state (stripping `FillColour` from a node type breaks every reference until the Canvas renderer replaces them).

Suggested migration order (escalating complexity):

1. **Treemap** — simplest (rectangles only, no multi-pass, no rotated labels beyond centered). ~258 lines of render code.
2. **Spiral** — flat node list (no tree walk), uses Line/Path + Discs. ~381 lines.
3. **Radial tree** — tree walk, edges + discs + rotated labels. ~419 lines.
4. **Bubble tree** — most complex (arc text, opacity, multi-pass z-ordering). ~403 lines + 210 for `bubble_font.go`.

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

### Utility File Disposition

- **`label.go`**: `TextColourFor(fill color.RGBA)` moves into `internal/canvas/` — contrast-aware label colour selection is a Canvas responsibility. When a shape has `ShowLabel: true`, the Canvas auto-selects a WCAG-contrasting label colour from the resolved fill. `ShouldShowLabel` stays viz-specific or generalizes.
- **`format.go`**: `FormatFromPath` moves into `internal/canvas/` — needed by `Render()` to select the backend from the output file extension.
- **`save.go`**: `saveContextPNG`/`saveContextJPG` are absorbed into the raster backend's `Finish()`.

### Bridge Code

Post-migration, the visualization-to-Canvas bridge code lives in each `cmd/codeviz/*_cmd.go`. The flow simplifies from layout → color application → render to layout → Ink creation → Canvas shape-adding → render. If Issue #152 (extract shared command workflow) lands, the bridge code consolidates further into a single integration point.
