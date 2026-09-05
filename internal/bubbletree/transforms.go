package bubbletree

import (
	"math"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

// ---------------------------------------------------------------------------
// Coordinate bounds helpers (used by scaling and layout)
// ---------------------------------------------------------------------------

// occupiedBounds returns the tight axis-aligned bounding box of the node's
// occupied area in its local coordinate frame, and whether any disc
// contributed to it. A node with no children and no visible label
// contributes nothing, so the returned bool is false.
func occupiedBounds(node *BubbleNode) (geometry.Rect, bool) {
	var (
		box geometry.Rect
		has bool
	)

	for _, c := range node.Children {
		box, has = expandBoundsForDisc(box, has, c.Geometry)
	}

	if node.ShowLabel && node.Geometry.Radius > 0 {
		box, has = expandBoundsForDisc(box, has, geometry.NewCircle(geometry.OriginPoint, node.Geometry.Radius))
	}

	return box, has
}

// expandBoundsForDisc returns box expanded to include circle's bounds, and
// true. When box has not yet received a disc (has is false), the disc's own
// bounds become box directly, avoiding a false union with an arbitrary
// starting rectangle.
//
//nolint:revive,nolintlint // flag-parameter: has is fold-accumulator state, not a behaviour switch
func expandBoundsForDisc(box geometry.Rect, has bool, circle geometry.Circle) (geometry.Rect, bool) {
	discBounds := circle.Bounds()

	if !has {
		return discBounds, true
	}

	if unioned, ok := box.Union(discBounds); ok {
		return unioned, true
	}

	return box, has
}

// ---------------------------------------------------------------------------
// Top-down coordinate assignment — scales local layout to pixel canvas
// ---------------------------------------------------------------------------

const canvasMarginFraction = 0.02 // 2% margin on each side

// scaleToFit assigns absolute pixel coordinates to the entire tree,
// scaling and translating so the tight bounding rectangle of the children
// fills the canvas (minus a small margin). Using a rectangle rather than the
// root bounding circle removes the large whitespace corners that a circle
// fit would leave on a non-square canvas.
func scaleToFit(node *BubbleNode, width, height float64) {
	if node.Geometry.Radius <= 0 {
		node.Geometry.Center = geometry.NewPoint(width/2, height/2)

		return
	}

	if len(node.Children) == 0 {
		node.Geometry.Center = geometry.NewPoint(width/2, height/2)
		node.Geometry.Radius = math.Min(width, height) * (1 - 2*canvasMarginFraction) / 2

		return
	}

	box, has := occupiedBounds(node)

	boxW := box.Width()
	boxH := box.Height()

	if !has || boxW <= 0 || boxH <= 0 {
		node.Geometry.Center = geometry.NewPoint(width/2, height/2)
		node.Geometry.Radius *= math.Min(width, height) / (2 * node.Geometry.Radius)

		return
	}

	usable := 1 - 2*canvasMarginFraction
	scale := math.Min(width*usable/boxW, height*usable/boxH)

	// Place the root node so that the bounding box centre maps to the canvas centre.
	boxCx := (box.Min.X + box.Max.X) / 2
	boxCy := (box.Min.Y + box.Max.Y) / 2

	node.Geometry.Center = geometry.NewPoint(width/2-boxCx*scale, height/2-boxCy*scale)
	node.Geometry.Radius *= scale

	applyScale(node, scale)
}

// applyScale recursively converts children from local to absolute coordinates.
func applyScale(parent *BubbleNode, scale float64) {
	for i := range parent.Children {
		child := &parent.Children[i]
		child.Geometry.Center = parent.Geometry.Center.Translate(geometry.Vector{
			X: child.Geometry.Center.X * scale,
			Y: child.Geometry.Center.Y * scale,
		})
		child.Geometry.Radius *= scale
		applyScale(child, scale)
	}
}

// OffsetNodes shifts every node in the tree by the provided offset.
func OffsetNodes(node *BubbleNode, offset geometry.Vector) {
	node.Geometry.Center = node.Geometry.Center.Translate(offset)

	for i := range node.Children {
		OffsetNodes(&node.Children[i], offset)
	}
}
