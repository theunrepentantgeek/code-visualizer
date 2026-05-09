# Canvas Abstraction Design

A drawing abstraction for `code-visualizer` that replaces duplicated per-visualization rendering code with a shared Canvas API.

## Problem

The current rendering layer has four visualization types (treemap, radial tree, bubble tree, spiral), each with paired raster (PNG/JPG via `fogleman/gg`) and SVG (direct XML) renderers.
This produces ~4600 lines in `internal/render/` with near-identical patterns for color resolution, shape drawing, and format dispatch.
The `cmd/codeviz/` layer adds another ~800 lines of duplicated `applyFillColours`/`applyBorderColours` logic.
Layout node types (`TreemapRectangle`, `RadialNode`, `BubbleNode`, `SpiralNode`) each carry `FillColour color.RGBA` and `BorderColour *color.RGBA` fields, embedding resolved color data into geometry.

PARKER: I verified the numbers. `internal/render/` has ~1718 lines across 10 files that would be replaced, plus another ~2576 lines across the 4 `*_cmd.go` files, a significant chunk of which is the duplicated `applyFillColours`/`applyBorderColours` pattern (each viz type has its own copy with only the node type varying). There are currently **twenty** `nolint:dupl` annotations in the cmd files — the linter has been telling us about this for a while. The problem statement is accurate and the duplication is real. This isn't a premature abstraction; it's a pattern that's been proven across four implementations.

## Approach

Introduce a new `internal/canvas/` package with:

- **Ink** — a smart colour resolver that maps metric values to RGBA via palette + mapping strategy.
- **Specs** — CSS-like templates that bundle inks with visual properties (line width, opacity, label style).
- **Canvas** — a retained-then-render drawing surface with layered z-ordering and two backend implementations (raster, SVG).

Layout node types become geometry-only (no color fields).
Visualizations set up Inks and Specs, add shapes to the Canvas with metric values, and call `Render()`.

PARKER: The three-layer decomposition (Ink → Spec → Canvas) is the right level of abstraction. Each layer has a clear single job: Ink resolves colours, Spec bundles visual style, Canvas handles drawing + z-order. This should hold up well as more viz types arrive — a 5th visualization becomes "write layout, create inks, fill canvas, call Render()." No new renderer code needed. That's a solid improvement over today where adding a viz means writing both PNG and SVG renderers from scratch (~300-400 lines each).

BISHOP: Strong approach. The three-layer decomposition (Ink → Spec → Canvas) correctly separates concerns: value resolution, style composition, and drawing. This directly addresses the duplication I catalogued in Issue #158 (raster/SVG unification) and eliminates the ~800 lines of `applyFillColours`/`applyBorderColours` from `cmd/codeviz/`. Stripping color fields from layout nodes fixes the "embedding resolved data into geometry" smell — layout should own geometry, not presentation. This is the Template Method pattern applied at a data-structure level: the Spec is the template, the shape supplies the variable input.

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

BEVAN: I think we need to allow for future evolution by including options for both kinds of Ink.

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

PARKER: Performance-wise, `Dip()` is fine. The current `BucketIndex()` does a linear scan through 9-13 boundaries — that's ~13 comparisons per shape. Even for a 100k-file repo that's 1.3M comparisons, well under a millisecond. The real work is in the I/O and layout math. Whether colour resolution happens during `applyFillColours` (now) or during `Render()` (proposed) doesn't change the total computation — it just moves when it happens. No concern here.

PARKER: One thing to consider: `NumericInk` takes `[]float64` which means the caller collects all values upfront for distribution analysis. The current code already does this (`collectNumericValues` walks the tree). So the data flow doesn't change, it just moves behind a cleaner API. Good.

BISHOP: The Ink type is the single best abstraction in this spec. It consolidates a pipeline that's currently scattered across three packages (`metric.ComputeBuckets` → `metric.BucketBoundaries.BucketIndex` → `palette.MapNumericToColour` for numeric; `palette.NewCategoricalMapper` → `CategoricalMapper.Map` for categorical). The metaphor is excellent — you "dip" a pen in ink to get a color, and the ink knows its palette. This is the Façade pattern done right: it hides multi-step orchestration behind a single method call.

BISHOP: However, the dual `Dip(float64)` / `DipCategory(string)` API is a **type-level code smell**. The caller must know at call-time which kind of Ink they're holding, which defeats part of the encapsulation benefit. If a visualization could swap between numeric and categorical fill inks (the same way the current `applyFillColours` dispatches on `metric.Kind`), the caller would need an `if ink.IsNumeric()` branch — re-creating the exact dispatch logic Ink is supposed to absorb. Bevan's `MetricValue` suggestion is the right fix: a single `Dip(MetricValue)` method lets Ink own the dispatch internally. I'd model `MetricValue` as:

```go
type MetricValue struct {
    Kind     metric.Kind
    Measure  float64
    Quantity int
    Category string
}
```

This also future-proofs for mixed-kind metrics and eliminates the `FillValue`/`FillCategory`/`BorderValue`/`BorderCategory` data clump on shapes (addressed below).

BISHOP: One more concern on Ink: it's a **value type** (struct, not pointer), which is correct for the `FixedInk` case but potentially expensive for `NumericInk` — the `boundaries *metric.BucketBoundaries` and `catMapper *palette.CategoricalMapper` fields are already pointer types, so copies share the underlying data. This is fine, but document this explicitly so future developers don't accidentally refactor the internal pointer fields to values, which would break the copy-is-cheap property. Consider a constructor-level comment: "Ink is safe to copy; internal state is shared."

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

BEVAN: I think Opacity should be part of an ink, configurable with a new InkOption - WithOpacity(). This will eliminate any confusion about what Opacity means on RectangleSpec or DiscSpec.

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

PARKER: `RectangleSpec` and `DiscSpec` are structurally identical — both have Fill, Border, BorderWidth, Opacity, ShowLabel, LabelInk, and LabelStyle. That's seven shared fields. When someone adds a new property (say, dash pattern or corner radius), they'll need to remember to add it to both. I'd extract a `ShapeStyle` struct with the common fields and embed it in both specs. Not critical for v1, but flag it so the first person who touches both specs knows to refactor.

PARKER: Agree with Bevan on moving Opacity into Ink via `WithOpacity()`. The current spec has Opacity on the shape but not on the Ink, which creates a confusing question: if a FixedInk produces `#FF0000` and the spec says `Opacity: 0.5`, is the final colour semi-transparent red? What if someone later adds opacity to Ink too — do they multiply? Putting opacity in one place (the Ink) eliminates that ambiguity entirely.

PARKER: `TextSpec` has no font family field. The current code hardcodes `sans-serif` for SVG and `goregular` for raster. That's fine for now, but the spec should note that font choice is intentionally fixed, not accidentally omitted. Otherwise someone will add it as a field, and then you're carrying font configuration through the whole pipeline for no real benefit.

BISHOP: **Data clump alert.** `RectangleSpec` and `DiscSpec` share 6 of 7 fields verbatim: `Fill`, `Border`, `BorderWidth`, `Opacity`, `ShowLabel`, `LabelInk`, `LabelStyle`. The only difference is the name. This is the classic "parallel types" smell — if you add a new visual property (e.g., a shadow or gradient), you must update both types identically. Extract the shared fields into an embedded base:

```go
type ShapeStyle struct {
    Fill        Ink
    Border      Ink
    BorderWidth float64
    ShowLabel   bool
    LabelInk    Ink
    LabelStyle  LabelStyle
}

type RectangleSpec struct {
    ShapeStyle
}

type DiscSpec struct {
    ShapeStyle
}
```

This also prepares for Bevan's opacity-to-Ink migration — once `Opacity` moves into `Ink` via `WithOpacity()`, the two specs become structurally identical except by name. The embedded `ShapeStyle` lets you add disc-specific or rectangle-specific fields later without losing the shared contract.

BISHOP: On Bevan's opacity note — I strongly agree. Opacity is a property of how an ink renders, not of the shape template. The current bubble renderer uses `bubbleDirAlpha = 0x30` on directory circles, which is logically "this fill ink is translucent." Moving it to `WithOpacity(0.18)` on the Ink itself means a FixedInk with alpha is just `FixedInk(color, WithOpacity(0.18))` — clean and self-documenting. The spec's `Opacity` field on specs describes a property of the *ink application*, not the *shape*, so it belongs on the Ink.

BISHOP: The CSS-class analogy for Specs is apt and well-chosen. This is the Flyweight pattern: many shapes share one Spec, reducing per-shape memory overhead and ensuring visual consistency. The spec mentions "shapes can override specific inks per-instance (the CSS-class-with-inline-override pattern)" but I don't see override fields on the shape structs themselves. If you want per-shape fill overrides, consider an `OverrideFill *Ink` field on `Rectangle`/`Disc`. Otherwise, drop the override claim from this section — you can always add it later without breaking the API.

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

BISHOP: Inferring format from the file extension is consistent with the current `render.FormatFromPath()` pattern — good continuity. However, coupling construction to a filesystem path means test code must invent fake paths to control format. Consider also accepting a `render.ImageFormat` directly: `NewCanvasWithFormat(width, height int, format ImageFormat)` for tests, with `NewCanvas` as the convenience wrapper. This costs one extra constructor but makes the Canvas fully testable without filesystem assumptions. The Canvas is a retained drawing surface — it shouldn't need to know where it's going until `Render()` time.

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

BEVAN: We are seeing the pair of Value+Category showing up a lot, and it's polluting the object model with the constant replication. I suggest we introduce MetricValue to wrap everything needed - containing (metric.Kind, Value float64, Category string). We may even want to split Measures and Quantities into separate fields (metric.Kind, Measure float64, Quantity int, Category string). We can then simplify Ink by combining the two Dip methods into one - Dip(MetricValue).

BISHOP: **Strongly endorse the `MetricValue` extraction.** Each shape currently carries 4 fields for what is logically 2 metric inputs (fill + border). With `MetricValue`, `Rectangle` shrinks from 10 fields to 7:

```go
type Rectangle struct {
    Spec     *RectangleSpec
    X, Y, W, H float64
    Fill     MetricValue  // for spec.Fill.Dip()
    Border   MetricValue  // for spec.Border.Dip()
    Label    string
}
```

This eliminates the "stringly typed" dispatch — the caller doesn't need to know whether to set `FillValue` or `FillCategory`, they just set `Fill`. The Ink's `Dip(MetricValue)` dispatches on `Kind` internally. This is the **Replace Primitive with Value Object** refactoring (Fowler), and it applies to all four shape types. For `FixedInk`, `Dip` simply ignores the `MetricValue` — no behavioral change.

BISHOP: One additional shape design issue: `Disc` carries an `Angle` field for label orientation, but this is a rendering concern, not geometry in the general sense. However, looking at the current `RadialNode` and `SpiralNode`, both carry `Angle` because label rotation depends on the node's angular position. This is legitimate — the angle *is* geometric, and the label renderer needs it. Leave as-is.

PARKER: Strongly agree with Bevan's MetricValue suggestion. Right now every shape type carries `FillValue float64`, `BorderValue float64`, `FillCategory string`, `BorderCategory string` — that's four fields per shape that exist solely to carry metric data in two different representations. The current codebase already has this split between `Quantity`/`Measure`/`Classification` in the model layer. A `MetricValue` struct that unifies these would cut the field count in half on shapes, eliminate the two-method `Dip`/`DipCategory` split on Ink, and make it impossible to accidentally pass a float to a categorical ink or vice versa. This is the kind of type-safety that prevents bugs in a codebase with multiple contributors. Do this before v1, not after — retrofitting it means changing every shape type and every call site.

### Drawing Methods

```go
func (c *Canvas) AddRectangle(layer Layer, r Rectangle)
func (c *Canvas) AddDisc(layer Layer, d Disc)
func (c *Canvas) AddLine(layer Layer, l Line)
func (c *Canvas) AddPath(layer Layer, p Path)
func (c *Canvas) Render() error
```

BISHOP: The `Add*` + `Render()` API is the **Command pattern** (retained drawing): shapes are recorded as data, then executed in batch. This is the right pattern because it enables z-ordering without requiring callers to draw in layer order — a direct fix for the current multi-pass rendering in `bubbletree.go` (dirs → files → labels) and `radialtree.go` (edges → discs → labels). The four `Add*` methods are appropriately specific — a `Shape` interface with a polymorphic `Add(Shape)` would save one method signature but add a type-switch inside Canvas, trading surface area for internal complexity. At four shape types, concrete methods win.

BISHOP: `Render() error` should accept the output path, not take it at construction time. A Canvas is a data structure; `Render` is the "flush to disk" operation. Signature: `Render(outputPath string) error`. This also means you can render the same Canvas to multiple formats (PNG + SVG of the same visualization) without reconstructing it — a useful capability for testing and for future "export all formats" features. If you keep path-at-construction, at minimum `Render` should still take a path to allow overriding.

BEVAN: Agreed - let's move the output path to `Render()`. We may need to pass an enum to the constructor to specify which backend to use.

BISHOP: Consider adding `AddText(layer Layer, t Text)` as a first-class shape — the current code uses standalone text for legend entries, axis labels, and titles. The `TextSpec` type is defined but no `Text` shape struct or `AddText` method exists. This looks like an accidental omission.

BEVAN: Agreed.

PARKER: The retained shape list deserves a memory estimate. Each `Rectangle` is ~112 bytes (8 float64s + 4 strings + pointer), `Disc` similar. For a 100k-file repo, that's ~11MB of shapes in memory at render time. Totally fine for a CLI tool. Even at 1M files (~110MB) it's manageable. This isn't a web server holding shapes in perpetuity — the Canvas lives for one render call and gets GC'd. No concern.

PARKER: Consider adding `AddText(layer Layer, t Text)` as a first-class shape method. The current spec handles text implicitly through specs with `ShowLabel: true`, but the radial and spiral renderers also draw standalone text (root labels, external labels) that isn't tied to a shape spec. Without `AddText`, those become awkward to express.

BEVAN: Agreed.

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

BISHOP: The layer system with integer gaps (0, 10, 20, 30) is the **Sparse Namespace** pattern — gaps leave room for insertion without renumbering. This is pragmatic and correct. The mapping table is well-thought-out: it correctly captures that the current multi-pass rendering in `bubbletree.go` (draw dir circles large→small, then file circles, then labels) becomes simply "add everything with the right layer, let Canvas sort." The "within a layer, insertion order" rule is important and well-specified — it's how the bubble tree's large-to-small directory circle ordering is preserved.

PARKER: The layer system is a clean replacement for the current multi-pass rendering pattern. Today each renderer does 2-3 explicit passes (e.g., edges → discs → labels in radial). The Canvas replaces those passes with layer numbers — same z-ordering, but the Canvas handles the sorting. Good tradeoff: callers declare intent, Canvas handles mechanics.

PARKER: The 10-unit gaps between layer constants (0, 10, 20, 30) leave room for future insertion. Smart — when someone needs a layer between Structure and Content (say, for a background gradient behind file shapes), they can use `Layer(15)` without renumbering. Just document that these gaps are intentional.

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

BISHOP: **The `LegendEntry` is too thin.** The current `render.LegendEntry` (in `legend.go`) carries `role`, `metricName`, `kind`, `buckets`, `palette`, and `categories` — all needed to render the gradient swatches, bucket boundary labels, and category legends. The spec's version only carries `Role` and `Ink`. The Ink encapsulates the palette and bucketing internally (unexported fields), but the legend renderer needs to *iterate* the bucket boundaries to draw individual swatches and format breakpoint labels (see `drawNumericSwatches` in `legend_png.go`). Either: (a) expose query methods on Ink (`Boundaries() []float64`, `Palette() ColourPalette`, `Categories() []string`) so the legend can introspect it, or (b) keep `LegendEntry` richer with explicit fields. Option (a) is cleaner — it keeps the Ink as the single source of truth and avoids data duplication between Ink and LegendEntry. But it does expand Ink's surface area beyond pure resolution. This is a trade-off worth making explicit in the design.

BISHOP: Also, `Role` as a plain `string` is **primitive obsession**. The valid values are a small closed set ("Fill", "Border", "Size"). Use a typed constant:

```go
type LegendRole string
const (
    LegendRoleFill   LegendRole = "Fill"
    LegendRoleBorder LegendRole = "Border"
    LegendRoleSize   LegendRole = "Size"
)
```

This prevents typos, enables exhaustive switch statements, and is consistent with how the codebase already types `LegendPosition` and `LegendOrientation` as typed strings.

The existing `ReserveLegendSpace()` pattern carries over — the Canvas knows the legend dimensions and can carve out space before the visualization fills the remaining area.

The legend is rendered as part of `Render()` on `LayerOverlay`.

PARKER: The legend design here is significantly simpler than the current `LegendInfo`/`LegendEntry` system in `internal/render/legend.go`. The current code carries `BucketBoundaries`, metric names, role labels, category swatches, and a 440-line measurement/drawing engine. The proposed `LegendConfig` has just Position, Orientation, and Entries. That's fine as an API, but make sure the implementation doesn't lose the breakpoint labels on numeric legends — those are useful context for reading the visualization. The spec says "Inks carry palette metadata" — confirm that includes the bucket boundaries needed for breakpoint display.

## Backend Implementations

The Canvas uses an internal backend interface.
Two implementations: raster (PNG/JPG via `fogleman/gg`) and SVG (direct XML).
The backend is chosen at Canvas creation time from the output path extension.

BEVAN: Should these implementations be segregated into subpackages underneath canvas for isolation?

BISHOP: **Yes — subpackages are the right call.** Placing the backends in `internal/canvas/raster/` and `internal/canvas/svg/` enforces that the backend interface is the *only* contract between the Canvas and its implementations. Without this, nothing prevents the raster backend from importing SVG helpers or vice versa — and in the current `internal/render/` package, exactly this kind of cross-contamination exists (shared module-level vars, ad-hoc helper reuse). Subpackages make the boundary compiler-enforced. The `backend` interface stays in the parent `canvas` package (unexported), and each subpackage exports a constructor that returns it. This is the **Ports and Adapters** pattern: the Canvas defines the port, each subpackage is an adapter.

PARKER: Pragmatically, no — not yet. Subpackages (`canvas/raster/`, `canvas/svg/`) add import path management overhead and force the backend interface to be exported (since subpackages can't access unexported interfaces in parent packages). Right now the backend interface is unexported, which is exactly right — it's an implementation detail. Keep `raster_backend.go` and `svg_backend.go` as separate files in `internal/canvas/`. The file-level separation gives enough isolation for a team this size. If the backends grow beyond ~300 lines each, or if you need to independently test them with their own test fixtures, revisit.

BEVAN: My prediction is that the backends will easily go over the 300 line limit, if not now, then in the near future. Splitting into subpackages is the right decision. 

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

BISHOP: **The backend interface surface is at the right level** — it deals in resolved colors and primitive geometry, not in Inks or Specs. This is a clean separation: the Canvas translates from the domain model (Ink, Spec, MetricValue) down to the rendering primitives (RGBA, coordinates). No implementation detail leaks upward.

BISHOP: **Long parameter lists.** `drawRectangle` takes 8 positional parameters and `drawDisc` takes 8 as well — this is the **Long Parameter List** smell. In Go, the pragmatic fix is a struct:

```go
type drawRectParams struct {
    x, y, w, h  float64
    fill, border color.RGBA
    borderWidth  float64
    opacity      float64
}
```

However, since this is an unexported interface with exactly two implementations that you control, the cost of the smell is low. If Bevan moves opacity into Ink (as recommended), `drawRectangle` drops to 7 params — still long, but tolerable for an internal interface. Flag for review during implementation; don't let it block the design.

BEVAN: A common programming mistake is to swap parameters around, this becomes a bigger problem with longer parameter lists like this. I suggest we introduce two new small structs - `Position{x,y}` and `Size{width,height}` - this serves to reduce both the number of parameters and the possibility of parameter-swap errors.

BISHOP: **`drawArcText` is oddly specific.** Every other `draw*` method is a geometric primitive (rectangle, disc, line, path, text). `drawArcText` is a composite operation — text curved along a circle. This breaks the conceptual uniformity of the interface. Consider whether arc text can be decomposed into `drawText` calls with per-character rotation (the SVG `<textPath>` approach doesn't decompose well, but the gg approach already does character-by-character arc placement in `bubble_font.go`). Alternatively, accept it as a pragmatic sixth primitive — the bubble tree needs it, and forcing decomposition would complicate both backends. I'd keep it, but document why it's there.

BISHOP: **`finish(outputPath string)` accepts the output path** — but if the Canvas constructor also takes the path (as currently specified), this is redundant. Pick one: either the Canvas owns the path and passes it to `finish()`, or `finish()` receives it and the Canvas is path-agnostic until render time. The latter is more flexible (see my note on `Render()` above).

BEVAN: As discussed above, we're removing the file path from the constructor. If we can defer selection of the back end until Render time, all the better.

PARKER: The backend interface is the riskiest part of this design, because it's the one thing that's hard to change later. Every new drawing capability (dashed lines, gradient fills, rounded rectangles, drop shadows) requires adding a method to this interface and implementing it in both backends. That's the cost of abstraction.

PARKER: The current method signatures are parameter-heavy — `drawRectangle` takes 8 arguments, `drawDisc` takes 8 arguments. That's fragile: add opacity support and every call site changes. Consider struct parameters (e.g., `drawRectangle(RectDrawCall)`) that can grow without breaking the signature. This is the difference between an interface that's painful to extend and one that accommodates change.

PARKER: `drawArcText` is a significant complexity sink. The raster implementation needs the entire `bubble_font.go` glyph-positioning engine (210 lines) — `computeGlyphPositions`, `loadBubbleFontFace`, `clampFontToArc`, `measureStringWidth`. The SVG implementation uses `<textPath>` which is a completely different mechanism. Make sure the spec acknowledges that this method carries the most implementation weight. It's the one most likely to have subtle rendering differences between backends.

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

PARKER: This is the right way to do it. Build the new package, prove it works with its own tests, ship it dark. Zero risk to existing functionality. The golden-file approach means you get pixel-level regression coverage from day one. One thing to add: write a mock/stub backend alongside the real ones for unit testing the Canvas's layer ordering and shape dispatch without rendering actual images. That lets you test "did the Canvas sort shapes into the right z-order?" without comparing PNGs.

### Stage 2: Migrate Visualizations

Migrate one visualization at a time. For each:

1. **Strip colour fields** from the layout node type (`FillColour`, `BorderColour`).
2. **Replace render functions** — delete the old `internal/render/` files for that viz type and replace with Canvas-based drawing code.
3. **Update the command** — replace `applyFillColours`/`applyBorderColours` with Ink creation and Canvas shape-adding.
4. **Run golden-file tests** — verify visual output matches (or update goldens where the output intentionally improves).

PARKER: The migration order (treemap → spiral → radial → bubble) is sound — it escalates complexity progressively. Treemap is rectangles-only, no multi-pass, no rotated text. Good warm-up. Bubble tree is last because arc text + opacity + multi-pass z-ordering makes it the hardest to get right.

PARKER: The main risk during migration is the atomic cut required for each viz. Step 2 says "strip colour fields from layout node" + "replace render functions" — those must happen in the same commit. If you strip `FillColour` from `TreemapRectangle` but haven't finished the Canvas renderer, nothing compiles. Go's type system is your friend here: the compiler will catch every missed reference. But plan each viz migration as a single PR, not a series of commits. You don't want the codebase in a half-migrated state that blocks other work.

PARKER: Consider keeping the old renderers alive (behind a build tag or a flag) for the first viz or two, until you're confident the Canvas output matches. If the golden files show unexpected deltas — and they will, because the rendering order within a layer may differ slightly from the hand-coded pass order — you want the ability to compare old vs new side by side.

Suggested migration order:

1. **Treemap** — simplest (rectangles only, no multi-pass, no rotated labels beyond centered).
2. **Spiral** — flat node list (no tree walk), uses Line/Path + Discs.
3. **Radial tree** — tree walk, edges + discs + rotated labels.
4. **Bubble tree** — most complex (arc text, opacity, multi-pass z-ordering).

PARKER: Worth noting the suggested order matches the current line counts: treemap PNG+SVG = ~258 lines, spiral = ~381, radial = ~419, bubble = ~403 + 210 for bubble_font.go. The complexity gradient is real.

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

PARKER: `label.go` has `ShouldShowLabel` and `TextColourFor` — these are shared logic that the Canvas internals will need. `format.go` has `FormatFromPath` — the Canvas's `NewCanvas` will need this. `save.go` has `saveContextPNG`/`saveContextJPG` — the raster backend's `finish()` will absorb these. Be explicit about which files move into `internal/canvas/` vs which get deleted. Ambiguity here leads to orphaned code or missing utilities.

PARKER: **Overall assessment.** This is a well-motivated, well-scoped design. The duplication it eliminates is real (~1700 lines of render code + ~800 lines of colour application), the API surface is appropriate (not over-engineered), and the migration path is realistic with golden-file tests as the safety net. The complexity budget is justified — the Canvas package will probably land at ~1000-1500 lines, which is a net reduction even before you count the ongoing cost of maintaining eight parallel renderers. My main concerns are: (1) use struct parameters for backend methods to keep the interface extensible, (2) address the MetricValue unification before v1 to avoid a second migration, and (3) acknowledge that `drawArcText` is the hard part and budget implementation time accordingly. Ship it.

BISHOP: **The migration path is well-ordered.** Treemap → Spiral → Radial → Bubble is correct complexity ordering — treemap is rectangles-only with no multi-pass, bubble tree is the hardest (arc text, alpha compositing, multi-pass z-ordering that the layer system must replicate exactly). The "one viz at a time" approach means the old and new code coexist during migration, which is safe because each viz's render path is already independent (separate files, no shared mutable state).

BISHOP: **Watch `label.go` carefully.** `TextColourFor(fill color.RGBA)` (contrast-aware label color selection) is a cross-cutting concern that the Canvas should absorb. When a shape has `ShowLabel: true`, the Canvas needs to determine label color from the resolved fill. This logic should live in the canvas package alongside Ink, not in a leftover utility file. `ShouldShowLabel` is treemap-specific (minimum-size threshold) and should either generalize or stay viz-specific.

BISHOP: **One structural risk not addressed:** the spec doesn't mention where the "visualization-to-Canvas bridge" code lives. Currently, `cmd/codeviz/treemap_cmd.go` does layout → color application → render. Post-migration, it does layout → Ink creation → Canvas shape-adding → render. That bridge code is simpler but still per-visualization. If Issue #152 (extract shared command workflow) lands first, the Canvas integration has a single place to wire into. If not, you'll create 4 new bridge implementations and then refactor them when #152 lands. Consider sequencing #152 before Stage 2.
