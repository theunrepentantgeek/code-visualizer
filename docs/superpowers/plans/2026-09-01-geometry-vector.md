# Geometry Vector Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Introduce `geometry.Vector` and replace raw displacement pairs and radial-tree relative positions without changing rendered output.

**Architecture:** Add a standard-library-only `internal/geometry` package. Keep `Vector` distinct from future absolute points, then use it for offsets and coordinates explicitly documented as relative to an origin.

**Tech Stack:** Go 1.26.1, `math`, Gomega, Goldie v2

**Stack position:** PR 1 of 5; base this branch on `main`.

---

### Task 1: Add the Vector contract

**Files:**
- Create: `internal/geometry/vector.go`
- Create: `internal/geometry/vector_test.go`

- [ ] **Step 1: Write failing table-driven tests**

Cover finite and non-finite `Valid` cases, arithmetic, dot product, length,
length-squared, zero-vector normalization, invalid normalization, and receiver
non-mutation:

```go
func TestVectorUnit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	unit, ok := (Vector{X: 3, Y: 4}).Unit()
	g.Expect(ok).To(BeTrue())
	g.Expect(unit).To(Equal(Vector{X: 0.6, Y: 0.8}))

	_, ok = (Vector{}).Unit()
	g.Expect(ok).To(BeFalse())
	_, ok = (Vector{X: math.NaN()}).Unit()
	g.Expect(ok).To(BeFalse())
}
```

- [ ] **Step 2: Run the focused test and verify failure**

Run `go test ./internal/geometry -run Vector`.

Expected: FAIL because package `internal/geometry` and `Vector` do not exist.

- [ ] **Step 3: Implement Vector**

```go
package geometry

import "math"

type Vector struct {
	X float64
	Y float64
}

// ZeroVector is the additive identity, representing no displacement.
var ZeroVector = Vector{}

func NewVector(x, y float64) Vector {
	return Vector{X: x, Y: y}
}

func NewRadialVector(angle, length float64) Vector {
	return Vector{X: length * math.Cos(angle), Y: length * math.Sin(angle)}
}

func (v Vector) Valid() bool {
	return !math.IsNaN(v.X) && !math.IsInf(v.X, 0) &&
		!math.IsNaN(v.Y) && !math.IsInf(v.Y, 0)
}

func (v Vector) Add(other Vector) Vector {
	return Vector{X: v.X + other.X, Y: v.Y + other.Y}
}

func (v Vector) Subtract(other Vector) Vector {
	return Vector{X: v.X - other.X, Y: v.Y - other.Y}
}

func (v Vector) Scale(factor float64) Vector {
	return Vector{X: v.X * factor, Y: v.Y * factor}
}

func (v Vector) Dot(other Vector) float64 {
	return v.X*other.X + v.Y*other.Y
}

func (v Vector) LengthSquared() float64 { return v.Dot(v) }
func (v Vector) Length() float64        { return math.Hypot(v.X, v.Y) }

// Unit scales v to length 1. Pre-scaling by the largest absolute component
// before computing Length avoids overflow to +Inf for near-MaxFloat64
// components and overflow of 1/Length for subnormal components, either of
// which would otherwise destroy the direction of v.
func (v Vector) Unit() (Vector, bool) {
	if !v.Valid() {
		return ZeroVector, false
	}

	scale := math.Max(math.Abs(v.X), math.Abs(v.Y))
	if scale == 0 {
		return ZeroVector, false
	}

	scaled := Vector{X: v.X / scale, Y: v.Y / scale}
	length := scaled.Length()

	return Vector{X: scaled.X / length, Y: scaled.Y / length}, true
}
```

- [ ] **Step 4: Run and format the package**

Run `gofumpt -w internal/geometry && go test ./internal/geometry`.

Expected: PASS.

- [ ] **Step 5: Commit the primitive**

```bash
git add internal/geometry
git commit -m "feat(geometry): add vector primitive" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 2: Use Vector for radial-tree offsets

**Files:**
- Modify: `internal/radialtree/node.go`
- Modify: `internal/radialtree/layout.go`
- Modify: `internal/radialtree/render.go`
- Modify: `internal/radialtree/layout_test.go`
- Modify: `internal/radialtree/render_internal_test.go`

- [ ] **Step 1: Update tests to construct and inspect a vector position**

Change `RadialNode` fixtures from `X`/`Y` fields to:

```go
Position: geometry.Vector{X: 30, Y: 40},
```

Use `node.Position.Length()` for distance-from-origin assertions. Keep all
existing expected coordinates unchanged.

- [ ] **Step 2: Run radial-tree tests and verify compile failure**

Run `go test ./internal/radialtree`.

Expected: FAIL because `RadialNode.Position` does not exist.

- [ ] **Step 3: Replace the relative coordinate pair**

Define:

```go
type RadialNode struct {
	Position    geometry.Vector
	DiscRadius  float64
	Angle       float64
	Label       string
	ShowLabel   bool
	IsDirectory bool
	Children    []RadialNode
}
```

In layout, assign `geometry.NewRadialVector(angle, radius)`.
In rendering, replace `node.X` and `node.Y` with `node.Position.X` and
`node.Position.Y`, and replace `nodeDistance` with `node.Position.Length()`.

- [ ] **Step 4: Run radial-tree tests**

Run `gofumpt -w internal/radialtree && go test ./internal/radialtree`.

Expected: PASS with existing assertions unchanged apart from field access.

- [ ] **Step 5: Commit the radial migration**

```bash
git add internal/radialtree
git commit -m "refactor(radialtree): model positions as vectors" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 3: Replace paired offset parameters

**Files:**
- Modify: `internal/bubbletree/transforms.go`
- Modify: `internal/bubbletree/transforms_test.go`
- Modify: `internal/scatter/layout.go`
- Modify: `internal/scatter/layout_test.go`
- Modify: `internal/treemap/layout.go`
- Modify: `internal/treemap/layout_test.go`
- Modify: `internal/bubbletree/stages.go`
- Modify: `internal/scatter/stages.go`
- Modify: `internal/treemap/stages.go`
- Modify: `internal/spiral/stages.go`

- [ ] **Step 1: Change offset-focused tests to pass Vector**

Use:

```go
offset := geometry.Vector{X: 11, Y: 17}
OffsetNodes(&root, offset)
OffsetLayout(&layout, offset)
OffsetRects(&root, offset)
```

Preserve existing expected final coordinates exactly.

- [ ] **Step 2: Run affected tests and verify compile failure**

Run:

```bash
go test ./internal/bubbletree ./internal/scatter ./internal/treemap ./internal/spiral
```

Expected: FAIL because the offset functions still accept two `float64` values.

- [ ] **Step 3: Change offset APIs and call sites**

Use one displacement argument:

```go
func OffsetNodes(node *BubbleNode, offset geometry.Vector)
func OffsetLayout(layout *ScatterLayout, offset geometry.Vector)
func OffsetRects(rect *TreemapRectangle, offset geometry.Vector)
```

Replace each `dx`/`dy` use with `offset.X`/`offset.Y`, including recursive calls.
At stage call sites construct `geometry.Vector{X: dx, Y: dy}`. In spiral, where
only vertical displacement is used, construct `geometry.Vector{Y: dy}` before
applying it to node coordinates.

- [ ] **Step 4: Run affected package tests**

Run:

```bash
gofumpt -w internal/bubbletree internal/scatter internal/treemap internal/spiral
go test ./internal/bubbletree ./internal/scatter ./internal/treemap ./internal/spiral
```

Expected: PASS.

- [ ] **Step 5: Commit displacement plumbing**

```bash
git add internal/bubbletree internal/scatter internal/treemap internal/spiral
git commit -m "refactor: pass layout offsets as vectors" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 4: Verify and open PR 1

**Files:**
- Verify only

- [ ] **Step 1: Run full tests without golden updates**

Run `task test`.

Expected: all packages PASS and no golden files change.

- [ ] **Step 2: Run repository CI through the required noisy-command agent**

Dispatch `task ci` to an Explore-equivalent agent and require only exit status,
failing checks, and file/line diagnostics.

Expected: exit status 0.

- [ ] **Step 3: Confirm a clean worktree**

Run `git status --short`.

Expected: no output.

- [ ] **Step 4: Open the first stacked PR**

Use the repository PR workflow with title `Introduce geometry Vector primitive`.
Record its branch as the base for the Point PR.
