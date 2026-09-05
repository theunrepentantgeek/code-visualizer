# Donut Ring Spacing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reduce each donut-tree ring to 90% of its radial slot so adjacent depth rings have a consistent transparent gap.

**Architecture:** Keep spacing entirely in `internal/donuttree` layout geometry. Preserve each ring's existing inner radius and radial slot, reduce only its outer radius, and let sector fills, metric borders, and labels consume the resulting `DonutNode` geometry without renderer-specific spacing logic.

**Tech Stack:** Go 1.26.1, Gomega, Goldie v2, fogleman/gg, SVG canvas backend

---

## File Structure

- Modify `internal/donuttree/layout.go`: define the fixed 90% ring-width ratio and apply it when calculating sector outer radii.
- Modify `internal/donuttree/layout_test.go`: specify slot boundaries, visible ring width, root-anchor contact, and inter-ring gaps.
- Modify `internal/donuttree/render_test.go`: verify fill polygons and label positions consume the narrowed layout geometry.
- Update `internal/goldentest/testdata/donut-tree-png.golden`: record the raster rendering with spaced rings.
- Update `internal/goldentest/testdata/donut-tree-svg.golden`: record the SVG rendering with spaced rings.
- Update `samples/donut-tree/code-visualizer.png`: refresh the checked-in donut-tree raster sample.
- Update `samples/donut-tree/code-visualizer.svg`: refresh the checked-in donut-tree SVG sample.
- Update `docs/content/docs/visualizations/donut-tree-thumb.png`: refresh the documentation thumbnail.

### Task 1: Specify and Implement Narrowed Ring Geometry

**Files:**
- Modify: `internal/donuttree/layout_test.go`
- Modify: `internal/donuttree/render_test.go`
- Modify: `internal/donuttree/layout.go:12-85`

- [ ] **Step 1: Write the failing layout test**

Add this test to `internal/donuttree/layout_test.go`:

```go
func TestLayoutLeavesTenPercentGapBetweenDepthRings(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	nested := directoryWithLines("nested", 10)
	parent := directoryWithLines("parent", 10)
	parent.Dirs = []*model.Directory{nested}
	root := &model.Directory{Name: "root", Dirs: []*model.Directory{parent}}

	layout := Layout(root, 600, filesystem.FileLines)
	firstRing := layout.Children[0]
	secondRing := firstRing.Children[0]
	slotWidth := layout.AnchorRadius

	g.Expect(firstRing.InnerRadius).To(BeNumerically("==", slotWidth))
	g.Expect(firstRing.OuterRadius-firstRing.InnerRadius).
		To(BeNumerically("~", slotWidth*0.9, 1e-12))
	g.Expect(secondRing.InnerRadius).To(BeNumerically("==", slotWidth*2))
	g.Expect(secondRing.OuterRadius-secondRing.InnerRadius).
		To(BeNumerically("~", slotWidth*0.9, 1e-12))
	g.Expect(secondRing.InnerRadius-firstRing.OuterRadius).
		To(BeNumerically("~", slotWidth*0.1, 1e-12))
}
```

- [ ] **Step 2: Write the failing rendering test**

Add this test to `internal/donuttree/render_test.go`:

```go
func TestRenderToCanvas_UsesNarrowedRingGeometryForSectorsAndLabels(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	root := donutRoot()
	layout := Layout(root, 600, filesystem.FileLines)
	is := BuildInks(root, stages.RequestedMetrics{}, filesystem.FileLines, palette.Neutral, "", "")

	calls := renderCalls(t, RenderToCanvas(layout, root, 600, 600, is, LabelMetrics{}))
	polygons := callsNamed(calls, "DrawPolygon")
	expectedOuterRadius := layout.AnchorRadius * 1.9

	g.Expect(polygons).NotTo(BeEmpty())
	g.Expect(layout.Center.DistanceTo(polygons[0].Points[0])).
		To(BeNumerically("~", expectedOuterRadius, 0.000001))

	var directoryLabel *mock.Call
	for index := range calls {
		if calls[index].Method == "DrawText" && calls[index].Text == "src" {
			directoryLabel = &calls[index]
			break
		}
	}

	if directoryLabel == nil {
		t.Fatal("expected src directory label")
	}

	expectedMidRadius := layout.AnchorRadius * 1.45
	g.Expect(layout.Center.DistanceTo(directoryLabel.Pos)).
		To(BeNumerically("~", expectedMidRadius, 0.000001))
}
```

- [ ] **Step 3: Run the focused tests to verify they fail**

Run:

```bash
go test ./internal/donuttree -run 'Test(LayoutLeavesTenPercentGapBetweenDepthRings|RenderToCanvas_UsesNarrowedRingGeometryForSectorsAndLabels)$' -count=1
```

Expected: FAIL because current rings occupy 100% of each radial slot, producing a zero inter-ring gap and larger sector and label radii.

- [ ] **Step 4: Implement the fixed ring-width ratio**

In `internal/donuttree/layout.go`, add a package constant after the imports:

```go
const donutRingWidthRatio = 0.9
```

Then change the radius calculation in `layoutChildren` to:

```go
innerRadius := float64(depth) * ringWidth
outerRadius := innerRadius + ringWidth*donutRingWidthRatio
```

Do not change `ringWidth`, `AnchorRadius`, child slot boundaries, angular allocation, or renderer code.

- [ ] **Step 5: Run all donut-tree tests**

Run:

```bash
go test ./internal/donuttree -count=1
```

Expected: PASS.

- [ ] **Step 6: Format the changed Go files**

Run:

```bash
gofumpt -w internal/donuttree/layout.go internal/donuttree/layout_test.go internal/donuttree/render_test.go
```

Expected: command exits successfully and `gofumpt -l` prints nothing for these files.

- [ ] **Step 7: Commit the geometry change**

```bash
git add internal/donuttree/layout.go internal/donuttree/layout_test.go internal/donuttree/render_test.go
git commit -m "Add spacing between donut rings" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 2: Refresh Donut Rendering Artifacts

**Files:**
- Modify: `internal/goldentest/testdata/donut-tree-png.golden`
- Modify: `internal/goldentest/testdata/donut-tree-svg.golden`
- Modify: `samples/donut-tree/code-visualizer.png`
- Modify: `samples/donut-tree/code-visualizer.svg`
- Modify: `docs/content/docs/visualizations/donut-tree-thumb.png`

- [ ] **Step 1: Confirm the donut golden test detects the visual change**

Run:

```bash
go test ./internal/goldentest -run '^TestGolden_DonutTree$' -count=1
```

Expected: FAIL with Goldie mismatches for `donut-tree-png` and `donut-tree-svg`.

- [ ] **Step 2: Update only the donut-tree golden snapshots**

Run:

```bash
GOLDIE_UPDATE=1 go test ./internal/goldentest -run '^TestGolden_DonutTree$' -count=1
```

Expected: PASS and only `internal/goldentest/testdata/donut-tree-png.golden` and `internal/goldentest/testdata/donut-tree-svg.golden` change.

- [ ] **Step 3: Regenerate the checked-in donut-tree samples**

Run:

```bash
task samples-donut-tree
```

Expected: PASS and only `samples/donut-tree/code-visualizer.png` and `samples/donut-tree/code-visualizer.svg` change.

- [ ] **Step 4: Regenerate the donut-tree documentation thumbnail**

Run:

```bash
task build
bin/codeviz donut-tree . \
  --config docs/site-images/donut-tree.yml \
  --output docs/content/docs/visualizations/donut-tree-thumb.png \
  --width 960 \
  --height 720
```

Expected: both commands exit successfully and `docs/content/docs/visualizations/donut-tree-thumb.png` changes.

- [ ] **Step 5: Verify the artifact change set is scoped**

Run:

```bash
git status --short
```

Expected: only the two donut golden files, two donut sample files, and donut documentation thumbnail are modified. Do not include `bin/codeviz` if it is ignored or untracked.

- [ ] **Step 6: Re-run the donut golden and package tests**

Run:

```bash
go test ./internal/donuttree ./internal/goldentest -run 'TestGolden_DonutTree|TestLayout|TestRender' -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit the refreshed artifacts**

```bash
git add \
  internal/goldentest/testdata/donut-tree-png.golden \
  internal/goldentest/testdata/donut-tree-svg.golden \
  samples/donut-tree/code-visualizer.png \
  samples/donut-tree/code-visualizer.svg \
  docs/content/docs/visualizations/donut-tree-thumb.png
git commit -m "Refresh donut ring spacing snapshots" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 3: Verify the Complete Change

**Files:**
- Verify only; no planned modifications.

- [ ] **Step 1: Run repository CI through an Explore agent**

Dispatch an Explore agent to run:

```bash
task ci
```

Ask it to return only the exit status, failing tests or linters with
`file:line` messages, or a one-line success note.

Expected: PASS with no failing tests or linters.

- [ ] **Step 2: Confirm the worktree is clean**

Run:

```bash
git status --short
```

Expected: no output.
