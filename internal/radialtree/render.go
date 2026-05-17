package radialtree

import (
	"cmp"
	"image/color"
	"math"
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	pkginks "github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

const (
	radialEdgeWidth = 0.5
	radialLabelGap  = 4.0
)

// RenderToCanvas walks the layout and model trees, adding shapes
// to the canvas. Returns the populated canvas.
func RenderToCanvas(
	nodes *RadialNode,
	root *model.Directory,
	canvasSize int,
	inks Inks,
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
func addRadialEdges(cv *canvas.Canvas, node RadialNode, cx, cy float64) {
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
	node   RadialNode
	file   *model.File
	sx, sy float64
	isDir  bool
}

// collectRadialDiscs recursively gathers all nodes with a positive DiscRadius,
// along with their corresponding model.File (nil for directories).
// INVARIANT: node.Children are ordered files-first, then subdirectories.
func collectRadialDiscs(
	node *RadialNode,
	dir *model.Directory,
	cx, cy float64,
) []radialDiscEntry {
	entries := make([]radialDiscEntry, 0)

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
	node *RadialNode,
	file *model.File,
	cx, cy float64,
) []radialDiscEntry {
	if node.DiscRadius <= 0 {
		return make([]radialDiscEntry, 0)
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
	nodes *RadialNode,
	root *model.Directory,
	cx, cy float64,
	inks Inks,
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
func addRadialDisc(cv *canvas.Canvas, e radialDiscEntry, inks Inks) {
	fillMV := pkginks.MetricValueForFile(e.file, inks.Fill)
	borderMV := pkginks.MetricValueForFile(e.file, inks.Border)

	fill := inks.Fill
	border := inks.Border

	if e.isDir {
		fill = canvas.FixedInk(radialDefaultDirFill)
		border = canvas.FixedInk(radialDefaultBorder)
	}

	discSpec := &canvas.DiscSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        fill,
			Border:      border,
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

// addRadialLabels recursively adds text labels for nodes with ShowLabel set.
func addRadialLabels(
	cv *canvas.Canvas,
	node RadialNode,
	cx, cy float64,
	inks Inks,
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
	node RadialNode,
	cx, cy float64,
	inks Inks,
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
	node RadialNode,
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
func radialEffectiveFill(node RadialNode, inks Inks) color.RGBA {
	if node.IsDirectory {
		return radialDefaultDirFill
	}

	return inks.Fill.Dip(canvas.MetricValue{})
}
