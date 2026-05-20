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
	edgeWidth = 0.5
	labelGap  = 4.0
)

// RenderToCanvas walks the layout and model trees, adding shapes to the canvas.
// canvasSize is the side length (pixels) of the square canvas.
func RenderToCanvas(
	nodes *RadialNode,
	root *model.Directory,
	canvasSize int,
	inks Inks,
) *canvas.Canvas {
	cv := canvas.NewCanvas(canvasSize, canvasSize)

	cx := float64(canvasSize) / 2.0
	cy := float64(canvasSize) / 2.0

	addBackground(cv, canvasSize)
	addEdges(cv, *nodes, cx, cy)
	addDiscs(cv, nodes, root, cx, cy, inks)
	addLabels(cv, *nodes, cx, cy, inks)

	return cv
}

// addBackground adds a white background rectangle.
func addBackground(cv *canvas.Canvas, canvasSize int) {
	bgSpec := &canvas.RectangleSpec{
		ShapeStyle: canvas.ShapeStyle{
			Fill:        canvas.FixedInk(bgColour),
			Border:      canvas.FixedInk(bgColour),
			BorderWidth: 0,
		},
	}

	cv.AddRectangle(canvas.LayerBackground, canvas.Rectangle{
		Spec: bgSpec,
		W:    float64(canvasSize), H: float64(canvasSize),
	})
}

// addEdges recursively adds edge lines from each node to its children.
func addEdges(cv *canvas.Canvas, node RadialNode, cx, cy float64) {
	px := cx + node.X
	py := cy + node.Y

	edgeSpec := &canvas.LineSpec{
		Stroke:      canvas.FixedInk(edgeColour),
		StrokeWidth: edgeWidth,
	}

	for _, child := range node.Children {
		chx := cx + child.X
		chy := cy + child.Y

		cv.AddLine(canvas.LayerStructure, canvas.Line{
			Spec: edgeSpec,
			X1:   px, Y1: py,
			X2: chx, Y2: chy,
		})

		addEdges(cv, child, cx, cy)
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
	inks Inks,
) {
	entries := collectDiscs(nodes, root, cx, cy)

	slices.SortFunc(entries, func(a, b discEntry) int {
		return cmp.Compare(b.node.DiscRadius, a.node.DiscRadius)
	})

	for _, e := range entries {
		addDisc(cv, e, inks)
	}
}

// addDisc adds a single disc shape to the canvas.
func addDisc(cv *canvas.Canvas, e discEntry, inks Inks) {
	fillMV := pkginks.MetricValueForFile(e.file, inks.Fill)
	borderMV := pkginks.MetricValueForFile(e.file, inks.Border)

	fill := inks.Fill
	border := inks.Border

	if e.isDir {
		fill = canvas.FixedInk(defaultDirFill)
		border = canvas.FixedInk(defaultBorder)
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

// addLabels recursively adds text labels for nodes with ShowLabel set.
func addLabels(
	cv *canvas.Canvas,
	node RadialNode,
	cx, cy float64,
	inks Inks,
) {
	// The root sits at dist==0 and has no meaningful angle; pass NaN so its
	// direct file children each use their own angle for orientation.
	addLabelsInner(cv, node, cx, cy, inks, math.NaN())
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
	inks Inks,
	parentDirAngle float64,
) {
	renderNodeLabel(cv, node, cx, cy, inks, parentDirAngle)

	childParentAngle := childParentAngleFor(node)
	for _, child := range node.Children {
		addLabelsInner(cv, child, cx, cy, inks, childParentAngle)
	}
}

// renderNodeLabel renders the label for a single node, if it has one.
func renderNodeLabel(
	cv *canvas.Canvas,
	node RadialNode,
	cx, cy float64,
	inks Inks,
	parentDirAngle float64,
) {
	if !node.ShowLabel || node.Label == "" {
		return
	}

	if nodeDistance(node) == 0 {
		addRootLabel(cv, node, cx, cy, inks)
	} else {
		addExternalLabel(cv, node, cx, cy, labelOrientAngle(node, parentDirAngle))
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

// addRootLabel adds a centred label on the root disc, using a contrasting
// text colour based on the effective fill.
func addRootLabel(
	cv *canvas.Canvas,
	node RadialNode,
	cx, cy float64,
	inks Inks,
) {
	fill := effectiveFill(node, inks)
	labelTextColour := canvas.TextColourFor(fill)

	labelSpec := &canvas.TextSpec{
		Ink:      canvas.FixedInk(labelTextColour),
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
		Ink:      canvas.FixedInk(labelColour),
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

// effectiveFill returns the fill colour for a node, resolving defaults.
// Used for computing label contrast colour on the root node.
func effectiveFill(node RadialNode, inks Inks) color.RGBA {
	if node.IsDirectory {
		return defaultDirFill
	}

	return inks.Fill.Dip(canvas.MetricValue{})
}
