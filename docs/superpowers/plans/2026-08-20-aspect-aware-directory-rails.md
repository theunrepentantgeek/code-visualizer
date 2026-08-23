# Aspect-Aware Treemap Directory Rails Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace unconditional tree-map directory headers with conditional top or left label rails that preserve child space and omit unreadable labels.

**Architecture:** A pure treemap chrome resolver measures and truncates directory names, chooses rail orientation from rectangle aspect ratio, and returns both rail and child-content bounds. The recursive layout stores that resolved metadata on each `TreemapRectangle`; rendering consumes it without remeasuring text, keeping PNG and SVG output consistent.

**Tech Stack:** Go 1.26.1, `internal/canvas/textlayout`, `nikolaydubina/treemap/layout`, Gomega, Goldie v2, Task

---

## File Map

| Path | Responsibility |
|---|---|
| `internal/treemap/directory_chrome.go` | New pure resolver for rail orientation, bounds, text fitting, and rune-safe truncation |
| `internal/treemap/directory_chrome_test.go` | New focused unit tests for all chrome and fitting rules |
| `internal/treemap/node.go` | Add directory chrome, orientation, and bounds metadata to layout rectangles |
| `internal/treemap/layout.go` | Resolve chrome before squarifying children; remove unconditional header geometry |
| `internal/treemap/layout_test.go` | Verify root, nested orientation, reclaimed space, containment, and offset behavior |
| `internal/treemap/node_test.go` | Replace the obsolete always-header assertion with directory-boundary invariants |
| `internal/treemap/inks.go` | Remove the old header-height alias; retain rail colors and border sizing |
| `internal/treemap/render.go` | Draw optional top/left rail metadata and correctly rotated fitted labels |
| `internal/treemap/render_focus_test.go` | Extend the shared capture backend to record text calls |
| `internal/treemap/render_directory_chrome_test.go` | New render tests for top, left, and omitted rails |
| `internal/goldentest/testdata/treemap-png.golden` | Updated raster snapshot |
| `internal/goldentest/testdata/treemap-svg.golden` | Updated SVG snapshot |

### Task 1: Build the Pure Directory Chrome Resolver

**Files:**
- Create: `internal/treemap/directory_chrome.go`
- Create: `internal/treemap/directory_chrome_test.go`
- Modify: `internal/treemap/node.go`

- [ ] **Step 1: Add failing resolver tests**

Create `internal/treemap/directory_chrome_test.go`:

```go
package treemap

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/textlayout"
)

func TestResolveDirectoryChrome_RootHasNoRail(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	got := resolveDirectoryChrome(
		RectangleBounds{X: 10, Y: 20, W: 100, H: 60},
		"root",
		true,
	)

	g.Expect(got.Orientation).To(Equal(DirectoryLabelNone))
	g.Expect(got.Text).To(BeEmpty())
	g.Expect(got.Rail).To(Equal(RectangleBounds{}))
	g.Expect(got.Content).To(Equal(RectangleBounds{X: 14, Y: 24, W: 92, H: 52}))
}

func TestResolveDirectoryChrome_ChoosesDominantAxis(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		rect RectangleBounds
		want DirectoryLabelOrientation
		rail RectangleBounds
		body RectangleBounds
	}{
		{
			name: "wide uses top",
			rect: RectangleBounds{X: 10, Y: 20, W: 100, H: 60},
			want: DirectoryLabelTop,
			rail: RectangleBounds{X: 10, Y: 20, W: 100, H: 20},
			body: RectangleBounds{X: 14, Y: 40, W: 92, H: 36},
		},
		{
			name: "square uses top",
			rect: RectangleBounds{X: 10, Y: 20, W: 100, H: 100},
			want: DirectoryLabelTop,
			rail: RectangleBounds{X: 10, Y: 20, W: 100, H: 20},
			body: RectangleBounds{X: 14, Y: 40, W: 92, H: 76},
		},
		{
			name: "tall uses left",
			rect: RectangleBounds{X: 10, Y: 20, W: 60, H: 100},
			want: DirectoryLabelLeft,
			rail: RectangleBounds{X: 10, Y: 20, W: 20, H: 100},
			body: RectangleBounds{X: 30, Y: 24, W: 36, H: 92},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			got := resolveDirectoryChrome(tc.rect, "source", false)

			g.Expect(got.Orientation).To(Equal(tc.want))
			g.Expect(got.Text).To(Equal("source"))
			g.Expect(got.Rail).To(Equal(tc.rail))
			g.Expect(got.Content).To(Equal(tc.body))
		})
	}
}

func TestResolveDirectoryChrome_OmitsRailWhenContentWouldBeTooSmall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		rect RectangleBounds
		body RectangleBounds
	}{
		{
			name: "wide but too short",
			rect: RectangleBounds{W: 100, H: 39},
			body: RectangleBounds{X: 4, Y: 4, W: 92, H: 31},
		},
		{
			name: "tall but too narrow",
			rect: RectangleBounds{W: 39, H: 100},
			body: RectangleBounds{X: 4, Y: 4, W: 31, H: 92},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			got := resolveDirectoryChrome(tc.rect, "source", false)

			g.Expect(got.Orientation).To(Equal(DirectoryLabelNone))
			g.Expect(got.Text).To(BeEmpty())
			g.Expect(got.Rail).To(Equal(RectangleBounds{}))
			g.Expect(got.Content).To(Equal(tc.body))
		})
	}
}

func TestFitDirectoryLabel_TruncatesToFourRunesAndEllipsis(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fourWidth, _ := textlayout.MeasureString("abcd…", directoryLabelFontSize)
	fiveWidth, _ := textlayout.MeasureString("abcde…", directoryLabelFontSize)

	got, ok := fitDirectoryLabel("abcdefgh", (fourWidth+fiveWidth)/2)

	g.Expect(ok).To(BeTrue())
	g.Expect(got).To(Equal("abcd…"))
}

func TestFitDirectoryLabel_OmitsWhenFourRunesAndEllipsisDoNotFit(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	minWidth, _ := textlayout.MeasureString("abcd…", directoryLabelFontSize)

	got, ok := fitDirectoryLabel("abcdefgh", minWidth-1)

	g.Expect(ok).To(BeFalse())
	g.Expect(got).To(BeEmpty())
}

func TestFitDirectoryLabel_TruncatesUnicodeByRune(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fourWidth, _ := textlayout.MeasureString("αβγδ…", directoryLabelFontSize)
	fiveWidth, _ := textlayout.MeasureString("αβγδε…", directoryLabelFontSize)

	got, ok := fitDirectoryLabel("αβγδεζη", (fourWidth+fiveWidth)/2)

	g.Expect(ok).To(BeTrue())
	g.Expect(got).To(Equal("αβγδ…"))
}

func TestFitDirectoryLabel_KeepsShortCompleteName(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	width, _ := textlayout.MeasureString("cmd", directoryLabelFontSize)

	got, ok := fitDirectoryLabel("cmd", width)

	g.Expect(ok).To(BeTrue())
	g.Expect(got).To(Equal("cmd"))
}
```

- [ ] **Step 2: Run the resolver tests and verify they fail**

Run:

```bash
go test ./internal/treemap -run 'TestResolveDirectoryChrome|TestFitDirectoryLabel' -count=1
```

Expected: FAIL because `RectangleBounds`, `DirectoryLabelOrientation`,
`resolveDirectoryChrome`, and `fitDirectoryLabel` do not exist.

- [ ] **Step 3: Add chrome metadata to layout rectangles**

Replace `internal/treemap/node.go` with:

```go
package treemap

// DirectoryLabelOrientation identifies the edge reserved for a directory label.
type DirectoryLabelOrientation uint8

const (
	DirectoryLabelNone DirectoryLabelOrientation = iota
	DirectoryLabelTop
	DirectoryLabelLeft
)

// RectangleBounds describes resolved geometry within a treemap rectangle.
type RectangleBounds struct {
	X float64
	Y float64
	W float64
	H float64
}

// DirectoryChrome is the resolved label rail and child-content geometry.
type DirectoryChrome struct {
	Orientation DirectoryLabelOrientation
	Text        string
	Rail        RectangleBounds
	Content     RectangleBounds
}

// TreemapRectangle is a positioned visual element in the rendered treemap.
type TreemapRectangle struct {
	X           float64
	Y           float64
	W           float64
	H           float64
	Label       string
	IsDirectory bool
	Chrome      DirectoryChrome
	Children    []TreemapRectangle
}
```

- [ ] **Step 4: Implement the pure resolver**

Create `internal/treemap/directory_chrome.go`:

```go
package treemap

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/textlayout"
)

const (
	directoryRailThickness  = 20.0
	directoryPadding        = 4.0
	directoryLabelFontSize  = 12.0
	minDirectoryContentSize = 20.0
	minTruncatedRunes       = 4
)

func resolveDirectoryChrome(
	rect RectangleBounds,
	name string,
	isRoot bool,
) DirectoryChrome {
	borderOnly := DirectoryChrome{Content: insetDirectoryBounds(rect)}
	if isRoot || name == "" {
		return borderOnly
	}

	var candidate DirectoryChrome
	if rect.W >= rect.H {
		candidate = DirectoryChrome{
			Orientation: DirectoryLabelTop,
			Rail: RectangleBounds{
				X: rect.X,
				Y: rect.Y,
				W: rect.W,
				H: directoryRailThickness,
			},
			Content: RectangleBounds{
				X: rect.X + directoryPadding,
				Y: rect.Y + directoryRailThickness,
				W: rect.W - 2*directoryPadding,
				H: rect.H - directoryRailThickness - directoryPadding,
			},
		}
	} else {
		candidate = DirectoryChrome{
			Orientation: DirectoryLabelLeft,
			Rail: RectangleBounds{
				X: rect.X,
				Y: rect.Y,
				W: directoryRailThickness,
				H: rect.H,
			},
			Content: RectangleBounds{
				X: rect.X + directoryRailThickness,
				Y: rect.Y + directoryPadding,
				W: rect.W - directoryRailThickness - directoryPadding,
				H: rect.H - 2*directoryPadding,
			},
		}
	}

	if candidate.Content.W < minDirectoryContentSize ||
		candidate.Content.H < minDirectoryContentSize {
		return borderOnly
	}

	railLength := candidate.Rail.W
	if candidate.Orientation == DirectoryLabelLeft {
		railLength = candidate.Rail.H
	}

	text, ok := fitDirectoryLabel(name, railLength-2*directoryPadding)
	if !ok {
		return borderOnly
	}

	candidate.Text = text

	return candidate
}

func insetDirectoryBounds(rect RectangleBounds) RectangleBounds {
	return RectangleBounds{
		X: rect.X + directoryPadding,
		Y: rect.Y + directoryPadding,
		W: rect.W - 2*directoryPadding,
		H: rect.H - 2*directoryPadding,
	}
}

func fitDirectoryLabel(name string, maxWidth float64) (string, bool) {
	if name == "" || maxWidth <= 0 {
		return "", false
	}

	runes := []rune(name)
	candidates := make([]string, 0, max(1, len(runes)-minTruncatedRunes+1))
	candidates = append(candidates, name)

	for keep := len(runes) - 1; keep >= minTruncatedRunes; keep-- {
		candidates = append(candidates, string(runes[:keep])+"…")
	}

	widths, _ := textlayout.MeasureStrings(candidates, directoryLabelFontSize)
	for i, width := range widths {
		if width <= maxWidth {
			return candidates[i], true
		}
	}

	return "", false
}
```

- [ ] **Step 5: Format and run the resolver tests**

Run:

```bash
task fmt
go test ./internal/treemap -run 'TestResolveDirectoryChrome|TestFitDirectoryLabel' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit the resolver**

```bash
git add internal/treemap/node.go internal/treemap/directory_chrome.go internal/treemap/directory_chrome_test.go
git commit -m "feat(treemap): resolve aspect-aware directory chrome" -m "Co-authored-by: Copilot App <223556219+Copilot@users.noreply.github.com>"
```

### Task 2: Integrate Chrome Resolution Into Recursive Layout

**Files:**
- Modify: `internal/treemap/layout.go`
- Modify: `internal/treemap/layout_test.go`
- Modify: `internal/treemap/node_test.go`

- [ ] **Step 1: Add failing layout integration tests**

Append these tests to `internal/treemap/layout_test.go`:

```go
func TestLayout_RootUsesHeaderFreePaddedInterior(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeFile("only.go", 100)},
	}

	rect := Layout(root, 200, 100, filesystem.FileSize)

	g.Expect(rect.Chrome.Orientation).To(Equal(DirectoryLabelNone))
	g.Expect(rect.Chrome.Content).To(Equal(RectangleBounds{X: 4, Y: 4, W: 192, H: 92}))
	g.Expect(rect.Children).To(HaveLen(1))
	g.Expect(rect.Children[0].Y).To(BeNumerically(">=", 4.0))
	g.Expect(rect.Children[0].Y).To(BeNumerically("<", directoryRailThickness))
}

func TestLayout_NestedDirectoryUsesTopRailWhenWide(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Dirs: []*model.Directory{{
			Name:  "source",
			Files: []*model.File{makeFile("only.go", 100)},
		}},
	}

	rect := Layout(root, 200, 100, filesystem.FileSize)
	dir := rect.Children[0]

	g.Expect(dir.IsDirectory).To(BeTrue())
	g.Expect(dir.Chrome.Orientation).To(Equal(DirectoryLabelTop))
	g.Expect(dir.Chrome.Text).To(Equal("source"))
	g.Expect(dir.Children[0].Y).To(BeNumerically(">=", dir.Y+directoryRailThickness))
}

func TestLayout_NestedDirectoryUsesLeftRailWhenTall(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Dirs: []*model.Directory{{
			Name:  "source",
			Files: []*model.File{makeFile("only.go", 100)},
		}},
	}

	rect := Layout(root, 100, 200, filesystem.FileSize)
	dir := rect.Children[0]

	g.Expect(dir.IsDirectory).To(BeTrue())
	g.Expect(dir.Chrome.Orientation).To(Equal(DirectoryLabelLeft))
	g.Expect(dir.Chrome.Text).To(Equal("source"))
	g.Expect(dir.Children[0].X).To(BeNumerically(">=", dir.X+directoryRailThickness))
}

func TestLayout_OmittedRailReturnsSpaceToChildren(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Dirs: []*model.Directory{{
			Name:  "source",
			Files: []*model.File{makeFile("only.go", 100)},
		}},
	}

	rect := Layout(root, 50, 50, filesystem.FileSize)
	dir := rect.Children[0]

	g.Expect(dir.Chrome.Orientation).To(Equal(DirectoryLabelNone))
	g.Expect(dir.Children).To(HaveLen(1))
	g.Expect(dir.Children[0].Y).To(BeNumerically("<", dir.Y+directoryRailThickness))
}

func TestLayout_ChildrenStayWithinResolvedContent(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Dirs: []*model.Directory{{
			Name: "source",
			Files: []*model.File{
				makeFile("a.go", 100),
				makeFile("b.go", 200),
			},
		}},
	}

	rect := Layout(root, 240, 120, filesystem.FileSize)
	dir := rect.Children[0]
	content := dir.Chrome.Content

	for _, child := range dir.Children {
		g.Expect(child.X).To(BeNumerically(">=", content.X))
		g.Expect(child.Y).To(BeNumerically(">=", content.Y))
		g.Expect(child.X + child.W).To(BeNumerically("<=", content.X+content.W))
		g.Expect(child.Y + child.H).To(BeNumerically("<=", content.Y+content.H))
	}
}
```

Extend `TestOffsetRects_ShiftsCoordinates` so its fixture contains chrome:

```go
rect := TreemapRectangle{
	X: 10, Y: 20, W: 100, H: 50,
	Chrome: DirectoryChrome{
		Orientation: DirectoryLabelTop,
		Rail:        RectangleBounds{X: 10, Y: 20, W: 100, H: 20},
		Content:     RectangleBounds{X: 14, Y: 40, W: 92, H: 26},
	},
}
```

Add these assertions after `OffsetRects(&rect, 30, 40)`:

```go
g.Expect(rect.Chrome.Rail).To(Equal(RectangleBounds{X: 40, Y: 60, W: 100, H: 20}))
g.Expect(rect.Chrome.Content).To(Equal(RectangleBounds{X: 44, Y: 80, W: 92, H: 26}))
```

- [ ] **Step 2: Run the integration tests and verify they fail**

Run:

```bash
go test ./internal/treemap -run 'TestLayout_|TestOffsetRects_ShiftsCoordinates' -count=1
```

Expected: FAIL because layout does not populate `Chrome`, still reserves a root
header, and does not offset chrome bounds.

- [ ] **Step 3: Resolve chrome before laying out children**

In `internal/treemap/layout.go`:

1. Remove `HeaderHeight` and `padding` from the constant block, leaving:

```go
const (
	siblingGap  = 2.0
	minFileSize = 1.0
)
```

2. Change the entry point and recursive function signature:

```go
func Layout(root *model.Directory, width, height int, sizeMetric metric.Name) TreemapRectangle {
	box := layout.Box{X: 0, Y: 0, W: float64(width), H: float64(height)}

	return layoutDir(root, box, sizeMetric, true)
}

func layoutDir(
	dir *model.Directory,
	box layout.Box,
	sizeMetric metric.Name,
	isRoot bool,
) TreemapRectangle {
	bounds := RectangleBounds{X: box.X, Y: box.Y, W: box.W, H: box.H}
	rect := TreemapRectangle{
		X: box.X, Y: box.Y, W: box.W, H: box.H,
		Label: dir.Name, IsDirectory: true,
		Chrome: resolveDirectoryChrome(bounds, dir.Name, isRoot),
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
		rect.Children = append(rect.Children, layoutChild(dir, c, b, sizeMetric))
	}

	return rect
}
```

3. Remove the old `contentArea` function.

4. Change the directory branch in `layoutChild`:

```go
if c.isDir {
	return layoutDir(dir.Dirs[c.dirIdx], b, sizeMetric, false)
}
```

5. Offset chrome bounds with the rectangle:

```go
func OffsetRects(rect *TreemapRectangle, dx, dy float64) {
	rect.X += dx
	rect.Y += dy

	if rect.IsDirectory {
		if rect.Chrome.Orientation != DirectoryLabelNone {
			rect.Chrome.Rail.X += dx
			rect.Chrome.Rail.Y += dy
		}

		rect.Chrome.Content.X += dx
		rect.Chrome.Content.Y += dy
	}

	for i := range rect.Children {
		OffsetRects(&rect.Children[i], dx, dy)
	}
}
```

- [ ] **Step 4: Replace the obsolete header test**

Replace `TestDirectoryHeaderBar` in `internal/treemap/node_test.go` with:

```go
func TestDirectoryWithoutRailKeepsStructuralSeparation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Dirs: []*model.Directory{
			{
				Name:  "a-very-long-directory-name",
				Files: []*model.File{makeFile("file.go", 100)},
			},
		},
	}

	rects := Layout(root, 50, 50, filesystem.FileSize)
	dirRect := findDirRect(rects, "a-very-long-directory-name")

	g.Expect(dirRect).NotTo(BeNil())
	if dirRect == nil {
		return
	}

	g.Expect(dirRect.Chrome.Orientation).To(Equal(DirectoryLabelNone))
	g.Expect(dirRect.Chrome.Content.X).To(BeNumerically(">", dirRect.X))
	g.Expect(dirRect.Chrome.Content.Y).To(BeNumerically(">", dirRect.Y))
}
```

- [ ] **Step 5: Format and run all treemap tests**

Run:

```bash
task fmt
go test ./internal/treemap -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit layout integration**

```bash
git add internal/treemap/layout.go internal/treemap/layout_test.go internal/treemap/node_test.go
git commit -m "feat(treemap): apply directory rails during layout" -m "Co-authored-by: Copilot App <223556219+Copilot@users.noreply.github.com>"
```

### Task 3: Render Top, Left, and Omitted Rails

**Files:**
- Modify: `internal/treemap/inks.go`
- Modify: `internal/treemap/render.go`
- Modify: `internal/treemap/render_focus_test.go`
- Create: `internal/treemap/render_directory_chrome_test.go`

- [ ] **Step 1: Extend the capture backend and add failing render tests**

In `internal/treemap/render_focus_test.go`, add:

```go
type textCall struct {
	pos      canvas.Position
	text     string
	fontSize float64
	anchor   canvas.TextAnchor
	rotation float64
}
```

Add a `texts []textCall` field to `captureBackend`, and replace its `DrawText`
method with:

```go
func (b *captureBackend) DrawText(
	pos canvas.Position,
	text string,
	_ color.RGBA,
	fontSize float64,
	anchor canvas.TextAnchor,
	rotation float64,
) {
	b.texts = append(b.texts, textCall{
		pos:      pos,
		text:     text,
		fontSize: fontSize,
		anchor:   anchor,
		rotation: rotation,
	})
}
```

Create `internal/treemap/render_directory_chrome_test.go`:

```go
package treemap_test

import (
	"image/color"
	"math"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/treemap"
)

func TestRenderToCanvas_DrawsTopDirectoryRail(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	backend := renderChromeFixture(t, treemap.DirectoryChrome{
		Orientation: treemap.DirectoryLabelTop,
		Text:        "source",
		Rail:        treemap.RectangleBounds{X: 10, Y: 10, W: 80, H: 20},
		Content:     treemap.RectangleBounds{X: 14, Y: 30, W: 72, H: 56},
	})

	g.Expect(backend.texts).To(ContainElement(textCall{
		pos:      canvas.Position{X: 14, Y: 20},
		text:     "source",
		fontSize: 12,
		anchor:   canvas.AnchorStart,
		rotation: 0,
	}))
	var foundRail bool
	for _, call := range backend.rectangles {
		if call.pos == (canvas.Position{X: 10, Y: 10}) &&
			call.size == (canvas.Size{W: 80, H: 20}) {
			foundRail = true
		}
	}
	g.Expect(foundRail).To(BeTrue())
}

func TestRenderToCanvas_DrawsLeftDirectoryRail(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	backend := renderChromeFixture(t, treemap.DirectoryChrome{
		Orientation: treemap.DirectoryLabelLeft,
		Text:        "source",
		Rail:        treemap.RectangleBounds{X: 10, Y: 10, W: 20, H: 80},
		Content:     treemap.RectangleBounds{X: 30, Y: 14, W: 56, H: 72},
	})

	g.Expect(backend.texts).To(ContainElement(textCall{
		pos:      canvas.Position{X: 20, Y: 86},
		text:     "source",
		fontSize: 12,
		anchor:   canvas.AnchorStart,
		rotation: -math.Pi / 2,
	}))
}

func TestRenderToCanvas_OmitsDirectoryRailAndLabel(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	backend := renderChromeFixture(t, treemap.DirectoryChrome{
		Orientation: treemap.DirectoryLabelNone,
		Content:     treemap.RectangleBounds{X: 14, Y: 14, W: 72, H: 72},
	})

	g.Expect(backend.texts).To(BeEmpty())
	for _, call := range backend.rectangles {
		g.Expect(call.size).NotTo(Equal(canvas.Size{W: 80, H: 20}))
		g.Expect(call.size).NotTo(Equal(canvas.Size{W: 20, H: 80}))
	}
}

func renderChromeFixture(t *testing.T, chrome treemap.DirectoryChrome) *captureBackend {
	t.Helper()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Dirs: []*model.Directory{{Name: "source"}},
	}
	rects := treemap.TreemapRectangle{
		X: 0, Y: 0, W: 100, H: 100,
		Label: "root", IsDirectory: true,
		Chrome: treemap.DirectoryChrome{
			Orientation: treemap.DirectoryLabelNone,
			Content:     treemap.RectangleBounds{X: 4, Y: 4, W: 92, H: 92},
		},
		Children: []treemap.TreemapRectangle{{
			X: 10, Y: 10, W: 80, H: 80,
			Label: "source", IsDirectory: true, Chrome: chrome,
		}},
	}
	is := treemap.Inks{
		Fill:   inks.FixedInk(color.RGBA{A: 255}),
		Border: inks.FixedInk(color.RGBA{A: 255}),
	}

	cv := treemap.RenderToCanvas(rects, root, 100, 100, is, "")
	backend := &captureBackend{}
	g.Expect(cv.RenderTo(backend)).To(Succeed())

	return backend
}
```

- [ ] **Step 2: Run the render tests and verify they fail**

Run:

```bash
go test ./internal/treemap -run 'TestRenderToCanvas_(Draws|Omits)DirectoryRail' -count=1
```

Expected: FAIL because rendering still draws a top header from `rect.Label`,
ignores `rect.Chrome`, and has no rotated left-label spec.

- [ ] **Step 3: Replace fixed header rendering with resolved rails**

In `internal/treemap/inks.go`, remove `headerHeight = HeaderHeight` from the
constant block:

```go
const (
	minBorderDim = 20.0
	midBorderDim = 100.0
)
```

In `internal/treemap/render.go`:

1. Add `"math"` to imports.

2. Replace `dirLabelSpec` with separate top and left specs:

```go
dirTopLabelSpec = &canvas.TextSpec{
	Ink:      inks.FixedInk(palette.White),
	FontSize: directoryLabelFontSize,
	Anchor:   canvas.AnchorStart,
}
dirLeftLabelSpec = &canvas.TextSpec{
	Ink:      inks.FixedInk(palette.White),
	FontSize: directoryLabelFontSize,
	Anchor:   canvas.AnchorStart,
	Rotation: -math.Pi / 2,
}
```

3. Replace the header and label portion at the start of `addDirectoryShapes`
with:

```go
if rect.Chrome.Orientation != DirectoryLabelNone {
	rail := rect.Chrome.Rail
	cv.AddRectangle(canvas.LayerStructure, canvas.Rectangle{
		Spec:  dirHeaderSpec,
		X:     rail.X,
		Y:     rail.Y,
		W:     rail.W,
		H:     rail.H,
		Focus: canvasmodel.Point{X: 0.5, Y: 0.5},
	})

	if rect.Chrome.Text != "" {
		spec := dirTopLabelSpec
		x := rail.X + directoryPadding
		y := rail.Y + rail.H/2

		if rect.Chrome.Orientation == DirectoryLabelLeft {
			spec = dirLeftLabelSpec
			x = rail.X + rail.W/2
			y = rail.Y + rail.H - directoryPadding
		}

		cv.AddText(canvas.LayerOverlay, canvas.Text{
			Spec:    spec,
			X:       x,
			Y:       y,
			Content: rect.Chrome.Text,
		})
	}
}
```

Leave the directory-border section unchanged so border-only directories remain
visible.

- [ ] **Step 4: Format and run all treemap tests**

Run:

```bash
task fmt
go test ./internal/treemap -count=1
```

Expected: PASS, including the existing weighted-focus, file-label, and output
format tests.

- [ ] **Step 5: Commit rendering**

```bash
git add internal/treemap/inks.go internal/treemap/render.go internal/treemap/render_focus_test.go internal/treemap/render_directory_chrome_test.go
git commit -m "feat(treemap): render aspect-aware directory rails" -m "Co-authored-by: Copilot App <223556219+Copilot@users.noreply.github.com>"
```

### Task 4: Update Golden Snapshots and Verify the Feature

**Files:**
- Modify: `internal/goldentest/testdata/treemap-png.golden`
- Modify: `internal/goldentest/testdata/treemap-svg.golden`

- [ ] **Step 1: Confirm the old golden snapshots fail**

Run:

```bash
go test ./internal/goldentest -run TestGolden_Treemap -count=1
```

Expected: FAIL for `treemap-png` and `treemap-svg` because the root header is
gone and nested directories now use resolved rails.

- [ ] **Step 2: Regenerate only the treemap snapshots**

Run:

```bash
GOLDIE_UPDATE=1 go test ./internal/goldentest -run TestGolden_Treemap -count=1
```

Expected: PASS and updates:

```text
internal/goldentest/testdata/treemap-png.golden
internal/goldentest/testdata/treemap-svg.golden
```

- [ ] **Step 3: Inspect the SVG snapshot**

Run:

```bash
rg '<text|rotate\\(-90' internal/goldentest/testdata/treemap-svg.golden
```

Expected: directory labels are absent for the root, visible nested labels use
their fitted text, and any left rail label is emitted with `rotate(-90.00 ...)`.
If the fixed 320×240 fixture produces no tall labeled directory, rely on the
focused render test for rotation and confirm the SVG otherwise reflects the
new header-free root and conditional rails.

- [ ] **Step 4: Run focused verification**

Run:

```bash
go test ./internal/treemap ./internal/goldentest -run 'Test(Layout|ResolveDirectoryChrome|FitDirectoryLabel|RenderToCanvas|Golden_Treemap)' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit the golden snapshots**

```bash
git add internal/goldentest/testdata/treemap-png.golden internal/goldentest/testdata/treemap-svg.golden
git commit -m "test(treemap): update directory rail snapshots" -m "Co-authored-by: Copilot App <223556219+Copilot@users.noreply.github.com>"
```

- [ ] **Step 6: Run repository CI through a low-noise task agent**

Per the repository workflow rules, dispatch an Explore or equivalent task
agent to run:

```bash
task ci
```

Ask it to return only the exit status, failing linter/test identities,
offending `file:line` messages, or a one-line success note.

Expected: exit status 0 with no lint, build, test, formatting, or dirty-tree
failures.

- [ ] **Step 7: Verify the final change set**

Run:

```bash
git --no-pager status --short
git --no-pager diff main...HEAD --stat
```

Expected: no tracked working-tree changes. The branch contains the approved
design, implementation plan, resolver, layout integration, rendering changes,
tests, and two updated treemap golden snapshots.
