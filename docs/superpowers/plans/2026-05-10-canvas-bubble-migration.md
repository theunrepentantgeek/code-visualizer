# Bubbletree Canvas Migration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the bubbletree visualization from the old `internal/render/` pipeline to the new `internal/canvas/` API, then delete the old render code.

**Architecture:** The bubbletree bridge (`bubble_canvas.go`) converts BubbleNode layout geometry + model.Directory metric data into Canvas shapes. Unlike treemap/radial which walk layout and model trees in parallel using positional indices, bubbletree uses *path-based lookup* — it indexes BubbleNodes by their `Path` field, then walks `model.Directory` looking up nodes by path. Directory circles get semi-transparent fills via `WithOpacity`. Directory labels use arc text via a new `AddArcText` method on Canvas (the backends already implement `DrawArcText`).

**Tech Stack:** Go 1.26.1, Canvas API (`internal/canvas`), bubbletree layout (`internal/bubbletree`)

---

## File Map

**Canvas API extension:**
- Modify: `internal/canvas/shape.go` — add `ArcText` shape struct
- Modify: `internal/canvas/text_spec.go` — add `ArcTextSpec` struct
- Modify: `internal/canvas/canvas.go` — add `shapeArcText` kind, `arcText` field in `layeredShape`, `AddArcText` method, `drawArcText` dispatcher
- Modify: `internal/canvas/canvas_test.go` — add test for `AddArcText`

**Bubbletree node cleanup:**
- Modify: `internal/bubbletree/node.go` — remove `FillColour` and `BorderColour` fields

**Bridge (new):**
- Create: `cmd/codeviz/bubble_canvas.go` — Canvas bridge functions

**Command rewire:**
- Modify: `cmd/codeviz/bubbletree_cmd.go` — replace old render pipeline with Canvas pipeline

**Old render deletion:**
- Delete: `internal/render/bubbletree.go`
- Delete: `internal/render/svg_bubble.go`
- Delete: `internal/render/bubbletree_test.go`
- Delete: `internal/render/bubble_font.go`
- Delete: `internal/render/testdata/bubble-tree.png` (golden file)

**Tests (new):**
- Create: `cmd/codeviz/bubble_canvas_test.go`

---

## Task 1: Add ArcText shape to Canvas API

The Canvas API has `DrawArcText` on the Backend interface (and both backends implement it), but there is no `AddArcText` method on Canvas to queue arc text shapes. We need to add the shape type, spec, and dispatch plumbing.

**Files:**
- Modify: `internal/canvas/text_spec.go` — add `ArcTextSpec`
- Modify: `internal/canvas/shape.go` — add `ArcText` struct
- Modify: `internal/canvas/canvas.go` — add kind, field, method, dispatcher
- Modify: `internal/canvas/canvas_test.go` — add test

- [ ] **Step 1: Add `ArcTextSpec` to `text_spec.go`**

Add after the `TextSpec` struct (after line 39):

```go
// ArcTextSpec defines the visual template for text curved along a circle arc.
type ArcTextSpec struct {
	Ink      Ink
	FontSize float64
}
```

- [ ] **Step 2: Add `ArcText` shape to `shape.go`**

Add after the `Path` struct (after line 41):

```go
// ArcText carries position and content for text curved along a circle arc.
type ArcText struct {
	Spec   *ArcTextSpec
	X, Y   float64 // circle centre
	Radius float64 // circle radius (label is inset from this)
	Text   string
}
```

- [ ] **Step 3: Add `shapeArcText` kind and plumbing to `canvas.go`**

Add `shapeArcText` to the shapeKind constants (after `shapePath`):

```go
	shapeArcText
```

Add `arcText` field to `layeredShape` (after `path  *Path`):

```go
	arcText *ArcText
```

Add the `AddArcText` method (after `AddPath`):

```go
// AddArcText records arc text on the given layer.
func (c *Canvas) AddArcText(layer Layer, a ArcText) {
	c.shapes = append(c.shapes, layeredShape{
		layer:   layer,
		order:   len(c.shapes),
		kind:    shapeArcText,
		arcText: &a,
	})
}
```

Add `shapeArcText` case to `dispatchShape` (after the `shapePath` case):

```go
	case shapeArcText:
		c.drawArcText(backend, s.arcText)
```

Add the `drawArcText` method (after `drawPath`):

```go
func (*Canvas) drawArcText(b Backend, a *ArcText) {
	ink := a.Spec.Ink.Dip(MetricValue{})
	b.DrawArcText(
		Position{X: a.X, Y: a.Y},
		a.Radius,
		a.Text,
		ink,
		a.Spec.FontSize,
	)
}
```

- [ ] **Step 4: Add test for `AddArcText`**

Add to `canvas_test.go`:

```go
func TestAddArcText_DispatchesToBackend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := NewCanvas(400, 400)
	spec := &ArcTextSpec{
		Ink:      FixedInk(color.RGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF}),
		FontSize: 14,
	}

	c.AddArcText(LayerOverlay, ArcText{
		Spec:   spec,
		X:      200,
		Y:      200,
		Radius: 100,
		Text:   "hello",
	})

	mock := newMockBackend()
	err := c.RenderTo(mock)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mock.calls).To(HaveLen(1))
	g.Expect(mock.calls[0].method).To(Equal("DrawArcText"))
	g.Expect(mock.calls[0].text).To(Equal("hello"))
	g.Expect(mock.calls[0].pos).To(Equal(Position{X: 200, Y: 200}))
}
```

- [ ] **Step 5: Run tests**

```bash
cd /home/bevan/github/code-visualizer && go test ./internal/canvas/... -v -run TestAddArcText
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/canvas/text_spec.go internal/canvas/shape.go internal/canvas/canvas.go internal/canvas/canvas_test.go
git commit -m "feat(canvas): add AddArcText shape for curved text labels

Add ArcTextSpec, ArcText shape, and Canvas.AddArcText() method with
dispatch to Backend.DrawArcText(). Needed by the bubbletree bridge
for curved directory labels."
```

---

## Task 2: Strip colour fields from BubbleNode

Remove `FillColour` and `BorderColour` from `BubbleNode` — colours are now handled by the Canvas Ink system.

**Files:**
- Modify: `internal/bubbletree/node.go`

- [ ] **Step 1: Remove colour fields**

Replace the BubbleNode struct (lines 20-30 in `internal/bubbletree/node.go`):

```go
// BubbleNode is a positioned visual element in the rendered bubble tree.
// X and Y are absolute pixel coordinates after layout; Radius is the circle radius in pixels.
type BubbleNode struct {
	X, Y        float64 // centre position in pixels
	Radius      float64 // circle radius in pixels
	Path        string  // model path — stable identifier for colour mapping
	Label       string  // display name
	ShowLabel   bool    // whether to render the label for this node
	IsDirectory bool    // true for directory nodes, false for file nodes
	Children    []BubbleNode
}
```

Remove the `"image/color"` import since it is no longer needed.

- [ ] **Step 2: Verify compilation**

```bash
cd /home/bevan/github/code-visualizer && go build ./internal/bubbletree/...
```

Expected: This will fail because bubbletree_cmd.go and render files reference the removed fields. That's expected — we fix those in Tasks 3-4.

- [ ] **Step 3: Commit (will be part of Task 4 commit since this breaks the build on its own)**

Do NOT commit yet — this change breaks compilation until Tasks 3 and 4 are complete.

---

## Task 3: Create bubble_canvas.go bridge

This is the core new file. It translates BubbleNode layout + model.Directory metric data into Canvas shapes. Key differences from the radial bridge:

1. **Path-based lookup**: Index BubbleNodes by `Path`, walk `model.Directory` looking up by path — NOT positional tree walk.
2. **Separate dir/file passes**: Dirs sorted by radius descending (for correct z-order), then files.
3. **Semi-transparent dir fills**: Use `WithOpacity(float64(0x30)/255.0)` on dir fill inks.
4. **Arc text labels**: Use `AddArcText` for directory labels, `AddText` for file labels.
5. **Arc font sizing**: Simple heuristic to compute font size that fits the circle arc.

**Files:**
- Create: `cmd/codeviz/bubble_canvas.go`

- [ ] **Step 1: Create the bridge file**

Create `cmd/codeviz/bubble_canvas.go`:

```go
package main

import (
	"cmp"
	"image/color"
	"math"
	"slices"
	"unicode/utf8"

	"github.com/theunrepentantgeek/code-visualizer/internal/bubbletree"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

var (
	bubbleDefaultFileFill = color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
	bubbleDefaultDirFill  = color.RGBA{R: 0x66, G: 0x99, B: 0xCC, A: 0xFF}
	bubbleDefaultBorder   = color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
	bubbleLabelColour     = color.RGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF}
	bubbleBgColour        = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
)

const (
	bubbleDirOpacity      = float64(0x30) / 255.0
	bubbleBorderWidth     = 0.5
	bubbleArcLabelInset   = 14.0
	bubbleMinArcFontSize  = 7.0
	bubbleDefaultFontSize = 14.0
	bubbleMaxArcFraction  = math.Pi / 2.0
)

// bubbleInks holds the Ink instances for a bubbletree render pass.
type bubbleInks struct {
	fill   canvas.Ink
	border canvas.Ink
}

// buildBubbleInks creates fill and border inks from metric configuration.
func buildBubbleInks(
	root *model.Directory,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
	borderMetric metric.Name,
	borderPaletteName palette.PaletteName,
) bubbleInks {
	inks := bubbleInks{
		fill:   canvas.FixedInk(bubbleDefaultFileFill),
		border: canvas.FixedInk(bubbleDefaultBorder),
	}

	inks.fill = buildMetricInk(root, fillMetric, fillPaletteName, bubbleDefaultFileFill)
	if borderMetric != "" {
		inks.border = buildMetricInk(root, borderMetric, borderPaletteName, bubbleDefaultBorder)
	}

	return inks
}

// renderBubbleToCanvas walks the BubbleNode tree and model tree using
// path-based lookup, adding shapes to the canvas.
func renderBubbleToCanvas(
	nodes *bubbletree.BubbleNode,
	root *model.Directory,
	width, height int,
	inks bubbleInks,
) *canvas.Canvas {
	cv := canvas.NewCanvas(width, height)

	addBubbleBackground(cv, width, height)

	dirs, files := indexBubbleNodes(nodes)
	addBubbleDirDiscs(cv, dirs, root, inks)
	addBubbleFileDiscs(cv, files, root, inks)
	addBubbleLabels(cv, *nodes)

	return cv
}

// addBubbleBackground adds a white background rectangle.
func addBubbleBackground(cv *canvas.Canvas, width, height int) {
	bgSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        canvas.FixedInk(bubbleBgColour),
			Border:      canvas.FixedInk(bubbleBgColour),
			BorderWidth: 0,
		},
	}

	cv.AddRectangle(canvas.LayerBackground, canvas.Rectangle{
		Spec: bgSpec,
		W:    float64(width), H: float64(height),
	})
}

// indexBubbleNodes recursively indexes all BubbleNodes by their Path,
// separating directories and files. Returns two maps.
func indexBubbleNodes(
	node *bubbletree.BubbleNode,
) (dirs map[string]*bubbletree.BubbleNode, files map[string]*bubbletree.BubbleNode) {
	dirs = make(map[string]*bubbletree.BubbleNode)
	files = make(map[string]*bubbletree.BubbleNode)

	indexBubbleNodesWalk(node, dirs, files)

	return dirs, files
}

func indexBubbleNodesWalk(
	node *bubbletree.BubbleNode,
	dirs map[string]*bubbletree.BubbleNode,
	files map[string]*bubbletree.BubbleNode,
) {
	for i := range node.Children {
		child := &node.Children[i]
		if child.IsDirectory {
			dirs[child.Path] = child
			indexBubbleNodesWalk(child, dirs, files)
		} else {
			files[child.Path] = child
		}
	}
}

// bubbleDirEntry holds a directory node for sorted drawing.
type bubbleDirEntry struct {
	node *bubbletree.BubbleNode
	mv   canvas.MetricValue
}

// addBubbleDirDiscs collects directory nodes from the model tree (via path lookup),
// sorts them largest-first, and adds semi-transparent discs to the canvas.
func addBubbleDirDiscs(
	cv *canvas.Canvas,
	dirIndex map[string]*bubbletree.BubbleNode,
	root *model.Directory,
	inks bubbleInks,
) {
	entries := collectBubbleDirEntries(dirIndex, root)

	slices.SortFunc(entries, func(a, b bubbleDirEntry) int {
		return cmp.Compare(b.node.Radius, a.node.Radius)
	})

	dirFill := canvas.FixedInk(bubbleDefaultDirFill, canvas.WithOpacity(bubbleDirOpacity))

	dirSpec := &canvas.DiscSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        dirFill,
			Border:      inks.border,
			BorderWidth: bubbleBorderWidth,
		},
	}

	for _, e := range entries {
		cv.AddDisc(canvas.LayerStructure, canvas.Disc{
			Spec:   dirSpec,
			X:      e.node.X,
			Y:      e.node.Y,
			Radius: e.node.Radius,
			Border: e.mv,
		})
	}
}

// collectBubbleDirEntries recursively walks model.Directory to find
// all directories that have a corresponding BubbleNode.
func collectBubbleDirEntries(
	dirIndex map[string]*bubbletree.BubbleNode,
	dir *model.Directory,
) []bubbleDirEntry {
	var entries []bubbleDirEntry

	for _, d := range dir.Dirs {
		if bn, ok := dirIndex[d.Path]; ok && bn.Radius > 0 {
			entries = append(entries, bubbleDirEntry{node: bn})
			entries = append(entries, collectBubbleDirEntries(dirIndex, d)...)
		}
	}

	return entries
}

// addBubbleFileDiscs walks the model tree via path lookup
// and adds file discs to the canvas.
func addBubbleFileDiscs(
	cv *canvas.Canvas,
	fileIndex map[string]*bubbletree.BubbleNode,
	root *model.Directory,
	inks bubbleInks,
) {
	addBubbleFileDiscsWalk(cv, fileIndex, root, inks)
}

func addBubbleFileDiscsWalk(
	cv *canvas.Canvas,
	fileIndex map[string]*bubbletree.BubbleNode,
	dir *model.Directory,
	inks bubbleInks,
) {
	fileSpec := &canvas.DiscSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        inks.fill,
			Border:      inks.border,
			BorderWidth: bubbleBorderWidth,
		},
	}

	for _, f := range dir.Files {
		bn, ok := fileIndex[f.Path]
		if !ok || bn.Radius <= 0 {
			continue
		}

		fillMV := metricValueForFile(f, inks.fill)
		borderMV := metricValueForFile(f, inks.border)

		cv.AddDisc(canvas.LayerContent, canvas.Disc{
			Spec:   fileSpec,
			X:      bn.X,
			Y:      bn.Y,
			Radius: bn.Radius,
			Fill:   fillMV,
			Border: borderMV,
		})
	}

	for _, d := range dir.Dirs {
		addBubbleFileDiscsWalk(cv, fileIndex, d, inks)
	}
}

// addBubbleLabels recursively adds labels for all nodes with ShowLabel set.
// Directory labels use arc text; file labels use centred text.
func addBubbleLabels(cv *canvas.Canvas, node bubbletree.BubbleNode) {
	if node.ShowLabel && node.Label != "" {
		if node.IsDirectory {
			addBubbleDirLabel(cv, node)
		} else {
			addBubbleFileLabel(cv, node)
		}
	}

	for _, child := range node.Children {
		addBubbleLabels(cv, child)
	}
}

// addBubbleDirLabel adds an arc text label curved along the top of a directory circle.
func addBubbleDirLabel(cv *canvas.Canvas, node bubbletree.BubbleNode) {
	fontSize := bubbleArcFontSize(node.Label, node.Radius)
	if fontSize == 0 {
		return
	}

	arcSpec := &canvas.ArcTextSpec{
		Ink:      canvas.FixedInk(bubbleLabelColour),
		FontSize: fontSize,
	}

	cv.AddArcText(canvas.LayerOverlay, canvas.ArcText{
		Spec:   arcSpec,
		X:      node.X,
		Y:      node.Y,
		Radius: node.Radius,
		Text:   node.Label,
	})
}

// addBubbleFileLabel adds a centred text label on a file circle.
func addBubbleFileLabel(cv *canvas.Canvas, node bubbletree.BubbleNode) {
	labelSpec := &canvas.TextSpec{
		Ink:      canvas.FixedInk(bubbleLabelColour),
		FontSize: 0,
		Anchor:   canvas.AnchorMiddle,
	}

	cv.AddText(canvas.LayerOverlay, canvas.Text{
		Spec:    labelSpec,
		X:       node.X,
		Y:       node.Y,
		Content: node.Label,
	})
}

// bubbleArcFontSize computes the font size for a label to fit within
// bubbleMaxArcFraction of the circle arc. Returns 0 if the label cannot fit
// at the minimum readable font size.
func bubbleArcFontSize(label string, radius float64) float64 {
	charCount := float64(utf8.RuneCountInString(label))
	if charCount == 0 {
		return 0
	}

	arcR := radius - bubbleArcLabelInset
	if arcR <= 0 {
		return 0
	}

	maxArcLen := arcR * bubbleMaxArcFraction
	// Each character is approximately 0.6 × fontSize wide.
	maxSize := maxArcLen / (charCount * 0.6)
	fontSize := min(bubbleDefaultFontSize, maxSize)

	if fontSize < bubbleMinArcFontSize {
		return 0
	}

	return fontSize
}
```

- [ ] **Step 2: Verify the file compiles in isolation**

Note: This will compile fully only after Task 2 (BubbleNode stripped) and Task 4 (cmd rewired) are complete. At this point, verify it has no syntax errors:

```bash
cd /home/bevan/github/code-visualizer && go vet ./cmd/codeviz/...
```

---

## Task 4: Rewire bubbletree_cmd.go to use Canvas

Replace the old render pipeline with the new Canvas pipeline. This involves:
1. Replacing `render.FormatFromPath` with `canvas.FormatFromPath`
2. Replacing `applyColoursAndRender` with Canvas-based rendering
3. Deleting all colour application functions (~240 lines)
4. Removing the `render` import

**Files:**
- Modify: `cmd/codeviz/bubbletree_cmd.go`

- [ ] **Step 1: Replace the `render` import with `canvas`**

In the imports section, replace:

```go
	"github.com/theunrepentantgeek/code-visualizer/internal/render"
```

with:

```go
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
```

- [ ] **Step 2: Replace `renderAndLog` method**

Replace the entire `renderAndLog` method (lines 164-197) with:

```go
func (c *BubbletreeCmd) renderAndLog(
	root *model.Directory,
	cfg *config.Bubbletree,
	width, height int,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
) error {
	size := metric.Name(ptrString(cfg.Size))
	files, dirs := countAll(root)

	slog.Info("Rendering image", "output", c.Output, "width", width, "height", height)

	borderMetric, borderPaletteName := c.resolveBorderMetricAndPalette(cfg)

	labels := c.resolveLabels(cfg)
	nodes := bubbletree.Layout(root, width, height, size, labels)
	inks := buildBubbleInks(root, fillMetric, fillPaletteName, borderMetric, borderPaletteName)
	cv := renderBubbleToCanvas(&nodes, root, width, height, inks)

	if err := cv.Render(c.Output); err != nil {
		return eris.Wrap(err, "render failed")
	}

	slog.Info("Rendered bubble tree",
		"files", files,
		"directories", dirs,
		"output", c.Output,
		"width", width,
		"height", height,
		"size_metric", string(size),
		"fill_metric", string(fillMetric),
		"fill_palette", string(fillPaletteName),
		"border_metric", string(borderMetric),
		"border_palette", string(borderPaletteName),
	)

	return nil
}
```

- [ ] **Step 3: Add `resolveBorderMetricAndPalette` helper**

Add after `resolveFillPalette` (around line 337):

```go
func (*BubbletreeCmd) resolveBorderMetricAndPalette(
	cfg *config.Bubbletree,
) (metric.Name, palette.PaletteName) {
	border := specMetric(cfg.Border)
	if border == "" {
		return "", ""
	}

	borderPaletteName := specPalette(cfg.Border)
	if borderPaletteName == "" {
		if p, ok := provider.Get(border); ok {
			borderPaletteName = p.DefaultPalette()
		} else {
			borderPaletteName = palette.Neutral
		}
	}

	return border, borderPaletteName
}
```

- [ ] **Step 4: Replace `render.FormatFromPath` with `canvas.FormatFromPath` in `validatePaths`**

In the `validatePaths` method, replace:

```go
	if _, err := render.FormatFromPath(c.Output); err != nil {
```

with:

```go
	if _, err := canvas.FormatFromPath(c.Output); err != nil {
```

- [ ] **Step 5: Delete old colour functions**

Delete the following functions entirely from `bubbletree_cmd.go`:

- `applyColoursAndRender` (lines 199-227)
- `applyBubbleFillColoursTop` (lines 371-395)
- `applyBorderColours` (lines 397-436)
- `indexBubbleNodesByPath` (lines 440-454)
- `applyBubbleFillColours` (lines 458-470)
- `applyBubbleFillColoursWalk` (lines 472-493)
- `applyCategoricalBubbleFillColours` (lines 497-508)
- `applyCategoricalBubbleFillColoursWalk` (lines 510-530)
- `applyBubbleBorderColours` (lines 534-546)
- `applyBubbleBorderColoursWalk` (lines 548-570)
- `applyCategoricalBubbleBorderColours` (lines 574-585)
- `applyCategoricalBubbleBorderColoursWalk` (lines 587-608)

After deletion, also remove any now-unused imports (`"github.com/theunrepentantgeek/code-visualizer/internal/palette"` may still be needed by `resolveFillPalette`; keep `"github.com/theunrepentantgeek/code-visualizer/internal/metric"` for `metric.Name`).

- [ ] **Step 6: Check for stale `//nolint:dupl` directives**

After removal, look at remaining `//nolint:dupl` comments:
- Line 96: `//nolint:dupl // parallel Run methods...` — keep this, it's still valid (Run() pattern shared with TreemapCmd/RadialTreeCmd/SpiralCmd)
- Line 247: `//nolint:dupl // mirrors TreemapCmd.validatePaths by design` — keep this, still valid

- [ ] **Step 7: Verify compilation**

```bash
cd /home/bevan/github/code-visualizer && go build ./cmd/codeviz/...
```

Expected: PASS

- [ ] **Step 8: Commit Tasks 2-4 together**

```bash
git add internal/bubbletree/node.go cmd/codeviz/bubble_canvas.go cmd/codeviz/bubbletree_cmd.go
git commit -m "feat: migrate bubbletree to Canvas API

Strip FillColour/BorderColour from BubbleNode.
Add bubble_canvas.go bridge with path-based colour lookup,
semi-transparent directory fills (WithOpacity), and arc text labels.
Rewire bubbletree_cmd.go to use Canvas pipeline.
Delete ~240 lines of old colour application functions."
```

---

## Task 5: Delete old render code

Remove the old bubbletree render files. After this, no code in the repo should reference these files.

**Files:**
- Delete: `internal/render/bubbletree.go`
- Delete: `internal/render/svg_bubble.go`
- Delete: `internal/render/bubbletree_test.go`
- Delete: `internal/render/bubble_font.go`
- Delete: `internal/render/testdata/bubble-tree.png` (golden file for deleted test)

- [ ] **Step 1: Delete the files**

```bash
cd /home/bevan/github/code-visualizer
rm internal/render/bubbletree.go
rm internal/render/svg_bubble.go
rm internal/render/bubbletree_test.go
rm internal/render/bubble_font.go
rm -f internal/render/testdata/bubble-tree.png
```

- [ ] **Step 2: Check for orphaned references**

Verify no remaining code references anything from the deleted files:

```bash
cd /home/bevan/github/code-visualizer && grep -rn 'RenderBubble\|renderBubbleSVG\|renderBubbleImage\|collectBubbleDirs\|collectBubbleFiles\|resolveDirFill\|resolveFileFill\|resolveBorder\|drawBubbleDirs\|drawBubbleFiles\|drawBubbleLabels\|drawBubbleDirLabel\|drawArcGlyph\|writeSVGBubble\|writeSVGArcDefs\|collectLabelledDirs\|computeArcFontSize\|computeGlyphPositions\|loadBubbleFontFace\|clampFontToArc\|measureStringWidth\|glyphPos\|charAdvance\|fixedAccum\|sampleBubbleTree\|bubbleDefaultFontSize\|bubbleMinFontSize\|bubbleMaxArcFraction\|bubbleLabelInset\|bubbleDirAlpha\|placeGlyphs\|collectAdvances\|sumAdvances' --include='*.go' .
```

Expected: No matches (or only matches inside comments/docs which are fine).

Note: If `svg_helpers.go` has functions only used by deleted bubble code (like `writeSVGText`, `colourToHex`), check that they are still used elsewhere. Currently `writeSVGText` and `colourToHex` are also used by the legend SVG code — so they should stay.

- [ ] **Step 3: Verify compilation**

```bash
cd /home/bevan/github/code-visualizer && go build ./...
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add -A internal/render/
git commit -m "refactor: delete old bubbletree render pipeline

Remove bubbletree.go, svg_bubble.go, bubble_font.go,
bubbletree_test.go, and the golden file.
All bubbletree rendering now goes through the Canvas API."
```

---

## Task 6: Add Canvas-based bubbletree tests

Write tests for the new bridge. Follow the same pattern as `radial_canvas_test.go`.

**Files:**
- Create: `cmd/codeviz/bubble_canvas_test.go`

- [ ] **Step 1: Create test file**

Create `cmd/codeviz/bubble_canvas_test.go`:

```go
package main

import (
	"bytes"
	"encoding/xml"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/bubbletree"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

func bubbleTestFile(name, ext string, size int64) *model.File {
	f := &model.File{Name: name, Extension: ext}
	f.SetQuantity(filesystem.FileSize, size)
	f.SetClassification(filesystem.FileType, ext)

	return f
}

func bubbleTestRoot() *model.Directory {
	return &model.Directory{
		Name: "project",
		Path: "project",
		Files: []*model.File{
			bubbleTestFile("readme.md", "md", 50),
		},
		Dirs: []*model.Directory{
			{
				Name: "src",
				Path: "project/src",
				Files: []*model.File{
					bubbleTestFile("main.go", "go", 200),
					bubbleTestFile("util.go", "go", 80),
				},
			},
		},
	}
}

func TestBuildBubbleInks_NumericFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			bubbleTestFile("a.go", "go", 100),
			bubbleTestFile("b.go", "go", 200),
		},
	}

	inks := buildBubbleInks(
		root, filesystem.FileSize, palette.Temperature, "", "",
	)

	g.Expect(inks.fill.Info().Kind).To(Equal(canvas.InkNumeric))
	g.Expect(inks.border.Info().Kind).To(Equal(canvas.InkFixed))
}

func TestBuildBubbleInks_CategoricalFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			bubbleTestFile("a.go", "go", 100),
			bubbleTestFile("b.rs", "rs", 200),
		},
	}

	inks := buildBubbleInks(
		root, filesystem.FileType, palette.Categorization, "", "",
	)

	g.Expect(inks.fill.Info().Kind).To(Equal(canvas.InkCategorical))
}

func TestBuildBubbleInks_WithBorder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			bubbleTestFile("a.go", "go", 100),
			bubbleTestFile("b.rs", "rs", 200),
		},
	}

	inks := buildBubbleInks(
		root,
		filesystem.FileSize, palette.Temperature,
		filesystem.FileSize, palette.Temperature,
	)

	g.Expect(inks.fill.Info().Kind).To(Equal(canvas.InkNumeric))
	g.Expect(inks.border.Info().Kind).NotTo(Equal(canvas.InkFixed))
}

func TestRenderBubbleToCanvas_PNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := bubbleTestRoot()
	nodes := bubbletree.Layout(root, 800, 600, filesystem.FileSize, bubbletree.LabelAll)
	inks := buildBubbleInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := renderBubbleToCanvas(&nodes, root, 800, 600, inks)

	out := filepath.Join(t.TempDir(), "bubble.png")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("png"))
}

func TestRenderBubbleToCanvas_SVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := bubbleTestRoot()
	nodes := bubbletree.Layout(root, 400, 300, filesystem.FileSize, bubbletree.LabelNone)
	inks := buildBubbleInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := renderBubbleToCanvas(&nodes, root, 400, 300, inks)

	out := filepath.Join(t.TempDir(), "bubble.svg")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	decoder := xml.NewDecoder(bytes.NewReader(data))

	var rootElement string

	for {
		tok, xmlErr := decoder.Token()
		if xmlErr != nil {
			break
		}

		if se, ok := tok.(xml.StartElement); ok {
			rootElement = se.Name.Local

			break
		}
	}

	g.Expect(rootElement).To(Equal("svg"))
}

func TestRenderBubbleToCanvas_EmptyDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Name: "empty"}
	nodes := bubbletree.Layout(root, 400, 300, filesystem.FileSize, bubbletree.LabelNone)
	inks := buildBubbleInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := renderBubbleToCanvas(&nodes, root, 400, 300, inks)

	out := filepath.Join(t.TempDir(), "empty.png")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())

	if info != nil {
		g.Expect(info.Size()).To(BeNumerically(">", 0))
	}
}

func TestBubbleArcFontSize_FitsLabel(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	size := bubbleArcFontSize("src", 200)
	g.Expect(size).To(BeNumerically(">", 0))
	g.Expect(size).To(BeNumerically("<=", bubbleDefaultFontSize))
}

func TestBubbleArcFontSize_TooSmallRadius(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	size := bubbleArcFontSize("very-long-directory-name-that-cannot-fit", 20)
	g.Expect(size).To(Equal(0.0))
}

func TestBubbleArcFontSize_EmptyLabel(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	size := bubbleArcFontSize("", 200)
	g.Expect(size).To(Equal(0.0))
}

func TestBubbleArcFontSize_ZeroRadius(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	size := bubbleArcFontSize("src", 0)
	g.Expect(size).To(Equal(0.0))
}
```

- [ ] **Step 2: Run tests**

```bash
cd /home/bevan/github/code-visualizer && go test ./cmd/codeviz/... -v -run TestBubble -count=1
```

Expected: All tests PASS

- [ ] **Step 3: Commit**

```bash
git add cmd/codeviz/bubble_canvas_test.go
git commit -m "test: add Canvas-based bubbletree tests

Cover ink construction, PNG/SVG rendering, empty directory handling,
and arc font size computation."
```

---

## Task 7: Run CI and fix lint issues

**Files:** Any files touched in Tasks 1-6

- [ ] **Step 1: Run full CI**

```bash
cd /home/bevan/github/code-visualizer && task ci
```

Expected: All green (build + tests + 76 linters).

- [ ] **Step 2: Fix any lint issues**

Common issues to watch for:
- **Orphaned helpers:** If any function from deleted render files was used by other render code (e.g., `collectBubbleDirs`, `collectBubbleFiles`, `resolveBorder`, `resolveFileFill`, `resolveDirFill` are used by both `bubbletree.go` and `svg_bubble.go` — since we're deleting both, they should all go).
- **Stale `//nolint:dupl`:** The `Run()` and `validatePaths()` methods still have duplicate partners in the other cmd files, so their `//nolint:dupl` directives should remain valid.
- **gci import order:** Run `task fmt` if imports are out of order.
- **funlen:** The new `renderAndLog` should be well under 65 lines. The bridge functions should all be small.
- **revive max-public-structs:** Check `bubble_canvas.go` doesn't exceed 5 public structs per file (it has 0 public structs — `bubbleInks`, `bubbleDirEntry` are unexported).
- **wsl_v5 blank lines:** Ensure blank lines after control flow blocks.

- [ ] **Step 3: Commit fixes if any**

```bash
git add -A
git commit -m "fix: address lint issues from bubbletree migration"
```

---

## Task 8: Push and create PR

- [ ] **Step 1: Verify only expected files are in the diff**

```bash
cd /home/bevan/github/code-visualizer && git diff --name-only main..HEAD
```

Expected files (and no others):
```
cmd/codeviz/bubble_canvas.go
cmd/codeviz/bubble_canvas_test.go
cmd/codeviz/bubbletree_cmd.go
internal/bubbletree/node.go
internal/canvas/canvas.go
internal/canvas/canvas_test.go
internal/canvas/shape.go
internal/canvas/text_spec.go
internal/render/bubble_font.go
internal/render/bubbletree.go
internal/render/bubbletree_test.go
internal/render/svg_bubble.go
internal/render/testdata/bubble-tree.png
```

Ensure NO `.agents/`, `.squad/`, `docs/superpowers/` or plan files appear.

- [ ] **Step 2: Push and create PR**

```bash
cd /home/bevan/github/code-visualizer
git push -u origin feature/canvas-bubble
gh pr create --title "feat: migrate bubbletree to Canvas API" \
  --body "## Summary

Migrates the bubbletree visualization from the old internal/render pipeline to the Canvas API.

### Changes
- **Canvas API**: Add ArcText shape type for curved text labels
- **BubbleNode**: Strip FillColour/BorderColour fields (colours handled by Canvas Ink system)
- **Bridge**: New bubble_canvas.go with path-based colour lookup, semi-transparent dir fills, arc text labels
- **Command**: Rewire bubbletree_cmd.go to use Canvas pipeline (~240 lines of old colour code deleted)
- **Cleanup**: Delete old render files (bubbletree.go, svg_bubble.go, bubble_font.go, bubbletree_test.go)
- **Tests**: New Canvas-based tests for ink construction, PNG/SVG rendering, empty dirs, arc font sizing

Part of Stage 2 Canvas migration (4/4 viz types complete)." \
  --base main
```
