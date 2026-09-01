# Geometry Rect Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Introduce `geometry.Rect` and consolidate repeated bounds and rectangle arithmetic across surface, pipelines, canvas, scatter, treemap, and bubble tree.

**Architecture:** Represent rectangles by ordered minimum and maximum points and provide explicit conversion from position-plus-size. Domain records retain labels and hierarchy while containing a Rect; raster clipping continues to use `image.Rectangle`.

**Tech Stack:** Go 1.26.1, `math`, Gomega, Goldie v2, `image.Rectangle`

**Stack position:** PR 4 of 5; base this branch on the Size PR branch.

---

### Task 1: Add the Rect contract

**Files:**
- Create: `internal/geometry/rect.go`
- Create: `internal/geometry/rect_test.go`

- [ ] **Step 1: Write failing Rect tests**

Cover XYWH construction, validity, empty axes, dimensions, size, centre,
inclusive containment, translation, positive/negative inset, over-inset failure,
expansion, union, invalid operands, NaN/Inf, unordered endpoints, and
non-mutation:

```go
func TestRectOperations(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	rect := Rect{Min: Point{X: 10, Y: 20}, Max: Point{X: 30, Y: 50}}

	g.Expect(rect.Size()).To(Equal(Size{Width: 20, Height: 30}))
	g.Expect(rect.Center()).To(Equal(Point{X: 20, Y: 35}))
	g.Expect(rect.Contains(rect.Min)).To(BeTrue())
	g.Expect(rect.Contains(rect.Max)).To(BeTrue())

	inset, ok := rect.Inset(5)
	g.Expect(ok).To(BeTrue())
	g.Expect(inset).To(Equal(Rect{
		Min: Point{X: 15, Y: 25},
		Max: Point{X: 25, Y: 45},
	}))
}
```

- [ ] **Step 2: Verify focused failure**

Run `go test ./internal/geometry -run Rect`.

Expected: FAIL because `Rect` is undefined.

- [ ] **Step 3: Implement Rect**

```go
package geometry

import "math"

type Rect struct {
	Min Point
	Max Point
}

func RectFromPositionSize(position Point, size Size) Rect {
	return Rect{
		Min: position,
		Max: Point{X: position.X + size.Width, Y: position.Y + size.Height},
	}
}

func (r Rect) Valid() bool {
	return r.Min.Valid() && r.Max.Valid() &&
		r.Min.X <= r.Max.X && r.Min.Y <= r.Max.Y
}

func (r Rect) Empty() bool {
	return r.Valid() && (r.Min.X == r.Max.X || r.Min.Y == r.Max.Y)
}

func (r Rect) Width() float64  { return r.Max.X - r.Min.X }
func (r Rect) Height() float64 { return r.Max.Y - r.Min.Y }
func (r Rect) Size() Size      { return Size{Width: r.Width(), Height: r.Height()} }
func (r Rect) Center() Point   { return Midpoint(r.Min, r.Max) }

func (r Rect) Contains(point Point) bool {
	return r.Valid() && point.Valid() &&
		point.X >= r.Min.X && point.X <= r.Max.X &&
		point.Y >= r.Min.Y && point.Y <= r.Max.Y
}

func (r Rect) Translate(offset Vector) Rect {
	return Rect{Min: r.Min.Translate(offset), Max: r.Max.Translate(offset)}
}

func (r Rect) Inset(amount float64) (Rect, bool) {
	if !r.Valid() || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return Rect{}, false
	}
	result := Rect{
		Min: Point{X: r.Min.X + amount, Y: r.Min.Y + amount},
		Max: Point{X: r.Max.X - amount, Y: r.Max.Y - amount},
	}
	return result, result.Valid()
}

func (r Rect) ExpandToInclude(point Point) (Rect, bool) {
	if !r.Valid() || !point.Valid() {
		return Rect{}, false
	}
	return Rect{
		Min: Point{X: min(r.Min.X, point.X), Y: min(r.Min.Y, point.Y)},
		Max: Point{X: max(r.Max.X, point.X), Y: max(r.Max.Y, point.Y)},
	}, true
}

func (r Rect) Union(other Rect) (Rect, bool) {
	if !r.Valid() || !other.Valid() {
		return Rect{}, false
	}
	return Rect{
		Min: Point{X: min(r.Min.X, other.Min.X), Y: min(r.Min.Y, other.Min.Y)},
		Max: Point{X: max(r.Max.X, other.Max.X), Y: max(r.Max.Y, other.Max.Y)},
	}, true
}
```

- [ ] **Step 4: Run and format geometry tests**

Run `gofumpt -w internal/geometry && go test ./internal/geometry`.

Expected: PASS.

- [ ] **Step 5: Commit Rect**

```bash
git add internal/geometry
git commit -m "feat(geometry): add rectangle primitive" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 2: Migrate surface and pipeline bounds

**Files:**
- Modify: `internal/surface/types.go`
- Modify: `internal/surface/mesh.go`
- Modify: `internal/surface/poisson.go`
- Modify: `internal/surface/mesh_internal_test.go`
- Modify: `internal/surface/mesh_refinement_test.go`
- Modify: `internal/surface/mesh_test.go`
- Modify: `internal/surface/poisson_test.go`
- Modify: `internal/stages/common.go`
- Modify: `internal/stages/canvas.go`
- Modify: `internal/stages/canvas_test.go`
- Modify: `internal/stages/dimensions_test.go`
- Modify: `internal/bubbletree/stages.go`
- Modify: `internal/radialtree/stages.go`
- Modify: `internal/spiral/stages.go`
- Modify: `internal/scatter/stages.go`
- Modify: `internal/treemap/stages.go`
- Modify: `internal/donuttree/stages.go`

- [ ] **Step 1: Update tests to construct geometry.Rect bounds**

Replace `surface.Rect` and `stages.DrawingBounds` literals with
`geometry.Rect{Min: ..., Max: ...}`. Preserve inclusive containment and every
existing width, height, and reserved-layout expectation.

- [ ] **Step 2: Verify focused compile failure**

Run `go test ./internal/surface ./internal/stages ./internal/bubbletree ./internal/radialtree ./internal/spiral ./internal/scatter ./internal/treemap ./internal/donuttree`.

Expected: FAIL until region and pipeline signatures use `geometry.Rect`.

- [ ] **Step 3: Replace duplicate bounds types**

Change `Region.Bounds()` to return `geometry.Rect` and
`Region.Contains` to accept `geometry.Point`. Delete `surface.Rect`.

Replace `DrawingBounds` with `geometry.Rect` in `CommonState`. Convert resolved
integer dimensions once:

```go
geometry.Rect{
	Min: geometry.Point{},
	Max: geometry.Point{X: float64(width), Y: float64(height)},
}
```

Use `Width`, `Height`, `Contains`, `Center`, `Inset`, and `Translate`. Perform
explicit integer conversion only when interacting with raster allocation or
existing integer command configuration.

- [ ] **Step 4: Run affected tests**

Run the package command from Step 2 after `gofumpt`.

Expected: PASS.

- [ ] **Step 5: Commit bounds migration**

```bash
git add internal/surface internal/stages internal/bubbletree internal/radialtree \
  internal/spiral internal/scatter internal/treemap internal/donuttree
git commit -m "refactor: use geometry rectangles for drawing bounds" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 3: Migrate scatter, treemap, and bubble bounds

**Files:**
- Modify: `internal/scatter/axis.go`
- Modify: `internal/scatter/axis_resolve.go`
- Modify: `internal/scatter/layout.go`
- Modify: `internal/scatter/render.go`
- Modify: `internal/scatter/axis_resolve_test.go`
- Modify: `internal/scatter/layout_test.go`
- Modify: `internal/treemap/node.go`
- Modify: `internal/treemap/layout.go`
- Modify: `internal/treemap/directory_chrome.go`
- Modify: `internal/treemap/labels.go`
- Modify: `internal/treemap/render.go`
- Modify: `internal/treemap/directory_chrome_test.go`
- Modify: `internal/treemap/labels_test.go`
- Modify: `internal/treemap/layout_test.go`
- Modify: `internal/treemap/render_directory_chrome_test.go`
- Modify: `internal/bubbletree/transforms.go`
- Modify: `internal/bubbletree/transforms_test.go`

- [ ] **Step 1: Rewrite bounds fixtures**

Replace `PlotRect`, `RectangleBounds`, `labelBounds`, and bubble `bounds`
fixtures with `geometry.Rect`. Keep domain records but give each a `Bounds`
field:

```go
type TreemapRectangle struct {
	Bounds       geometry.Rect
	VisibleDepth int
	Label        string
	IsDirectory  bool
	Chrome       DirectoryChrome
	Children     []TreemapRectangle
}
```

- [ ] **Step 2: Verify package compile failures**

Run `go test ./internal/scatter ./internal/treemap ./internal/bubbletree`.

Expected: FAIL until production code uses `geometry.Rect`.

- [ ] **Step 3: Consolidate rectangle operations**

Delete `PlotRect`, `RectangleBounds`, `labelBounds`, and bubble `bounds`.
Represent directory rail/content rectangles as `geometry.Rect`. Convert
third-party `layout.Box` values through `RectFromPositionSize`.

Use `Rect.Translate` for offsets, `Inset` for label/content padding,
`ExpandToInclude`/`Union` for occupied bounds, and `Size`/`Center` instead of
repeated coordinate arithmetic. Preserve existing behavior for empty bounds by
tracking whether an accumulator has received its first rectangle rather than
using MaxFloat sentinels.

- [ ] **Step 4: Run package tests**

Run `gofumpt -w internal/scatter internal/treemap internal/bubbletree && go test ./internal/scatter ./internal/treemap ./internal/bubbletree`.

Expected: PASS with unchanged layout numbers.

- [ ] **Step 5: Commit visualization rectangle migration**

```bash
git add internal/scatter internal/treemap internal/bubbletree
git commit -m "refactor: consolidate visualization rectangles" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 4: Make canvas rectangles first-class

**Files:**
- Modify: `internal/canvas/rectangle.go`
- Modify: `internal/canvas/block_label.go`
- Modify: `internal/canvas/model/backend.go`
- Modify: `internal/canvas/mock/backend.go`
- Modify: `internal/canvas/raster/backend.go`
- Modify: `internal/canvas/svg/backend.go`
- Modify: `internal/canvas/canvas_test.go`
- Modify: `internal/canvas/raster/backend_test.go`
- Modify: `internal/canvas/raster/gradient_test.go`
- Modify: `internal/canvas/svg/backend_test.go`
- Modify: `internal/canvas/svg/gradient_test.go`
- Modify: `internal/legend/render.go`
- Modify: `internal/legend/render_test.go`

- [ ] **Step 1: Update rectangle call expectations**

Change mock/backend and retained-shape tests to use one `geometry.Rect`. Keep all
expected SVG attributes and raster extents unchanged.

- [ ] **Step 2: Verify canvas compile failure**

Run `go test ./internal/canvas/... ./internal/legend`.

Expected: FAIL until rectangle APIs accept `geometry.Rect`.

- [ ] **Step 3: Replace position-plus-size rectangle APIs**

Change backend `DrawRectangle` to receive `geometry.Rect`. Change retained
`canvas.Rectangle` and `BlockLabel` geometry to a `Bounds geometry.Rect` field.
Backends derive position and size through `Min`, `Width`, and `Height`. Continue
using `image.Rect` for pixel clipping.

- [ ] **Step 4: Run canvas and legend tests**

Run `gofumpt -w internal/canvas internal/legend && go test ./internal/canvas/... ./internal/legend`.

Expected: PASS.

- [ ] **Step 5: Commit canvas rectangles**

```bash
git add internal/canvas internal/legend
git commit -m "refactor(canvas): use geometry rectangles" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 5: Verify and open PR 4

**Files:**
- Verify only

- [ ] Run `task test`; expect all packages PASS and no golden changes.
- [ ] Dispatch `task ci` through the required noisy-command agent; expect exit status 0.
- [ ] Run `git status --short`; expect no output.
- [ ] Open a PR titled `Introduce geometry Rect primitive`, with the Size PR branch as its base.
