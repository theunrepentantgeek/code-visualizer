# Donut Tree Compact Captions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Render donut-tree caption lines as a normally spaced, centered block with radial-tree-compatible baseline orientation.

**Architecture:** Replace donut tree's per-glyph concentric-arc renderer with one centered canvas text element per caption line. Compute the block at the sector midpoint, offset lines along the tangent by the measured line height, rotate baselines radially using the radial tree's right/left-half rule, and fit the rotated block in both sector dimensions.

**Tech Stack:** Go 1.26.1, retained-mode canvas text, `internal/canvas/textlayout`, Gomega, Goldie v2

---

## File Structure

- Modify `internal/donuttree/labels.go`: calculate compact multiline caption geometry, radial baseline orientation, and two-dimensional font fitting.
- Modify `internal/donuttree/labels_test.go`: replace concentric-arc/per-glyph expectations with compact block, orientation, and fitting regression tests.
- Update `internal/goldentest/testdata/donut-tree-png.golden`: accept the corrected raster rendering.
- Update `internal/goldentest/testdata/donut-tree-svg.golden`: accept the corrected SVG rendering.
- Update `samples/donut-tree/code-visualizer.png`: refresh only the donut-tree raster sample.
- Update `samples/donut-tree/code-visualizer.svg`: refresh only the donut-tree SVG sample.

### Task 1: Compact Multiline Caption Geometry

**Files:**
- Modify: `internal/donuttree/labels_test.go`
- Modify: `internal/donuttree/labels.go`

- [ ] **Step 1: Replace the concentric-arc regression test with a compact right-half block test**

Add the `textlayout` import to `internal/donuttree/labels_test.go`, then replace `TestAddSectorLabel_RendersLinesOnConcentricArcsWithCommonMidpoint` with:

```go
func TestAddSectorLabel_RendersCompactRadialLinesCenteredOnSector(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	node := DonutNode{
		StartAngle:  0,
		SweepAngle:  math.Pi / 2,
		InnerRadius: 100,
		OuterRadius: 140,
	}
	center := canvas.Position{X: 200, Y: 200}
	lines := []string{"src", "120", "go"}
	cv := canvas.NewCanvas(400, 400)
	addSectorLabel(cv, node, center, lines, inks.FixedInk(donutLabelColour))

	backend := mock.NewBackend()
	g.Expect(cv.RenderTo(backend)).To(Succeed())
	calls := callsNamed(backend.Calls, "DrawText")
	g.Expect(calls).To(HaveLen(len(lines)))

	midpoint := node.StartAngle + node.SweepAngle/2
	midRadius := (node.InnerRadius + node.OuterRadius) / 2
	blockCenter := canvas.Position{
		X: center.X + midRadius*math.Cos(midpoint),
		Y: center.Y + midRadius*math.Sin(midpoint),
	}
	_, lineHeight := textlayout.MeasureString(lines[0], calls[0].FontSize)

	for index, call := range calls {
		offset := (float64(index) - float64(len(lines)-1)/2) * lineHeight
		g.Expect(call.Text).To(Equal(lines[index]))
		g.Expect(call.Anchor).To(Equal(canvas.AnchorMiddle))
		g.Expect(call.Rotation).To(BeNumerically("~", midpoint, 0.001))
		g.Expect(call.Pos.X).To(BeNumerically("~", blockCenter.X-offset*math.Sin(midpoint), 0.001))
		g.Expect(call.Pos.Y).To(BeNumerically("~", blockCenter.Y+offset*math.Cos(midpoint), 0.001))
	}
}
```

- [ ] **Step 2: Run the focused test to verify RED**

Run:

```bash
go test ./internal/donuttree -run '^TestAddSectorLabel_RendersCompactRadialLinesCenteredOnSector$' -count=1
```

Expected: FAIL because the current implementation emits one `DrawText` call per glyph rather than one per line.

- [ ] **Step 3: Replace the lower/upper curved-glyph tests with a left-half orientation test**

Remove `TestAddSectorLabel_InvertsLowerRightGlyphOrderAndRotation` and `TestAddSectorLabel_DoesNotInvertUpperLeftGlyphOrderOrRotation`. Remove the now-unused `strings` import. Add:

```go
func TestAddSectorLabel_FlipsRadialBaselineOnLeftHalf(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	node := DonutNode{
		StartAngle:  math.Pi / 2,
		SweepAngle:  math.Pi / 2,
		InnerRadius: 100,
		OuterRadius: 140,
	}
	cv := canvas.NewCanvas(400, 400)
	addSectorLabel(
		cv,
		node,
		canvas.Position{X: 200, Y: 200},
		[]string{"src", "120"},
		inks.FixedInk(donutLabelColour),
	)

	backend := mock.NewBackend()
	g.Expect(cv.RenderTo(backend)).To(Succeed())
	calls := callsNamed(backend.Calls, "DrawText")
	g.Expect(calls).To(HaveLen(2))

	midpoint := node.StartAngle + node.SweepAngle/2
	for _, call := range calls {
		g.Expect(call.Rotation).To(BeNumerically("~", midpoint+math.Pi, 0.001))
	}
}
```

- [ ] **Step 4: Run both geometry tests to verify RED**

Run:

```bash
go test ./internal/donuttree -run 'TestAddSectorLabel_(RendersCompactRadialLinesCenteredOnSector|FlipsRadialBaselineOnLeftHalf)$' -count=1
```

Expected: FAIL because current captions are tangential, per-glyph text on separate radii.

- [ ] **Step 5: Implement compact line placement and radial orientation**

In `internal/donuttree/labels.go`, replace `addSectorLabel` and `addSectorLabelLine` with:

```go
func addSectorLabel(cv *canvas.Canvas, node DonutNode, center canvas.Position, lines []string, ink inks.Ink) {
	fontSize := sectorLabelFontSize(node, lines)
	if fontSize == 0 {
		return
	}

	midpoint := node.StartAngle + node.SweepAngle/2
	midRadius := (node.InnerRadius + node.OuterRadius) / 2
	blockCenter := canvas.Position{
		X: center.X + midRadius*math.Cos(midpoint),
		Y: center.Y + midRadius*math.Sin(midpoint),
	}

	rotation := midpoint
	if math.Cos(midpoint) < 0 {
		rotation += math.Pi
	}

	_, lineHeight := textlayout.MeasureStrings(lines, fontSize)
	spec := &canvas.TextSpec{
		Ink:      ink,
		FontSize: fontSize,
		Anchor:   canvas.AnchorMiddle,
		Rotation: rotation,
	}

	for index, line := range lines {
		offset := (float64(index) - float64(len(lines)-1)/2) * lineHeight
		cv.AddText(canvas.LayerOverlay, canvas.Text{
			Spec:    spec,
			X:       blockCenter.X - offset*math.Sin(rotation),
			Y:       blockCenter.Y + offset*math.Cos(rotation),
			Content: line,
		})
	}
}
```

Delete `stringsToRunes`, `isLowerHalf`, and `reverseStrings`; they become unused when curved glyph placement is removed.

- [ ] **Step 6: Run the geometry tests to verify GREEN**

Run:

```bash
go test ./internal/donuttree -run 'TestAddSectorLabel_(RendersCompactRadialLinesCenteredOnSector|FlipsRadialBaselineOnLeftHalf)$' -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit compact caption geometry**

Run:

```bash
git add internal/donuttree/labels.go internal/donuttree/labels_test.go
git commit -m "fix: compact donut tree captions"
```

Expected: one commit containing only the focused renderer and test changes.

### Task 2: Fit the Rotated Caption Block in Both Dimensions

**Files:**
- Modify: `internal/donuttree/labels_test.go`
- Modify: `internal/donuttree/labels.go`

- [ ] **Step 1: Replace the old row-fitting test with radial and tangential fitting cases**

Replace `TestSectorLabelFontSize_FitsAllRowsAndRejectsInsufficientRadialSpace` with:

```go
func TestSectorLabelFontSize_FitsRotatedBlockInBothDimensions(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	wideLine := DonutNode{SweepAngle: math.Pi, InnerRadius: 100, OuterRadius: 120}
	narrowArc := DonutNode{SweepAngle: 0.1, InnerRadius: 100, OuterRadius: 140}
	roomy := DonutNode{SweepAngle: math.Pi / 2, InnerRadius: 100, OuterRadius: 140}

	g.Expect(sectorLabelFontSize(wideLine, []string{"wide"})).To(
		BeNumerically(">=", donutMinimumLabelFontSize),
	)
	g.Expect(sectorLabelFontSize(wideLine, []string{"wide"})).To(
		BeNumerically("<", donutDefaultLabelFontSize),
	)
	g.Expect(sectorLabelFontSize(narrowArc, []string{"a", "b", "c", "d"})).To(BeZero())
	g.Expect(sectorLabelFontSize(roomy, []string{"src", "120"})).To(Equal(donutDefaultLabelFontSize))
}
```

- [ ] **Step 2: Run the font-fitting test to verify RED**

Run:

```bash
go test ./internal/donuttree -run '^TestSectorLabelFontSize_FitsRotatedBlockInBothDimensions$' -count=1
```

Expected: FAIL because the current fitting logic treats line width as tangential arc usage and line count as radial usage—the opposite of the new radial-baseline geometry.

- [ ] **Step 3: Implement two-dimensional rotated-block fitting**

Replace `sectorLabelFontSize` in `internal/donuttree/labels.go` with:

```go
func sectorLabelFontSize(node DonutNode, lines []string) float64 {
	if len(lines) == 0 || node.SweepAngle <= 0 || node.OuterRadius <= node.InnerRadius {
		return 0
	}

	widths, lineHeight := textlayout.MeasureStrings(lines, donutDefaultLabelFontSize)
	if lineHeight <= 0 {
		return 0
	}

	maxWidth := 0.0
	for _, width := range widths {
		maxWidth = max(maxWidth, width)
	}
	if maxWidth <= 0 {
		return 0
	}

	ringWidth := node.OuterRadius - node.InnerRadius
	midRadius := (node.InnerRadius + node.OuterRadius) / 2
	availableArcLength := midRadius * node.SweepAngle
	blockHeight := lineHeight * float64(len(lines))

	fontSize := min(
		donutDefaultLabelFontSize,
		donutDefaultLabelFontSize*ringWidth/maxWidth,
		donutDefaultLabelFontSize*availableArcLength/blockHeight,
	)
	if fontSize < donutMinimumLabelFontSize {
		return 0
	}

	return fontSize
}
```

- [ ] **Step 4: Run all donut-tree tests to verify GREEN**

Run:

```bash
go test ./internal/donuttree -count=1
```

Expected: PASS.

- [ ] **Step 5: Check formatting and static editor diagnostics**

Run:

```bash
gofumpt -w internal/donuttree/labels.go internal/donuttree/labels_test.go
gofumpt -l internal/donuttree/labels.go internal/donuttree/labels_test.go
```

Expected: no output from the second command. Check both modified files in the editor diagnostics; expected: no errors.

- [ ] **Step 6: Commit font fitting**

Run:

```bash
git add internal/donuttree/labels.go internal/donuttree/labels_test.go
git commit -m "fix: fit radial donut caption blocks"
```

Expected: one commit containing the fitting regression and minimal implementation.

### Task 3: Refresh Donut-Tree Visual Baselines and Verify the Branch

**Files:**
- Update: `internal/goldentest/testdata/donut-tree-png.golden`
- Update: `internal/goldentest/testdata/donut-tree-svg.golden`
- Update: `samples/donut-tree/code-visualizer.png`
- Update: `samples/donut-tree/code-visualizer.svg`

- [ ] **Step 1: Run the donut-tree golden test before updating snapshots**

Run:

```bash
go test ./internal/goldentest -run '^TestGolden_DonutTree$' -count=1
```

Expected: FAIL for the PNG and SVG snapshots because caption geometry intentionally changed.

- [ ] **Step 2: Update only donut-tree golden snapshots**

Run:

```bash
GOLDIE_UPDATE=1 go test ./internal/goldentest -run '^TestGolden_DonutTree$' -count=1
```

Expected: PASS and modifications only to `internal/goldentest/testdata/donut-tree-png.golden` and `internal/goldentest/testdata/donut-tree-svg.golden` among golden files.

- [ ] **Step 3: Regenerate only donut-tree samples**

Run:

```bash
task samples-donut-tree
```

Expected: successful PNG and SVG generation, with no non-donut sample changes.

- [ ] **Step 4: Inspect the refreshed raster sample**

Open `samples/donut-tree/code-visualizer.png` and verify captions are compact multiline blocks, baselines point radially, left-half captions are flipped upright, and lines do not overlap sector boundaries.

- [ ] **Step 5: Verify the focused golden test against its updated snapshots**

Run:

```bash
go test ./internal/goldentest -run '^TestGolden_DonutTree$' -count=1
```

Expected: PASS.

- [ ] **Step 6: Run full CI through an Explore subagent**

Ask an `Explore` subagent to run `task ci` and return only the exit status, failing test/linter identities, offending file and line messages, or a one-line success note.

Expected: exit status 0 with no failing tests or linters and no files reformatted by CI.

- [ ] **Step 7: Confirm the final diff is scoped and preserves unrelated state**

Run:

```bash
git status --short
git diff --check
git diff --name-only HEAD~2
```

Expected: no whitespace errors. Compare `git status --short` with the initial clean status and verify no unrelated files—including `.custom-gcl.yml`, `.golangci.yml`, `Taskfile.yml`, or samples for other visualizations—were changed by this implementation. The visual-companion `.superpowers/` directory is a brainstorming artifact and must not be committed.

- [ ] **Step 8: Commit visual baselines**

Run:

```bash
git add internal/goldentest/testdata/donut-tree-png.golden \
  internal/goldentest/testdata/donut-tree-svg.golden \
  samples/donut-tree/code-visualizer.png \
  samples/donut-tree/code-visualizer.svg
git commit -m "test: update donut caption baselines"
```

Expected: one commit containing only donut-tree golden and sample artifacts.
