# Surface Palette-Band Subdivision Design

## Purpose

Improve the visual quality of discrete topographic surfaces. Instead of
assigning one average metric value to a complete mesh triangle, preserve the
linearly interpolated metric field across that triangle and split it at every
numeric palette breakpoint. Each resulting polygon receives exactly one
palette-band colour.

The geometry is reusable by every visualization that adopts
`internal/surface`; spiral is the first consumer. Bubbletree remains out of
scope.

## User Experience

- Surface vertices retain their precise observed or interpolated metric values.
- A numeric palette band changes exactly where the linearly interpolated metric
  crosses its bucket breakpoint on a triangle edge.
- A triangle can produce any number of convex, flat-filled fragments. This
  includes skipped palette bands when endpoint values span multiple
  breakpoints.
- In the common two-low/one-high case, the result is a high-band triangle and
  a low-band quadrilateral joined at the two interpolated edge crossings.
- Fragments have no border or contour stroke. Their shared edges form clean
  colour transitions without adding visual contour lines.
- Fixed and categorical surface inks retain the current single-colour triangle
  output because they have no ordered numeric breakpoints.
- Existing background-colour region-boundary overlays remain in place after
  all surface fragments render, retaining pixel-smooth outer and inner
  boundaries.

## Architecture

### `internal/surface`

Add a package-level geometry operation conceptually shaped as:

```go
type Polygon struct {
    Points []Point
    Value  float64
}

func SubdivideTriangle(triangle Triangle, breakpoints []float64) []Polygon
```

The exact exported names may follow local naming conventions, but the
interface has these responsibilities:

- Accept one valid triangle and strictly increasing, finite numeric
  breakpoints.
- Treat `Point.Value` as a scalar field that varies linearly through the
  triangle.
- Clip the triangle into one polygon for every intersected bucket interval.
- Return deterministic polygons in ascending band order, each with a
  representative metric value that resolves to that interval through the
  existing bucket semantics.
- Omit degenerate zero-area results and never mutate the source triangle.

`internal/surface` remains independent of palettes, inks, canvas, and
visualization-specific data. It owns only the geometry and scalar
interpolation.

### `internal/inks`

Provide a narrow public way to obtain numeric bucket breakpoints from an
`Ink`. It returns no breakpoints for fixed or categorical inks. The accessor
must expose the exact `metric.BucketBoundaries.Boundaries` used by
`numericInk`, so geometry and colour mapping agree at every threshold without
duplicating bucketing logic in visualizations.

### Visualization renderers

A surface-enabled renderer:

1. Determines whether its surface ink is numeric and, if so, obtains the
   numeric breakpoints.
2. Calls the reusable surface subdivision operation for each mesh triangle.
3. Emits one `canvas.Polygon` for each returned fragment on its surface layer,
   resolving the fragment value through the existing surface ink.
4. Sets fragment border width to zero.
5. For a non-numeric ink, emits the existing one polygon for the triangle
   using its average value.

Spiral keeps its current layer ordering: background, surface fragments,
background-colour boundary overlays, guide track, discs, then labels.

## Subdivision Algorithm

For every triangle:

1. Use each breakpoint to define numeric palette intervals matching
   `BucketIndex`: values below a breakpoint belong to the lower interval, and
   a value equal to a breakpoint belongs to the upper interval.
2. Clip the triangle's linearly interpolated scalar field to each intersected
   interval. Intersections on an edge use:

   ```text
   t = (breakpoint - valueA) / (valueB - valueA)
   point = A + t * (B - A)
   ```

3. Assign the fragment a representative value in its interval. For an
   intermediate interval this can be a finite value strictly inside its bounds;
   the first and final intervals use an existing in-interval vertex value when
   needed. The representative must resolve through the same numeric ink to the
   intended bucket.
4. Discard fragments with fewer than three distinct points, non-finite
   coordinates or values, or zero area.

The implementation must handle breakpoint equality, repeated corner values,
and an edge that crosses multiple breakpoints. Invalid or unsorted caller
breakpoints produce no subdivision rather than malformed output.

## Error Handling and Compatibility

- Mesh generation, scalar interpolation, and region handling are unchanged.
- A triangle with no crossed breakpoints produces one polygon in its existing
  bucket.
- Fixed and categorical inks preserve the current fallback rendering path.
- A malformed numeric breakpoint list is rejected by the geometry helper;
  renderers continue with the original triangle rather than rendering a
  partial surface.
- The implementation adds no new user configuration or CLI options.

## Testing

### Surface geometry tests

- One breakpoint with two vertices in the low band and one in the high band
  produces the expected triangle and quadrilateral, including exact edge
  crossing coordinates.
- Multiple breakpoints on one edge produce ordered fragments for every crossed
  band.
- A vertex exactly on a breakpoint follows `BucketIndex` upper-band semantics.
- Equal vertex values, degenerate input, invalid values, invalid breakpoint
  sequences, and zero-area fragments are safely handled.
- Results are deterministic and leave the input triangle unchanged.

### Ink tests

- Numeric inks expose their exact computed bucket boundaries.
- Fixed and categorical inks report that no numeric breakpoints are available.

### Spiral rendering tests

- A numeric surface triangle crossing one and several bands emits the expected
  number of polygons, in the expected colours and layer order.
- Generated fragments have no borders.
- Non-numeric ink rendering remains one polygon per triangle.
- Existing boundary paths remain after all surface polygons and before the
  foreground.
- Update PNG and SVG Goldie snapshots to show the cleaner banded surface.

## Non-goals

- Continuous colour gradients or per-pixel shading.
- Visible contour-line strokes between palette bands.
- Changing surface metrics, palettes, mesh density, or boundary geometry.
- Adding surfaces to other visualizations in this change.
