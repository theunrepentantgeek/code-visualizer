# Surface Boundary Overlays Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove raster gaps at surface rims by overlaying every region boundary with a continuous background-colour path.

**Architecture:** Surface regions optionally provide closed, ordered boundary loops sampled at the same density used for mesh seeding. Spiral requests its annulus loops after polygon rendering and draws each as a 2px path in the existing background colour, before its track, discs, and labels. This keeps boundary treatment reusable without breaking custom regions that do not support overlays.

**Tech Stack:** Go 1.26, Gomega, fogleman/gg raster backend, SVG backend, existing canvas paths.

---

### Task 1: Expose reusable region boundary loops

**Files:**
- Modify: `internal/surface/types.go:33-160`
- Modify: `internal/surface/mesh.go:310-330`
- Test: `internal/surface/mesh_internal_test.go`

- [ ] **Step 1: Write failing boundary-loop tests**

```go
func TestBoundaryLoops_ReturnsClosedInnerAndOuterAnnulusLoops(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	annulus := Annulus{CX: 20, CY: 30, InnerRadius: 10, OuterRadius: 20}

	loops := BoundaryLoops(annulus, MaxBoundarySegmentLength)

	g.Expect(loops).To(HaveLen(2))
	g.Expect(loops[0]).To(HaveLen(126))
	g.Expect(loops[1]).To(HaveLen(63))
	g.Expect(Distance(loops[0][0], loops[0][len(loops[0])-1])).To(BeNumerically("<=", 1.0))
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/surface -run '^TestBoundaryLoops_ReturnsClosedInnerAndOuterAnnulusLoops$' -count=1`

Expected: FAIL because `BoundaryLoops` does not exist.

- [ ] **Step 3: Add the boundary-loop contract and implementations**

```go
type boundaryLoopProvider interface {
	BoundaryLoops(maximumSegmentLength float64) [][]Point
}

func BoundaryLoops(region Region, maximumSegmentLength float64) [][]Point {
	provider, ok := region.(boundaryLoopProvider)
	if !ok {
		return nil
	}
	return provider.BoundaryLoops(maximumSegmentLength)
}

func (a Annulus) BoundaryLoops(maximumSegmentLength float64) [][]Point {
	loops := [][]Point{
		circularBoundaryPoints(a.CX, a.CY, a.OuterRadius, maximumSegmentLength),
	}
	if a.InnerRadius > 0 {
		loops = append(loops, circularBoundaryPoints(
			a.CX, a.CY, a.InnerRadius, maximumSegmentLength,
		))
	}
	return loops
}
```

Make `Rect.BoundaryLoops` return its perimeter as a single ordered loop. Make
`boundarySamples` flatten `BoundaryLoops(region, MaxBoundarySegmentLength)` and
continue to reject non-finite or exact duplicate coordinates. Keeping this
provider optional preserves custom `Region` implementations that only support
meshing without a rendered boundary overlay.

- [ ] **Step 4: Run the focused surface tests**

Run: `go test ./internal/surface -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the reusable boundary contract**

```bash
git add internal/surface/types.go internal/surface/mesh.go internal/surface/mesh_internal_test.go
git commit -m "feat(surface): expose boundary loops"
```

### Task 2: Overlay spiral surface boundaries

**Files:**
- Modify: `internal/spiral/render.go:34-90`
- Modify: `internal/spiral/surface.go:9-35`
- Modify: `internal/spiral/surface_test.go:100-160`
- Modify: `internal/canvas/mock/backend.go:91-96`
- Test: `internal/spiral/surface_test.go`

- [ ] **Step 1: Write a failing rendering-order test**

```go
func TestRenderToCanvas_OverlaysSurfaceBoundaryLoops(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	layout, buckets, triangles := surfaceRenderFixture()
	layout.CX, layout.CY, layout.A, layout.B, layout.MaxTheta = 40, 50, 20, 2, 10

	backend := mock.NewBackend()
	RenderToCanvas(layout, buckets, 160, 120, Inks{
		Fill: numericInk(), Border: numericInk(),
	}, triangles, numericInk()).RenderTo(backend)

	g.Expect(backend.Calls[len(triangles)+1].Method).To(Equal("DrawPath"))
	g.Expect(backend.Calls[len(triangles)+2].Method).To(Equal("DrawPath"))
	g.Expect(backend.Calls[len(triangles)+3].Method).To(Equal("DrawPath"))
	g.Expect(backend.Calls[len(triangles)+1].Fill).To(Equal(color.RGBA{
		R: 255, G: 255, B: 255, A: 255,
	}))
	g.Expect(backend.Calls[len(triangles)+1].StrokeWidth).To(Equal(2.0))
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/spiral -run '^TestRenderToCanvas_OverlaysSurfaceBoundaryLoops$' -count=1`

Expected: FAIL because only the guide-track path is currently rendered.

- [ ] **Step 3: Render continuous boundary overlays after surface polygons**

```go
const surfaceBoundaryWidth = 2.0

func addSurfaceBoundaries(cv *canvas.Canvas, region surface.Region) {
	spec := &canvas.LineSpec{
		Stroke:      inks.FixedInk(bgColour),
		StrokeWidth: surfaceBoundaryWidth,
	}
	for _, loop := range surface.BoundaryLoops(region, surface.MaxBoundarySegmentLength) {
		points := make([]canvas.Position, len(loop)+1)
		for index, point := range loop {
			points[index] = canvas.Position{X: point.X, Y: point.Y}
		}
		points[len(loop)] = points[0]
		cv.AddPath(canvas.LayerSurface, canvas.Path{Spec: spec, Points: points})
	}
}
```

Update the mock backend's `DrawPath` to store the supplied stroke in `Call.Fill`
and supplied width in `Call.StrokeWidth`, enabling the regression assertion.
Use the same `surfaceAnnulus(layout)` helper in both `BuildSurface` and
`RenderToCanvas`, and call `addSurfaceBoundaries` only when triangles are
present. Preserve `addTrack` after the boundary paths so the spiral foreground
remains above the surface treatment.

- [ ] **Step 4: Run focused spiral and surface tests**

Run: `go test ./internal/surface ./internal/spiral -count=1`

Expected: PASS.

- [ ] **Step 5: Regenerate and inspect the manual spiral sample**

Run:

```bash
task build
bin/codeviz spiral . --config samples/spiral/code-visualizer.yml \
  --output samples/spiral/code-visualizer.png \
  --footer 'Generated by github.com/theunrepentantgeek/code-visualizer'
```

Expected: Both annulus rims are continuous white 2px arcs with no
triangle-by-triangle gaps. Do not stage the manual sample artifact.

- [ ] **Step 6: Commit the spiral overlay**

```bash
git add internal/spiral/render.go internal/spiral/surface.go internal/spiral/surface_test.go
git commit -m "fix(surface): overlay boundary paths"
```

### Task 3: Refresh committed surface snapshots and verify integration

**Files:**
- Modify: `internal/goldentest/testdata/spiral-surface-shared-png.golden`
- Modify: `internal/goldentest/testdata/spiral-surface-shared-svg.golden`
- Modify: `internal/goldentest/testdata/spiral-surface-distinct-png.golden`
- Modify: `internal/goldentest/testdata/spiral-surface-distinct-svg.golden`

- [ ] **Step 1: Regenerate only spiral surface Goldie fixtures**

Run:

```bash
GOLDIE_UPDATE=1 go test ./internal/goldentest \
  -run '^TestGolden_SpiralSurface(Shared|Distinct)$' -count=1
```

Expected: PASS and only the four listed fixtures change.

- [ ] **Step 2: Run complete verification**

Run:

```bash
task test
task build
task fmt:check
task lint
```

Expected: all commands exit 0.

- [ ] **Step 3: Commit updated snapshots**

```bash
git add internal/goldentest/testdata/spiral-surface-*.golden
git commit -m "test(surface): cover boundary overlays"
```
