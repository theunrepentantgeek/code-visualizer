package bubbletree

import (
	"math"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

// ---------------------------------------------------------------------------
// encloses
// ---------------------------------------------------------------------------

func TestEncloses_ContainedCircle(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	outer := enclosure{center: geometry.Point{X: 0, Y: 0}, radius: 10}
	inner := enclosure{center: geometry.Point{X: 1, Y: 1}, radius: 2}

	g.Expect(encloses(outer, inner)).To(BeTrue())
}

func TestEncloses_SameCircle(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := enclosure{center: geometry.Point{X: 5, Y: 5}, radius: 3}

	g.Expect(encloses(c, c)).To(BeTrue())
}

func TestEncloses_OuterTooSmall(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	outer := enclosure{center: geometry.Point{X: 0, Y: 0}, radius: 3}
	inner := enclosure{center: geometry.Point{X: 0, Y: 0}, radius: 5}

	g.Expect(encloses(outer, inner)).To(BeFalse())
}

func TestEncloses_TouchingExternally(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := enclosure{center: geometry.Point{X: 0, Y: 0}, radius: 3}
	b := enclosure{center: geometry.Point{X: 6, Y: 0}, radius: 3} // centres 6 apart, radii sum to 6

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
	a := enclosure{center: geometry.Point{X: 0, Y: 0}, radius: 10}
	b := enclosure{center: geometry.Point{X: 1, Y: 0}, radius: 2}

	result := enclosingTwo(a, b)

	g.Expect(result.radius).To(BeNumerically("~", a.radius, 1e-9))
	g.Expect(result.center.X).To(BeNumerically("~", a.center.X, 1e-9))
	g.Expect(result.center.Y).To(BeNumerically("~", a.center.Y, 1e-9))
}

func TestEnclosingTwo_BContainsA(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// A is entirely inside B.
	a := enclosure{center: geometry.Point{X: 1, Y: 0}, radius: 2}
	b := enclosure{center: geometry.Point{X: 0, Y: 0}, radius: 10}

	result := enclosingTwo(a, b)

	g.Expect(result.radius).To(BeNumerically("~", b.radius, 1e-9))
}

func TestEnclosingTwo_EqualCirclesSideBySide(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Two circles of radius 1 with centres at (-1,0) and (1,0).
	// The minimum enclosing circle has centre at (0,0) and radius 2.
	a := enclosure{center: geometry.Point{X: -1, Y: 0}, radius: 1}
	b := enclosure{center: geometry.Point{X: 1, Y: 0}, radius: 1}

	result := enclosingTwo(a, b)

	g.Expect(result.center.X).To(BeNumerically("~", 0.0, 1e-9))
	g.Expect(result.center.Y).To(BeNumerically("~", 0.0, 1e-9))
	g.Expect(result.radius).To(BeNumerically("~", 2.0, 1e-9))

	// Verify it actually encloses both.
	g.Expect(encloses(result, a)).To(BeTrue())
	g.Expect(encloses(result, b)).To(BeTrue())
}

func TestEnclosingTwo_DifferentRadii(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := enclosure{center: geometry.Point{X: 0, Y: 0}, radius: 1}
	b := enclosure{center: geometry.Point{X: 3, Y: 0}, radius: 2}

	result := enclosingTwo(a, b)

	// Enclosing circle must contain both.
	g.Expect(encloses(result, a)).To(BeTrue())
	g.Expect(encloses(result, b)).To(BeTrue())
}

func TestEnclosingTwo_PreservesPrePointEvaluationOrder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := enclosure{center: geometry.Point{X: -100.7, Y: -100.7}, radius: 0.1}
	b := enclosure{center: geometry.Point{X: -19.7, Y: 100.1}, radius: 0.2}

	result := enclosingTwo(a, b)

	g.Expect(result).To(Equal(enclosure{
		center: geometry.Point{X: -60.181295176031675, Y: -0.2536305104587626},
		radius: 108.41084241312738,
	}))
}

// ---------------------------------------------------------------------------
// enclosingThree
// ---------------------------------------------------------------------------

func TestEnclosingThree_EnclosesAllThree(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Three equal circles at vertices of an equilateral triangle.
	s := math.Sqrt(3) / 2 // side = 2, so height = sqrt(3)
	a := enclosure{center: geometry.Point{X: 0, Y: 0}, radius: 1}
	b := enclosure{center: geometry.Point{X: 2, Y: 0}, radius: 1}
	c := enclosure{center: geometry.Point{X: 1, Y: 2 * s}, radius: 1}

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
	a := enclosure{center: geometry.Point{X: 0, Y: 0}, radius: 1}
	b := enclosure{center: geometry.Point{X: 4, Y: 0}, radius: 1}
	c := enclosure{center: geometry.Point{X: 8, Y: 0}, radius: 1}

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

	node := BubbleNode{Position: geometry.Point{X: 3, Y: 4}, Radius: 5}

	result := computeEnclosing([]BubbleNode{node})

	g.Expect(result.center.X).To(BeNumerically("~", 3.0, 1e-9))
	g.Expect(result.center.Y).To(BeNumerically("~", 4.0, 1e-9))
	g.Expect(result.radius).To(BeNumerically("~", 5.0, 1e-9))
}

func TestComputeEnclosing_TwoNodes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	nodes := []BubbleNode{
		{Position: geometry.Point{X: -2, Y: 0}, Radius: 1},
		{Position: geometry.Point{X: 2, Y: 0}, Radius: 1},
	}

	result := computeEnclosing(nodes)

	// Must enclose both nodes as enclosure circles.
	for _, n := range nodes {
		e := enclosure{center: n.Position, radius: n.Radius}
		g.Expect(encloses(result, e)).To(
			BeTrue(),
			"enclosing circle must contain node at (%v,%v)",
			n.Position.X,
			n.Position.Y,
		)
	}
}

func TestComputeEnclosing_ThreeNodes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	nodes := []BubbleNode{
		{Position: geometry.Point{X: 0, Y: 0}, Radius: 1},
		{Position: geometry.Point{X: 4, Y: 0}, Radius: 1},
		{Position: geometry.Point{X: 2, Y: 4}, Radius: 1},
	}

	result := computeEnclosing(nodes)

	for _, n := range nodes {
		e := enclosure{center: n.Position, radius: n.Radius}
		g.Expect(encloses(result, e)).To(
			BeTrue(),
			"enclosing circle must contain node at (%v,%v)",
			n.Position.X,
			n.Position.Y,
		)
	}
}

func TestComputeEnclosing_OneNodeInsideAnother(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// The large circle already contains the small one; result should equal the large circle.
	nodes := []BubbleNode{
		{Position: geometry.Point{X: 0, Y: 0}, Radius: 10},
		{Position: geometry.Point{X: 1, Y: 0}, Radius: 2},
	}

	result := computeEnclosing(nodes)

	for _, n := range nodes {
		e := enclosure{center: n.Position, radius: n.Radius}
		g.Expect(encloses(result, e)).To(BeTrue())
	}
}
