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

	outer := geometry.Circle{Center: geometry.Point{X: 0, Y: 0}, Radius: 10}
	inner := geometry.Circle{Center: geometry.Point{X: 1, Y: 1}, Radius: 2}

	g.Expect(enclosesWithin(outer, inner, welzlTolerance)).To(BeTrue())
}

func TestEncloses_SameCircle(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := geometry.Circle{Center: geometry.Point{X: 5, Y: 5}, Radius: 3}

	g.Expect(enclosesWithin(c, c, welzlTolerance)).To(BeTrue())
}

func TestEncloses_OuterTooSmall(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	outer := geometry.Circle{Center: geometry.Point{X: 0, Y: 0}, Radius: 3}
	inner := geometry.Circle{Center: geometry.Point{X: 0, Y: 0}, Radius: 5}

	g.Expect(enclosesWithin(outer, inner, welzlTolerance)).To(BeFalse())
}

func TestEncloses_TouchingExternally(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := geometry.Circle{Center: geometry.Point{X: 0, Y: 0}, Radius: 3}
	b := geometry.Circle{Center: geometry.Point{X: 6, Y: 0}, Radius: 3} // centres 6 apart, radii sum to 6

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
	a := geometry.Circle{Center: geometry.Point{X: 0, Y: 0}, Radius: 10}
	b := geometry.Circle{Center: geometry.Point{X: 1, Y: 0}, Radius: 2}

	result := enclosingTwo(a, b)

	g.Expect(result.Radius).To(BeNumerically("~", a.Radius, 1e-9))
	g.Expect(result.Center.X).To(BeNumerically("~", a.Center.X, 1e-9))
	g.Expect(result.Center.Y).To(BeNumerically("~", a.Center.Y, 1e-9))
}

func TestEnclosingTwo_BContainsA(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// A is entirely inside B.
	a := geometry.Circle{Center: geometry.Point{X: 1, Y: 0}, Radius: 2}
	b := geometry.Circle{Center: geometry.Point{X: 0, Y: 0}, Radius: 10}

	result := enclosingTwo(a, b)

	g.Expect(result.Radius).To(BeNumerically("~", b.Radius, 1e-9))
}

func TestEnclosingTwo_EqualCirclesSideBySide(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Two circles of radius 1 with centres at (-1,0) and (1,0).
	// The minimum enclosing circle has centre at (0,0) and radius 2.
	a := geometry.Circle{Center: geometry.Point{X: -1, Y: 0}, Radius: 1}
	b := geometry.Circle{Center: geometry.Point{X: 1, Y: 0}, Radius: 1}

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

	a := geometry.Circle{Center: geometry.Point{X: 0, Y: 0}, Radius: 1}
	b := geometry.Circle{Center: geometry.Point{X: 3, Y: 0}, Radius: 2}

	result := enclosingTwo(a, b)

	// Enclosing circle must contain both.
	g.Expect(enclosesWithin(result, a, welzlTolerance)).To(BeTrue())
	g.Expect(enclosesWithin(result, b, welzlTolerance)).To(BeTrue())
}

func TestEnclosingTwo_PreservesPrePointEvaluationOrder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := geometry.Circle{Center: geometry.Point{X: -100.7, Y: -100.7}, Radius: 0.1}
	b := geometry.Circle{Center: geometry.Point{X: -19.7, Y: 100.1}, Radius: 0.2}

	result := enclosingTwo(a, b)

	g.Expect(result).To(Equal(geometry.Circle{
		Center: geometry.Point{X: -60.181295176031675, Y: -0.2536305104587626},
		Radius: 108.41084241312738,
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
	a := geometry.Circle{Center: geometry.Point{X: 0, Y: 0}, Radius: 1}
	b := geometry.Circle{Center: geometry.Point{X: 2, Y: 0}, Radius: 1}
	c := geometry.Circle{Center: geometry.Point{X: 1, Y: 2 * s}, Radius: 1}

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
	a := geometry.Circle{Center: geometry.Point{X: 0, Y: 0}, Radius: 1}
	b := geometry.Circle{Center: geometry.Point{X: 4, Y: 0}, Radius: 1}
	c := geometry.Circle{Center: geometry.Point{X: 8, Y: 0}, Radius: 1}

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

	node := BubbleNode{Geometry: geometry.Circle{Center: geometry.Point{X: 3, Y: 4}, Radius: 5}}

	result := computeEnclosing([]BubbleNode{node})

	g.Expect(result.Center.X).To(BeNumerically("~", 3.0, 1e-9))
	g.Expect(result.Center.Y).To(BeNumerically("~", 4.0, 1e-9))
	g.Expect(result.Radius).To(BeNumerically("~", 5.0, 1e-9))
}

func TestComputeEnclosing_TwoNodes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	nodes := []BubbleNode{
		{Geometry: geometry.Circle{Center: geometry.Point{X: -2, Y: 0}, Radius: 1}},
		{Geometry: geometry.Circle{Center: geometry.Point{X: 2, Y: 0}, Radius: 1}},
	}

	result := computeEnclosing(nodes)

	// Must enclose both nodes as enclosure circles.
	for _, n := range nodes {
		e := geometry.Circle{Center: n.Geometry.Center, Radius: n.Geometry.Radius}
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
		{Geometry: geometry.Circle{Center: geometry.Point{X: 0, Y: 0}, Radius: 1}},
		{Geometry: geometry.Circle{Center: geometry.Point{X: 4, Y: 0}, Radius: 1}},
		{Geometry: geometry.Circle{Center: geometry.Point{X: 2, Y: 4}, Radius: 1}},
	}

	result := computeEnclosing(nodes)

	for _, n := range nodes {
		e := geometry.Circle{Center: n.Geometry.Center, Radius: n.Geometry.Radius}
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
		{Geometry: geometry.Circle{Center: geometry.Point{X: 0, Y: 0}, Radius: 10}},
		{Geometry: geometry.Circle{Center: geometry.Point{X: 1, Y: 0}, Radius: 2}},
	}

	result := computeEnclosing(nodes)

	for _, n := range nodes {
		e := geometry.Circle{Center: n.Geometry.Center, Radius: n.Geometry.Radius}
		g.Expect(enclosesWithin(result, e, welzlTolerance)).To(BeTrue())
	}
}
