# Spiral Surface Design

## Purpose

Add an optional, topographic surface to spiral visualizations. The surface
interpolates a selected metric between the timeline's existing bucket discs,
rendering discrete palette-coloured contour bands beneath the guide track,
discs, and labels.

The first integration is the spiral visualization. Bubbletree is explicitly
out of scope because its densely clustered nodes would not produce a useful
surface. The surface generation package will be visualization-agnostic so
scatter and radial tree can adopt it later.

## User Experience

### Surface appearance

- The surface is an annulus: it appears only in the data-supported region
  occupied by spiral nodes. The empty central region and the area beyond the
  outer spiral remain the normal background.
- It is composed of flat-filled triangles. Each triangle is assigned one
  discrete palette colour, producing contour bands rather than a smooth
  gradient.
- The existing guide track, discs, and labels all render above the surface
  unchanged.
- A triangle's hairline border is its fill colour, eliminating anti-aliased
  seams between adjacent triangles.

### Metrics, palettes, and legends

- By default, the enabled surface uses the spiral fill metric, its palette,
  and its exact bucket boundaries. A disc and its surrounding matching
  contour band therefore use the same colour.
- A surface may instead use a distinct metric and optional palette. Its
  values have independent bucket boundaries and palette mapping.
- When the surface metric is distinct from the disc fill metric, the legend
  includes an additional entry for that metric. When it matches the fill
  metric, the existing fill legend is reused and no duplicate is added.

### CLI and configuration

Add these spiral command options:

```text
--surface
    Enable the surface using the configured fill metric, palette, and
    bucket boundaries.

--surface-metric <metric[,palette]>
    Enable the surface using this metric and optional palette instead.
```

`--surface-metric` implies `--surface`. A surface requires a usable numeric
or measure metric; enabling it without a fill metric and without
`--surface-metric` is a validation error. Surface configuration is persisted
under `spiral.surface`, with an enable flag and optional `MetricSpec`, and
uses the existing config merge and CLI override conventions.

## Architecture

### `internal/surface`

Introduce a visualization-independent package that accepts original
positioned metric points, an explicit rendering region, and returns the
filled triangles to render:

```go
type Point struct {
    X, Y  float64
    Value float64
}

type Triangle struct {
    Points [3]Point
    Band   int
}

type Region interface {
    Contains(x, y float64) bool
    Bounds() Rect
}
```

The package owns sampling, triangulation, interpolation, boundary filtering,
and band assignment. It does not know about spiral buckets, canvas layers,
palettes, or legends. The spiral package supplies original node coordinates,
resolved metric values, and an annular region centred at `CX, CY`, with inner
radius `A` and outer radius `A + B*MaxTheta`. It maps triangle bands to
colours using its existing ink/bucket machinery, and emits canvas polygons.

### Canvas polygon support

Add a `canvas.Polygon` shape and specification, backed by:

```go
DrawPolygon(points []Position, fill Fill, border color.RGBA, borderWidth float64)
```

on `canvas/model.Backend`. Implement it in raster and SVG backends. The
surface polygons use solid fills; the generic primitive preserves the
existing `Fill` abstraction for future callers.

Add `LayerSurface = 5`, between `LayerBackground = 0` and
`LayerStructure = 10`. The spiral renderer writes surface triangles to that
layer before adding the existing guide track.

## Mesh Generation

### Constants

```go
const (
    maxTriangleEdge    = 8.0 // pixels
    poissonMinDistance = 4.0 // pixels
    idwPower           = 2.0
)
```

`maxTriangleEdge` is the hard geometry constraint. The Poisson minimum
distance creates organic blue-noise infill rather than visible directional
lattice artifacts.

### Algorithm

1. Start from the original spiral-node positions. These are the only points
   that carry observed metric values.
2. Generate infill points by Poisson-disk sampling within the supplied
   `Region`. Seed the spatial index with every original point, and reject
   candidates closer than `poissonMinDistance` to any accepted or original
   point.
3. Run one Delaunay triangulation over originals and accepted infill points.
   Use `github.com/fogleman/delaunay`.
4. Discard a triangle if any of its edges exceeds `maxTriangleEdge`. This
   removes triangles spanning the empty central core. Also discard triangles
   whose centroid is outside the supplied region, preserving its inner and
   outer boundaries.
5. Assign values to vertices:
   - Original vertices retain their observed metric value.
   - Infill vertices receive an inverse-distance-weighted average from the
     nearest original points, using `idwPower`.
6. Calculate a triangle's value as the arithmetic mean of its three vertex
   values. Map that value through the selected metric's existing bucket
   boundaries to select its discrete colour band.

The sampler must produce a sufficiently dense point set that all accepted
triangles satisfy the 8-pixel maximum. If a boundary triangle still exceeds
the limit, it is discarded rather than rendered.

## Pipeline Integration

1. Extend spiral config, command parsing, validation, and metric resolution
   for the surface settings.
2. Include the effective surface metric in requested-metric collection,
   aggregation, and provider resolution.
3. Build a surface-specific numeric mapper only when its metric differs from
   the fill metric; otherwise reuse the fill ink's mapper and bucket
   boundaries.
4. After layout and before `RenderToCanvas`, convert spiral nodes to original
   surface points and generate the mesh.
5. Render generated triangles on `LayerSurface`, then retain the existing
   background, track, discs, and label calls and their order.
6. Extend legend construction with the distinct surface metric only when
   needed.

Surface generation is disabled by default, preserving all current spiral
output when no surface option is supplied.

## Error Handling

- Reject `--surface` when neither a fill metric nor a surface metric is
  configured.
- Validate an explicit surface metric and palette using the existing
  `MetricSpec` validation path.
- Reject categorical surface metrics because contour bands require a numeric
  quantity or measure.
- If too few original points can form a mesh, skip surface rendering and log
  a clear warning; continue rendering the normal spiral rather than emitting
  a malformed partial surface.

## Testing

### `internal/surface` unit tests

- Poisson-disk samples satisfy the configured minimum-distance constraint,
  including distance from originals.
- Accepted triangles have no edge longer than `maxTriangleEdge`.
- Triangles crossing the spiral core or external boundary are excluded.
- Original vertices retain their observed values.
- IDW results for infill vertices match known weighted examples.
- Triangle values map to expected discrete bands.

### Integration and rendering tests

- CLI/config tests cover default enablement, metric override, validation, and
  config merge behaviour.
- Spiral stage tests verify requested-metric collection, shared fill mapper
  reuse, and independent surface mapper creation.
- Canvas raster and SVG tests cover filled polygon geometry and seam-closing
  borders.
- Goldie snapshots cover spiral output with a shared fill/surface metric and
  a distinct surface metric, in PNG and SVG, including the additional legend.

## Non-goals

- Surface support for scatter, radial tree, or other visualizations in this
  change.
- Bubbletree surface support.
- Smooth gradients, contour-line overlays, or configurable mesh density.
- Per-pixel rasterization, which would not preserve SVG vector output.
