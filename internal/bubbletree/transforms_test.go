package bubbletree

import (
	"math"
	"testing"

	. "github.com/onsi/gomega"
)

// ---------------------------------------------------------------------------
// expandBoundsForDisc
// ---------------------------------------------------------------------------

func TestExpandBoundsForDisc_SingleDisc(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	box := newEmptyBounds()
	expandBoundsForDisc(&box, 5, 3, 2)

	g.Expect(box.minX).To(BeNumerically("~", 3.0, 1e-9))
	g.Expect(box.maxX).To(BeNumerically("~", 7.0, 1e-9))
	g.Expect(box.minY).To(BeNumerically("~", 1.0, 1e-9))
	g.Expect(box.maxY).To(BeNumerically("~", 5.0, 1e-9))
}

func TestExpandBoundsForDisc_MultipleDiscs(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	box := newEmptyBounds()
	expandBoundsForDisc(&box, 0, 0, 1)  // bounds: (-1,-1)..(1,1)
	expandBoundsForDisc(&box, 5, 0, 2)  // extends maxX to 7, maxY to 2, minY to -2
	expandBoundsForDisc(&box, 0, -4, 1) // extends minY to -5
	expandBoundsForDisc(&box, -3, 0, 0) // extends minX to -3

	g.Expect(box.minX).To(BeNumerically("~", -3.0, 1e-9))
	g.Expect(box.maxX).To(BeNumerically("~", 7.0, 1e-9))
	g.Expect(box.minY).To(BeNumerically("~", -5.0, 1e-9))
	g.Expect(box.maxY).To(BeNumerically("~", 2.0, 1e-9))
}

func TestExpandBoundsForDisc_ZeroRadius(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	box := newEmptyBounds()
	expandBoundsForDisc(&box, 3, 7, 0) // zero-radius "point"

	g.Expect(box.minX).To(BeNumerically("~", 3.0, 1e-9))
	g.Expect(box.maxX).To(BeNumerically("~", 3.0, 1e-9))
	g.Expect(box.minY).To(BeNumerically("~", 7.0, 1e-9))
	g.Expect(box.maxY).To(BeNumerically("~", 7.0, 1e-9))
}

// ---------------------------------------------------------------------------
// occupiedBounds
// ---------------------------------------------------------------------------

func TestOccupiedBounds_NoChildren_NoLabel(t *testing.T) {
	t.Parallel()

	// A leaf node with no children and ShowLabel=false returns an "empty"
	// bounds (MaxFloat64 for min, -MaxFloat64 for max). The caller is
	// expected to guard against degenerate boxes; we just verify the
	// values are the empty-bounds sentinel.
	node := BubbleNode{X: 5, Y: 5, Radius: 3, ShowLabel: false}

	box := occupiedBounds(&node)

	// minX should equal the initial empty sentinel.
	if box.minX != math.MaxFloat64 {
		t.Errorf("expected minX == MaxFloat64 for empty bounds, got %v", box.minX)
	}
}

func TestOccupiedBounds_WithChildren(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	parent := BubbleNode{
		X: 0, Y: 0, Radius: 10,
		Children: []BubbleNode{
			{X: -3, Y: 0, Radius: 2}, // covers X: -5 .. -1
			{X: 4, Y: 0, Radius: 1},  // covers X:  3 ..  5
		},
	}

	box := occupiedBounds(&parent)

	g.Expect(box.minX).To(BeNumerically("~", -5.0, 1e-9))
	g.Expect(box.maxX).To(BeNumerically("~", 5.0, 1e-9))
}

func TestOccupiedBounds_ShowLabelIncludesRoot(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// When ShowLabel is true, the root node's own circle is added to bounds.
	parent := BubbleNode{
		X: 0, Y: 0, Radius: 8, ShowLabel: true,
		Children: []BubbleNode{
			{X: 1, Y: 0, Radius: 1},
		},
	}

	box := occupiedBounds(&parent)

	// The root circle contributes radius 8 around (0,0).
	g.Expect(box.minX).To(BeNumerically("<=", -8.0+1e-9))
	g.Expect(box.maxX).To(BeNumerically(">=", 8.0-1e-9))
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
		X: 10, Y: 20, Radius: 15,
		Children: []BubbleNode{
			{X: 2, Y: 0, Radius: 1},
		},
	}

	applyScale(&parent, 3)

	child := parent.Children[0]
	g.Expect(child.X).To(BeNumerically("~", 16.0, 1e-9))
	g.Expect(child.Y).To(BeNumerically("~", 20.0, 1e-9))
	g.Expect(child.Radius).To(BeNumerically("~", 3.0, 1e-9))
}

func TestApplyScale_NestedChildren(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	grandchild := BubbleNode{X: 1, Y: 0, Radius: 0.5}
	child := BubbleNode{
		X: 2, Y: 0, Radius: 2,
		Children: []BubbleNode{grandchild},
	}
	parent := BubbleNode{
		X: 0, Y: 0, Radius: 10,
		Children: []BubbleNode{child},
	}

	applyScale(&parent, 2)

	// child: X = 0 + 2*2 = 4, Y = 0, Radius = 4
	g.Expect(parent.Children[0].X).To(BeNumerically("~", 4.0, 1e-9))
	g.Expect(parent.Children[0].Radius).To(BeNumerically("~", 4.0, 1e-9))

	// grandchild: X = 4 + 1*2 = 6, Y = 0, Radius = 1
	g.Expect(parent.Children[0].Children[0].X).To(BeNumerically("~", 6.0, 1e-9))
	g.Expect(parent.Children[0].Children[0].Radius).To(BeNumerically("~", 1.0, 1e-9))
}

// ---------------------------------------------------------------------------
// OffsetNodes
// ---------------------------------------------------------------------------

func TestOffsetNodes_RootAndChildren(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	child := BubbleNode{X: 5, Y: 5, Radius: 1}
	root := BubbleNode{
		X: 10, Y: 10, Radius: 20,
		Children: []BubbleNode{child},
	}

	OffsetNodes(&root, 3, -2)

	g.Expect(root.X).To(BeNumerically("~", 13.0, 1e-9))
	g.Expect(root.Y).To(BeNumerically("~", 8.0, 1e-9))
	g.Expect(root.Children[0].X).To(BeNumerically("~", 8.0, 1e-9))
	g.Expect(root.Children[0].Y).To(BeNumerically("~", 3.0, 1e-9))
}
