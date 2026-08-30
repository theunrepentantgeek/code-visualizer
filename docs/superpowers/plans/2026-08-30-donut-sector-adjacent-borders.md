# Donut Sector Adjacent Borders Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Render neighboring donut-sector metric borders beside one another instead of on the same centerline.

**Architecture:** Keep each sector's current polygon as an unchanged fill-only shape. When border metrics are enabled, add a second transparent polygon whose circular edges are inset by half the stroke width and whose radial edges are parallel offsets inside the sector; the centered stroke then reaches, but does not cross, the logical sector boundary.

**Tech Stack:** Go 1.26, canvas polygon primitives, fogleman/gg raster rendering, SVG backend, Gomega, Goldie v2

---

### Task 1: Prove the overlapping-centerline regression

**Files:**
- Modify: `internal/donuttree/render_test.go:153-184`

- [ ] **Step 1: Replace the border-width-only test with a failing geometry regression**

Add a two-sibling fixture with a configured border metric, render it through the
mock backend, and assert that rendering produces one unchanged fill polygon and
one border polygon per sector:

```go
func TestRenderToCanvas_InsetsMetricBordersInsideAdjacentSectors(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	left := donutDirectory("left", 100)
	right := donutDirectory("right", 100)
	root := donutDirectory("root", 200)
	root.Dirs = []*model.Directory{left, right}

	const borderMetric = metric.Name("file-freshness.sum")
	left.SetQuantity(borderMetric, 1)
	right.SetQuantity(borderMetric, 2)

	layout := Layout(root, 600, filesystem.FileLines)
	is := BuildInks(
		root,
		stages.CollectRequestedMetrics(borderMetric),
		filesystem.FileLines,
		palette.Neutral,
		borderMetric,
		palette.GoodBad,
	)
	polygons := callsNamed(renderCalls(t, RenderToCanvas(
		layout, root, 600, 600, is, LabelMetrics{Size: filesystem.FileLines},
	)), "DrawPolygon")

	g.Expect(polygons).To(HaveLen(4))
	for index := 0; index < len(polygons); index += 2 {
		fill := polygons[index]
		border := polygons[index+1]
		g.Expect(fill.BorderWidth).To(BeZero())
		g.Expect(fill.Points).To(Equal(sectorPoints(layout.Children[index/2], layout.Center)))
		g.Expect(border.Fill.A).To(BeZero())
		g.Expect(border.BorderWidth).To(Equal(donutSectorBorderWidth))
	}

	leftBorder := polygons[1].Points
	rightBorder := polygons[3].Points
	leftEnd := leftBorder[len(leftBorder)/2-1]
	rightStart := rightBorder[0]
	g.Expect(math.Hypot(leftEnd.X-rightStart.X, leftEnd.Y-rightStart.Y)).
		To(BeNumerically("~", donutSectorBorderWidth, 0.000001))
}
```

Retain the existing no-border assertions in a separate
`TestRenderToCanvas_OmitsBorderPolygonsUnlessConfigured` test and assert that
the no-border case still emits exactly one polygon per directory with zero
border width.

- [ ] **Step 2: Run the focused test to verify it fails**

Run:

```bash
go test ./internal/donuttree -run 'TestRenderToCanvas_(InsetsMetricBordersInsideAdjacentSectors|OmitsBorderPolygonsUnlessConfigured)$'
```

Expected: FAIL because current rendering emits only two polygons and centers
both sector borders on their shared boundary.

- [ ] **Step 3: Commit the failing regression test**

```bash
git add internal/donuttree/render_test.go
git commit -m "test: reproduce overlapping donut sector borders" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 2: Render inset border polygons

**Files:**
- Modify: `internal/donuttree/render.go:75-131`
- Modify: `internal/donuttree/render_test.go:225-262`

- [ ] **Step 1: Add focused geometry tests for inset arcs and narrow sectors**

Add a test that calls `insetSectorPoints` directly and verifies all points are
finite, outer samples use `OuterRadius - donutSectorBorderWidth/2`, inner
samples use `InnerRadius + donutSectorBorderWidth/2`, and radial endpoints are
half a stroke width from the original radial edge:

```go
func TestInsetSectorPoints_KeepsBorderStrokeInsideSector(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	node := DonutNode{
		StartAngle: 0, SweepAngle: math.Pi / 3,
		InnerRadius: 40, OuterRadius: 80,
	}
	center := canvas.Position{X: 120, Y: 160}
	points := insetSectorPoints(node, center, donutSectorBorderWidth)
	halfWidth := donutSectorBorderWidth / 2
	steps := sectorSteps(node.SweepAngle)
	outerCount := steps + 1

	for _, point := range points {
		g.Expect(point.X).NotTo(Or(BeNaN(), BeNumerically("==", math.Inf(1)), BeNumerically("==", math.Inf(-1))))
		g.Expect(point.Y).NotTo(Or(BeNaN(), BeNumerically("==", math.Inf(1)), BeNumerically("==", math.Inf(-1))))
	}

	g.Expect(math.Hypot(points[0].X-center.X, points[0].Y-center.Y)).
		To(BeNumerically("~", node.OuterRadius-halfWidth, 0.000001))
	g.Expect(math.Hypot(points[outerCount].X-center.X, points[outerCount].Y-center.Y)).
		To(BeNumerically("~", node.InnerRadius+halfWidth, 0.000001))
	g.Expect(math.Abs(points[0].Y-center.Y)).To(BeNumerically("~", halfWidth, 0.000001))
	g.Expect(math.Abs(points[outerCount].Y-center.Y)).To(BeNumerically("~", halfWidth, 0.000001))
}
```

Add a table entry whose sweep is smaller than the two requested angular insets
and assert the result remains finite and has a positive effective sweep.

- [ ] **Step 2: Run the geometry test to verify it fails**

Run:

```bash
go test ./internal/donuttree -run TestInsetSectorPoints_KeepsBorderStrokeInsideSector
```

Expected: FAIL to compile because `insetSectorPoints`,
`donutSectorBorderWidth`, and `sectorSteps` do not exist.

- [ ] **Step 3: Split fill and border drawing**

In `internal/donuttree/render.go`, define the fixed stroke width and transparent
ink, then use separate polygon specs:

```go
const donutSectorBorderWidth = 1.0

var transparentInk = inks.FixedInk(color.RGBA{})

fillSpec := &canvas.PolygonSpec{
	ShapeStyle: canvas.ShapeStyle{Fill: is.Fill, Border: is.Border},
}
borderSpec := &canvas.PolygonSpec{
	ShapeStyle: canvas.ShapeStyle{
		Fill: transparentInk, Border: is.Border, BorderWidth: donutSectorBorderWidth,
	},
}
```

For each node, always add the existing fill polygon with `fillSpec`. Only when
`is.HasBorderMetric` is true, add a second polygon using
`insetSectorPoints(node, center, donutSectorBorderWidth)`, the node's border
metric value, and `borderSpec`.

- [ ] **Step 4: Implement the inset geometry**

Extract the sampling count so fill and border polygons retain matching arc
resolution:

```go
func sectorSteps(sweepAngle float64) int {
	return max(2, int(math.Ceil(sweepAngle/(2*math.Pi)*64)))
}
```

Implement `insetSectorPoints` by setting `halfWidth := borderWidth / 2`,
`outerRadius := node.OuterRadius - halfWidth`, and
`innerRadius := node.InnerRadius + halfWidth`. For each radius, calculate the
radial-edge angular offset with:

```go
func radialEdgeInset(radius, halfWidth float64) float64 {
	if radius <= 0 {
		return 0
	}

	return math.Asin(math.Min(halfWidth/radius, 1))
}
```

Set the start angle to `node.StartAngle + max(outerInset, innerInset)` and the
end angle to `node.EndAngle() - max(outerInset, innerInset)`. If these cross,
collapse them around the sector midpoint while preserving a small positive
sweep using `math.Nextafter(midpoint, node.EndAngle())`. Sample the outer arc
forward and inner arc backward exactly as `sectorPoints` does, then close the
polygon by appending its first point.

- [ ] **Step 5: Run focused donut rendering tests**

Run:

```bash
go test ./internal/donuttree -run 'Test(RenderToCanvas|SectorPoints|InsetSectorPoints)'
```

Expected: PASS.

- [ ] **Step 6: Format and commit the implementation**

```bash
gofumpt -w internal/donuttree/render.go internal/donuttree/render_test.go
git add internal/donuttree/render.go internal/donuttree/render_test.go
git commit -m "fix: keep donut sector borders adjacent" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 3: Update golden render expectations

**Files:**
- Modify: `internal/goldentest/testdata/donut-tree-png.golden`
- Modify: `internal/goldentest/testdata/donut-tree-svg.golden`

- [ ] **Step 1: Run the donut golden tests and confirm only border output differs**

Run:

```bash
go test ./internal/goldentest -run TestVisualizations/donut-tree -count=1
```

Expected: FAIL with PNG and SVG golden mismatches confined to the donut-tree
fixture.

- [ ] **Step 2: Regenerate the donut goldens using the repository's existing update mechanism**

Run:

```bash
UPDATE_GOLDEN=true go test ./internal/goldentest -run TestVisualizations/donut-tree -count=1
```

Expected: PASS and changes only to
`internal/goldentest/testdata/donut-tree-{png,svg}.golden`.

- [ ] **Step 3: Re-run the golden test without update mode**

Run:

```bash
go test ./internal/goldentest -run TestVisualizations/donut-tree -count=1
```

Expected: PASS.

- [ ] **Step 4: Commit the revised golden files**

```bash
git add internal/goldentest/testdata/donut-tree-png.golden \
  internal/goldentest/testdata/donut-tree-svg.golden
git commit -m "test: update adjacent donut border goldens" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 4: Verify and publish

**Files:**
- No source changes expected

- [ ] **Step 1: Run the complete donut package tests**

Run:

```bash
go test ./internal/donuttree ./internal/goldentest
```

Expected: PASS.

- [ ] **Step 2: Run repository CI through an output-summarizing subagent**

In accordance with repository workflow rules, dispatch an Explore-equivalent
subagent to run:

```bash
task ci
```

Expected: exit status 0, no failing tests, and no failing linters.

- [ ] **Step 3: Review the final branch diff**

Run:

```bash
git status --short
git diff --check
git log --oneline main..HEAD
git diff --stat main...HEAD
```

Expected: a clean worktree; design, test, implementation, and golden commits;
no whitespace errors; and changes confined to the approved files.

- [ ] **Step 4: Push and open the pull request**

```bash
git push -u origin HEAD
printf '%s\n' \
  '## Summary' \
  '- draw metric borders on inset polygons so adjacent strokes meet without overlapping' \
  '- preserve the existing donut fill and label geometry' \
  '- add geometry regression coverage and update donut PNG/SVG goldens' \
  '' \
  '## Verification' \
  '- `go test ./internal/donuttree ./internal/goldentest`' \
  '- `task ci`' \
  > /tmp/donut-adjacent-borders-pr.md
gh pr create \
  --base main \
  --title "Fix overlapping donut sector borders" \
  --body-file /tmp/donut-adjacent-borders-pr.md
```

The PR body must explain the shared-centerline root cause, the inset-border
geometry, the unchanged fill behavior, and successful targeted and CI checks.
