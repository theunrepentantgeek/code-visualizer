package main

import (
	"cmp"
	"image/color"
	"math"
	"slices"

	"github.com/bevan/code-visualizer/internal/canvas"
	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
	"github.com/bevan/code-visualizer/internal/radialtree"
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
	node   radialtree.RadialNode
	file   *model.File
	sx, sy float64
	isDir  bool
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
		addRadialDisc(cv, e, inks)
	}
}

// addRadialDisc adds a single disc shape to the canvas.
func addRadialDisc(cv *canvas.Canvas, e radialDiscEntry, inks radialInks) {
	fillMV := radialMetricValue(e.file, inks.fill)
	borderMV := radialMetricValue(e.file, inks.border)

	fill := inks.fill
	if e.isDir {
		fill = canvas.FixedInk(radialDefaultDirFill)
	}

	discSpec := &canvas.DiscSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        fill,
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

// radialMetricValue builds a MetricValue from a file's data for the given ink.
// For directory nodes (file == nil), returns an empty MetricValue.
func radialMetricValue(file *model.File, ink canvas.Ink) canvas.MetricValue {
	if file == nil {
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
		Ink:      canvas.FixedInk(labelColour),
		Anchor:   canvas.AnchorMiddle,
		FontSize: 0,
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
		FontSize: 0,
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
