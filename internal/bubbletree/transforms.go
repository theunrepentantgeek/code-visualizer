package bubbletree

import "math"

// ---------------------------------------------------------------------------
// Coordinate bounds helpers (used by scaling and layout)
// ---------------------------------------------------------------------------

type bounds struct {
	minX float64
	minY float64
	maxX float64
	maxY float64
}

func newEmptyBounds() bounds {
	return bounds{
		minX: math.MaxFloat64,
		minY: math.MaxFloat64,
		maxX: -math.MaxFloat64,
		maxY: -math.MaxFloat64,
	}
}

func expandBoundsForDisc(box *bounds, x, y, radius float64) {
	if x-radius < box.minX {
		box.minX = x - radius
	}

	if y-radius < box.minY {
		box.minY = y - radius
	}

	if x+radius > box.maxX {
		box.maxX = x + radius
	}

	if y+radius > box.maxY {
		box.maxY = y + radius
	}
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
	if node.Radius <= 0 {
		node.X = width / 2
		node.Y = height / 2

		return
	}

	if len(node.Children) == 0 {
		node.X = width / 2
		node.Y = height / 2
		node.Radius = math.Min(width, height) * (1 - 2*canvasMarginFraction) / 2

		return
	}

	box := occupiedBounds(node)

	boxW := box.maxX - box.minX
	boxH := box.maxY - box.minY

	if boxW <= 0 || boxH <= 0 {
		node.X = width / 2
		node.Y = height / 2
		node.Radius *= math.Min(width, height) / (2 * node.Radius)

		return
	}

	usable := 1 - 2*canvasMarginFraction
	scale := math.Min(width*usable/boxW, height*usable/boxH)

	// Place the root node so that the bounding box centre maps to the canvas centre.
	boxCx := (box.minX + box.maxX) / 2
	boxCy := (box.minY + box.maxY) / 2

	node.X = width/2 - boxCx*scale
	node.Y = height/2 - boxCy*scale
	node.Radius *= scale

	applyScale(node, scale)
}

// occupiedBounds returns the tight axis-aligned bounding box of the node's
// occupied area in its local coordinate frame.
func occupiedBounds(node *BubbleNode) bounds {
	box := newEmptyBounds()

	for _, c := range node.Children {
		expandBoundsForDisc(&box, c.X, c.Y, c.Radius)
	}

	if node.ShowLabel && node.Radius > 0 {
		expandBoundsForDisc(&box, 0, 0, node.Radius)
	}

	return box
}

// applyScale recursively converts children from local to absolute coordinates.
func applyScale(parent *BubbleNode, scale float64) {
	for i := range parent.Children {
		child := &parent.Children[i]
		child.X = parent.X + child.X*scale
		child.Y = parent.Y + child.Y*scale
		child.Radius *= scale
		applyScale(child, scale)
	}
}

// OffsetNodes shifts every node in the tree by (dx, dy).
func OffsetNodes(node *BubbleNode, dx, dy float64) {
	node.X += dx
	node.Y += dy

	for i := range node.Children {
		OffsetNodes(&node.Children[i], dx, dy)
	}
}
