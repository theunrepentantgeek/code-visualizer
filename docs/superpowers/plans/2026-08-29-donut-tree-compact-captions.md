# Donut Tree Compact Captions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Keep donut-tree caption lines normally spaced while aligning each compact block on a radius and orienting its text baselines tangentially.

**Architecture:** Retain one canvas text element per caption line and the sector-midpoint block center. Change line offsets to follow the sector midpoint radius, rotate text by pi over two so baselines are tangential, preserve the lower-half readability flip, and swap the font-fitting dimensions to match the revised geometry.

**Tech Stack:** Go 1.26.1, retained-mode canvas text, `internal/canvas/textlayout`, Gomega, Goldie v2

---

## File Structure

- Modify `internal/donuttree/labels.go`: revise compact block positions, rotations, and fitting dimensions.
- Modify `internal/donuttree/labels_test.go`: encode radial centerline, tangential baseline, lower-half flip, and revised fitting behavior.
- Update `internal/goldentest/testdata/donut-tree-png.golden`: accept corrected raster orientation.
- Update `internal/goldentest/testdata/donut-tree-svg.golden`: accept corrected SVG orientation.
- Update `samples/donut-tree/code-visualizer.png`: refresh only the donut-tree raster sample.
- Update `samples/donut-tree/code-visualizer.svg`: refresh only the donut-tree SVG sample.

### Task 1: Radial Caption Centerline and Tangential Baselines

**Files:**
- Modify: `internal/donuttree/labels_test.go`
- Modify: `internal/donuttree/labels.go`

- [ ] **Step 1: Revise the upper-half geometry regression**

Rename `TestAddSectorLabel_RendersCompactRadialLinesCenteredOnSector` to `TestAddSectorLabel_RendersCompactLinesAlongSectorRadius`. Change its sector to `StartAngle: math.Pi` and `SweepAngle: math.Pi/2`. Keep its block-center, line-content, and anchor assertions, but assert the following rotation and positions:

```go
expectedRotation := midpoint + math.Pi/2
for index, call := range calls {
	offset := (float64(index) - float64(len(lines)-1)/2) * lineHeight
	g.Expect(call.Text).To(Equal(lines[index]))
	g.Expect(call.Anchor).To(Equal(canvas.AnchorMiddle))
	g.Expect(call.Rotation).To(BeNumerically("~", expectedRotation, 0.001))
	g.Expect(call.Pos.X).To(BeNumerically("~", blockCenter.X+offset*math.Cos(midpoint), 0.001))
	g.Expect(call.Pos.Y).To(BeNumerically("~", blockCenter.Y+offset*math.Sin(midpoint), 0.001))
}
```

- [ ] **Step 2: Revise the readability-flip regression**

Rename `TestAddSectorLabel_FlipsRadialBaselineOnLeftHalf` to `TestAddSectorLabel_FlipsTangentialBaselineOnLowerHalf`. Use `StartAngle: 0` and `SweepAngle: math.Pi/2`, retaining the existing radii and two-line assertion. Assert:

```go
midpoint := node.StartAngle + node.SweepAngle/2
for _, call := range calls {
	g.Expect(call.Rotation).To(BeNumerically("~", midpoint+3*math.Pi/2, 0.001))
}
```

- [ ] **Step 3: Run the focused tests to verify RED**

Run:

```bash
go test ./internal/donuttree -run 'TestAddSectorLabel_(RendersCompactLinesAlongSectorRadius|FlipsTangentialBaselineOnLowerHalf)$' -count=1
```

Expected: FAIL because current line positions are tangential and current baselines are radial.

- [ ] **Step 4: Implement radial offsets and tangential rotation**

In `addSectorLabel`, replace the rotation and line-position calculations with:

```go
rotation := midpoint + math.Pi/2
if isLowerHalf(midpoint) {
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
		X:       blockCenter.X + offset*math.Cos(midpoint),
		Y:       blockCenter.Y + offset*math.Sin(midpoint),
		Content: line,
	})
}
```

Restore only the normalized lower-half helper:

```go
func isLowerHalf(angle float64) bool {
	angle = math.Mod(angle, 2*math.Pi)
	if angle < 0 {
		angle += 2 * math.Pi
	}

	return angle > 0 && angle < math.Pi
}
```

Do not restore the obsolete per-glyph helpers.

- [ ] **Step 5: Run focused and package tests to verify GREEN**

Run:

```bash
go test ./internal/donuttree -run 'TestAddSectorLabel_(RendersCompactLinesAlongSectorRadius|FlipsTangentialBaselineOnLowerHalf)$' -count=1
go test ./internal/donuttree -count=1
```

Expected: PASS.

- [ ] **Step 6: Format and commit geometry**

Run:

```bash
gofumpt -w internal/donuttree/labels.go internal/donuttree/labels_test.go
gofumpt -l internal/donuttree/labels.go internal/donuttree/labels_test.go
git diff --check
git add internal/donuttree/labels.go internal/donuttree/labels_test.go
git commit -m "fix: orient donut captions along radii"
```

Expected: no formatter or whitespace output, followed by one focused commit.

### Task 2: Fit the Revised Caption Geometry

**Files:**
- Modify: `internal/donuttree/labels_test.go`
- Modify: `internal/donuttree/labels.go`

- [ ] **Step 1: Revise font-fitting regression cases**

Replace `TestSectorLabelFontSize_FitsRotatedBlockInBothDimensions` with:

```go
func TestSectorLabelFontSize_FitsRadialCaptionBlockInBothDimensions(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	thinRing := DonutNode{SweepAngle: math.Pi, InnerRadius: 100, OuterRadius: 120}
	narrowArc := DonutNode{SweepAngle: 0.05, InnerRadius: 100, OuterRadius: 140}
	roomy := DonutNode{SweepAngle: math.Pi / 2, InnerRadius: 100, OuterRadius: 140}

	g.Expect(sectorLabelFontSize(thinRing, []string{"a", "b"})).To(BeNumerically("<", donutDefaultLabelFontSize))
	g.Expect(sectorLabelFontSize(thinRing, []string{"a", "b"})).To(BeNumerically(">=", donutMinimumLabelFontSize))
	g.Expect(sectorLabelFontSize(narrowArc, []string{"wide label"})).To(BeZero())
	g.Expect(sectorLabelFontSize(roomy, []string{"src", "120"})).To(Equal(donutDefaultLabelFontSize))
}
```

- [ ] **Step 2: Run the fitting test to verify RED**

Run:

```bash
go test ./internal/donuttree -run '^TestSectorLabelFontSize_FitsRadialCaptionBlockInBothDimensions$' -count=1
```

Expected: FAIL because current fitting applies text width to ring thickness and block height to arc length.

- [ ] **Step 3: Swap fitting dimensions**

Keep current validation and measurements in `sectorLabelFontSize`, but replace the size calculation with:

```go
ringWidth := node.OuterRadius - node.InnerRadius
midRadius := (node.InnerRadius + node.OuterRadius) / 2
availableArcLength := midRadius * node.SweepAngle
blockHeight := lineHeight * float64(len(lines))

fontSize := min(
	donutDefaultLabelFontSize,
	donutDefaultLabelFontSize*availableArcLength/maxWidth,
	donutDefaultLabelFontSize*ringWidth/blockHeight,
)
```

Retain the existing below-minimum omission unchanged.

- [ ] **Step 4: Run package tests and format checks**

Run:

```bash
go test ./internal/donuttree -count=1
gofumpt -w internal/donuttree/labels.go internal/donuttree/labels_test.go
gofumpt -l internal/donuttree/labels.go internal/donuttree/labels_test.go
git diff --check
```

Expected: tests pass and checks produce no output.

- [ ] **Step 5: Commit revised fitting**

Run:

```bash
git add internal/donuttree/labels.go internal/donuttree/labels_test.go
git commit -m "fix: fit radial donut caption stacks"
```

Expected: one focused commit.

### Task 3: Refresh Visual Baselines and Verify

**Files:**
- Update: `internal/goldentest/testdata/donut-tree-png.golden`
- Update: `internal/goldentest/testdata/donut-tree-svg.golden`
- Update: `samples/donut-tree/code-visualizer.png`
- Update: `samples/donut-tree/code-visualizer.svg`

- [ ] **Step 1: Verify expected golden RED**

Run:

```bash
go test ./internal/goldentest -run '^TestGolden_DonutTree$' -count=1
```

Expected: FAIL for PNG and SVG because caption positions and rotations changed.

- [ ] **Step 2: Update only donut-tree goldens and samples**

Run:

```bash
GOLDIE_UPDATE=1 go test ./internal/goldentest -run '^TestGolden_DonutTree$' -count=1
task samples-donut-tree
```

Expected: PASS; only the two donut-tree goldens and two donut-tree samples change.

- [ ] **Step 3: Inspect output and verify focused GREEN**

Inspect `samples/donut-tree/code-visualizer.png` and SVG. Confirm compact blocks follow radial centerlines, text baselines are tangential, lower-half text is upright, line spacing is normal, and labels do not visibly cross sector boundaries. Then run:

```bash
go test ./internal/goldentest -run '^TestGolden_DonutTree$' -count=1
git diff --check
```

Expected: PASS with no whitespace errors.

- [ ] **Step 4: Commit visual baselines**

Run:

```bash
git add internal/goldentest/testdata/donut-tree-png.golden \
  internal/goldentest/testdata/donut-tree-svg.golden \
  samples/donut-tree/code-visualizer.png \
  samples/donut-tree/code-visualizer.svg
git commit -m "test: update radial donut captions"
```

Expected: one commit containing only donut-tree visual artifacts.

- [ ] **Step 5: Run full CI and scope verification**

Run `task ci` through an Explore subagent and require exit status 0 with no failing tests or linters. Then compare final `git status --short` with the initial clean state and verify `.custom-gcl.yml`, `.golangci.yml`, `Taskfile.yml`, and all non-donut samples remain unchanged.
