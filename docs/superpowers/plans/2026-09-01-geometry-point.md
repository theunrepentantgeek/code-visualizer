# Geometry Point Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Introduce `geometry.Point`, migrate absolute positions, and separate surface samples and gradient coordinates from pixel geometry.

**Architecture:** Build `Point` on the Vector PR while preserving strict location-versus-displacement semantics. Use composition for `surface.Sample` and keep normalized `GradientPoint` distinct at the canvas boundary.

**Tech Stack:** Go 1.26.1, `math`, Gomega, Goldie v2, canvas raster/SVG backends

**Stack position:** PR 2 of 5; base this branch on the Vector PR branch.

---

### Task 1: Add the Point contract

**Files:**
- Create: `internal/geometry/point.go`
- Create: `internal/geometry/point_test.go`

- [ ] **Step 1: Write failing Point tests**

Cover `Valid`, translation, `VectorTo`, both distances, midpoint, interpolation,
NaN/Inf propagation, distance symmetry, endpoint interpolation, non-mutation,
`OriginPoint`, and the `NewPoint` factory:

```go
func TestPointVectorSemantics(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	start := Point{X: 2, Y: 3}
	end := Point{X: 5, Y: 7}

	g.Expect(start.VectorTo(end)).To(Equal(Vector{X: 3, Y: 4}))
	g.Expect(start.Translate(start.VectorTo(end))).To(Equal(end))
	g.Expect(start.DistanceTo(end)).To(Equal(5.0))
	g.Expect(start.DistanceSquaredTo(end)).To(Equal(25.0))
	g.Expect(Midpoint(start, end)).To(Equal(Point{X: 3.5, Y: 5}))
	g.Expect(Lerp(start, end, 0)).To(Equal(start))
	g.Expect(Lerp(start, end, 1)).To(Equal(end))
	g.Expect(OriginPoint).To(Equal(Point{X: 0, Y: 0}))
	g.Expect(NewPoint(2, 3)).To(Equal(start))
}
```

- [ ] **Step 2: Verify focused failure**

Run `go test ./internal/geometry -run Point`.

Expected: FAIL because `Point` is undefined.

- [ ] **Step 3: Implement Point**

```go
package geometry

import "math"

type Point struct {
	X float64
	Y float64
}

// OriginPoint is the point at (0, 0). It's a var, not a const, because Go
// structs cannot be declared const.
var OriginPoint = Point{}

// NewPoint constructs a Point from Cartesian coordinates.
func NewPoint(x, y float64) Point {
	return Point{X: x, Y: y}
}

func (p Point) Valid() bool {
	return !math.IsNaN(p.X) && !math.IsInf(p.X, 0) &&
		!math.IsNaN(p.Y) && !math.IsInf(p.Y, 0)
}

func (p Point) Translate(v Vector) Point {
	return Point{X: p.X + v.X, Y: p.Y + v.Y}
}

func (p Point) VectorTo(other Point) Vector {
	return Vector{X: other.X - p.X, Y: other.Y - p.Y}
}

func (p Point) DistanceSquaredTo(other Point) float64 {
	if !p.Valid() || !other.Valid() {
		return math.NaN()
	}
	return p.VectorTo(other).LengthSquared()
}

func (p Point) DistanceTo(other Point) float64 {
	if !p.Valid() || !other.Valid() {
		return math.NaN()
	}
	return p.VectorTo(other).Length()
}

func Midpoint(a, b Point) Point { return Lerp(a, b, 0.5) }

func Lerp(a, b Point, fraction float64) Point {
	return a.Translate(a.VectorTo(b).Scale(fraction))
}
```

`OriginPoint` mirrors `ZeroVector`: a `var`, not a `const`, because Go structs
cannot be declared `const`. `NewPoint` mirrors `NewVector` as the ordinary
two-coordinate construction factory. Callers use `NewPoint(x, y)` at absolute-point
construction sites and `OriginPoint` wherever a point semantically means the
coordinate origin, rather than repeating `Point{X: ..., Y: ...}` or `Point{}`
literals.

- [ ] **Step 4: Run and format geometry tests**

Run `gofumpt -w internal/geometry && go test ./internal/geometry`.

Expected: PASS.

- [ ] **Step 5: Commit Point**

```bash
git add internal/geometry
git commit -m "feat(geometry): add point primitive" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 2: Migrate the canvas geometry boundary

**Files:**
- Modify: `internal/canvas/model/backend.go`
- Modify: `internal/canvas/geometry.go`
- Modify: `internal/canvas/model/fill.go`
- Modify: `internal/canvas/model/fill_test.go`
- Modify: `internal/canvas/mock/backend.go`
- Modify: `internal/canvas/raster/backend.go`
- Modify: `internal/canvas/raster/gradient_fill.go`
- Modify: `internal/canvas/svg/backend.go`
- Modify: `internal/canvas/disc.go`
- Modify: `internal/canvas/line.go`
- Modify: `internal/canvas/path.go`
- Modify: `internal/canvas/filled_path.go`
- Modify: `internal/canvas/polygon.go`
- Modify: `internal/canvas/text.go`
- Modify: `internal/canvas/text_spec.go`
- Modify: `internal/canvas/canvas_test.go`
- Modify: `internal/canvas/raster/backend_test.go`
- Modify: `internal/canvas/svg/backend_test.go`
- Modify: `internal/canvas/raster/gradient_test.go`
- Modify: `internal/canvas/svg/gradient_test.go`

- [ ] **Step 1: Update backend tests to use geometry.Point**

Replace absolute `model.Position` and `canvas.Position` values with
`geometry.Point`. Rename normalized fill fixtures from `model.Point` to
`model.GradientPoint`. Do not change any numeric expectation.

- [ ] **Step 2: Verify canvas compile failure**

Run `go test ./internal/canvas/...`.

Expected: FAIL because backend signatures still use `model.Position` and
`GradientPoint` does not exist.

- [ ] **Step 3: Separate gradient coordinates**

In `internal/canvas/model/fill.go` define:

```go
type GradientPoint struct {
	X float64
	Y float64
}
```

Change `RadialGradientFill.Focus` and shape focus fields to `GradientPoint`.
At raster and SVG rendering boundaries, calculate absolute focus into a
`geometry.Point` before reading `X` and `Y`.

- [ ] **Step 4: Change canvas APIs to geometry.Point**

Change every backend position, path, polygon, line, and text signature to
`geometry.Point` or slices of it. Delete the concrete `model.Position`
definition. Remove `canvas.Position`; do not retain a compatibility alias.
Convert canvas retained shapes from coordinate pairs where they are pure
positions:

```go
type Line struct {
	Spec *LineSpec
	From geometry.Point
	To   geometry.Point
}

type Polygon struct {
	Spec   *PolygonSpec
	Points []geometry.Point
	Fill   inks.MetricValue
	Border inks.MetricValue
}
```

Use `Position geometry.Point` for text and absolute path records. Keep rectangle
and disc paired fields until their dedicated PRs.

- [ ] **Step 5: Run canvas tests**

Run `gofumpt -w internal/canvas && go test ./internal/canvas/...`.

Expected: PASS with raster and SVG expectations unchanged.

- [ ] **Step 6: Commit the canvas migration**

```bash
git add internal/canvas
git commit -m "refactor(canvas): use geometry points" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 3: Replace surface.Point with surface.Sample

**Files:**
- Modify: `internal/surface/types.go`
- Modify: `internal/surface/interpolation.go`
- Modify: `internal/surface/interpolation_test.go`
- Modify: `internal/surface/mesh.go`
- Modify: `internal/surface/mesh_internal_test.go`
- Modify: `internal/surface/mesh_refinement_test.go`
- Modify: `internal/surface/mesh_test.go`
- Modify: `internal/surface/poisson.go`
- Modify: `internal/surface/poisson_test.go`
- Modify: `internal/surface/subdivide.go`
- Modify: `internal/surface/subdivide_test.go`
- Modify: `internal/spiral/surface.go`
- Modify: `internal/spiral/surface_test.go`

- [ ] **Step 1: Rewrite surface tests around Sample.Position**

Rename `Point` fixtures to `Sample`, place coordinates under
`Position: geometry.Point{...}`, and change geometric assertions from
`sample.X`/`sample.Y` to `sample.Position.X`/`sample.Position.Y`. Change
`Distance(a, b)` assertions to `a.Position.DistanceTo(b.Position)`.

- [ ] **Step 2: Run surface tests and verify compile failure**

Run `go test ./internal/surface ./internal/spiral -run 'Surface|Mesh|Poisson|Interpolation|Subdivide'`.

Expected: FAIL because `Sample` and its `Position` field do not exist.

- [ ] **Step 3: Introduce Sample and migrate algorithms**

Replace `surface.Point` with:

```go
type Sample struct {
	Position    geometry.Point
	Value       float64
	unsupported bool
	Original    bool
}
```

Change `Triangle.Points` to `[3]Sample`, `Polygon.Points` to `[]Sample`, region
boundary loops to `[][]Sample`, interpolation and Poisson slices to `[]Sample`,
and Delaunay conversion to read `sample.Position`.

Use `geometry.Midpoint`, `Point.DistanceTo`, `Point.DistanceSquaredTo`,
`Point.VectorTo`, and `Point.Translate` for existing coordinate arithmetic.
Keep interpolation value/state on `Sample`. Remove the package-level `Distance`
function.

- [ ] **Step 4: Run surface and spiral tests**

Run `gofumpt -w internal/surface internal/spiral && go test ./internal/surface ./internal/spiral`.

Expected: PASS.

- [ ] **Step 5: Commit the surface migration**

```bash
git add internal/surface internal/spiral
git commit -m "refactor(surface): separate samples from points" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 4: Migrate visualization absolute positions

**Files:**
- Modify: `internal/bubbletree/node.go`
- Modify: `internal/bubbletree/layout.go`
- Modify: `internal/bubbletree/packing.go`
- Modify: `internal/bubbletree/transforms.go`
- Modify: `internal/bubbletree/render.go`
- Modify: `internal/bubbletree/geometry_test.go`
- Modify: `internal/bubbletree/layout_test.go`
- Modify: `internal/bubbletree/packing_test.go`
- Modify: `internal/bubbletree/render_test.go`
- Modify: `internal/bubbletree/transforms_test.go`
- Modify: `internal/spiral/node.go`
- Modify: `internal/spiral/layout.go`
- Modify: `internal/spiral/discsize.go`
- Modify: `internal/spiral/labels.go`
- Modify: `internal/spiral/render.go`
- Modify: `internal/spiral/discsize_test.go`
- Modify: `internal/spiral/labels_test.go`
- Modify: `internal/spiral/layout_test.go`
- Modify: `internal/spiral/render_test.go`
- Modify: `internal/spiral/stages_test.go`
- Modify: `internal/scatter/layout.go`
- Modify: `internal/scatter/render.go`
- Modify: `internal/scatter/layout_test.go`
- Modify: `internal/donuttree/node.go`
- Modify: `internal/donuttree/layout.go`
- Modify: `internal/donuttree/labels.go`
- Modify: `internal/donuttree/render.go`
- Modify: `internal/donuttree/labels_test.go`
- Modify: `internal/donuttree/layout_test.go`
- Modify: `internal/donuttree/render_test.go`
- Modify: `internal/legend/render.go`
- Modify: `internal/legend/render_test.go`

- [ ] **Step 1: Update fixtures to compose absolute positions**

Use fields such as:

```go
Position: geometry.Point{X: 120, Y: 80},
Center:   geometry.Point{X: 300, Y: 300},
```

Preserve all existing expected numeric values and mock backend call order.

- [ ] **Step 2: Verify visualization compile failures**

Run:

```bash
go test ./internal/bubbletree ./internal/spiral ./internal/scatter ./internal/donuttree ./internal/legend
```

Expected: FAIL until production records and call sites use the new fields.

- [ ] **Step 3: Migrate absolute position fields and arithmetic**

Use `Position geometry.Point` for bubble, spiral, and scatter records, and
`Center geometry.Point` for donut layout. Keep radial-tree `Position` as
`geometry.Vector`, converting it to an absolute point only by translating the
canvas centre:

```go
absolute := center.Translate(node.Position)
```

Use `Translate`, `VectorTo`, `DistanceTo`, `DistanceSquaredTo`, `Midpoint`, and
`Lerp` wherever they directly replace existing arithmetic. Do not convert
angles, scalar axis coordinates, or gradient coordinates.

- [ ] **Step 4: Run visualization tests**

Run the package command from Step 2 after `gofumpt`.

Expected: PASS.

- [ ] **Step 5: Commit visualization plumbing**

```bash
git add internal/bubbletree internal/spiral internal/scatter internal/donuttree internal/legend
git commit -m "refactor: use geometry points for visual positions" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 5: Verify and open PR 2

**Files:**
- Verify only

- [ ] Run `task test`; expect all packages PASS and no golden changes.
- [ ] Dispatch `task ci` through the required noisy-command agent; expect exit status 0.
- [ ] Run `git status --short`; expect no output.
- [ ] Open a PR titled `Introduce geometry Point primitive`, with the Vector PR branch as its base.
