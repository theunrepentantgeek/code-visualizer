# Treemap Depth Header Palette Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Color treemap directory rails with a five-step dark slate palette selected by visible nesting depth.

**Architecture:** Layout stores a self-contained visible depth on each directory rectangle, with the hidden root at `-1` and its immediate children at `0`. Rendering prebuilds one immutable rail spec per private treemap color and selects it with modulo arithmetic, preserving the existing allocation-conscious canvas pipeline.

**Tech Stack:** Go 1.26.1, fogleman/gg canvas backends, Gomega, Goldie v2, Task

---

## File Map

| Path | Responsibility |
|---|---|
| `internal/treemap/node.go` | Add visible directory depth to layout rectangles |
| `internal/treemap/layout.go` | Assign root, child, and nested directory depths |
| `internal/treemap/layout_test.go` | Verify depth assignment and offset stability |
| `internal/treemap/inks.go` | Define the private five-color rail palette |
| `internal/treemap/header_palette_test.go` | Verify exact colors, order, and white-text contrast |
| `internal/treemap/render.go` | Prebuild depth-colored rail specs and select by modulo |
| `internal/treemap/render_directory_chrome_test.go` | Verify rendered top/left/same-depth/wrapped rail colors |
| `internal/goldentest/testdata/treemap-png.golden` | Updated raster snapshot |
| `internal/goldentest/testdata/treemap-svg.golden` | Updated SVG snapshot |
| `samples/tree-map/code-visualizer.png` | Updated tree-map raster sample |
| `samples/tree-map/code-visualizer.svg` | Updated tree-map SVG sample |

### Task 1: Assign Visible Directory Depth During Layout

**Files:**
- Modify: `internal/treemap/node.go`
- Modify: `internal/treemap/layout.go`
- Modify: `internal/treemap/layout_test.go`

- [ ] **Step 1: Add failing depth tests**

Append to `internal/treemap/layout_test.go`:

```go
func TestLayoutAssignsVisibleDirectoryDepth(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeFile("root.go", 25),
		},
		Dirs: []*model.Directory{
			{
				Name: "src",
				Files: []*model.File{
					makeFile("src.go", 25),
				},
				Dirs: []*model.Directory{
					{
						Name:  "internal",
						Files: []*model.File{makeFile("internal.go", 25)},
					},
				},
			},
			{
				Name:  "cmd",
				Files: []*model.File{makeFile("main.go", 25)},
			},
		},
	}

	rects := Layout(root, 600, 400, filesystem.FileSize)
	src := findDirRect(rects, "src")
	cmd := findDirRect(rects, "cmd")

	g.Expect(rects.VisibleDepth).To(Equal(-1))
	g.Expect(src).NotTo(BeNil())
	g.Expect(cmd).NotTo(BeNil())
	if src == nil || cmd == nil {
		return
	}

	g.Expect(src.VisibleDepth).To(Equal(0))
	g.Expect(cmd.VisibleDepth).To(Equal(0))

	internal := findDirRect(*src, "internal")
	g.Expect(internal).NotTo(BeNil())
	if internal == nil {
		return
	}

	g.Expect(internal.VisibleDepth).To(Equal(1))

	var rootFile *TreemapRectangle
	for i := range rects.Children {
		if !rects.Children[i].IsDirectory {
			rootFile = &rects.Children[i]

			break
		}
	}

	g.Expect(rootFile).NotTo(BeNil())
	g.Expect(rootFile.VisibleDepth).To(Equal(0))
}

func TestOffsetRectsPreservesVisibleDepth(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	rect := TreemapRectangle{
		X: 10, Y: 20, W: 100, H: 50,
		IsDirectory: true,
		VisibleDepth: 3,
		Chrome: DirectoryChrome{
			Orientation: DirectoryLabelTop,
			Rail:        RectangleBounds{X: 10, Y: 20, W: 100, H: 20},
			Content:     RectangleBounds{X: 14, Y: 40, W: 92, H: 26},
		},
	}

	OffsetRects(&rect, 30, 40)

	g.Expect(rect.VisibleDepth).To(Equal(3))
}
```

The root-file assertion documents that file rectangles leave the field at the
Go zero value and do not participate in directory depth selection.

- [ ] **Step 2: Run the depth tests and verify they fail**

Run:

```bash
go test ./internal/treemap -run 'TestLayoutAssignsVisibleDirectoryDepth|TestOffsetRectsPreservesVisibleDepth' -count=1
```

Expected: FAIL because `TreemapRectangle.VisibleDepth` does not exist.

- [ ] **Step 3: Add visible depth metadata**

In `internal/treemap/node.go`, add the field to `TreemapRectangle`:

```go
type TreemapRectangle struct {
	X            float64
	Y            float64
	W            float64
	H            float64
	Label        string
	IsDirectory  bool
	VisibleDepth int
	Chrome       DirectoryChrome
	Children     []TreemapRectangle
}
```

- [ ] **Step 4: Thread depth through recursive layout**

Update `internal/treemap/layout.go` so the root uses `-1`, nested directories
increment from their parent, and files remain unchanged:

```go
func layoutRoot(root *model.Directory, box layout.Box, sizeMetric metric.Name) TreemapRectangle {
	return layoutDirectory(root, box, sizeMetric, -1, directoryChromeBorderOnly(RectangleBounds{
		X: box.X,
		Y: box.Y,
		W: box.W,
		H: box.H,
	}))
}

func layoutDir(
	dir *model.Directory,
	box layout.Box,
	sizeMetric metric.Name,
	visibleDepth int,
) TreemapRectangle {
	return layoutDirectory(dir, box, sizeMetric, visibleDepth, resolveDirectoryChrome(RectangleBounds{
		X: box.X,
		Y: box.Y,
		W: box.W,
		H: box.H,
	}, dir.Name))
}

func layoutDirectory(
	dir *model.Directory,
	box layout.Box,
	sizeMetric metric.Name,
	visibleDepth int,
	chrome DirectoryChrome,
) TreemapRectangle {
	rect := TreemapRectangle{
		X:            box.X,
		Y:            box.Y,
		W:            box.W,
		H:            box.H,
		Label:        dir.Name,
		IsDirectory:  true,
		VisibleDepth: visibleDepth,
		Chrome:       chrome,
	}

	children := collectChildren(dir, sizeMetric)
	if len(children) == 0 {
		return rect
	}

	contentBox := layout.Box{
		X: rect.Chrome.Content.X,
		Y: rect.Chrome.Content.Y,
		W: rect.Chrome.Content.W,
		H: rect.Chrome.Content.H,
	}
	if contentBox.W <= 0 || contentBox.H <= 0 {
		return rect
	}

	areas := make([]float64, len(children))
	for i, c := range children {
		areas[i] = c.area
	}

	boxes := layout.Squarify(contentBox, areas)
	rect.Children = make([]TreemapRectangle, 0, len(children))

	for i, c := range children {
		b := insetBox(boxes[i], siblingGap/2)
		rect.Children = append(
			rect.Children,
			layoutChild(dir, c, b, sizeMetric, rect.VisibleDepth),
		)
	}

	return rect
}

func layoutChild(
	dir *model.Directory,
	c child,
	b layout.Box,
	sizeMetric metric.Name,
	parentVisibleDepth int,
) TreemapRectangle {
	if c.isDir {
		return layoutDir(dir.Dirs[c.dirIdx], b, sizeMetric, parentVisibleDepth+1)
	}

	f := dir.Files[c.fileIdx]

	return TreemapRectangle{
		X: b.X, Y: b.Y, W: b.W, H: b.H,
		Label: f.Name,
	}
}
```

- [ ] **Step 5: Format and run treemap tests**

Run:

```bash
GOENV_VERSION=1.26.1 task fmt
go test ./internal/treemap -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit layout depth**

```bash
git add internal/treemap/node.go internal/treemap/layout.go internal/treemap/layout_test.go
git commit -m "feat(treemap): track visible directory depth" \
  -m "Co-authored-by: Copilot App <223556219+Copilot@users.noreply.github.com>" \
  -m "Copilot-Session: 0113bb97-3cb7-4511-a2a7-f156a4d83cf6"
```

### Task 2: Render Rails With the Depth Palette

**Files:**
- Modify: `internal/treemap/inks.go`
- Create: `internal/treemap/header_palette_test.go`
- Modify: `internal/treemap/render.go`
- Modify: `internal/treemap/render_directory_chrome_test.go`

- [ ] **Step 1: Add failing palette definition tests**

Create `internal/treemap/header_palette_test.go`:

```go
package treemap

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

func TestHeaderFillsAreOrderedDarkSlatesWithReadableWhiteText(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	want := [...]color.RGBA{
		{R: 0x20, G: 0x26, B: 0x31, A: 0xFF},
		{R: 0x2F, G: 0x3B, B: 0x4D, A: 0xFF},
		{R: 0x3D, G: 0x52, B: 0x68, A: 0xFF},
		{R: 0x51, G: 0x6A, B: 0x7D, A: 0xFF},
		{R: 0x5F, G: 0x78, B: 0x88, A: 0xFF},
	}

	g.Expect(headerFills).To(Equal(want))

	for _, fill := range headerFills {
		g.Expect(palette.ContrastRatio(fill, palette.White)).To(BeNumerically(">=", 4.5))
	}
}
```

- [ ] **Step 2: Add failing render color tests**

In `internal/treemap/render_directory_chrome_test.go`:

1. Import canvas model and define the expected colors:

```go
canvasmodel "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"

//nolint:gochecknoglobals // fixed expected values shared by depth subtests
var expectedHeaderFills = [...]color.RGBA{
	{R: 0x20, G: 0x26, B: 0x31, A: 0xFF},
	{R: 0x2F, G: 0x3B, B: 0x4D, A: 0xFF},
	{R: 0x3D, G: 0x52, B: 0x68, A: 0xFF},
	{R: 0x51, G: 0x6A, B: 0x7D, A: 0xFF},
	{R: 0x5F, G: 0x78, B: 0x88, A: 0xFF},
}
```

2. Change `renderDirectoryChrome` to accept `visibleDepth int`, and set it on
the nested directory fixture:

```go
func renderDirectoryChrome(
	t *testing.T,
	chrome treemap.DirectoryChrome,
	visibleDepth int,
) *captureBackend {
	t.Helper()

	root := &model.Directory{
		Name: "",
		Dirs: []*model.Directory{
			{
				Name:  "source",
				Files: []*model.File{makeTestFile("main.go", "go", 100)},
			},
		},
	}
	rects := treemap.TreemapRectangle{
		X: 0, Y: 0, W: 100, H: 100,
		IsDirectory:  true,
		VisibleDepth: -1,
		Chrome: treemap.DirectoryChrome{
			Orientation: treemap.DirectoryLabelNone,
			Content:     treemap.RectangleBounds{X: 4, Y: 4, W: 92, H: 92},
		},
		Children: []treemap.TreemapRectangle{
			{
				X: 10, Y: 10, W: 80, H: 80,
				Label:        "source",
				IsDirectory:  true,
				VisibleDepth: visibleDepth,
				Chrome:       chrome,
				Children: []treemap.TreemapRectangle{
					{X: 14, Y: 14, W: 72, H: 72},
				},
			},
		},
	}
	is := treemap.Inks{
		Fill:   inks.FixedInk(color.RGBA{R: 0x88, A: 0xFF}),
		Border: inks.FixedInk(color.RGBA{A: 0xFF}),
	}
	cv := treemap.RenderToCanvas(rects, root, 100, 100, is, filesystem.FileSize)
	backend := &captureBackend{}
	NewGomegaWithT(t).Expect(cv.RenderTo(backend)).To(Succeed())

	return backend
}
```

Update the three existing callers to pass `0`.

3. Add a fill-aware helper:

```go
func hasRectangleWithFill(
	rectangles []rectangleCall,
	pos canvas.Position,
	size canvas.Size,
	fill color.RGBA,
) bool {
	want := canvasmodel.SolidFill{Color: fill}

	for _, rectangle := range rectangles {
		if rectangle.pos == pos && rectangle.size == size && rectangle.fill == want {
			return true
		}
	}

	return false
}
```

4. Add the depth and wrapping tests:

```go
func TestRenderToCanvas_SelectsDirectoryRailFillByVisibleDepth(t *testing.T) {
	t.Parallel()

	for depth := range expectedHeaderFills {
		t.Run(strconv.Itoa(depth), func(t *testing.T) {
			t.Parallel()

			g := NewGomegaWithT(t)
			backend := renderDirectoryChrome(t, treemap.DirectoryChrome{
				Orientation: treemap.DirectoryLabelTop,
				Text:        "source",
				Rail:        treemap.RectangleBounds{X: 10, Y: 10, W: 80, H: 20},
				Content:     treemap.RectangleBounds{X: 14, Y: 30, W: 72, H: 56},
			}, depth)

			g.Expect(hasRectangleWithFill(
				backend.rectangles,
				canvas.Position{X: 10, Y: 10},
				canvas.Size{Width: 80, Height: 20},
				expectedHeaderFills[depth],
			)).To(BeTrue())
		})
	}
}

func TestRenderToCanvas_WrapsDirectoryRailFillAfterPalette(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	backend := renderDirectoryChrome(t, treemap.DirectoryChrome{
		Orientation: treemap.DirectoryLabelLeft,
		Text:        "source",
		Rail:        treemap.RectangleBounds{X: 10, Y: 10, W: 20, H: 80},
		Content:     treemap.RectangleBounds{X: 30, Y: 14, W: 56, H: 72},
	}, len(expectedHeaderFills))

	g.Expect(hasRectangleWithFill(
		backend.rectangles,
		canvas.Position{X: 10, Y: 10},
		canvas.Size{Width: 20, Height: 80},
		expectedHeaderFills[0],
	)).To(BeTrue())
}
```

Add `"strconv"` to imports.

- [ ] **Step 3: Run the focused tests and verify they fail**

Run:

```bash
go test ./internal/treemap -run 'TestHeaderFills|TestRenderToCanvas_(Selects|Wraps)' -count=1
```

Expected: FAIL because `headerFills` does not exist and rendering still uses
one fixed header spec.

- [ ] **Step 4: Define the private five-color palette**

In `internal/treemap/inks.go`, replace `headerFill` with:

```go
var (
	structuralBorder = color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
	headerFills = [...]color.RGBA{
		{R: 0x20, G: 0x26, B: 0x31, A: 0xFF},
		{R: 0x2F, G: 0x3B, B: 0x4D, A: 0xFF},
		{R: 0x3D, G: 0x52, B: 0x68, A: 0xFF},
		{R: 0x51, G: 0x6A, B: 0x7D, A: 0xFF},
		{R: 0x5F, G: 0x78, B: 0x88, A: 0xFF},
	}
	defaultFill = color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
)
```

Retain the existing global-state lint rationale or add a focused
`//nolint:gochecknoglobals` comment for the read-only structural colors.

- [ ] **Step 5: Prebuild and select rail specs**

In `internal/treemap/render.go`:

1. Remove the single `dirHeaderSpec` global.

2. Add:

```go
type dirRailSpecs [len(headerFills)]*canvas.RectangleSpec

func buildDirRailSpecs() dirRailSpecs {
	var specs dirRailSpecs

	for i, fill := range headerFills {
		fillInk := inks.FixedInk(fill)
		specs[i] = &canvas.RectangleSpec{
			ShapeStyle: canvas.ShapeStyle{
				Fill:        fillInk,
				Border:      fillInk,
				BorderWidth: 0,
			},
		}
	}

	return specs
}

func dirRailSpecForDepth(specs dirRailSpecs, visibleDepth int) *canvas.RectangleSpec {
	return specs[visibleDepth%len(specs)]
}
```

3. In `RenderToCanvas`, build the specs and pass them into the recursive walk:

```go
railSpecs := buildDirRailSpecs()
dirSpecs := buildDirBorderSpecs()
fileSpecs := buildFileRectSpecs(is)
addRect(cv, rects, root, is, sizeMetric, railSpecs, dirSpecs, fileSpecs)
```

4. Add `railSpecs dirRailSpecs` to `addRect`, pass it recursively, and pass it
to `addDirectoryShapes`.

5. Add `railSpecs dirRailSpecs` to `addDirectoryShapes`, then select the rail
spec:

```go
if rect.Chrome.Orientation != DirectoryLabelNone {
	rail := rect.Chrome.Rail
	cv.AddRectangle(canvas.LayerStructure, canvas.Rectangle{
		Spec:  dirRailSpecForDepth(railSpecs, rect.VisibleDepth),
		X:     rail.X,
		Y:     rail.Y,
		W:     rail.W,
		H:     rail.H,
		Focus: canvasmodel.Point{X: 0.5, Y: 0.5},
	})
}
```

Leave label and directory-border rendering unchanged.

- [ ] **Step 6: Add same-depth sibling coverage**

Add a render test using two directory children at `VisibleDepth: 0`, with
distinct rail positions. Render through the capture backend and assert both
rail geometries use `expectedHeaderFills[0]`:

```go
func TestRenderToCanvas_UsesSameRailFillForSameDepthSiblings(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	root := &model.Directory{
		Dirs: []*model.Directory{
			{Name: "src"},
			{Name: "cmd"},
		},
	}
	chrome := func(x float64) treemap.DirectoryChrome {
		return treemap.DirectoryChrome{
			Orientation: treemap.DirectoryLabelTop,
			Text:        "dir",
			Rail:        treemap.RectangleBounds{X: x, Y: 10, W: 35, H: 20},
			Content:     treemap.RectangleBounds{X: x + 4, Y: 30, W: 27, H: 56},
		}
	}
	rects := treemap.TreemapRectangle{
		X: 0, Y: 0, W: 100, H: 100,
		IsDirectory: true,
		VisibleDepth: -1,
		Chrome: treemap.DirectoryChrome{
			Orientation: treemap.DirectoryLabelNone,
			Content:     treemap.RectangleBounds{X: 4, Y: 4, W: 92, H: 92},
		},
		Children: []treemap.TreemapRectangle{
			{
				X: 10, Y: 10, W: 35, H: 80,
				Label: "src", IsDirectory: true, VisibleDepth: 0,
				Chrome: chrome(10),
			},
			{
				X: 55, Y: 10, W: 35, H: 80,
				Label: "cmd", IsDirectory: true, VisibleDepth: 0,
				Chrome: chrome(55),
			},
		},
	}
	is := treemap.Inks{
		Fill:   inks.FixedInk(color.RGBA{R: 0x88, A: 0xFF}),
		Border: inks.FixedInk(color.RGBA{A: 0xFF}),
	}
	cv := treemap.RenderToCanvas(rects, root, 100, 100, is, filesystem.FileSize)
	backend := &captureBackend{}
	g.Expect(cv.RenderTo(backend)).To(Succeed())

	for _, x := range []float64{10, 55} {
		g.Expect(hasRectangleWithFill(
			backend.rectangles,
			canvas.Position{X: x, Y: 10},
			canvas.Size{Width: 35, Height: 20},
			expectedHeaderFills[0],
		)).To(BeTrue())
	}
}
```

The existing orientation-none test continues to verify that border-only
directories emit no rail while preserving their structural border.

- [ ] **Step 7: Format and run all treemap tests**

Run:

```bash
GOENV_VERSION=1.26.1 task fmt
go test ./internal/treemap -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit depth-aware rendering**

```bash
git add internal/treemap/inks.go internal/treemap/header_palette_test.go \
  internal/treemap/render.go internal/treemap/render_directory_chrome_test.go
git commit -m "feat(treemap): color directory rails by depth" \
  -m "Co-authored-by: Copilot App <223556219+Copilot@users.noreply.github.com>" \
  -m "Copilot-Session: 0113bb97-3cb7-4511-a2a7-f156a4d83cf6"
```

### Task 3: Update Visual Artifacts and Verify the Branch

**Files:**
- Modify: `internal/goldentest/testdata/treemap-png.golden`
- Modify: `internal/goldentest/testdata/treemap-svg.golden`
- Modify: `samples/tree-map/code-visualizer.png`
- Modify: `samples/tree-map/code-visualizer.svg`

- [ ] **Step 1: Record unrelated sample modifications**

Run:

```bash
git status --short samples
```

Expected: other visualization samples may already be modified. Record those
paths and do not restore, stage, or commit them.

- [ ] **Step 2: Confirm old treemap goldens fail**

Run:

```bash
go test ./internal/goldentest -run TestGolden_Treemap -count=1
```

Expected: FAIL for treemap PNG and SVG because rail colors changed.

- [ ] **Step 3: Regenerate and verify only treemap goldens**

Run:

```bash
GOLDIE_UPDATE=1 go test ./internal/goldentest -run TestGolden_Treemap -count=1
go test ./internal/goldentest -run TestGolden_Treemap -count=1
```

Expected: PASS; only the two treemap golden files change.

- [ ] **Step 4: Regenerate only tree-map samples**

Run:

```bash
task samples-tree-map
```

Expected: only `samples/tree-map/code-visualizer.png` and
`samples/tree-map/code-visualizer.svg` are rewritten by this command. Existing
changes under other sample directories remain untouched.

- [ ] **Step 5: Inspect the SVG artifacts**

Run:

```bash
rg 'fill="rgba\\((32,38,49|47,59,77|61,82,104|81,106,125|95,120,136),1\\.000\\)"' \
  internal/goldentest/testdata/treemap-svg.golden \
  samples/tree-map/code-visualizer.svg
```

Expected: multiple slate fills appear, corresponding to different visible
depths. Confirm white text and the existing `rotate(-90.00 ...)` left-rail
labels remain.

- [ ] **Step 6: Commit only related visual artifacts**

Before staging, run:

```bash
git status --short
```

Then stage exactly:

```bash
git add internal/goldentest/testdata/treemap-png.golden \
  internal/goldentest/testdata/treemap-svg.golden \
  samples/tree-map/code-visualizer.png \
  samples/tree-map/code-visualizer.svg
git diff --cached --name-only
```

Expected staged paths: exactly the four files listed above. Do not stage other
modified samples.

Commit:

```bash
git commit -m "test(treemap): update depth palette snapshots" \
  -m "Co-authored-by: Copilot App <223556219+Copilot@users.noreply.github.com>" \
  -m "Copilot-Session: 0113bb97-3cb7-4511-a2a7-f156a4d83cf6"
```

- [ ] **Step 7: Run focused verification**

Run:

```bash
go test ./internal/treemap ./internal/goldentest \
  -run 'Test(Layout|HeaderFills|RenderToCanvas|Golden_Treemap)' -count=1
```

Expected: PASS.

- [ ] **Step 8: Run repository gates without disturbing unrelated samples**

Because pre-existing unrelated sample modifications make
`task verify-no-changes` unsuitable, dispatch a low-noise task agent to run:

```bash
PATH="$PWD/tools:$PATH" task tidy
PATH="$PWD/tools:$PATH" task build
PATH="$PWD/tools:$PATH" task test
PATH="$PWD/tools:$PATH" task lint
```

Ask it to return only exit statuses, failing test/linter identities and
`file:line` messages, or a one-line success note. Expected: all four commands
exit 0.

Afterward run:

```bash
git status --short
```

Expected: only the pre-existing, unrelated sample modifications remain.

- [ ] **Step 9: Push the updated PR branch**

Run:

```bash
git push origin theunrepentantgeek-design-treemap-directory-labels
```

Expected: the existing pull request branch advances without force-pushing.
