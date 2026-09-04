package bubbletree

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

// ---------------------------------------------------------------------------
// expandBoundsForDisc
// ---------------------------------------------------------------------------

func TestExpandBoundsForDisc_SingleDisc(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	box, has := expandBoundsForDisc(geometry.Rect{}, false, geometry.Point{X: 5, Y: 3}, 2)

	g.Expect(has).To(BeTrue())
	g.Expect(box.Min.X).To(BeNumerically("~", 3.0, 1e-9))
	g.Expect(box.Max.X).To(BeNumerically("~", 7.0, 1e-9))
	g.Expect(box.Min.Y).To(BeNumerically("~", 1.0, 1e-9))
	g.Expect(box.Max.Y).To(BeNumerically("~", 5.0, 1e-9))
}

func TestExpandBoundsForDisc_MultipleDiscs(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	box, has := geometry.Rect{}, false
	box, has = expandBoundsForDisc(box, has, geometry.Point{X: 0, Y: 0}, 1)  // bounds: (-1,-1)..(1,1)
	box, has = expandBoundsForDisc(box, has, geometry.Point{X: 5, Y: 0}, 2)  // extends maxX to 7, maxY to 2, minY to -2
	box, has = expandBoundsForDisc(box, has, geometry.Point{X: 0, Y: -4}, 1) // extends minY to -5
	box, has = expandBoundsForDisc(box, has, geometry.Point{X: -3, Y: 0}, 0) // extends minX to -3

	g.Expect(has).To(BeTrue())
	g.Expect(box.Min.X).To(BeNumerically("~", -3.0, 1e-9))
	g.Expect(box.Max.X).To(BeNumerically("~", 7.0, 1e-9))
	g.Expect(box.Min.Y).To(BeNumerically("~", -5.0, 1e-9))
	g.Expect(box.Max.Y).To(BeNumerically("~", 2.0, 1e-9))
}

func TestExpandBoundsForDisc_ZeroRadius(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	box, has := expandBoundsForDisc(geometry.Rect{}, false, geometry.Point{X: 3, Y: 7}, 0) // zero-radius "point"

	g.Expect(has).To(BeTrue())
	g.Expect(box.Min.X).To(BeNumerically("~", 3.0, 1e-9))
	g.Expect(box.Max.X).To(BeNumerically("~", 3.0, 1e-9))
	g.Expect(box.Min.Y).To(BeNumerically("~", 7.0, 1e-9))
	g.Expect(box.Max.Y).To(BeNumerically("~", 7.0, 1e-9))
}

// ---------------------------------------------------------------------------
// occupiedBounds
// ---------------------------------------------------------------------------

func TestOccupiedBounds_NoChildren_NoLabel(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// A leaf node with no children and ShowLabel=false contributes nothing,
	// so occupiedBounds reports it received no rectangle.
	node := BubbleNode{Position: geometry.Point{X: 5, Y: 5}, Radius: 3, ShowLabel: false}

	_, has := occupiedBounds(&node)

	g.Expect(has).To(BeFalse())
}

func TestOccupiedBounds_WithChildren(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	parent := BubbleNode{
		Position: geometry.Point{X: 0, Y: 0}, Radius: 10,
		Children: []BubbleNode{
			{Position: geometry.Point{X: -3, Y: 0}, Radius: 2}, // covers X: -5 .. -1
			{Position: geometry.Point{X: 4, Y: 0}, Radius: 1},  // covers X:  3 ..  5
		},
	}

	box, has := occupiedBounds(&parent)

	g.Expect(has).To(BeTrue())
	g.Expect(box.Min.X).To(BeNumerically("~", -5.0, 1e-9))
	g.Expect(box.Max.X).To(BeNumerically("~", 5.0, 1e-9))
}

func TestOccupiedBounds_ShowLabelIncludesRoot(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// When ShowLabel is true, the root node's own circle is added to bounds.
	parent := BubbleNode{
		Position: geometry.Point{X: 0, Y: 0}, Radius: 8, ShowLabel: true,
		Children: []BubbleNode{
			{Position: geometry.Point{X: 1, Y: 0}, Radius: 1},
		},
	}

	box, has := occupiedBounds(&parent)

	g.Expect(has).To(BeTrue())

	// The root circle contributes radius 8 around (0,0).
	g.Expect(box.Min.X).To(BeNumerically("<=", -8.0+1e-9))
	g.Expect(box.Max.X).To(BeNumerically(">=", 8.0-1e-9))
}

// ---------------------------------------------------------------------------
// applyScale
// ---------------------------------------------------------------------------

func TestApplyScale_ChildPositionUpdated(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Parent placed at (10, 20); child in local frame at (2, 0) with radius 1.
	// After applyScale(scale=3): child should be at (10+2*3, 20+0*3)=(16,20) with radius 3.
	parent := BubbleNode{
		Position: geometry.Point{X: 10, Y: 20}, Radius: 15,
		Children: []BubbleNode{
			{Position: geometry.Point{X: 2, Y: 0}, Radius: 1},
		},
	}

	applyScale(&parent, 3)

	child := parent.Children[0]
	g.Expect(child.Position.X).To(BeNumerically("~", 16.0, 1e-9))
	g.Expect(child.Position.Y).To(BeNumerically("~", 20.0, 1e-9))
	g.Expect(child.Radius).To(BeNumerically("~", 3.0, 1e-9))
}

func TestApplyScale_NestedChildren(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	grandchild := BubbleNode{Position: geometry.Point{X: 1, Y: 0}, Radius: 0.5}
	child := BubbleNode{
		Position: geometry.Point{X: 2, Y: 0}, Radius: 2,
		Children: []BubbleNode{grandchild},
	}
	parent := BubbleNode{
		Position: geometry.Point{X: 0, Y: 0}, Radius: 10,
		Children: []BubbleNode{child},
	}

	applyScale(&parent, 2)

	// child: X = 0 + 2*2 = 4, Y = 0, Radius = 4
	g.Expect(parent.Children[0].Position.X).To(BeNumerically("~", 4.0, 1e-9))
	g.Expect(parent.Children[0].Radius).To(BeNumerically("~", 4.0, 1e-9))

	// grandchild: X = 4 + 1*2 = 6, Y = 0, Radius = 1
	g.Expect(parent.Children[0].Children[0].Position.X).To(BeNumerically("~", 6.0, 1e-9))
	g.Expect(parent.Children[0].Children[0].Radius).To(BeNumerically("~", 1.0, 1e-9))
}

// ---------------------------------------------------------------------------
// OffsetNodes
// ---------------------------------------------------------------------------

func TestOffsetNodes_RootAndChildren(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	child := BubbleNode{Position: geometry.Point{X: 5, Y: 5}, Radius: 1}
	root := BubbleNode{
		Position: geometry.Point{X: 10, Y: 10}, Radius: 20,
		Children: []BubbleNode{child},
	}

	OffsetNodes(&root, geometry.Vector{X: 3, Y: -2})

	g.Expect(root.Position.X).To(BeNumerically("~", 13.0, 1e-9))
	g.Expect(root.Position.Y).To(BeNumerically("~", 8.0, 1e-9))
	g.Expect(root.Children[0].Position.X).To(BeNumerically("~", 8.0, 1e-9))
	g.Expect(root.Children[0].Position.Y).To(BeNumerically("~", 3.0, 1e-9))
}
