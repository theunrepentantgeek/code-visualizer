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

	outer := geometry.NewCircle(geometry.NewPoint(0, 0), 10)
	inner := geometry.NewCircle(geometry.NewPoint(1, 1), 2)

	g.Expect(enclosesWithin(outer, inner, welzlTolerance)).To(BeTrue())
}

func TestEncloses_SameCircle(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := geometry.NewCircle(geometry.NewPoint(5, 5), 3)

	g.Expect(enclosesWithin(c, c, welzlTolerance)).To(BeTrue())
}

func TestEncloses_OuterTooSmall(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	outer := geometry.NewCircle(geometry.NewPoint(0, 0), 3)
	inner := geometry.NewCircle(geometry.NewPoint(0, 0), 5)

	g.Expect(enclosesWithin(outer, inner, welzlTolerance)).To(BeFalse())
}

func TestEncloses_TouchingExternally(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := geometry.NewCircle(geometry.NewPoint(0, 0), 3)
	b := geometry.NewCircle(geometry.NewPoint(6, 0), 3) // centres 6 apart, radii sum to 6

	// Neither encloses the other.
	g.Expect(enclosesWithin(a, b, welzlTolerance)).To(BeFalse())
	g.Expect(enclosesWithin(b, a, welzlTolerance)).To(BeFalse())
}

// ---------------------------------------------------------------------------
// enclosingTwo
// ---------------------------------------------------------------------------

func TestEnclosingTwo_AContainsB(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// B is entirely inside A, so the enclosing circle should equal A.
	a := geometry.NewCircle(geometry.NewPoint(0, 0), 10)
	b := geometry.NewCircle(geometry.NewPoint(1, 0), 2)

	result := enclosingTwo(a, b)

	g.Expect(result.Radius).To(BeNumerically("~", a.Radius, 1e-9))
	g.Expect(result.Center.X).To(BeNumerically("~", a.Center.X, 1e-9))
	g.Expect(result.Center.Y).To(BeNumerically("~", a.Center.Y, 1e-9))
}

func TestEnclosingTwo_BContainsA(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// A is entirely inside B.
	a := geometry.NewCircle(geometry.NewPoint(1, 0), 2)
	b := geometry.NewCircle(geometry.NewPoint(0, 0), 10)

	result := enclosingTwo(a, b)

	g.Expect(result.Radius).To(BeNumerically("~", b.Radius, 1e-9))
}

func TestEnclosingTwo_EqualCirclesSideBySide(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Two circles of radius 1 with centres at (-1,0) and (1,0).
	// The minimum enclosing circle has centre at (0,0) and radius 2.
	a := geometry.NewCircle(geometry.NewPoint(-1, 0), 1)
	b := geometry.NewCircle(geometry.NewPoint(1, 0), 1)

	result := enclosingTwo(a, b)

	g.Expect(result.Center.X).To(BeNumerically("~", 0.0, 1e-9))
	g.Expect(result.Center.Y).To(BeNumerically("~", 0.0, 1e-9))
	g.Expect(result.Radius).To(BeNumerically("~", 2.0, 1e-9))

	// Verify it actually encloses both.
	g.Expect(enclosesWithin(result, a, welzlTolerance)).To(BeTrue())
	g.Expect(enclosesWithin(result, b, welzlTolerance)).To(BeTrue())
}

func TestEnclosingTwo_DifferentRadii(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := geometry.NewCircle(geometry.NewPoint(0, 0), 1)
	b := geometry.NewCircle(geometry.NewPoint(3, 0), 2)

	result := enclosingTwo(a, b)

	// Enclosing circle must contain both.
	g.Expect(enclosesWithin(result, a, welzlTolerance)).To(BeTrue())
	g.Expect(enclosesWithin(result, b, welzlTolerance)).To(BeTrue())
}

func TestEnclosingTwo_PreservesPrePointEvaluationOrder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := geometry.NewCircle(geometry.NewPoint(-100.7, -100.7), 0.1)
	b := geometry.NewCircle(geometry.NewPoint(-19.7, 100.1), 0.2)

	result := enclosingTwo(a, b)

	g.Expect(result).To(Equal(geometry.NewCircle(
		geometry.NewPoint(-60.181295176031675, -0.2536305104587626),
		108.41084241312738,
	)))
}

// ---------------------------------------------------------------------------
// enclosingThree
// ---------------------------------------------------------------------------

func TestEnclosingThree_EnclosesAllThree(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Three equal circles at vertices of an equilateral triangle.
	s := math.Sqrt(3) / 2 // side = 2, so height = sqrt(3)
	a := geometry.NewCircle(geometry.NewPoint(0, 0), 1)
	b := geometry.NewCircle(geometry.NewPoint(2, 0), 1)
	c := geometry.NewCircle(geometry.NewPoint(1, 2*s), 1)

	result := enclosingThree(a, b, c)

	g.Expect(enclosesWithin(result, a, welzlTolerance)).To(BeTrue())
	g.Expect(enclosesWithin(result, b, welzlTolerance)).To(BeTrue())
	g.Expect(enclosesWithin(result, c, welzlTolerance)).To(BeTrue())
}

func TestEnclosingThree_CollinearCircles(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Three circles in a row — degenerate (no unique circumscribed circle).
	// Should fall back gracefully and still enclose all three.
	a := geometry.NewCircle(geometry.NewPoint(0, 0), 1)
	b := geometry.NewCircle(geometry.NewPoint(4, 0), 1)
	c := geometry.NewCircle(geometry.NewPoint(8, 0), 1)

	result := enclosingThree(a, b, c)

	g.Expect(enclosesWithin(result, a, welzlTolerance)).To(BeTrue())
	g.Expect(enclosesWithin(result, b, welzlTolerance)).To(BeTrue())
	g.Expect(enclosesWithin(result, c, welzlTolerance)).To(BeTrue())
}

// ---------------------------------------------------------------------------
// computeEnclosing
// ---------------------------------------------------------------------------

func TestComputeEnclosing_Empty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	result := computeEnclosing([]BubbleNode{})

	g.Expect(result.Radius).To(Equal(0.0))
}

func TestComputeEnclosing_SingleNode(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	node := BubbleNode{Geometry: geometry.NewCircle(geometry.NewPoint(3, 4), 5)}

	result := computeEnclosing([]BubbleNode{node})

	g.Expect(result.Center.X).To(BeNumerically("~", 3.0, 1e-9))
	g.Expect(result.Center.Y).To(BeNumerically("~", 4.0, 1e-9))
	g.Expect(result.Radius).To(BeNumerically("~", 5.0, 1e-9))
}

func TestComputeEnclosing_TwoNodes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	nodes := []BubbleNode{
		{Geometry: geometry.NewCircle(geometry.NewPoint(-2, 0), 1)},
		{Geometry: geometry.NewCircle(geometry.NewPoint(2, 0), 1)},
	}

	result := computeEnclosing(nodes)

	// Must enclose both nodes as enclosure circles.
	for _, n := range nodes {
		e := geometry.NewCircle(n.Geometry.Center, n.Geometry.Radius)
		g.Expect(enclosesWithin(result, e, welzlTolerance)).To(
			BeTrue(),
			"enclosing circle must contain node at (%v,%v)",
			n.Geometry.Center.X,
			n.Geometry.Center.Y,
		)
	}
}

func TestComputeEnclosing_ThreeNodes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	nodes := []BubbleNode{
		{Geometry: geometry.NewCircle(geometry.NewPoint(0, 0), 1)},
		{Geometry: geometry.NewCircle(geometry.NewPoint(4, 0), 1)},
		{Geometry: geometry.NewCircle(geometry.NewPoint(2, 4), 1)},
	}

	result := computeEnclosing(nodes)

	for _, n := range nodes {
		e := geometry.NewCircle(n.Geometry.Center, n.Geometry.Radius)
		g.Expect(enclosesWithin(result, e, welzlTolerance)).To(
			BeTrue(),
			"enclosing circle must contain node at (%v,%v)",
			n.Geometry.Center.X,
			n.Geometry.Center.Y,
		)
	}
}

func TestComputeEnclosing_OneNodeInsideAnother(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// The large circle already contains the small one; result should equal the large circle.
	nodes := []BubbleNode{
		{Geometry: geometry.NewCircle(geometry.NewPoint(0, 0), 10)},
		{Geometry: geometry.NewCircle(geometry.NewPoint(1, 0), 2)},
	}

	result := computeEnclosing(nodes)

	for _, n := range nodes {
		e := geometry.NewCircle(n.Geometry.Center, n.Geometry.Radius)
		g.Expect(enclosesWithin(result, e, welzlTolerance)).To(BeTrue())
	}
}
