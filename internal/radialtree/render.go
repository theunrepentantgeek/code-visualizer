package radialtree

import (
	"cmp"
	"math"
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	canvasmodel "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

const (
	edgeWidth = 0.5
	labelGap  = 4.0
)

// RenderToCanvas walks the layout and model trees, adding shapes to a
// canvasWidth×canvasHeight canvas. Node coordinates are stored relative to the
// tree centre, so (cx, cy) is the screen-space point the root is drawn at and
// every node is translated by adding it.
func RenderToCanvas(
	nodes *RadialNode,
	root *model.Directory,
	canvasWidth int,
	canvasHeight int,
	cx float64,
	cy float64,
	is Inks,
) *canvas.Canvas {
	cv := canvas.NewCanvas(canvasWidth, canvasHeight)
	center := geometry.NewPoint(cx, cy)

	addBackground(cv, canvasWidth, canvasHeight)
	addEdges(cv, *nodes, center)
	addDiscs(cv, nodes, root, center, is)
	addLabels(cv, *nodes, center, is)

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
		Spec:   bgSpec,
		Bounds: geometry.Rect{Max: geometry.Point{X: float64(canvasWidth), Y: float64(canvasHeight)}},
		Focus:  canvasmodel.GradientPoint{X: 0.5, Y: 0.5},
	})
}

// addEdges recursively adds edge lines from each node to its children.
func addEdges(cv *canvas.Canvas, node RadialNode, center geometry.Point) {
	edgeSpec := &canvas.LineSpec{
		Stroke:      inks.FixedInk(edgeColour),
		StrokeWidth: edgeWidth,
	}
	addEdgesInner(cv, node, center, edgeSpec)
}

// addEdgesInner is the recursive worker for addEdges. It accepts a pre-allocated
// edgeSpec so the single allocation is not repeated for every node in the tree.
func addEdgesInner(cv *canvas.Canvas, node RadialNode, center geometry.Point, edgeSpec *canvas.LineSpec) {
	position := center.Translate(node.Position)

	for _, child := range node.Children {
		childPosition := center.Translate(child.Position)

		cv.AddLine(canvas.LayerStructure, canvas.Line{
			Spec: edgeSpec,
			From: position,
			To:   childPosition,
		})

		addEdgesInner(cv, child, center, edgeSpec)
	}
}

// discEntry holds a node and its screen position for deferred drawing.
type discEntry struct {
	node      RadialNode
	file      *model.File
	directory *model.Directory
	position  geometry.Point
	isDir     bool
}

// collectDiscs recursively gathers all nodes with a positive DiscRadius,
// along with their corresponding model.File (nil for directories).
// INVARIANT: node.Children are ordered files-first, then subdirectories.
func collectDiscs(
	node *RadialNode,
	dir *model.Directory,
	center geometry.Point,
) []discEntry {
	entries := make([]discEntry, 0)

	if node.DiscRadius > 0 {
		position := center.Translate(node.Position)
		entries = append(entries, discEntry{
			node:      *node,
			position:  position,
			isDir:     node.IsDirectory,
			directory: dir,
		})
	}

	fileIdx := 0
	dirIdx := 0

	for i := range node.Children {
		child := &node.Children[i]
		if child.IsDirectory && dirIdx < len(dir.Dirs) {
			entries = append(entries, collectDiscs(child, dir.Dirs[dirIdx], center)...)
			dirIdx++
		} else if !child.IsDirectory && fileIdx < len(dir.Files) {
			childEntries := collectDiscsLeaf(child, dir.Files[fileIdx], center)
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
	center geometry.Point,
) []discEntry {
	if node.DiscRadius <= 0 {
		return make([]discEntry, 0)
	}

	position := center.Translate(node.Position)

	return []discEntry{{
		node:     *node,
		file:     file,
		position: position,
	}}
}

// addDiscs collects all discs, sorts them largest-first so smaller nodes are
// never obscured, then adds them to the canvas.
func addDiscs(
	cv *canvas.Canvas,
	nodes *RadialNode,
	root *model.Directory,
	center geometry.Point,
	is Inks,
) {
	entries := collectDiscs(nodes, root, center)

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
			Fill:        is.DirectoryFill,
			Border:      is.DirectoryBorder,
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
		fillMV = inks.MetricValueForDirectory(e.directory, is.DirectoryFill)
		borderMV = inks.MetricValueForDirectory(e.directory, is.DirectoryBorder)
	}

	cv.AddDisc(canvas.LayerContent, canvas.Disc{
		Spec: spec,
		Geometry: geometry.Circle{
			Center: e.position,
			Radius: e.node.DiscRadius,
		},
		Angle:  e.node.Angle,
		Fill:   fillMV,
		Border: borderMV,
	})
}

// addLabels recursively adds text labels for nodes with ShowLabel set.
func addLabels(
	cv *canvas.Canvas,
	node RadialNode,
	center geometry.Point,
	is Inks,
) {
	labelInk := inks.FixedInk(labelColour)
	// The root sits at dist==0 and has no meaningful angle; pass NaN so its
	// direct file children each use their own angle for orientation.
	addLabelsInner(cv, node, center, is, math.NaN(), labelInk)
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
	center geometry.Point,
	is Inks,
	parentDirAngle float64,
	labelInk inks.Ink,
) {
	renderNodeLabel(cv, node, center, is, parentDirAngle, labelInk)

	childParentAngle := childParentAngleFor(node)
	for _, child := range node.Children {
		addLabelsInner(cv, child, center, is, childParentAngle, labelInk)
	}
}

// renderNodeLabel renders the label for a single node, if it has one.
func renderNodeLabel(
	cv *canvas.Canvas,
	node RadialNode,
	center geometry.Point,
	is Inks,
	parentDirAngle float64,
	labelInk inks.Ink,
) {
	if !node.ShowLabel || node.Label == "" {
		return
	}

	if node.Position.Length() == 0 {
		addRootLabel(cv, node, center, is, labelInk)
	} else {
		addExternalLabel(cv, node, center, labelOrientAngle(node, parentDirAngle), labelInk)
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
	if node.IsDirectory && node.Position.Length() > 0 {
		return node.Angle
	}

	return math.NaN()
}

// addRootLabel adds a centred label on the root disc.
// The label uses the same dark labelColour as external labels because the
// root disc is often very small; most of the text sits on the white
// background where white text would be invisible.
func addRootLabel(
	cv *canvas.Canvas,
	node RadialNode,
	center geometry.Point,
	_ Inks,
	labelInk inks.Ink,
) {
	labelSpec := &canvas.TextSpec{
		Ink:      labelInk,
		Anchor:   canvas.AnchorMiddle,
		FontSize: 0,
	}

	cv.AddText(canvas.LayerOverlay, canvas.Text{
		Spec:     labelSpec,
		Position: center.Translate(node.Position),
		Content:  node.Label,
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
	center geometry.Point,
	orientAngle float64,
	labelInk inks.Ink,
) {
	dist := node.Position.Length()
	labelRadius := dist + node.DiscRadius + labelGap
	labelDisplacement := geometry.NewVector(
		labelRadius*math.Cos(node.Angle),
		labelRadius*math.Sin(node.Angle),
	)

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
		Spec:     labelSpec,
		Position: center.Translate(labelDisplacement),
		Content:  node.Label,
	})
}
