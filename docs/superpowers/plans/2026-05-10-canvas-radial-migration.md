# Canvas Radial Tree Migration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the radial tree visualization from the old `internal/render/` pipeline to the new Canvas abstraction.

**Architecture:** The radial tree uses a tree of `RadialNode` values (each with children) plus a parallel `model.Directory` tree for metric data. The migration strips colour fields from `RadialNode` (making it geometry-only), creates a Canvas bridge file that walks both trees in parallel to build Canvas shapes with metric-driven inks, rewires the command to call the bridge, and deletes the old render code. This follows the exact same pattern established by the treemap and spiral migrations.

**Tech Stack:** Go 1.26+, Canvas API (`internal/canvas`), radialtree layout (`internal/radialtree`), Kong CLI, Gomega test assertions.

---

## File Structure

| File | Action | Responsibility |
|------|--------|----------------|
| `internal/radialtree/node.go` | Modify | Strip `FillColour`, `BorderColour` fields (geometry-only) |
| `cmd/codeviz/radial_canvas.go` | Create | Canvas bridge: inks, shapes, tree walk, labels, edges |
| `cmd/codeviz/radial_canvas_test.go` | Create | Tests for bridge functions |
| `cmd/codeviz/radialtree_cmd.go` | Modify | Rewire rendering to Canvas, delete colour functions |
| `internal/render/radialtree.go` | Delete | Old PNG rendering |
| `internal/render/svg_radial.go` | Delete | Old SVG rendering |
| `internal/render/radialtree_test.go` | Delete | Old render tests |

---

### Task 1: Strip colour fields from RadialNode

**Files:**
- Modify: `internal/radialtree/node.go`

- [ ] **Step 1: Remove FillColour and BorderColour from RadialNode**

Edit `internal/radialtree/node.go` to remove the colour fields and the `image/color` import:

```go
package radialtree

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/viz"
)

// LabelMode is an alias for [viz.LabelMode].
type LabelMode = viz.LabelMode

const (
	LabelAll         = viz.LabelAll
	LabelFoldersOnly = viz.LabelFoldersOnly
	LabelNone        = viz.LabelNone
)

// RadialNode is a positioned visual element in the rendered radial tree.
// X and Y are pixel offsets from the canvas centre (canvas centre = origin).
type RadialNode struct {
	X, Y        float64 // pixel position relative to canvas centre
	DiscRadius  float64 // radius of the node disc in pixels
	Angle       float64 // angle in radians (0 = right/east, π/2 = down, in screen coordinates)
	Label       string  // display name
	ShowLabel   bool    // whether to render the label for this node
	IsDirectory bool    // true for directory nodes, false for file nodes
	Children    []RadialNode
}
```

- [ ] **Step 2: Verify the build breaks**

Run: `cd /home/bevan/github/code-visualizer && go build ./...`

Expected: Compilation errors in `radialtree_cmd.go` (references to `FillColour`, `BorderColour`) and in `internal/render/radialtree.go` and `internal/render/svg_radial.go`. This confirms we found all the consumers.

- [ ] **Step 3: Commit**

```bash
git add internal/radialtree/node.go
git commit -m "refactor(radial): strip colour fields from RadialNode

RadialNode is now geometry-only. FillColour and BorderColour are
removed — colour resolution moves to the Canvas bridge layer.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 2: Create Canvas bridge

**Files:**
- Create: `cmd/codeviz/radial_canvas.go`

This is the core of the migration. The radial tree has THREE visual element types: edges (lines connecting parent→child), discs (filled circles per node), and labels (rotated text). It also has a background. The tree structure means we walk `RadialNode` and `model.Directory` in parallel — files first, then subdirectories — matching the ordering invariant.

Key difference from treemap: radial tree uses a recursive tree walk (not flat list like spiral). Key difference from spiral: radial uses `model.File` metric data (like treemap), not pre-aggregated bucket data.

- [ ] **Step 1: Create the bridge file with ink construction**

Create `cmd/codeviz/radial_canvas.go`:

```go
package main

import (
	"image/color"
	"math"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/radialtree"
)

var (
	radialDefaultFileFill = color.RGBA{R: 0xCC, G: 0xCC, B: 0xCC, A: 0xFF}
	radialDefaultDirFill  = color.RGBA{R: 0x44, G: 0x44, B: 0x44, A: 0xFF}
	radialDefaultBorder   = color.RGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xFF}
	radialEdgeColour      = color.RGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xFF}
	radialLabelColour     = color.RGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xFF}
	radialBgColour        = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
)

const (
	radialEdgeWidth = 0.5
	radialLabelGap  = 4.0
)

// radialInks holds the Ink instances for a radial tree render pass.
type radialInks struct {
	fill   canvas.Ink
	border canvas.Ink
}

// buildRadialInks creates fill and border inks from metric configuration.
// Uses the same buildMetricInk helper as the treemap bridge since both
// visualizations derive colours from the model.Directory tree.
func buildRadialInks(
	root *model.Directory,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
	borderMetric metric.Name,
	borderPaletteName palette.PaletteName,
) radialInks {
	inks := radialInks{
		fill:   canvas.FixedInk(radialDefaultFileFill),
		border: canvas.FixedInk(radialDefaultBorder),
	}

	inks.fill = buildMetricInk(root, fillMetric, fillPaletteName, radialDefaultFileFill)
	if borderMetric != "" {
		inks.border = buildMetricInk(root, borderMetric, borderPaletteName, radialDefaultBorder)
	}

	return inks
}

// renderRadialToCanvas walks the layout and model trees, adding shapes
// to the canvas. Returns the populated canvas.
func renderRadialToCanvas(
	nodes *radialtree.RadialNode,
	root *model.Directory,
	canvasSize int,
	inks radialInks,
) *canvas.Canvas {
	cv := canvas.NewCanvas(canvasSize, canvasSize)

	cx := float64(canvasSize) / 2.0
	cy := float64(canvasSize) / 2.0

	addRadialBackground(cv, canvasSize)
	addRadialEdges(cv, *nodes, cx, cy)
	addRadialDiscs(cv, nodes, root, cx, cy, inks)
	addRadialLabels(cv, *nodes, cx, cy, inks)

	return cv
}

// addRadialBackground adds a white background rectangle.
func addRadialBackground(cv *canvas.Canvas, canvasSize int) {
	bgSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        canvas.FixedInk(radialBgColour),
			Border:      canvas.FixedInk(radialBgColour),
			BorderWidth: 0,
		},
	}

	cv.AddRectangle(canvas.LayerBackground, canvas.Rectangle{
		Spec: bgSpec,
		W:    float64(canvasSize), H: float64(canvasSize),
	})
}

// addRadialEdges recursively adds edge lines from each node to its children.
func addRadialEdges(cv *canvas.Canvas, node radialtree.RadialNode, cx, cy float64) {
	px := cx + node.X
	py := cy + node.Y

	edgeSpec := &canvas.LineSpec{
		Stroke:      canvas.FixedInk(radialEdgeColour),
		StrokeWidth: radialEdgeWidth,
	}

	for _, child := range node.Children {
		chx := cx + child.X
		chy := cy + child.Y

		cv.AddLine(canvas.LayerStructure, canvas.Line{
			Spec: edgeSpec,
			X1:   px, Y1: py,
			X2: chx, Y2: chy,
		})

		addRadialEdges(cv, child, cx, cy)
	}
}

// radialDiscEntry holds a node and its screen position for deferred drawing.
type radialDiscEntry struct {
	node     radialtree.RadialNode
	file     *model.File
	sx, sy   float64
	isDir    bool
}

// collectRadialDiscs recursively gathers all nodes with a positive DiscRadius,
// along with their corresponding model.File (nil for directories).
// INVARIANT: node.Children are ordered files-first, then subdirectories.
func collectRadialDiscs(
	node *radialtree.RadialNode,
	dir *model.Directory,
	cx, cy float64,
) []radialDiscEntry {
	var entries []radialDiscEntry

	if node.DiscRadius > 0 {
		entries = append(entries, radialDiscEntry{
			node:  *node,
			sx:    cx + node.X,
			sy:    cy + node.Y,
			isDir: node.IsDirectory,
		})
	}

	fileIdx := 0
	dirIdx := 0

	for i := range node.Children {
		child := &node.Children[i]
		if child.IsDirectory && dirIdx < len(dir.Dirs) {
			entries = append(entries, collectRadialDiscs(child, dir.Dirs[dirIdx], cx, cy)...)
			dirIdx++
		} else if !child.IsDirectory && fileIdx < len(dir.Files) {
			childEntries := collectRadialDiscsLeaf(child, dir.Files[fileIdx], cx, cy)
			entries = append(entries, childEntries...)
			fileIdx++
		}
	}

	return entries
}

// collectRadialDiscsLeaf collects a single file node (leaf).
func collectRadialDiscsLeaf(
	node *radialtree.RadialNode,
	file *model.File,
	cx, cy float64,
) []radialDiscEntry {
	if node.DiscRadius <= 0 {
		return nil
	}

	return []radialDiscEntry{{
		node: *node,
		file: file,
		sx:   cx + node.X,
		sy:   cy + node.Y,
	}}
}

// addRadialDiscs collects all discs, sorts them largest-first so smaller
// nodes are never obscured, then adds them to the canvas.
func addRadialDiscs(
	cv *canvas.Canvas,
	nodes *radialtree.RadialNode,
	root *model.Directory,
	cx, cy float64,
	inks radialInks,
) {
	entries := collectRadialDiscs(nodes, root, cx, cy)

	slices.SortFunc(entries, func(a, b radialDiscEntry) int {
		return cmp.Compare(b.node.DiscRadius, a.node.DiscRadius)
	})

	for _, e := range entries {
		fillMV := radialMetricValue(e.file, e.isDir, inks.fill)
		borderMV := radialMetricValue(e.file, e.isDir, inks.border)

		discSpec := &canvas.DiscSpec{
			ShapeStyle: canvas.ShapeStyle{
				Fill:        radialFillInk(e.isDir, inks),
				Border:      inks.border,
				BorderWidth: 1.0,
			},
		}

		cv.AddDisc(canvas.LayerContent, canvas.Disc{
			Spec:   discSpec,
			X:      e.sx,
			Y:      e.sy,
			Radius: e.node.DiscRadius,
			Angle:  e.node.Angle,
			Fill:   fillMV,
			Border: borderMV,
		})
	}
}

// radialFillInk returns the fill ink, using the directory default
// for directory nodes and the metric-driven ink for file nodes.
func radialFillInk(isDir bool, inks radialInks) canvas.Ink {
	if isDir {
		return canvas.FixedInk(radialDefaultDirFill)
	}

	return inks.fill
}

// radialMetricValue builds a MetricValue from a file's data for the given ink.
// For directory nodes (file == nil), returns an empty MetricValue.
func radialMetricValue(file *model.File, isDir bool, ink canvas.Ink) canvas.MetricValue {
	if isDir || file == nil {
		return canvas.MetricValue{}
	}

	return metricValueForFile(file, ink)
}

// addRadialLabels recursively adds text labels for nodes with ShowLabel set.
func addRadialLabels(
	cv *canvas.Canvas,
	node radialtree.RadialNode,
	cx, cy float64,
	inks radialInks,
) {
	if node.ShowLabel && node.Label != "" {
		dist := math.Sqrt(node.X*node.X + node.Y*node.Y)

		if dist == 0 {
			addRadialRootLabel(cv, node, cx, cy, inks)
		} else {
			addRadialExternalLabel(cv, node, cx, cy)
		}
	}

	for _, child := range node.Children {
		addRadialLabels(cv, child, cx, cy, inks)
	}
}

// addRadialRootLabel adds a centred label on the root disc, using a
// contrasting text colour based on the effective fill.
func addRadialRootLabel(
	cv *canvas.Canvas,
	node radialtree.RadialNode,
	cx, cy float64,
	inks radialInks,
) {
	fill := radialEffectiveFill(node, inks)
	labelColour := canvas.TextColourFor(fill)

	labelSpec := &canvas.TextSpec{
		Ink:    canvas.FixedInk(labelColour),
		Anchor: canvas.AnchorMiddle,
	}

	cv.AddText(canvas.LayerOverlay, canvas.Text{
		Spec:    labelSpec,
		X:       cx + node.X,
		Y:       cy + node.Y,
		Content: node.Label,
	})
}

// addRadialExternalLabel adds a radially-oriented label outside the disc.
func addRadialExternalLabel(
	cv *canvas.Canvas,
	node radialtree.RadialNode,
	cx, cy float64,
) {
	dist := math.Sqrt(node.X*node.X + node.Y*node.Y)
	labelRadius := dist + node.DiscRadius + radialLabelGap
	lx := cx + labelRadius*math.Cos(node.Angle)
	ly := cy + labelRadius*math.Sin(node.Angle)

	angle := math.Mod(node.Angle, 2*math.Pi)
	if angle < 0 {
		angle += 2 * math.Pi
	}

	var anchor canvas.TextAnchor
	var rotation float64

	if angle <= math.Pi/2 || angle > 3*math.Pi/2 {
		anchor = canvas.AnchorStart
		rotation = node.Angle
	} else {
		anchor = canvas.AnchorEnd
		rotation = node.Angle + math.Pi
	}

	labelSpec := &canvas.TextSpec{
		Ink:      canvas.FixedInk(radialLabelColour),
		Anchor:   anchor,
		Rotation: rotation,
	}

	cv.AddText(canvas.LayerOverlay, canvas.Text{
		Spec:    labelSpec,
		X:       lx,
		Y:       ly,
		Content: node.Label,
	})
}

// radialEffectiveFill returns the fill colour for a node, resolving defaults.
// Used for computing label contrast colour on the root node.
func radialEffectiveFill(node radialtree.RadialNode, inks radialInks) color.RGBA {
	if node.IsDirectory {
		return radialDefaultDirFill
	}

	return inks.fill.Dip(canvas.MetricValue{})
}
```

Note: This file reuses `buildMetricInk` and `metricValueForFile` from `treemap_canvas.go` — they operate on the shared `model.Directory` tree and are already available in the same package.

- [ ] **Step 2: Add missing imports**

The file uses `cmp` and `slices` from the standard library. Make sure these are in the import block:

```go
import (
	"cmp"
	"image/color"
	"math"
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/radialtree"
)
```

Wait — `provider` is not used directly in this file (it's used via `buildMetricInk` in `treemap_canvas.go`). Remove the `provider` import. Also `metric` is not directly used. Check all imports carefully. The actual imports needed are:

```go
import (
	"cmp"
	"image/color"
	"math"
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/radialtree"
)
```

Actually, `buildRadialInks` takes `metric.Name` and `palette.PaletteName` parameters, so we need those:

```go
import (
	"cmp"
	"image/color"
	"math"
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/radialtree"
)
```

- [ ] **Step 3: Commit**

```bash
git add cmd/codeviz/radial_canvas.go
git commit -m "feat(radial): add Canvas bridge for radial tree

Create radial_canvas.go with ink construction, tree walk, and shape
generation for the Canvas pipeline. Uses four layers: background
(rect), structure (edges), content (discs sorted by size), and
overlay (radially-oriented labels).

Reuses buildMetricInk and metricValueForFile from the treemap bridge
since both viz types derive colours from the model.Directory tree.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 3: Rewire command to Canvas pipeline

**Files:**
- Modify: `cmd/codeviz/radialtree_cmd.go`

This task replaces `applyColoursAndRender` (and its `render.RenderRadial` call) with the Canvas pipeline, switches `render.FormatFromPath` to `canvas.FormatFromPath`, and deletes all six colour application functions.

- [ ] **Step 1: Replace the rendering pipeline**

Replace the `applyColoursAndRender` method with a new `renderAndLog` that uses Canvas:

```go
func (c *RadialCmd) renderAndLog(
	root *model.Directory,
	cfg *config.Radial,
	files, dirs, canvasSize int,
	fillMetric metric.Name,
	fillPaletteName palette.PaletteName,
) error {
	discSize := metric.Name(ptrString(cfg.DiscSize))
	labels := c.resolveLabels(cfg)
	nodes := radialtree.Layout(root, canvasSize, discSize, labels)

	borderMetric, borderPaletteName := c.resolveBorderMetricAndPalette(cfg)

	inks := buildRadialInks(root, fillMetric, fillPaletteName, borderMetric, borderPaletteName)

	slog.Info("Rendering image", "output", c.Output, "canvas_size", canvasSize)

	cv := renderRadialToCanvas(&nodes, root, canvasSize, inks)

	if err := cv.Render(c.Output); err != nil {
		return eris.Wrap(err, "render failed")
	}

	legendStr := ptrString(cfg.Legend)
	if legendStr != "" && legendStr != "none" {
		slog.Warn("Legend rendering not yet supported in Canvas pipeline; legend will be omitted")
	}

	slog.Info("Rendered radial tree",
		"files", files,
		"directories", dirs,
		"output", c.Output,
		"canvas_size", canvasSize,
		"disc_metric", string(discSize),
		"fill_metric", string(fillMetric),
		"fill_palette", string(fillPaletteName),
		"border_metric", string(borderMetric),
		"border_palette", string(borderPaletteName),
	)

	return nil
}
```

- [ ] **Step 2: Add resolveBorderMetricAndPalette method**

Add a `resolveBorderMetricAndPalette` method that extracts border metric and palette from config (same pattern as spiral):

```go
func (*RadialCmd) resolveBorderMetricAndPalette(cfg *config.Radial) (metric.Name, palette.PaletteName) {
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

- [ ] **Step 3: Replace render.FormatFromPath with canvas.FormatFromPath**

In `validatePaths()`, change:

```go
// OLD:
if _, err := render.FormatFromPath(c.Output); err != nil {
// NEW:
if _, err := canvas.FormatFromPath(c.Output); err != nil {
```

- [ ] **Step 4: Update import block**

Replace the import block. Remove `render` import, add `canvas` import:

```go
import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/export"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/radialtree"
	"github.com/theunrepentantgeek/code-visualizer/internal/scan"
)
```

- [ ] **Step 5: Delete the old colour application functions**

Delete these six functions from `radialtree_cmd.go`:
- `applyRadialFillColoursTop` (lines 364–388)
- `applyRadialFillColours` (lines 436–458)
- `applyCategoricalRadialFillColours` (lines 464–486)
- `applyRadialBorderColours` (lines 492–515)
- `applyCategoricalRadialBorderColours` (lines 521–544)
- `applyBorderColours` (lines 391–430)

Also delete the old `applyColoursAndRender` method (lines 193–220) and the old `renderAndLog` (lines 159–190), replacing them with the new `renderAndLog`.

- [ ] **Step 6: Update the Run method call**

The `Run` method currently calls:

```go
return c.renderAndLog(root, cfg, files, dirs, canvasSize, fillMetric, fillPaletteName)
```

This already matches the new `renderAndLog` signature, so no change is needed.

- [ ] **Step 7: Verify build succeeds**

Run: `cd /home/bevan/github/code-visualizer && go build ./...`

Expected: Clean build with no errors.

- [ ] **Step 8: Commit**

```bash
git add cmd/codeviz/radialtree_cmd.go
git commit -m "feat(radial): rewire command to Canvas pipeline

Replace applyColoursAndRender with Canvas-based renderAndLog.
Delete 6 colour application functions (~180 lines).
Switch render.FormatFromPath to canvas.FormatFromPath.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 4: Delete old render code

**Files:**
- Delete: `internal/render/radialtree.go`
- Delete: `internal/render/svg_radial.go`
- Delete: `internal/render/radialtree_test.go`

- [ ] **Step 1: Delete the files**

```bash
git rm internal/render/radialtree.go
git rm internal/render/svg_radial.go
git rm internal/render/radialtree_test.go
```

- [ ] **Step 2: Verify build succeeds**

Run: `cd /home/bevan/github/code-visualizer && go build ./...`

Expected: Clean build. No other files import `RenderRadial`.

- [ ] **Step 3: Commit**

```bash
git commit -m "refactor(render): delete old radial tree renderer

Remove radialtree.go (227 lines), svg_radial.go (194 lines), and
radialtree_test.go (223 lines). All radial rendering now goes
through the Canvas pipeline.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 5: Add Canvas-based tests

**Files:**
- Create: `cmd/codeviz/radial_canvas_test.go`

- [ ] **Step 1: Create the test file**

Create `cmd/codeviz/radial_canvas_test.go`:

```go
package main

import (
	"bytes"
	"encoding/xml"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/radialtree"
)

func makeRadialTestFile(name, ext string, size int64) *model.File {
	f := &model.File{Name: name, Extension: ext}
	f.SetQuantity(filesystem.FileSize, size)
	f.SetClassification(filesystem.FileType, ext)

	return f
}

func sampleRadialRoot() *model.Directory {
	return &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeRadialTestFile("small.go", "go", 100),
			makeRadialTestFile("medium.py", "py", 500),
			makeRadialTestFile("large.rs", "rs", 2000),
		},
	}
}

func TestBuildRadialInks_Numeric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := sampleRadialRoot()
	inks := buildRadialInks(root, filesystem.FileSize, palette.Temperature, "", "")

	g.Expect(inks.fill.Info().Kind).To(Equal(canvas.InkNumeric))
	g.Expect(inks.border.Info().Kind).To(Equal(canvas.InkFixed))
}

func TestBuildRadialInks_Categorical(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := sampleRadialRoot()
	inks := buildRadialInks(root, filesystem.FileType, palette.Categorization, "", "")

	g.Expect(inks.fill.Info().Kind).To(Equal(canvas.InkCategorical))
	g.Expect(inks.border.Info().Kind).To(Equal(canvas.InkFixed))
}

func TestBuildRadialInks_DefaultFallback(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := sampleRadialRoot()
	inks := buildRadialInks(root, "", "", "", "")

	g.Expect(inks.fill.Info().Kind).To(Equal(canvas.InkFixed))
	g.Expect(inks.border.Info().Kind).To(Equal(canvas.InkFixed))
}

func TestRenderRadialToCanvas_PNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := sampleRadialRoot()
	nodes := radialtree.Layout(root, 800, filesystem.FileSize, radialtree.LabelAll)
	inks := buildRadialInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := renderRadialToCanvas(&nodes, root, 800, inks)

	out := filepath.Join(t.TempDir(), "radial.png")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("png"))
}

func TestRenderRadialToCanvas_SVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := sampleRadialRoot()
	nodes := radialtree.Layout(root, 400, filesystem.FileSize, radialtree.LabelAll)
	inks := buildRadialInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := renderRadialToCanvas(&nodes, root, 400, inks)

	out := filepath.Join(t.TempDir(), "radial.svg")
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

func TestRenderRadialToCanvas_JPG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := sampleRadialRoot()
	nodes := radialtree.Layout(root, 400, filesystem.FileSize, radialtree.LabelAll)
	inks := buildRadialInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := renderRadialToCanvas(&nodes, root, 400, inks)

	out := filepath.Join(t.TempDir(), "radial.jpg")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("jpeg"))
}

func TestRenderRadialToCanvas_NestedDirs(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeRadialTestFile("root.go", "go", 100)},
		Dirs: []*model.Directory{
			{
				Name:  "sub",
				Files: []*model.File{makeRadialTestFile("child.go", "go", 500)},
			},
		},
	}

	nodes := radialtree.Layout(root, 800, filesystem.FileSize, radialtree.LabelAll)
	inks := buildRadialInks(root, filesystem.FileSize, palette.Temperature, "", "")
	cv := renderRadialToCanvas(&nodes, root, 800, inks)

	out := filepath.Join(t.TempDir(), "nested.png")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info.Size()).To(BeNumerically(">", 0))
}

func TestRenderRadialToCanvas_EmptyDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Name: "empty"}
	nodes := radialtree.Layout(root, 400, filesystem.FileSize, radialtree.LabelAll)
	inks := buildRadialInks(root, "", "", "", "")
	cv := renderRadialToCanvas(&nodes, root, 400, inks)

	out := filepath.Join(t.TempDir(), "empty.png")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(info.Size()).To(BeNumerically(">", 0))
}
```

- [ ] **Step 2: Run tests**

Run: `cd /home/bevan/github/code-visualizer && go test ./cmd/codeviz/ -run TestBuildRadialInks -v && go test ./cmd/codeviz/ -run TestRenderRadialToCanvas -v`

Expected: All 8 tests pass.

- [ ] **Step 3: Commit**

```bash
git add cmd/codeviz/radial_canvas_test.go
git commit -m "test(radial): add Canvas-based radial tree tests

8 tests covering ink construction (numeric, categorical, default),
end-to-end output (PNG, SVG, JPG), nested directories, and empty
directory edge case.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 6: Run CI

- [ ] **Step 1: Run full CI**

Run: `cd /home/bevan/github/code-visualizer && task ci`

Expected: Build passes, all tests pass (23+ packages), 0 lint issues.

- [ ] **Step 2: Fix any issues**

If lint or test failures occur, fix them. Common issues:
- `funlen` violations (max 65 lines) — extract helpers
- `gci` import ordering — run `task fmt`
- `revive` max-public-structs — check struct count per file
- Unused imports or variables

---

### Task 7: Create pull request

- [ ] **Step 1: Verify diff is clean**

Run: `git diff --name-only main..HEAD`

Expected files:
```
cmd/codeviz/radial_canvas.go
cmd/codeviz/radial_canvas_test.go
cmd/codeviz/radialtree_cmd.go
internal/radialtree/node.go
internal/render/radialtree.go
internal/render/radialtree_test.go
internal/render/svg_radial.go
```

No `.agents/`, `.squad/`, `docs/superpowers/` or other non-radial files.

- [ ] **Step 2: Push and create PR**

```bash
git push -u origin feature/canvas-radial
gh pr create \
  --title "feat: migrate radial tree visualization to Canvas pipeline" \
  --body "## Summary

Migrates the radial tree visualization from the old internal/render/ pipeline
to the new Canvas abstraction (Stage 2, third of four viz types).

## Changes

### RadialNode (geometry-only)
- Stripped FillColour/BorderColour fields — colour resolution moves to bridge

### Canvas bridge (radial_canvas.go)
- Ink construction via buildRadialInks (reuses buildMetricInk from treemap bridge)
- Recursive tree walk pairing RadialNode + model.Directory for correct file↔node mapping
- Four layers: background (rect), structure (edge lines), content (discs sorted largest-first), overlay (rotated labels)
- Root label uses contrasting text colour; external labels radially oriented with half-plane flip

### Command rewiring (radialtree_cmd.go)
- Replaced applyColoursAndRender with Canvas-based renderAndLog
- Deleted 6 old colour application functions
- Switched render.FormatFromPath → canvas.FormatFromPath

### Render cleanup
- Deleted radialtree.go (PNG), svg_radial.go (SVG), radialtree_test.go

### Testing
- 8 new Canvas-based tests (ink types, output formats, nested dirs, empty dir)

### Known regression
Legend rendering temporarily unavailable for radial tree.
Warning logged when legend position is configured." \
  --base main
```
