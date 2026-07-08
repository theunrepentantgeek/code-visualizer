package bubbletree

import (
	"math"
	"testing"

	. "github.com/onsi/gomega"
)

// ---------------------------------------------------------------------------
// encloses
// ---------------------------------------------------------------------------

func TestEncloses_ContainedCircle(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	outer := enclosure{x: 0, y: 0, radius: 10}
	inner := enclosure{x: 1, y: 1, radius: 2}

	g.Expect(encloses(outer, inner)).To(BeTrue())
}

func TestEncloses_SameCircle(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := enclosure{x: 5, y: 5, radius: 3}

	g.Expect(encloses(c, c)).To(BeTrue())
}

func TestEncloses_OuterTooSmall(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	outer := enclosure{x: 0, y: 0, radius: 3}
	inner := enclosure{x: 0, y: 0, radius: 5}

	g.Expect(encloses(outer, inner)).To(BeFalse())
}

func TestEncloses_TouchingExternally(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := enclosure{x: 0, y: 0, radius: 3}
	b := enclosure{x: 6, y: 0, radius: 3} // centres 6 apart, radii sum to 6

	// Neither encloses the other.
	g.Expect(encloses(a, b)).To(BeFalse())
	g.Expect(encloses(b, a)).To(BeFalse())
}

// ---------------------------------------------------------------------------
// enclosingTwo
// ---------------------------------------------------------------------------

func TestEnclosingTwo_AContainsB(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// B is entirely inside A, so the enclosing circle should equal A.
	a := enclosure{x: 0, y: 0, radius: 10}
	b := enclosure{x: 1, y: 0, radius: 2}

	result := enclosingTwo(a, b)

	g.Expect(result.radius).To(BeNumerically("~", a.radius, 1e-9))
	g.Expect(result.x).To(BeNumerically("~", a.x, 1e-9))
	g.Expect(result.y).To(BeNumerically("~", a.y, 1e-9))
}

func TestEnclosingTwo_BContainsA(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// A is entirely inside B.
	a := enclosure{x: 1, y: 0, radius: 2}
	b := enclosure{x: 0, y: 0, radius: 10}

	result := enclosingTwo(a, b)

	g.Expect(result.radius).To(BeNumerically("~", b.radius, 1e-9))
}

func TestEnclosingTwo_EqualCirclesSideBySide(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Two circles of radius 1 with centres at (-1,0) and (1,0).
	// The minimum enclosing circle has centre at (0,0) and radius 2.
	a := enclosure{x: -1, y: 0, radius: 1}
	b := enclosure{x: 1, y: 0, radius: 1}

	result := enclosingTwo(a, b)

	g.Expect(result.x).To(BeNumerically("~", 0.0, 1e-9))
	g.Expect(result.y).To(BeNumerically("~", 0.0, 1e-9))
	g.Expect(result.radius).To(BeNumerically("~", 2.0, 1e-9))

	// Verify it actually encloses both.
	g.Expect(encloses(result, a)).To(BeTrue())
	g.Expect(encloses(result, b)).To(BeTrue())
}

func TestEnclosingTwo_DifferentRadii(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := enclosure{x: 0, y: 0, radius: 1}
	b := enclosure{x: 3, y: 0, radius: 2}

	result := enclosingTwo(a, b)

	// Enclosing circle must contain both.
	g.Expect(encloses(result, a)).To(BeTrue())
	g.Expect(encloses(result, b)).To(BeTrue())
}

// ---------------------------------------------------------------------------
// enclosingThree
// ---------------------------------------------------------------------------

func TestEnclosingThree_EnclosesAllThree(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Three equal circles at vertices of an equilateral triangle.
	s := math.Sqrt(3) / 2 // side = 2, so height = sqrt(3)
	a := enclosure{x: 0, y: 0, radius: 1}
	b := enclosure{x: 2, y: 0, radius: 1}
	c := enclosure{x: 1, y: 2 * s, radius: 1}

	result := enclosingThree(a, b, c)

	g.Expect(encloses(result, a)).To(BeTrue())
	g.Expect(encloses(result, b)).To(BeTrue())
	g.Expect(encloses(result, c)).To(BeTrue())
}

func TestEnclosingThree_CollinearCircles(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Three circles in a row — degenerate (no unique circumscribed circle).
	// Should fall back gracefully and still enclose all three.
	a := enclosure{x: 0, y: 0, radius: 1}
	b := enclosure{x: 4, y: 0, radius: 1}
	c := enclosure{x: 8, y: 0, radius: 1}

	result := enclosingThree(a, b, c)

	g.Expect(encloses(result, a)).To(BeTrue())
	g.Expect(encloses(result, b)).To(BeTrue())
	g.Expect(encloses(result, c)).To(BeTrue())
}

// ---------------------------------------------------------------------------
// computeEnclosing
// ---------------------------------------------------------------------------

func TestComputeEnclosing_Empty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	result := computeEnclosing([]BubbleNode{})

	g.Expect(result.radius).To(Equal(0.0))
}

func TestComputeEnclosing_SingleNode(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	node := BubbleNode{X: 3, Y: 4, Radius: 5}

	result := computeEnclosing([]BubbleNode{node})

	g.Expect(result.x).To(BeNumerically("~", 3.0, 1e-9))
	g.Expect(result.y).To(BeNumerically("~", 4.0, 1e-9))
	g.Expect(result.radius).To(BeNumerically("~", 5.0, 1e-9))
}

func TestComputeEnclosing_TwoNodes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	nodes := []BubbleNode{
		{X: -2, Y: 0, Radius: 1},
		{X: 2, Y: 0, Radius: 1},
	}

	result := computeEnclosing(nodes)

	// Must enclose both nodes as enclosure circles.
	for _, n := range nodes {
		e := enclosure{x: n.X, y: n.Y, radius: n.Radius}
		g.Expect(encloses(result, e)).To(BeTrue(), "enclosing circle must contain node at (%v,%v)", n.X, n.Y)
	}
}

func TestComputeEnclosing_ThreeNodes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	nodes := []BubbleNode{
		{X: 0, Y: 0, Radius: 1},
		{X: 4, Y: 0, Radius: 1},
		{X: 2, Y: 4, Radius: 1},
	}

	result := computeEnclosing(nodes)

	for _, n := range nodes {
		e := enclosure{x: n.X, y: n.Y, radius: n.Radius}
		g.Expect(encloses(result, e)).To(BeTrue(), "enclosing circle must contain node at (%v,%v)", n.X, n.Y)
	}
}

func TestComputeEnclosing_OneNodeInsideAnother(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// The large circle already contains the small one; result should equal the large circle.
	nodes := []BubbleNode{
		{X: 0, Y: 0, Radius: 10},
		{X: 1, Y: 0, Radius: 2},
	}

	result := computeEnclosing(nodes)

	for _, n := range nodes {
		e := enclosure{x: n.X, y: n.Y, radius: n.Radius}
		g.Expect(encloses(result, e)).To(BeTrue())
	}
}
