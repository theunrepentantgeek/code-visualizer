# Spiral Surface Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an optional, banded topographic surface beneath spiral timeline nodes, with a shared or independently selected numeric metric.

**Architecture:** Build `internal/surface` as a visualization-independent pipeline: deterministic Poisson-disk infill constrained by a caller-provided region, one Delaunay triangulation, IDW interpolation from original observations, and edge/region filtering. Extend canvas with a retained polygon primitive, then have spiral resolve and aggregate the surface metric, reuse or create a numeric ink, emit polygons at a new intermediate layer, and append an independent legend entry only when necessary.

**Tech Stack:** Go 1.26.1, `github.com/fogleman/delaunay` v0.0.0-20180910191513-63f09b4c883d, fogleman/gg raster backend, direct SVG XML backend, Gomega, Goldie v2.

---

## File structure

| Path | Responsibility |
| --- | --- |
| `internal/surface/types.go` | Public point, rectangle, region, triangle, and mesh constants. |
| `internal/surface/poisson.go` | Deterministic, region-constrained Poisson-disk point generation. |
| `internal/surface/mesh.go` | Delaunay conversion, IDW interpolation, triangle filtering, and band-value output. |
| `internal/surface/*_test.go` | Unit coverage for spacing, region constraints, interpolation, and mesh guarantees. |
| `internal/canvas/model/backend.go` | Add the polygon dispatch method. |
| `internal/canvas/spec.go`, `shape.go`, `canvas.go`, `layer.go` | Retained polygon shape/spec, layer constant, and canvas insertion method. |
| `internal/canvas/raster/backend.go`, `internal/canvas/svg/backend.go` | Draw filled/stroked polygons for raster and SVG output. |
| `internal/canvas/mock/backend.go` | Record polygon dispatches for canvas/spiral tests. |
| `internal/config/spiral.go` | Persist enablement and optional surface `MetricSpec`. |
| `cmd/codeviz/spiral_cmd.go` | Parse, override, and validate `--surface` and `--surface-metric`. |
| `internal/spiral/*` | Resolve/aggregate the metric, build surface ink, generate the annular mesh, render it, and build the legend. |
| `internal/legend/*` | Support an explicit `RoleSurface` entry without changing other visualizations. |
| `internal/goldentest/viz_golden_test.go` | Add stable PNG/SVG snapshot cases for shared and distinct surface metrics. |
| `docs/content/docs/visualizations/spiral.md` | Document the CLI/config feature and its numeric-metric constraint. |

### Task 1: Add the surface module and deterministic Poisson-disk sampler

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`
- Create: `internal/surface/types.go`
- Create: `internal/surface/poisson.go`
- Create: `internal/surface/poisson_test.go`

- [ ] **Step 1: Add the Delaunay dependency**

Run:

```bash
go get github.com/fogleman/delaunay@v0.0.0-20180910191513-63f09b4c883d
go mod tidy
```

Expected: `go.mod` lists `github.com/fogleman/delaunay` as a direct dependency, and `go.sum` contains its checksums.

- [ ] **Step 2: Write the failing surface sampler tests**

Create `internal/surface/poisson_test.go` with tests using a simple rectangular region. Cover these exact assertions:

```go
func TestSample_RespectsMinimumDistanceFromOriginalsAndSamples(t *testing.T) {
    originals := []surface.Point{{X: 8, Y: 8}, {X: 32, Y: 32}}
    samples := surface.Sample(surface.Rect{MinX: 0, MinY: 0, MaxX: 40, MaxY: 40},
        originals, 4, 17)

    all := append(slices.Clone(originals), samples...)
    for i := range all {
        for j := i + 1; j < len(all); j++ {
            g.Expect(surface.Distance(all[i], all[j])).To(BeNumerically(">=", 4))
        }
    }
}

func TestSample_OnlyReturnsPointsInsideRegion(t *testing.T) {
    region := surface.Annulus{CX: 20, CY: 20, InnerRadius: 8, OuterRadius: 18}
    samples := surface.Sample(region, nil, 4, 17)

    for _, sample := range samples {
        g.Expect(region.Contains(sample.X, sample.Y)).To(BeTrue())
    }
}

func TestSample_IsDeterministicForSeed(t *testing.T) {
    region := surface.Rect{MinX: 0, MinY: 0, MaxX: 40, MaxY: 40}
    g.Expect(surface.Sample(region, nil, 4, 17)).
        To(Equal(surface.Sample(region, nil, 4, 17)))
}
```

Use `NewGomegaWithT(t)`, `t.Parallel()`, and package `surface_test`.

- [ ] **Step 3: Run the sampler tests and confirm they fail**

Run:

```bash
go test ./internal/surface -run 'TestSample_' -count=1
```

Expected: compile failure because `internal/surface` does not exist.

- [ ] **Step 4: Implement the surface types**

Create `internal/surface/types.go` with these stable public types and constants:

```go
package surface

const (
    MaxTriangleEdge    = 8.0
    PoissonMinDistance = 4.0
    IDWPower           = 2.0
)

type Point struct {
    X, Y  float64
    Value float64
    Original bool
}

type Rect struct {
    MinX, MinY, MaxX, MaxY float64
}

func (r Rect) Bounds() Rect { return r }
func (r Rect) Contains(x, y float64) bool {
    return x >= r.MinX && x <= r.MaxX && y >= r.MinY && y <= r.MaxY
}

type Region interface {
    Bounds() Rect
    Contains(x, y float64) bool
}

type Annulus struct {
    CX, CY                     float64
    InnerRadius, OuterRadius   float64
}

type Triangle struct {
    Points [3]Point
    Value  float64
}
```

Implement `Annulus.Bounds`, `Annulus.Contains`, and `Distance(a, b Point) float64`. Treat either radius boundary as inside so sampling and centroid filtering cannot introduce artificial gaps.

- [ ] **Step 5: Implement deterministic Poisson sampling**

Create `internal/surface/poisson.go`. Implement Bridson sampling with:

- `func Sample(region Region, originals []Point, minimumDistance float64, seed uint64) []Point`
- a grid cell size of `minimumDistance / math.Sqrt2`;
- a `math/rand/v2.New(math/rand/v2.NewPCG(seed, seed^0x9e3779b97f4a7c15))` generator;
- all originals inserted into the occupancy grid before candidates are considered;
- candidates generated uniformly in the annulus `[minimumDistance, 2*minimumDistance]` around an active point;
- candidate rejection if it is outside `region` or closer than `minimumDistance` to an original or accepted sample;
- a deterministic active-list selection and 30 attempts per active point.

Return only infill points, with `Original: false`; do not return or mutate `originals`. A zero/negative distance or empty region returns no samples.

- [ ] **Step 6: Run the sampler tests and format**

Run:

```bash
gofumpt -w internal/surface
go test ./internal/surface -run 'TestSample_' -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit the sampler foundation**

```bash
git add go.mod go.sum internal/surface/types.go internal/surface/poisson.go internal/surface/poisson_test.go
git commit -m "feat(surface): add Poisson disk sampling"
```

### Task 2: Build the Delaunay mesh, IDW values, and region-safe triangles

**Files:**
- Create: `internal/surface/mesh.go`
- Create: `internal/surface/mesh_test.go`

- [ ] **Step 1: Write failing mesh tests**

Create `internal/surface/mesh_test.go` covering:

```go
func TestBuild_PreservesOriginalPointValues(t *testing.T) {
    originals := []surface.Point{
        {X: 0, Y: 0, Value: 10, Original: true},
        {X: 8, Y: 0, Value: 20, Original: true},
        {X: 0, Y: 8, Value: 30, Original: true},
    }
    triangles := surface.Build(surface.Rect{MinX: 0, MinY: 0, MaxX: 8, MaxY: 8}, originals, 7)
    g.Expect(triangles).NotTo(BeEmpty())
    for _, triangle := range triangles {
        for _, point := range triangle.Points {
            if point.Original {
                g.Expect(point.Value).To(BeElementOf(10.0, 20.0, 30.0))
            }
        }
    }
}

func TestInterpolate_UsesInverseDistanceWeightedOriginalValues(t *testing.T) {
    originals := []surface.Point{
        {X: 0, Y: 0, Value: 0, Original: true},
        {X: 4, Y: 0, Value: 8, Original: true},
    }
    g.Expect(surface.Interpolate(surface.Point{X: 1, Y: 0}, originals)).
        To(BeNumerically("~", 0.8, 1e-9))
}

func TestBuild_ExcludesCoreAndLongEdgeTriangles(t *testing.T) {
    region := surface.Annulus{CX: 20, CY: 20, InnerRadius: 8, OuterRadius: 18}
    triangles := surface.Build(region, annularOriginals(), 7)
    for _, triangle := range triangles {
        centroid := triangleCentroid(triangle)
        g.Expect(region.Contains(centroid.X, centroid.Y)).To(BeTrue())
        g.Expect(surface.LongestEdge(triangle)).To(BeNumerically("<=", surface.MaxTriangleEdge))
    }
}
```

The `triangleCentroid` helper must average the three point coordinates. Use enough `annularOriginals()` points around both rings to produce triangles; no test may depend on random output because `Build` receives an explicit seed.

- [ ] **Step 2: Run mesh tests and confirm they fail**

Run:

```bash
go test ./internal/surface -run 'Test(Build|Interpolate)_' -count=1
```

Expected: compile failure because `Build`, `Interpolate`, and `LongestEdge` do not exist.

- [ ] **Step 3: Implement interpolation and triangulation**

Create `internal/surface/mesh.go`:

```go
func Interpolate(point Point, originals []Point) float64
func Build(region Region, originals []Point, seed uint64) []Triangle
func LongestEdge(triangle Triangle) float64
```

`Interpolate` must:

1. Return the observed value immediately for `point.Original`.
2. Return an exactly matching original's value when its distance is zero.
3. Compute `sum(weight*original.Value) / sum(weight)` using
   `weight = 1 / math.Pow(distance, IDWPower)`.
4. Return zero only when `originals` is empty.

`Build` must:

1. Return no triangles for fewer than three originals.
2. Copy originals, mark them `Original: true`, append
   `Sample(region, originals, PoissonMinDistance, seed)`.
3. Interpolate every appended point using the original-only slice.
4. Convert those points to `[]delaunay.Point`, call
   `delaunay.Triangulate`, and convert each successive triple from
   `triangulation.Triangles` into a `Triangle`.
5. Set each triangle value to the arithmetic mean of its three vertex values.
6. Keep a triangle only when all three source points and its centroid are
   in `region` and `LongestEdge(triangle) <= MaxTriangleEdge`.

Do not derive colour or palette bands in this package: it returns metric
values so callers can apply their own existing `inks.Ink` mapping.

- [ ] **Step 4: Run all surface tests**

Run:

```bash
gofumpt -w internal/surface
go test ./internal/surface -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit mesh generation**

```bash
git add internal/surface/mesh.go internal/surface/mesh_test.go
git commit -m "feat(surface): generate interpolated triangle meshes"
```

### Task 3: Add retained canvas polygons and backend implementations

**Files:**
- Modify: `internal/canvas/model/backend.go`
- Modify: `internal/canvas/spec.go`
- Modify: `internal/canvas/shape.go`
- Modify: `internal/canvas/canvas.go`
- Modify: `internal/canvas/layer.go`
- Modify: `internal/canvas/mock/backend.go`
- Modify: `internal/canvas/raster/backend.go`
- Modify: `internal/canvas/raster/backend_test.go`
- Modify: `internal/canvas/svg/backend.go`
- Modify: `internal/canvas/svg/backend_test.go`
- Modify: `internal/canvas/canvas_test.go`
- Modify: `internal/treemap/render_focus_test.go`
- Modify: `internal/bubbletree/render_test.go`

- [ ] **Step 1: Write failing canvas dispatch/layer tests**

Add a canvas test that constructs a polygon at `LayerSurface`, then a path at
`LayerStructure`, renders to `mock.NewBackend()`, and asserts:

```go
g.Expect(backend.Calls).To(HaveLen(2))
g.Expect(backend.Calls[0].Method).To(Equal("DrawPolygon"))
g.Expect(backend.Calls[1].Method).To(Equal("DrawPath"))
g.Expect(backend.Calls[0].Fill).To(Equal(color.RGBA{R: 12, G: 34, B: 56, A: 255}))
```

Use triangle points `(1,1)`, `(9,1)`, `(1,9)` and a `PolygonSpec` whose fill
and border are the same fixed ink. This test proves retained-shape dispatch
and the exact z-order required by the surface.

Add raster and SVG tests that call `DrawPolygon` with the same triangle:

- raster: render to a temporary PNG and assert pixel `(2,2)` has the supplied
  fill colour;
- SVG: finish to a temporary SVG and assert it contains
  `<polygon points="1.00,1.00 9.00,1.00 1.00,9.00"` plus matching fill,
  stroke, and `stroke-width="0.5"`.

- [ ] **Step 2: Run the targeted tests and confirm compile failures**

Run:

```bash
go test ./internal/canvas ./internal/canvas/raster ./internal/canvas/svg -count=1
```

Expected: compile failures for `Polygon`, `PolygonSpec`, `LayerSurface`, and
`DrawPolygon`.

- [ ] **Step 3: Add the canvas model and retained shape**

Make these exact interface and public API additions:

```go
// internal/canvas/model/backend.go
DrawPolygon(points []Position, fill, border Fill, borderWidth float64)

// internal/canvas/spec.go
type PolygonSpec struct{ ShapeStyle }

// internal/canvas/shape.go
type Polygon struct {
    Spec   *PolygonSpec
    Points []Position
    Fill   inks.MetricValue
    Border inks.MetricValue
}

// internal/canvas/canvas.go
func (c *Canvas) AddPolygon(layer Layer, p Polygon)

// internal/canvas/layer.go
LayerSurface Layer = 5
```

`Polygon.drawTo` resolves fill and border exactly as `Disc.drawTo` does, then
calls `DrawPolygon`. `AddPolygon` must append a value copy, matching
`AddRectangle`/`AddDisc`. Record point coordinates and border width in the
mock `Call`, adding `Points []model.Position` and `BorderWidth float64`.

Update the two local `captureBackend` test doubles in
`internal/treemap/render_focus_test.go` and
`internal/bubbletree/render_test.go` with no-op `DrawPolygon` methods so
they continue satisfying `canvas.Backend`.

- [ ] **Step 4: Implement the raster and SVG drawing methods**

In `rasterBackend.DrawPolygon`:

```go
if len(points) < 3 {
    return
}
r.dc.MoveTo(points[0].X, points[0].Y)
for _, point := range points[1:] {
    r.dc.LineTo(point.X, point.Y)
}
r.dc.ClosePath()
r.dc.SetColor(nrgba(model.SolidColor(fill)))
r.dc.FillPreserve()
if borderWidth > 0 {
    r.dc.SetColor(nrgba(model.SolidColor(border)))
    r.dc.SetLineWidth(borderWidth)
    r.dc.Stroke()
}
```

In `svgBackend.DrawPolygon`, return for fewer than three points; otherwise
write a single `<polygon>` element with two-decimal `x,y` coordinate pairs,
`svgFillAttr(fill)`, `colourCSS(model.SolidColor(border))`, and the supplied
stroke width. Preserve `Fill` support rather than assuming a solid fill.

- [ ] **Step 5: Run focused canvas tests and format**

Run:

```bash
gofumpt -w internal/canvas internal/treemap/render_focus_test.go internal/bubbletree/render_test.go
go test ./internal/canvas ./internal/canvas/raster ./internal/canvas/svg ./internal/treemap ./internal/bubbletree -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit polygon support**

```bash
git add internal/canvas internal/treemap/render_focus_test.go internal/bubbletree/render_test.go
git commit -m "feat(canvas): add polygon primitive"
```

### Task 4: Resolve and aggregate the surface metric and expose it in config/CLI/legend

**Files:**
- Modify: `internal/config/spiral.go`
- Modify: `internal/config/override.go`
- Modify: `internal/config/config_test.go`
- Modify: `cmd/codeviz/spiral_cmd.go`
- Modify: `cmd/codeviz/main_test.go`
- Modify: `internal/spiral/state.go`
- Modify: `internal/spiral/timebucket.go`
- Modify: `internal/spiral/aggregation.go`
- Modify: `internal/spiral/stages.go`
- Modify: `internal/spiral/inks.go`
- Modify: `internal/spiral/stages_test.go`
- Modify: `internal/legend/config.go`
- Modify: `internal/legend/legend.go`
- Modify: `internal/legend/legend_test.go`
- Modify: `internal/bubbletree/stages.go`
- Modify: `internal/radialtree/stages.go`
- Modify: `internal/scatter/stages.go`
- Modify: `internal/treemap/stages.go`

- [ ] **Step 1: Write failing CLI/config and metric tests**

Add tests for these cases:

```go
// Kong parses --surface and --surface-metric file-lines,terrain.
g.Expect(cli.Spiral.Surface).To(BeTrue())
g.Expect(cli.Spiral.SurfaceMetric).To(Equal(config.MetricSpec{
    Metric: "file-lines", Palette: palette.Terrain,
}))

// --surface with no configured fill errors.
g.Expect((&SpiralCmd{Surface: true}).validateConfig(&config.Spiral{})).
    To(MatchError(ContainSubstring("surface requires a numeric fill metric or surface metric")))

// categorical explicit surface metric errors.
g.Expect((&SpiralCmd{}).validateConfig(&config.Spiral{
    SurfaceMetric: &config.MetricSpec{Metric: "file-type"},
})).To(MatchError(ContainSubstring("surface metric must be numeric")))

// Resolution adds only the distinct metric to Requested.
g.Expect(common.Requested.BaseMetrics).To(ConsistOf(
    metric.Name("file-lines"), metric.Name("file-size"),
))

// Aggregation populates SurfaceValue for the selected metric.
g.Expect(viz.Buckets[0].SurfaceValue).To(Equal(42.0))
```

Add a legend test asserting a distinct surface metric creates
`RoleSurface`, while a surface metric equal to fill creates no duplicate
entry.

- [ ] **Step 2: Run targeted tests and confirm failures**

Run:

```bash
go test ./cmd/codeviz ./internal/config ./internal/spiral ./internal/legend -count=1
```

Expected: compile failures for the surface config/state/legend fields and
methods.

- [ ] **Step 3: Add configuration and CLI wiring**

Use this configuration shape:

```go
type Spiral struct {
    // existing fields
    Surface       *bool       `yaml:"surface,omitempty"        json:"surface,omitempty"`
    SurfaceMetric *MetricSpec `yaml:"surfaceMetric,omitempty"  json:"surfaceMetric,omitempty"`
}

func (s *Spiral) SurfaceEnabled() bool {
    return (s.Surface != nil && *s.Surface) || (s.SurfaceMetric != nil && !s.SurfaceMetric.IsZero())
}
func (s *Spiral) OverrideSurface(v bool) {
    if v { s.Surface = &v }
}
func (s *Spiral) OverrideSurfaceMetric(v MetricSpec) {
    overrideMetricSpec(&s.SurfaceMetric, v)
}
```

Add `Surface bool` and `SurfaceMetric config.MetricSpec` to `SpiralCmd`:

```go
Surface bool `help:"Render a banded metric surface behind the spiral." optional:""`
SurfaceMetric config.MetricSpec `help:"Surface metric: metric[,palette]; implies --surface." name:"surface-metric" optional:""`
```

Call the new overrides from `applyOverrides`. Validation must:

1. return early when `SurfaceEnabled()` is false;
2. select `SurfaceMetric` when non-empty, otherwise `Fill`;
3. reject an absent selected metric;
4. call `MetricSpec.Validate("surface metric")`;
5. call `provider.ResolveForValidation`, then reject its
   `ResolvedMetric.ResultKind == metric.Classification`.

Do not add a false-valued CLI override: existing CLI boolean options only
override config when asserted true.

- [ ] **Step 4: Add resolved state, aggregation, ink selection, and legend role**

Make these data-flow changes:

```go
// spiral.State
SurfaceEnabled bool
SurfaceMetric  metric.Name
SurfacePalette palette.PaletteName
SurfaceInk     inks.Ink

// spiral.TimeBucket
SurfaceValue float64

// legend/config.go
RoleSurface Role = "Surface"
```

In `ResolveMetrics`, resolve the selected effective surface metric and palette:

- disabled: leave `SurfaceMetric` empty;
- metric override: use its metric and `stages.ResolveFillPalette`;
- enabled without override: copy `FillMetric` and `FillPalette`.

Extend requested-metric collection to deduplicate size, fill, border, and
surface. Extend `AggregateBucketMetrics`/`aggregateBucket` to receive
`surfaceMetric` and calculate `SurfaceValue` with `bucketNumericValue`.

In `BuildInksStage`, set `SurfaceInk = Inks.Fill` when surface and fill
metrics match. Otherwise use `buildBucketInk` with
`func(b *TimeBucket) float64 { return b.SurfaceValue }`, no categorical
accessor, and the resolved surface palette.

Refactor `legend.Build` to accept an `[]Entry` argument instead of
fill/border/size positional arguments; build the same ordered fill, border,
and conditional size entries at every visualization call site
(`bubbletree`, `radialtree`, `scatter`, `spiral`, and `treemap`). Spiral
appends `Entry{Role: legend.RoleSurface, MetricName: string(p.SurfaceMetric),
Ink: p.SurfaceInk}` only when `SurfaceMetric != "" && SurfaceMetric !=
FillMetric`. This keeps the shared legend package extensible and prevents the
distinct surface mapper from being mislabeled as fill.

- [ ] **Step 5: Run targeted tests and format**

Run:

```bash
gofumpt -w internal/config cmd/codeviz internal/spiral internal/legend
go test ./cmd/codeviz ./internal/config ./internal/spiral ./internal/legend -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit CLI, metric, and legend plumbing**

```bash
git add internal/config cmd/codeviz internal/spiral internal/legend
git commit -m "feat(spiral): configure surface metrics"
```

### Task 5: Render the annular surface beneath spiral content

**Files:**
- Create: `internal/spiral/surface.go`
- Create: `internal/spiral/surface_test.go`
- Modify: `internal/spiral/render.go`
- Modify: `internal/spiral/render_test.go`
- Modify: `internal/spiral/stages.go`

- [ ] **Step 1: Write failing spiral surface render tests**

Create `internal/spiral/surface_test.go` with:

```go
func TestBuildSurface_UsesSpiralAnnulus(t *testing.T) {
    layout := spiral.Layout(sampleTimeBuckets(), 320, 320, spiral.Hourly, spiral.LabelNone)
    triangles := spiral.BuildSurface(layout, surfaceValues(len(layout.Nodes)), 17)

    g.Expect(triangles).NotTo(BeEmpty())
    region := surface.Annulus{
        CX: layout.CX, CY: layout.CY, InnerRadius: layout.A,
        OuterRadius: layout.A + layout.B*layout.MaxTheta,
    }
    for _, triangle := range triangles {
        centroid := triangleCentroid(triangle)
        g.Expect(region.Contains(centroid.X, centroid.Y)).To(BeTrue())
    }
}

func TestRenderToCanvas_SurfaceRendersBeforeTrackAndDiscs(t *testing.T) {
    cv := spiral.RenderToCanvas(layout, buckets, 320, 320, inks, triangles, surfaceInk)
    backend := mock.NewBackend()
    g.Expect(cv.RenderTo(backend)).To(Succeed())
    g.Expect(backend.Calls[0].Method).To(Equal("DrawRectangle"))
    g.Expect(backend.Calls[1].Method).To(Equal("DrawPolygon"))
    g.Expect(firstIndex(backend.Calls, "DrawPath")).To(BeNumerically(">", 1))
    g.Expect(firstIndex(backend.Calls, "DrawDisc")).To(BeNumerically(">", firstIndex(backend.Calls, "DrawPath")))
}
```

Use a numeric surface ink and assert a polygon fill equals
`surfaceInk.Dip(metricValue(triangle.Value, "", surfaceInk))`. Define
`firstIndex(calls []mock.Call, method string) int` in this test file to
return the first matching index (or `-1`), then add a
no-surface test that calls the normal render path and asserts no
`DrawPolygon` occurs.

- [ ] **Step 2: Run tests and confirm failure**

Run:

```bash
go test ./internal/spiral -run 'Test(BuildSurface|RenderToCanvas_Surface)' -count=1
```

Expected: compile failure because `BuildSurface` and the surface render
arguments do not exist.

- [ ] **Step 3: Implement surface conversion and rendering**

Create `internal/spiral/surface.go`:

```go
func BuildSurface(layout SpiralLayout, values []float64, seed uint64) []surface.Triangle
```

It must:

1. return nil when fewer than three nodes exist or `len(values) != len(layout.Nodes)`;
2. convert every node/value pair to an original `surface.Point`;
3. construct the annulus with inner radius `layout.A` and outer radius
   `layout.A + layout.B*layout.MaxTheta`;
4. call `surface.Build(region, originals, seed)`.

Use a stable seed derived from the node count and layout dimensions; never use
wall-clock randomness, so PNG/SVG snapshots are repeatable.

Change `RenderToCanvas` to accept `triangles []surface.Triangle` and
`surfaceInk inks.Ink`. Keep `addBackground` first, then call:

```go
addSurface(cv, triangles, surfaceInk)
addTrack(cv, layout)
addDiscs(cv, layout.Nodes, buckets, is)
addLabels(cv, layout.Nodes)
```

`addSurface` must allocate one shared `canvas.PolygonSpec`:

```go
spec := &canvas.PolygonSpec{
    ShapeStyle: canvas.ShapeStyle{
        Fill: surfaceInk, Border: surfaceInk, BorderWidth: 0.5,
    },
}
```

For every triangle, convert its points to `[]canvas.Position` and add it on
`canvas.LayerSurface` with both `Fill` and `Border` set to
`metricValue(triangle.Value, "", surfaceInk)`.

In `RenderStage`, conditionally call `BuildSurface` with the buckets'
`SurfaceValue`s only when `p.SurfaceEnabled`, then pass those triangles and
`p.SurfaceInk` to `RenderToCanvas`; otherwise pass nil and retain the
pre-existing rendering result.

- [ ] **Step 4: Run focused spiral tests**

Run:

```bash
gofumpt -w internal/spiral
go test ./internal/spiral -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit spiral rendering**

```bash
git add internal/spiral
git commit -m "feat(spiral): render banded metric surface"
```

### Task 6: Add end-to-end snapshots, document usage, and validate the change

**Files:**
- Modify: `internal/goldentest/viz_golden_test.go`
- Create: `internal/goldentest/testdata/TestGolden_SpiralSurfaceShared-png.golden`
- Create: `internal/goldentest/testdata/TestGolden_SpiralSurfaceShared-svg.golden`
- Create: `internal/goldentest/testdata/TestGolden_SpiralSurfaceDistinct-png.golden`
- Create: `internal/goldentest/testdata/TestGolden_SpiralSurfaceDistinct-svg.golden`
- Modify: `docs/content/docs/visualizations/spiral.md`

- [ ] **Step 1: Write the failing golden tests**

In `internal/goldentest/viz_golden_test.go`, add two render closures and tests:

```go
func TestGolden_SpiralSurfaceShared(t *testing.T) {
    runVizGolden(t, "spiral-surface-shared", renderSpiralSurfaceShared)
}

func TestGolden_SpiralSurfaceDistinct(t *testing.T) {
    runVizGolden(t, "spiral-surface-distinct", renderSpiralSurfaceDistinct)
}
```

Both must configure a numeric spiral fill (`file-lines,terrain`) and a local
`enabled := true` assigned as `Surface: &enabled`. The distinct case additionally sets
`SurfaceMetric: &config.MetricSpec{Metric: "file-size", Palette:
palette.Temperature}`. Reuse `renderSpiral`'s synthetic history and pipeline
setup, with the surface config applied before `spiral.ResolveMetrics`.

- [ ] **Step 2: Run golden tests and confirm they fail for missing snapshots**

Run:

```bash
go test ./internal/goldentest -run 'TestGolden_SpiralSurface' -count=1
```

Expected: FAIL because Goldie cannot find the four named golden files.

- [ ] **Step 3: Generate and inspect golden snapshots**

Run:

```bash
GOLDIE_UPDATE=1 go test ./internal/goldentest -run 'TestGolden_SpiralSurface' -count=1
go test ./internal/goldentest -run 'TestGolden_SpiralSurface' -count=1
```

Expected: the first command writes four PNG/SVG goldens; the second command
passes. Confirm the distinct SVG contains a `Surface` legend label and the
shared SVG contains only one fill/surface metric legend entry.

- [ ] **Step 4: Document the feature**

Update `docs/content/docs/visualizations/spiral.md` with:

- `--surface` behavior and its requirement for a numeric `--fill`;
- `--surface-metric file-lines,terrain` behavior and the fact it implies
  enablement;
- the equivalent YAML:

```yaml
spiral:
  fill:
    metric: file-lines
    palette: terrain
  surface: true
  surfaceMetric:
    metric: file-size
    palette: temperature
```

- the annular extent, discrete bands, and foreground ordering;
- the additional legend entry for a distinct surface metric.

- [ ] **Step 5: Run all required validation**

Run:

```bash
task fmt:check
task test
task build
```

Then dispatch the repository-required lint command to an Explore-equivalent
agent, requesting only exit status, failing linter/test counts and identities,
file:line messages, or a one-line clean result:

```bash
task lint
```

Expected: every command exits successfully. Do not strip `--verbose` from the
Taskfile lint command.

- [ ] **Step 6: Commit tests, documentation, and goldens**

```bash
git add internal/goldentest docs/content/docs/visualizations/spiral.md
git commit -m "test: cover spiral metric surfaces"
```
