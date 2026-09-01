# Geometry Circle Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Introduce `geometry.Circle`, consolidate centre/radius records and calculations, and remove the remaining duplicate primitive implementations.

**Architecture:** Compose Circle from Point and keep exact generic predicates separate from algorithm-specific tolerances. Visualization records retain metadata while containing a Circle; annuli and donut sectors remain domain types.

**Tech Stack:** Go 1.26.1, `math`, Gomega, Goldie v2

**Stack position:** PR 5 of 5; base this branch on the Rect PR branch.

---

### Task 1: Add the Circle contract

**Files:**
- Create: `internal/geometry/circle.go`
- Create: `internal/geometry/circle_test.go`

- [ ] **Step 1: Write failing Circle tests**

Cover zero-value validity, negative/NaN/Inf radius, invalid centre, centre and
boundary containment, disjoint/tangent/overlapping intersections, enclosure,
bounds, translation, invalid predicates, and receiver non-mutation:

```go
func TestCirclePredicates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	circle := Circle{Center: Point{X: 10, Y: 20}, Radius: 5}

	g.Expect(circle.Contains(Point{X: 15, Y: 20})).To(BeTrue())
	g.Expect(circle.Contains(Point{X: 15.1, Y: 20})).To(BeFalse())
	g.Expect(circle.Intersects(Circle{
		Center: Point{X: 20, Y: 20},
		Radius: 5,
	})).To(BeTrue())
	g.Expect(circle.Bounds()).To(Equal(Rect{
		Min: Point{X: 5, Y: 15},
		Max: Point{X: 15, Y: 25},
	}))
}
```

- [ ] **Step 2: Verify focused failure**

Run `go test ./internal/geometry -run Circle`.

Expected: FAIL because `Circle` is undefined.

- [ ] **Step 3: Implement Circle**

```go
package geometry

import "math"

type Circle struct {
	Center Point
	Radius float64
}

func (c Circle) Valid() bool {
	return c.Center.Valid() &&
		!math.IsNaN(c.Radius) && !math.IsInf(c.Radius, 0) &&
		c.Radius >= 0
}

func (c Circle) Contains(point Point) bool {
	if !c.Valid() || !point.Valid() {
		return false
	}
	return c.Center.DistanceSquaredTo(point) <= c.Radius*c.Radius
}

func (c Circle) Encloses(other Circle) bool {
	if !c.Valid() || !other.Valid() || other.Radius > c.Radius {
		return false
	}
	distance := c.Center.DistanceTo(other.Center)
	return distance+other.Radius <= c.Radius
}

func (c Circle) Intersects(other Circle) bool {
	if !c.Valid() || !other.Valid() {
		return false
	}
	radii := c.Radius + other.Radius
	return c.Center.DistanceSquaredTo(other.Center) <= radii*radii
}

func (c Circle) Bounds() Rect {
	offset := Vector{X: c.Radius, Y: c.Radius}
	return Rect{
		Min: c.Center.Translate(offset.Scale(-1)),
		Max: c.Center.Translate(offset),
	}
}

func (c Circle) Translate(offset Vector) Circle {
	return Circle{Center: c.Center.Translate(offset), Radius: c.Radius}
}
```

- [ ] **Step 4: Run and format geometry tests**

Run `gofumpt -w internal/geometry && go test ./internal/geometry`.

Expected: PASS.

- [ ] **Step 5: Commit Circle**

```bash
git add internal/geometry
git commit -m "feat(geometry): add circle primitive" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 2: Migrate canvas discs

**Files:**
- Modify: `internal/canvas/disc.go`
- Modify: `internal/canvas/model/backend.go`
- Modify: `internal/canvas/mock/backend.go`
- Modify: `internal/canvas/raster/backend.go`
- Modify: `internal/canvas/svg/backend.go`
- Modify: `internal/canvas/canvas_test.go`
- Modify: `internal/canvas/raster/backend_test.go`
- Modify: `internal/canvas/svg/backend_test.go`

- [ ] **Step 1: Change disc fixtures and mock calls to Circle**

Use:

```go
Geometry: geometry.Circle{
	Center: geometry.Point{X: 100, Y: 80},
	Radius: 20,
},
```

Preserve every expected draw call, SVG attribute, and raster extent.

- [ ] **Step 2: Verify canvas compile failure**

Run `go test ./internal/canvas/...`.

Expected: FAIL because disc APIs still accept separate centre and radius values.

- [ ] **Step 3: Replace disc geometry**

Change `canvas.Disc` to contain `Geometry geometry.Circle`. Change
`Backend.DrawDisc` and all implementations/mocks to accept a Circle. Use
`circle.Center`, `circle.Radius`, and `circle.Bounds()` in gradient and clipping
calculations.

- [ ] **Step 4: Run canvas tests**

Run `gofumpt -w internal/canvas && go test ./internal/canvas/...`.

Expected: PASS with identical backend output.

- [ ] **Step 5: Commit canvas Circle migration**

```bash
git add internal/canvas
git commit -m "refactor(canvas): use geometry circles for discs" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 3: Migrate bubble circles and enclosing calculations

**Files:**
- Modify: `internal/bubbletree/node.go`
- Modify: `internal/bubbletree/geometry.go`
- Modify: `internal/bubbletree/packing.go`
- Modify: `internal/bubbletree/layout.go`
- Modify: `internal/bubbletree/transforms.go`
- Modify: `internal/bubbletree/render.go`
- Modify: `internal/bubbletree/geometry_test.go`
- Modify: `internal/bubbletree/layout_stage_test.go`
- Modify: `internal/bubbletree/layout_test.go`
- Modify: `internal/bubbletree/packing_test.go`
- Modify: `internal/bubbletree/render_test.go`
- Modify: `internal/bubbletree/transforms_test.go`

- [ ] **Step 1: Rewrite bubble fixtures around Geometry**

Change bubble node fixtures to:

```go
Geometry: geometry.Circle{
	Center: geometry.Point{X: 10, Y: 20},
	Radius: 5,
},
```

Keep layout and enclosure expectations numerically identical.

- [ ] **Step 2: Verify bubble-tree compile failure**

Run `go test ./internal/bubbletree`.

Expected: FAIL until `BubbleNode` and algorithms use Circle.

- [ ] **Step 3: Replace duplicate circle records**

Give `BubbleNode` a `Geometry geometry.Circle` field. Replace private
`enclosure` with `geometry.Circle` throughout Welzl's algorithm. Use
`Circle.Bounds` for occupied bounds and `Translate` when applying layout
offsets. Use the local tolerance-aware helper below instead of the exact
`Circle.Encloses` inside Welzl's algorithm.

For the algorithm's existing `1e-6` enclosure tolerance, keep a local helper:

```go
func enclosesWithin(outer, inner geometry.Circle, tolerance float64) bool {
	if !outer.Valid() || !inner.Valid() {
		return false
	}
	return outer.Center.DistanceTo(inner.Center)+inner.Radius <=
		outer.Radius+tolerance
}
```

Do not move this tolerance into `geometry.Circle.Encloses`.

- [ ] **Step 4: Run bubble-tree tests**

Run `gofumpt -w internal/bubbletree && go test ./internal/bubbletree`.

Expected: PASS.

- [ ] **Step 5: Commit bubble Circle migration**

```bash
git add internal/bubbletree
git commit -m "refactor(bubbletree): model nodes as geometry circles" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 4: Migrate remaining disc-bearing visualizations

**Files:**
- Modify: `internal/scatter/layout.go`
- Modify: `internal/scatter/render.go`
- Modify: `internal/scatter/layout_test.go`
- Modify: `internal/radialtree/node.go`
- Modify: `internal/radialtree/layout.go`
- Modify: `internal/radialtree/render.go`
- Modify: `internal/radialtree/layout_test.go`
- Modify: `internal/radialtree/render_internal_test.go`
- Modify: `internal/spiral/node.go`
- Modify: `internal/spiral/layout.go`
- Modify: `internal/spiral/discsize.go`
- Modify: `internal/spiral/render.go`
- Modify: `internal/spiral/discsize_test.go`
- Modify: `internal/spiral/layout_test.go`
- Modify: `internal/spiral/render_test.go`
- Modify: `internal/spiral/stages_test.go`
- Modify: `internal/donuttree/node.go`
- Modify: `internal/donuttree/layout.go`
- Modify: `internal/donuttree/render.go`
- Modify: `internal/donuttree/layout_test.go`
- Modify: `internal/donuttree/render_test.go`

- [ ] **Step 1: Update fixtures to contain Circle**

Use `Geometry geometry.Circle` for scatter and spiral laid-out discs. For radial
nodes, retain the relative `geometry.Vector` and radius until converting to an
absolute Circle during rendering:

```go
circle := geometry.Circle{
	Center: canvasCenter.Translate(node.Position),
	Radius: node.DiscRadius,
}
```

For donut layout, retain annulus-sector radii and angles, but use
`geometry.Circle{Center: layout.Center, Radius: radius}` for boundary and anchor
calculations.

- [ ] **Step 2: Verify package compile failures**

Run `go test ./internal/scatter ./internal/radialtree ./internal/spiral ./internal/donuttree`.

Expected: FAIL until records and render calls use Circle.

- [ ] **Step 3: Migrate circle concepts without flattening domain records**

Replace centre/radius pairs only where they form one disc. Keep spiral radius,
radial offset, angles, annular inner/outer radii, labels, metrics, and hierarchy
as domain fields. Use `Contains`, `Intersects`, `Bounds`, and `Translate` only
where they exactly replace existing formulas.

- [ ] **Step 4: Run visualization tests**

Run the package command from Step 2 after `gofumpt`.

Expected: PASS.

- [ ] **Step 5: Commit remaining Circle plumbing**

```bash
git add internal/scatter internal/radialtree internal/spiral internal/donuttree
git commit -m "refactor: use geometry circles for visual discs" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 5: Remove transitional geometry definitions

**Files:**
- Modify or delete: `internal/canvas/geometry.go`
- Search all production `*.go` files under `internal/`

- [ ] **Step 1: Prove no duplicate primitive definitions remain**

Run:

```bash
git grep -nE '^type (Point|Position|Size|Rect|PlotRect|RectangleBounds|Circle|enclosure|bounds) struct' -- 'internal/*.go' 'internal/**/*.go'
```

Expected before cleanup: only intentional domain types plus any transitional
aliases or duplicate definitions identified during the stack.

- [ ] **Step 2: Remove transitional aliases and duplicates**

Delete obsolete `canvas.Position`, `canvas.Size`, canvas-model equivalents,
surface Rect, scatter PlotRect, treemap RectangleBounds, bubble bounds/enclosure,
and any compatibility-only aliases. Keep `GradientPoint`, `surface.Sample`,
`image.Rectangle`, and domain records explicitly retained by the design.

- [ ] **Step 3: Verify the final type inventory**

Run the search from Step 1 again.

Expected: `geometry.Point`, `geometry.Size`, `geometry.Rect`, and
`geometry.Circle` are the sole reusable definitions; remaining matches have
domain-specific semantics documented by their names.

- [ ] **Step 4: Run all tests**

Run `task test`.

Expected: PASS with no golden changes.

- [ ] **Step 5: Commit cleanup**

```bash
git add internal
git commit -m "refactor: remove duplicate geometry primitives" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 6: Verify and open PR 5

**Files:**
- Verify only

- [ ] Dispatch `task ci` through the required noisy-command agent; expect exit status 0.
- [ ] Run `git status --short`; expect no output.
- [ ] Open a PR titled `Introduce geometry Circle primitive`, with the Rect PR branch as its base.
- [ ] Confirm the five PR bases form `main → Vector → Point → Size → Rect → Circle`.
