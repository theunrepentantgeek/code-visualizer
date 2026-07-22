package radialtree

import (
	"cmp"
	"math"
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	canvasmodel "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

const (
	edgeWidth = 0.5
	labelGap  = 4.0
)

// RenderToCanvas walks the layout and model trees, adding shapes to the canvas.
// canvasSize is the side length (pixels) of the square content area.
// topOffset is the number of pixels to reserve at the top (e.g. for a title);
// the content centre is shifted down by topOffset so it fits below the reserved area.
func RenderToCanvas(
	nodes *RadialNode,
	root *model.Directory,
	canvasWidth int,
	canvasHeight int,
	canvasSize int,
	topOffset int,
	is Inks,
) *canvas.Canvas {
	cv := canvas.NewCanvas(canvasWidth, canvasHeight)

	cx := float64(canvasWidth) / 2.0
	cy := float64(canvasSize)/2.0 + float64(topOffset)

	addBackground(cv, canvasWidth, canvasHeight)
	addEdges(cv, *nodes, cx, cy)
	addDiscs(cv, nodes, root, cx, cy, is)
	addLabels(cv, *nodes, cx, cy, is)

	return cv
}

// addBackground adds a white background rectangle.
func addBackground(cv *canvas.Canvas, canvasWidth, canvasHeight int) {
	bgSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        inks.FixedInk(bgColour),
			Border:      inks.FixedInk(bgColour),
			BorderWidth: 0,
		},
	}

	cv.AddRectangle(canvas.LayerBackground, canvas.Rectangle{
		Spec:  bgSpec,
		W:     float64(canvasWidth),
		H:     float64(canvasHeight),
		Focus: canvasmodel.Point{X: 0.5, Y: 0.5},
	})
}

// addEdges recursively adds edge lines from each node to its children.
func addEdges(cv *canvas.Canvas, node RadialNode, cx, cy float64) {
	edgeSpec := &canvas.LineSpec{
		Stroke:      inks.FixedInk(edgeColour),
		StrokeWidth: edgeWidth,
	}
	addEdgesInner(cv, node, cx, cy, edgeSpec)
}

// addEdgesInner is the recursive worker for addEdges. It accepts a pre-allocated
// edgeSpec so the single allocation is not repeated for every node in the tree.
func addEdgesInner(cv *canvas.Canvas, node RadialNode, cx, cy float64, edgeSpec *canvas.LineSpec) {
	px := cx + node.X
	py := cy + node.Y

	for _, child := range node.Children {
		chx := cx + child.X
		chy := cy + child.Y

		cv.AddLine(canvas.LayerStructure, canvas.Line{
			Spec: edgeSpec,
			X1:   px, Y1: py,
			X2: chx, Y2: chy,
		})

		addEdgesInner(cv, child, cx, cy, edgeSpec)
	}
}

// discEntry holds a node and its screen position for deferred drawing.
type discEntry struct {
	node   RadialNode
	file   *model.File
	sx, sy float64
	isDir  bool
}

// collectDiscs recursively gathers all nodes with a positive DiscRadius,
// along with their corresponding model.File (nil for directories).
// INVARIANT: node.Children are ordered files-first, then subdirectories.
func collectDiscs(
	node *RadialNode,
	dir *model.Directory,
	cx, cy float64,
) []discEntry {
	entries := make([]discEntry, 0)

	if node.DiscRadius > 0 {
		entries = append(entries, discEntry{
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
			entries = append(entries, collectDiscs(child, dir.Dirs[dirIdx], cx, cy)...)
			dirIdx++
		} else if !child.IsDirectory && fileIdx < len(dir.Files) {
			childEntries := collectDiscsLeaf(child, dir.Files[fileIdx], cx, cy)
			entries = append(entries, childEntries...)
			fileIdx++
		}
	}

	return entries
}

// collectDiscsLeaf collects a single file node (leaf).
func collectDiscsLeaf(
	node *RadialNode,
	file *model.File,
	cx, cy float64,
) []discEntry {
	if node.DiscRadius <= 0 {
		return make([]discEntry, 0)
	}

	return []discEntry{{
		node: *node,
		file: file,
		sx:   cx + node.X,
		sy:   cy + node.Y,
	}}
}

// addDiscs collects all discs, sorts them largest-first so smaller nodes are
// never obscured, then adds them to the canvas.
func addDiscs(
	cv *canvas.Canvas,
	nodes *RadialNode,
	root *model.Directory,
	cx, cy float64,
	is Inks,
) {
	entries := collectDiscs(nodes, root, cx, cy)

	slices.SortFunc(entries, func(a, b discEntry) int {
		return cmp.Compare(b.node.DiscRadius, a.node.DiscRadius)
	})

	// Pre-allocate the two spec variants so they are not re-created per disc.
	fileSpec := &canvas.DiscSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        is.Fill,
			Border:      is.Border,
			BorderWidth: 1.0,
		},
	}
	dirSpec := &canvas.DiscSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        inks.FixedInk(defaultDirFill),
			Border:      inks.FixedInk(defaultBorder),
			BorderWidth: 1.0,
		},
	}

	for _, e := range entries {
		addDisc(cv, e, is, fileSpec, dirSpec)
	}
}

// addDisc adds a single disc shape to the canvas.
func addDisc(cv *canvas.Canvas, e discEntry, is Inks, fileSpec, dirSpec *canvas.DiscSpec) {
	fillMV := inks.MetricValueForFile(e.file, is.Fill)
	borderMV := inks.MetricValueForFile(e.file, is.Border)

	spec := fileSpec
	if e.isDir {
		spec = dirSpec
	}

	cv.AddDisc(canvas.LayerContent, canvas.Disc{
		Spec:   spec,
		X:      e.sx,
		Y:      e.sy,
		Radius: e.node.DiscRadius,
		Angle:  e.node.Angle,
		Fill:   fillMV,
		Border: borderMV,
	})
}

// addLabels recursively adds text labels for nodes with ShowLabel set.
func addLabels(
	cv *canvas.Canvas,
	node RadialNode,
	cx, cy float64,
	is Inks,
) {
	labelInk := inks.FixedInk(labelColour)
	// The root sits at dist==0 and has no meaningful angle; pass NaN so its
	// direct file children each use their own angle for orientation.
	addLabelsInner(cv, node, cx, cy, is, math.NaN(), labelInk)
}

// addLabelsInner recurses the node tree, rendering labels.
// parentDirAngle is the angle of the nearest ancestor directory node in
// radians, or math.NaN() when there is no such ancestor (e.g. for the root
// node itself and its direct children). File labels inherit the parent
// directory angle so that all files within a given directory use a consistent
// left/right orientation even when they straddle the 12 o'clock or 6 o'clock
// meridian.
func addLabelsInner(
	cv *canvas.Canvas,
	node RadialNode,
	cx, cy float64,
	is Inks,
	parentDirAngle float64,
	labelInk inks.Ink,
) {
	renderNodeLabel(cv, node, cx, cy, is, parentDirAngle, labelInk)

	childParentAngle := childParentAngleFor(node)
	for _, child := range node.Children {
		addLabelsInner(cv, child, cx, cy, is, childParentAngle, labelInk)
	}
}

// renderNodeLabel renders the label for a single node, if it has one.
func renderNodeLabel(
	cv *canvas.Canvas,
	node RadialNode,
	cx, cy float64,
	is Inks,
	parentDirAngle float64,
	labelInk inks.Ink,
) {
	if !node.ShowLabel || node.Label == "" {
		return
	}

	if nodeDistance(node) == 0 {
		addRootLabel(cv, node, cx, cy, is, labelInk)
	} else {
		addExternalLabel(cv, node, cx, cy, labelOrientAngle(node, parentDirAngle), labelInk)
	}
}

// labelOrientAngle returns the angle to use for orienting a non-root node's
// label. File nodes inherit their parent directory's angle when available so
// sibling files share a consistent left/right orientation.
func labelOrientAngle(node RadialNode, parentDirAngle float64) float64 {
	if !node.IsDirectory && !math.IsNaN(parentDirAngle) {
		return parentDirAngle
	}

	return node.Angle
}

// childParentAngleFor returns the parentDirAngle to pass to a node's children.
// Only non-root directories propagate their angle; root and file nodes pass NaN.
func childParentAngleFor(node RadialNode) float64 {
	if node.IsDirectory && nodeDistance(node) > 0 {
		return node.Angle
	}

	return math.NaN()
}

// nodeDistance returns the node's distance from the canvas centre.
func nodeDistance(node RadialNode) float64 {
	return math.Sqrt(node.X*node.X + node.Y*node.Y)
}

// addRootLabel adds a centred label on the root disc.
// The label uses the same dark labelColour as external labels because the
// root disc is often very small; most of the text sits on the white
// background where white text would be invisible.
func addRootLabel(
	cv *canvas.Canvas,
	node RadialNode,
	cx, cy float64,
	_ Inks,
	labelInk inks.Ink,
) {
	labelSpec := &canvas.TextSpec{
		Ink:      labelInk,
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

// addExternalLabel adds a radially-oriented label outside the disc.
// orientAngle controls the left/right side determination (anchor and rotation);
// the label is still positioned at node.Angle from the canvas centre.
// Pass node.Angle for the default per-node behaviour, or a parent directory's
// angle to keep all sibling file labels on the same side.
func addExternalLabel(
	cv *canvas.Canvas,
	node RadialNode,
	cx, cy float64,
	orientAngle float64,
	labelInk inks.Ink,
) {
	dist := math.Sqrt(node.X*node.X + node.Y*node.Y)
	labelRadius := dist + node.DiscRadius + labelGap
	lx := cx + labelRadius*math.Cos(node.Angle)
	ly := cy + labelRadius*math.Sin(node.Angle)

	angle := math.Mod(orientAngle, 2*math.Pi)
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
		Ink:      labelInk,
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
