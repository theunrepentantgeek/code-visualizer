# Geometry Size Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Introduce `geometry.Size` and use it for resolved floating-point width/height pairs while preserving configuration and rendering behavior.

**Architecture:** Add a literal-friendly dimensions value to `internal/geometry`. Migrate canvas and layout boundaries where width and height travel together; retain integer and optional configuration dimensions at their existing domain boundaries.

**Tech Stack:** Go 1.26.1, `math`, Gomega, Goldie v2

**Stack position:** PR 3 of 5; base this branch on the Point PR branch.

---

### Task 1: Add the Size contract

**Files:**
- Create: `internal/geometry/size.go`
- Create: `internal/geometry/size_test.go`

- [ ] **Step 1: Write failing Size tests**

Cover the zero value, finite positive dimensions, one zero dimension, negative
dimensions, NaN/Inf, area, scaling, aspect ratio, invalid aspect ratio, and
receiver non-mutation:

```go
func TestSizeAspectRatio(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ratio, ok := (Size{Width: 16, Height: 9}).AspectRatio()
	g.Expect(ok).To(BeTrue())
	g.Expect(ratio).To(Equal(16.0 / 9.0))

	_, ok = (Size{Width: 16}).AspectRatio()
	g.Expect(ok).To(BeFalse())
	_, ok = (Size{Width: -1, Height: 9}).AspectRatio()
	g.Expect(ok).To(BeFalse())
}
```

- [ ] **Step 2: Verify focused failure**

Run `go test ./internal/geometry -run Size`.

Expected: FAIL because `Size` is undefined.

- [ ] **Step 3: Implement Size**

```go
package geometry

import "math"

type Size struct {
	Width  float64
	Height float64
}

func (s Size) Valid() bool {
	return !math.IsNaN(s.Width) && !math.IsInf(s.Width, 0) &&
		!math.IsNaN(s.Height) && !math.IsInf(s.Height, 0) &&
		s.Width >= 0 && s.Height >= 0
}

func (s Size) Empty() bool {
	return s.Valid() && (s.Width == 0 || s.Height == 0)
}

func (s Size) Area() float64 { return s.Width * s.Height }

func (s Size) Scale(factor float64) Size {
	return Size{Width: s.Width * factor, Height: s.Height * factor}
}

func (s Size) AspectRatio() (float64, bool) {
	if !s.Valid() || s.Height == 0 {
		return 0, false
	}
	return s.Width / s.Height, true
}
```

- [ ] **Step 4: Run and format geometry tests**

Run `gofumpt -w internal/geometry && go test ./internal/geometry`.

Expected: PASS.

- [ ] **Step 5: Commit Size**

```bash
git add internal/geometry
git commit -m "feat(geometry): add size primitive" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 2: Replace canvas Size

**Files:**
- Modify: `internal/canvas/model/backend.go`
- Modify: `internal/canvas/geometry.go`
- Modify: `internal/canvas/mock/backend.go`
- Modify: `internal/canvas/raster/backend.go`
- Modify: `internal/canvas/svg/backend.go`
- Modify: `internal/canvas/rectangle.go`
- Modify: `internal/canvas/canvas_test.go`
- Modify: `internal/canvas/raster/backend_test.go`
- Modify: `internal/canvas/svg/backend_test.go`

- [ ] **Step 1: Change backend test calls to geometry.Size**

Replace `model.Size` and `canvas.Size` literals with:

```go
geometry.Size{Width: 200, Height: 100}
```

Do not alter expected SVG numbers, raster bounds, or mock calls.

- [ ] **Step 2: Verify canvas compile failure**

Run `go test ./internal/canvas/...`.

Expected: FAIL because backend methods still require the old size type.

- [ ] **Step 3: Migrate backend signatures**

Change `DrawRectangle` and all implementations/mocks to accept
`geometry.Size`. Delete `model.Size` and remove the `canvas.Size` alias from
`internal/canvas/geometry.go`. Where rectangle retained data still has `W` and
`H`, construct:

```go
size := geometry.Size{Width: rectangle.W, Height: rectangle.H}
```

The Rect PR will replace the entire retained rectangle.

- [ ] **Step 4: Run canvas tests**

Run `gofumpt -w internal/canvas && go test ./internal/canvas/...`.

Expected: PASS with identical output assertions.

- [ ] **Step 5: Commit canvas Size migration**

```bash
git add internal/canvas
git commit -m "refactor(canvas): use geometry sizes" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 3: Group resolved layout dimensions

**Files:**
- Modify: `internal/canvas/canvas.go`
- Modify: `internal/canvas/canvas_test.go`
- Modify: `internal/canvas/legendlayout/layout.go`
- Modify: `internal/canvas/legendlayout/layout_test.go`
- Modify: `internal/legend/reserve.go`
- Modify: `internal/legend/reserve_test.go`

- [ ] **Step 1: Add tests that pass one Size value**

Change helpers that currently receive paired floating-point dimensions to accept:

```go
available := geometry.Size{Width: 640, Height: 480}
```

Keep all expected origins, reserved space, and text measurements unchanged.

- [ ] **Step 2: Verify focused compile failure**

Run `go test ./internal/canvas/... ./internal/legend`.

Expected: FAIL until production signatures accept `geometry.Size`.

- [ ] **Step 3: Migrate dimension pairs**

Use `geometry.Size` where resolved floating-point width and height are passed or
stored together. Keep `canvas.NewCanvas(width, height int)` and pipeline
`CommonState.Width`/`Height` unchanged because they are resolved integer output
dimensions and changing the constructor is not needed by consumers. Convert to
`geometry.Size` at floating-point layout boundaries.

Do not modify `config.ImageSize`: its optional `*int` fields represent user
configuration rather than geometry.

- [ ] **Step 4: Run affected tests**

Run `gofumpt -w internal/canvas internal/legend && go test ./internal/canvas/... ./internal/legend`.

Expected: PASS.

- [ ] **Step 5: Commit layout Size plumbing**

```bash
git add internal/canvas internal/legend
git commit -m "refactor: group layout dimensions as geometry sizes" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 4: Verify and open PR 3

**Files:**
- Verify only

- [ ] Run `task test`; expect all packages PASS and no golden changes.
- [ ] Dispatch `task ci` through the required noisy-command agent; expect exit status 0.
- [ ] Run `git status --short`; expect no output.
- [ ] Open a PR titled `Introduce geometry Size primitive`, with the Point PR branch as its base.
