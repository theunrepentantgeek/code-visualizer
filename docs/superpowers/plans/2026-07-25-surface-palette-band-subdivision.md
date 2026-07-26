# Surface Palette-Band Subdivision Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Render numeric surface mesh triangles as borderless polygons split exactly at palette bucket breakpoints, preserving discrete palette contours without average-colour triangle patches.

**Architecture:** `internal/surface` will own deterministic scalar-field clipping and return flat polygon fragments with representative values. `internal/inks` will expose the numeric ink's exact bucket boundaries without leaking palette logic into surface geometry. Spiral will call these two APIs, emit all fragments on `LayerSurface`, and preserve its existing boundary-overlay and foreground ordering.

**Tech Stack:** Go 1.26, Gomega, Goldie v2, fogleman/gg raster output, SVG backend.

---

## File Structure

| File | Responsibility |
| --- | --- |
| `internal/surface/types.go` | Add the reusable `Polygon` geometry result type. |
| `internal/surface/subdivide.go` | Clip one linearly interpolated scalar triangle to palette-band intervals. |
| `internal/surface/subdivide_test.go` | Unit-test exact crossings, multi-band fragments, equality, invalid input, and determinism. |
| `internal/inks/introspection.go` | Add a public numeric-breakpoint accessor to the ink package. |
| `internal/inks/introspection_test.go` | Verify exact numeric-boundary exposure and non-numeric fallback. |
| `internal/spiral/render.go` | Render numeric triangles as borderless band fragments; retain the existing fallback for other inks. |
| `internal/spiral/surface_test.go` | Verify rendered fragments, colours, zero borders, fallback output, and layer ordering. |
| `internal/goldentest/testdata/spiral-surface-*.golden` | Update committed PNG/SVG snapshots for the improved surface output. |

### Task 1: Expose Numeric Bucket Breakpoints

**Files:**
- Modify: `internal/inks/introspection.go`
- Modify: `internal/inks/introspection_test.go`

- [ ] **Step 1: Write failing ink introspection tests**

Add these tests to `internal/inks/introspection_test.go`, using the existing
Gomega test style and imports:

```go
func TestNumericBreakpoints_ReturnsNumericInkBoundaries(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    values := []float64{1, 2, 3, 4, 5}
    pal := palette.GetPalette(palette.Temperature)
    ink := inks.NumericInk(
        "metric",
        values,
        pal,
    )

    expected := metric.ComputeBuckets(values, len(pal.Colours)).Boundaries
    g.Expect(inks.NumericBreakpoints(ink)).To(Equal(expected))
}

func TestNumericBreakpoints_ReturnsNilForNonNumericInks(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    g.Expect(inks.NumericBreakpoints(inks.FixedInk(color.RGBA{}))).To(BeNil())
    g.Expect(inks.NumericBreakpoints(
        inks.CategoricalInk("metric", []string{"go"}, palette.GetPalette(palette.Categorization)),
    )).To(BeNil())
}
```

Import `internal/metric` so the assertion uses the repository's canonical
bucket calculation rather than duplicating palette-dependent expectations.

- [ ] **Step 2: Run the focused ink tests and confirm they fail**

Run:

```bash
go test ./internal/inks -run 'TestNumericBreakpoints' -count=1
```

Expected: compilation failure because `inks.NumericBreakpoints` does not yet
exist.

- [ ] **Step 3: Implement the narrow breakpoint accessor**

In `internal/inks/introspection.go`, add:

```go
// NumericBreakpoints returns a copy of the numeric bucket breakpoints used by ink.
// It returns nil when ink does not resolve ordered numeric values.
func NumericBreakpoints(ink Ink) []float64 {
    numeric, ok := ink.(*numericInk)
    if !ok {
        return nil
    }

    return append([]float64(nil), numeric.boundaries.Boundaries...)
}
```

Keep `numericInk` unexported and do not add bucket or palette dependencies to
the `Ink` interface. The returned copy prevents callers from mutating ink
state.

- [ ] **Step 4: Strengthen the test for copy safety**

Extend `TestNumericBreakpoints_ReturnsNumericInkBoundaries`:

```go
breakpoints := inks.NumericBreakpoints(ink)
breakpoints[0] = -1
g.Expect(inks.NumericBreakpoints(ink)[0]).NotTo(Equal(-1.0))
```

- [ ] **Step 5: Run the focused ink tests and format**

Run:

```bash
gofumpt -w internal/inks/introspection.go internal/inks/introspection_test.go
go test ./internal/inks -run 'TestNumericBreakpoints' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit the accessor**

```bash
git add internal/inks/introspection.go internal/inks/introspection_test.go
git commit -m "feat(inks): expose numeric bucket breakpoints"
```

### Task 2: Define and Test Surface Palette-Band Geometry

**Files:**
- Modify: `internal/surface/types.go`
- Create: `internal/surface/subdivide.go`
- Create: `internal/surface/subdivide_test.go`

- [ ] **Step 1: Add the result type**

In `internal/surface/types.go`, after `Triangle`, add:

```go
// Polygon is a flat-filled fragment of a scalar triangle.
type Polygon struct {
    Points []Point
    Value  float64
}
```

`Value` must resolve to the palette interval occupied by the polygon.

- [ ] **Step 2: Write failing tests for the common two-low/one-high case**

Create `internal/surface/subdivide_test.go` with:

```go
package surface_test

import (
    "testing"

    . "github.com/onsi/gomega"

    "github.com/theunrepentantgeek/code-visualizer/internal/surface"
)

func TestSubdivideTriangle_SplitsTwoLowOneHighVerticesAtBreakpoint(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    triangle := surface.Triangle{Points: [3]surface.Point{
        {X: 0, Y: 0, Value: 0},
        {X: 4, Y: 0, Value: 0},
        {X: 0, Y: 4, Value: 2},
    }}

    polygons := surface.SubdivideTriangle(triangle, []float64{1})

    g.Expect(polygons).To(HaveLen(2))
    g.Expect(polygons[0].Points).To(ConsistOf(
        surface.Point{X: 0, Y: 0, Value: 0},
        surface.Point{X: 4, Y: 0, Value: 0},
        surface.Point{X: 2, Y: 2, Value: 1},
        surface.Point{X: 0, Y: 2, Value: 1},
    ))
    g.Expect(polygons[1].Points).To(ConsistOf(
        surface.Point{X: 0, Y: 2, Value: 1},
        surface.Point{X: 2, Y: 2, Value: 1},
        surface.Point{X: 0, Y: 4, Value: 2},
    ))
    g.Expect(polygons[0].Value).To(BeNumerically("<", 1))
    g.Expect(polygons[1].Value).To(BeNumerically(">=", 1))
}
```

The points are asserted as a set because winding direction is an
implementation detail; retain deterministic ascending-band output by asserting
the lower polygon first.

- [ ] **Step 3: Add failing tests for multiple bands and boundary semantics**

Add:

```go
func TestSubdivideTriangle_ReturnsEveryCrossedPaletteBand(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    triangle := surface.Triangle{Points: [3]surface.Point{
        {X: 0, Y: 0, Value: 0},
        {X: 6, Y: 0, Value: 0},
        {X: 0, Y: 6, Value: 3},
    }}

    polygons := surface.SubdivideTriangle(triangle, []float64{1, 2})

    g.Expect(polygons).To(HaveLen(3))
    g.Expect(polygons[0].Value).To(BeNumerically("<", 1))
    g.Expect(polygons[1].Value).To(And(BeNumerically(">=", 1), BeNumerically("<", 2)))
    g.Expect(polygons[2].Value).To(BeNumerically(">=", 2))
    g.Expect(polygons[1].Points).To(ContainElement(surface.Point{X: 0, Y: 2, Value: 1}))
    g.Expect(polygons[1].Points).To(ContainElement(surface.Point{X: 0, Y: 4, Value: 2}))
}

func TestSubdivideTriangle_AssignsBreakpointVertexToUpperBand(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    triangle := surface.Triangle{Points: [3]surface.Point{
        {X: 0, Y: 0, Value: 0},
        {X: 2, Y: 0, Value: 1},
        {X: 0, Y: 2, Value: 1},
    }}

    polygons := surface.SubdivideTriangle(triangle, []float64{1})

    g.Expect(polygons).To(HaveLen(2))
    g.Expect(polygons[1].Points).To(ContainElements(
        surface.Point{X: 2, Y: 0, Value: 1},
        surface.Point{X: 0, Y: 2, Value: 1},
    ))
}
```

- [ ] **Step 4: Add failing defensive and compatibility tests**

Add:

```go
func TestSubdivideTriangle_LeavesUncrossedTriangleWhole(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)
    triangle := surface.Triangle{Points: [3]surface.Point{
        {X: 0, Y: 0, Value: 2},
        {X: 2, Y: 0, Value: 2.5},
        {X: 0, Y: 2, Value: 3},
    }, Value: 2.5}

    g.Expect(surface.SubdivideTriangle(triangle, []float64{1})).To(Equal([]surface.Polygon{{
        Points: triangle.Points[:],
        Value:  triangle.Value,
    }}))
}

func TestSubdivideTriangle_ReturnsNilForInvalidBreakpointsOrGeometry(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)
    triangle := surface.Triangle{Points: [3]surface.Point{
        {X: 0, Y: 0, Value: 0},
        {X: 2, Y: 0, Value: 1},
        {X: 0, Y: 2, Value: 2},
    }}

    g.Expect(surface.SubdivideTriangle(triangle, []float64{2, 1})).To(BeNil())
    g.Expect(surface.SubdivideTriangle(triangle, []float64{1, 1})).To(BeNil())
    g.Expect(surface.SubdivideTriangle(surface.Triangle{Points: [3]surface.Point{
        {X: 0, Y: 0, Value: 0},
        {X: 2, Y: 0, Value: math.NaN()},
        {X: 0, Y: 2, Value: 2},
    }}, []float64{1})).To(BeNil())
    g.Expect(surface.SubdivideTriangle(surface.Triangle{Points: [3]surface.Point{
        {X: 0, Y: 0, Value: 0},
        {X: 2, Y: 0, Value: 1},
        {X: 4, Y: 0, Value: 2},
    }}, []float64{1})).To(BeNil())
}
```

Import `math` for the NaN case. Set `triangle.Value` to `2.5` in the uncrossed
fixture so its expected single polygon has a stable representative value.

- [ ] **Step 5: Run the focused surface tests and confirm they fail**

Run:

```bash
go test ./internal/surface -run 'TestSubdivideTriangle' -count=1
```

Expected: compilation failure because `surface.Polygon` and
`surface.SubdivideTriangle` do not yet exist.

- [ ] **Step 6: Implement band clipping**

Create `internal/surface/subdivide.go`. Implement the public entry point and
private helpers with these exact responsibilities:

```go
func SubdivideTriangle(triangle Triangle, breakpoints []float64) []Polygon
func validBreakpoints(breakpoints []float64) bool
func clipBelow(points []Point, breakpoint float64) []Point
func clipAtOrAbove(points []Point, breakpoint float64) []Point
func edgeIntersection(start, end Point, breakpoint float64) Point
func polygonArea(points []Point) float64
func representativeValue(points []Point, lower, upper *float64) (float64, bool)
```

Use Sutherland-Hodgman clipping against scalar half-planes. For band `i`,
clip the original triangle first to its lower bound with `>=` and then to its
upper bound with `<`. For every directed edge:

```go
fraction := (breakpoint - start.Value) / (end.Value - start.Value)
intersection := Point{
    X: start.X + fraction*(end.X-start.X),
    Y: start.Y + fraction*(end.Y-start.Y),
    Value: breakpoint,
}
```

Do not divide when the edge values are equal. Retain an endpoint only when it
satisfies the corresponding band predicate; add the intersection only if the
edge changes predicate sides. Deduplicate adjacent equal points after each
clip, remove a repeated closing point, then reject polygons with fewer than
three points or an area at most a small epsilon.

Build the intervals as `(-Inf, b0)`, `[b0, b1)`, ..., `[bn, +Inf)`. A
representative value must map to the interval under `BucketIndex`: use a
strict interior value for bounded intervals; use an existing fragment vertex
that is below the first bound or at/above the final bound for unbounded
intervals. If no valid representative exists, omit the fragment.

When `breakpoints` is empty, return one polygon whose points are a copied
triangle slice and whose value is `triangle.Value`. Return `nil` for invalid
breakpoints, non-finite triangle points, or a degenerate triangle. Never
change `triangle.Points`.

- [ ] **Step 7: Run the focused surface tests and format**

Run:

```bash
gofumpt -w internal/surface/types.go internal/surface/subdivide.go internal/surface/subdivide_test.go
go test ./internal/surface -run 'TestSubdivideTriangle' -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit reusable surface geometry**

```bash
git add internal/surface/types.go internal/surface/subdivide.go internal/surface/subdivide_test.go
git commit -m "feat(surface): subdivide triangles by palette bands"
```

### Task 3: Render Numeric Surface Fragments in Spiral

**Files:**
- Modify: `internal/spiral/render.go`
- Modify: `internal/spiral/surface_test.go`

- [ ] **Step 1: Write failing spiral rendering tests**

Replace the one-polygon-per-triangle assertions in
`TestRenderToCanvas_RendersSurfaceBeforeSpiralForeground` with a fixture whose
vertices cross known numeric boundaries:

```go
triangles := []surface.Triangle{{
    Points: [3]surface.Point{
        {X: 20, Y: 30, Value: 1},
        {X: 40, Y: 30, Value: 1},
        {X: 20, Y: 50, Value: 3},
    },
    Value: 5.0 / 3.0,
}}
surfaceInk := inks.NumericInk(
    "surface",
    []float64{1, 2, 3},
    palette.GetPalette(palette.Temperature),
)
```

Assert two `DrawPolygon` calls occur before the boundary path, their fills
equal `surfaceInk.Dip(inks.MeasureValue(1))` and
`surfaceInk.Dip(inks.MeasureValue(2))`, and both calls have
`BorderWidth == 0`.

Add a separate fixed-ink regression:

```go
func TestRenderToCanvas_RendersOneSurfacePolygonForNonNumericInk(t *testing.T) {
    // Render the same triangle with inks.FixedInk(red).
    // Assert exactly one DrawPolygon, its fill is red, and its border width is 0.5.
}
```

The fallback keeps the established flat-triangle path unchanged; its existing
hairline same-colour border therefore remains valid.

- [ ] **Step 2: Run the focused spiral tests and confirm they fail**

Run:

```bash
go test ./internal/spiral -run 'TestRenderToCanvas_(RendersSurfaceBeforeSpiralForeground|RendersOneSurfacePolygonForNonNumericInk)' -count=1
```

Expected: FAIL because the renderer still emits one average-colour polygon
with a `0.5` border for numeric inks.

- [ ] **Step 3: Refactor `addSurface` to branch by ink kind**

In `internal/spiral/render.go`, keep the existing nil and empty-triangle
guard. Add:

```go
breakpoints := inks.NumericBreakpoints(surfaceInk)
if surfaceInk.Info().Kind != inks.KindNumeric {
    addFlatSurface(cv, triangles, surfaceInk)
    return
}
```

Extract the existing loop into:

```go
func addFlatSurface(cv *canvas.Canvas, triangles []surface.Triangle, surfaceInk inks.Ink)
```

Retain its current `PolygonSpec` border width of `0.5`.

Add:

```go
func addBandedSurface(
    cv *canvas.Canvas,
    triangles []surface.Triangle,
    surfaceInk inks.Ink,
    breakpoints []float64,
)
```

Use a single `canvas.PolygonSpec` with:

```go
ShapeStyle: canvas.ShapeStyle{
    Fill:        surfaceInk,
    Border:      surfaceInk,
    BorderWidth: 0,
},
```

For every triangle, call `surface.SubdivideTriangle`. If it returns `nil`
because of invalid geometry, skip that triangle; otherwise convert each
fragment point to `canvas.Position` and emit:

```go
cv.AddPolygon(canvas.LayerSurface, canvas.Polygon{
    Spec:   bandedSpec,
    Points: points,
    Fill:   metricValue(fragment.Value, "", surfaceInk),
    Border: metricValue(fragment.Value, "", surfaceInk),
})
```

Call `addBandedSurface` from `addSurface` only for numeric inks. This keeps
surface geometry generic and ensures all future visualization renderers can
reuse it.

- [ ] **Step 4: Run focused spiral tests and format**

Run:

```bash
gofumpt -w internal/spiral/render.go internal/spiral/surface_test.go
go test ./internal/spiral -run 'TestRenderToCanvas_(RendersSurfaceBeforeSpiralForeground|RendersOneSurfacePolygonForNonNumericInk|OverlaysAnnulusBoundaryBeforeGuideTrack)' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit spiral integration**

```bash
git add internal/spiral/render.go internal/spiral/surface_test.go
git commit -m "feat(spiral): render palette band fragments"
```

### Task 4: Regenerate Golden Coverage and Verify the Change

**Files:**
- Modify: `internal/goldentest/testdata/spiral-surface-shared-png.golden`
- Modify: `internal/goldentest/testdata/spiral-surface-shared-svg.golden`
- Modify: `internal/goldentest/testdata/spiral-surface-distinct-png.golden`
- Modify: `internal/goldentest/testdata/spiral-surface-distinct-svg.golden`
- Modify only if generated output requires it: `samples/spiral/code-visualizer.png`, `samples/spiral/code-visualizer.svg`

- [ ] **Step 1: Run the focused surface and spiral packages**

Run:

```bash
go test ./internal/surface ./internal/inks ./internal/spiral -count=1
```

Expected: golden tests fail and identify the four surface snapshot files that
need regeneration; all non-golden unit tests pass.

- [ ] **Step 2: Regenerate only the approved golden fixtures**

Run:

```bash
GOLDIE_UPDATE=1 go test ./internal/goldentest ./internal/spiral -count=1
```

Inspect:

```bash
git --no-pager diff --stat -- internal/goldentest/testdata
git --no-pager diff --check
```

Expected: only the four committed spiral-surface PNG/SVG golden fixtures
change. Do not stage user-managed sample artifacts unless the user explicitly
asks for their refresh.

- [ ] **Step 3: Re-run focused verification**

Run:

```bash
go test ./internal/surface ./internal/inks ./internal/spiral ./internal/goldentest -count=1
task build
task fmt:check
```

Expected: all commands pass.

- [ ] **Step 4: Run lint through the required concise subagent workflow**

Delegate `task lint` to an Explore-equivalent subagent and request only:

```text
Return the exit status; failing linter/test count and identities; each
file:line and message; or one line stating no issues.
```

Expected: report whether the pre-existing lint findings remain, without
capturing verbose lint output in the main session.

- [ ] **Step 5: Commit snapshots**

```bash
git add internal/goldentest/testdata/spiral-surface-shared-png.golden \
  internal/goldentest/testdata/spiral-surface-shared-svg.golden \
  internal/goldentest/testdata/spiral-surface-distinct-png.golden \
  internal/goldentest/testdata/spiral-surface-distinct-svg.golden
git commit -m "test(surface): update palette band snapshots"
```

- [ ] **Step 6: Confirm the final diff excludes manual artifacts**

Run:

```bash
git --no-pager status --short
git --no-pager log --oneline -4
```

Expected: implementation commits contain only source, tests, and four golden
fixtures. Preserve existing manual sample and `.superpowers` changes without
staging or reverting them.
