package bubbletree

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

// ---------------------------------------------------------------------------
// tangentPositions
// ---------------------------------------------------------------------------

func TestTangentPositions_SymmetricCircles(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Two equal circles 20 apart (centre-to-centre), new circle radius 5.
	a := BubbleNode{Geometry: geometry.NewCircle(geometry.NewPoint(0, 0), 5)}
	b := BubbleNode{Geometry: geometry.NewCircle(geometry.NewPoint(20, 0), 5)}

	p1, p2, ok := tangentPositions(5, a, b)

	g.Expect(ok).To(BeTrue())

	// Both candidate positions must be at the correct tangent distances from a and b.
	expectDist := func(p geometry.Point, center BubbleNode, r float64) {
		t.Helper()

		da := a.Geometry.Radius + r + siblingPadding
		db := center.Geometry.Radius + r + siblingPadding
		distA := p.DistanceTo(a.Geometry.Center)
		distCenter := p.DistanceTo(center.Geometry.Center)

		g.Expect(distA).To(BeNumerically("~", da, 1e-9))
		g.Expect(distCenter).To(BeNumerically("~", db, 1e-9))
	}

	expectDist(p1, b, 5)
	expectDist(p2, b, 5)

	// p1 and p2 are reflections across the x-axis.
	g.Expect(p1.X).To(BeNumerically("~", p2.X, 1e-9))
	g.Expect(p1.Y).To(BeNumerically("~", -p2.Y, 1e-9))
}

func TestTangentPositions_CoincidentCentres_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := BubbleNode{Geometry: geometry.NewCircle(geometry.NewPoint(0, 0), 5)}
	b := BubbleNode{Geometry: geometry.NewCircle(geometry.NewPoint(0, 0), 5)}

	_, _, ok := tangentPositions(5, a, b)

	g.Expect(ok).To(BeFalse())
}

func TestTangentPositions_TooFarApart_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Circles with radius 1, 1000 units apart — far exceeds tangent reach.
	a := BubbleNode{Geometry: geometry.NewCircle(geometry.NewPoint(0, 0), 1)}
	b := BubbleNode{Geometry: geometry.NewCircle(geometry.NewPoint(1000, 0), 1)}

	_, _, ok := tangentPositions(1, a, b)

	g.Expect(ok).To(BeFalse())
}

func TestTangentPositions_OneInsideOther_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Circle b is entirely inside circle a; no external tangent circle fits.
	a := BubbleNode{Geometry: geometry.NewCircle(geometry.NewPoint(0, 0), 50)}
	b := BubbleNode{Geometry: geometry.NewCircle(geometry.NewPoint(1, 0), 1)}

	_, _, ok := tangentPositions(1, a, b)

	g.Expect(ok).To(BeFalse())
}

func TestTangentPositions_PreservesPrePointEvaluationOrder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := BubbleNode{Geometry: geometry.NewCircle(geometry.NewPoint(-10.7, -10.7), 0.1)}
	b := BubbleNode{Geometry: geometry.NewCircle(geometry.NewPoint(-5.3, 0.1), 0.2)}

	p1, p2, ok := tangentPositions(3.9, a, b)

	g.Expect(ok).To(BeTrue())
	g.Expect(p1).To(Equal(geometry.NewPoint(
		-4.770151522985639,
		-6.980202016284957,
	)))
	g.Expect(p2).To(Equal(geometry.NewPoint(
		-11.282070699236581,
		-3.724242428159486,
	)))
}

func TestPlaceFallback_PreservesPrePointDistanceEvaluationOrder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	circles := []BubbleNode{
		{Geometry: geometry.NewCircle(geometry.NewPoint(-1000.3, -100.7), 0.1)},
		{Geometry: geometry.NewCircle(geometry.OriginPoint, 0.1)},
	}

	placeFallback(circles, 1)

	g.Expect(circles[1].Geometry.Center).To(Equal(geometry.NewPoint(
		-743.6777670569004,
		681.2697533617113,
	)))
}

// ---------------------------------------------------------------------------
// anyOverlap
// ---------------------------------------------------------------------------

func TestAnyOverlap_NoCircles(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(anyOverlap(geometry.NewPoint(0, 0), 5, nil, -1, -1)).To(BeFalse())
}

func TestAnyOverlap_FarAway_NoOverlap(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	placed := []BubbleNode{
		{Geometry: geometry.NewCircle(geometry.NewPoint(0, 0), 5)},
		{Geometry: geometry.NewCircle(geometry.NewPoint(100, 0), 5)},
	}

	g.Expect(anyOverlap(geometry.NewPoint(0, 50), 5, placed, -1, -1)).To(BeFalse())
}

func TestAnyOverlap_DirectOverlap(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	placed := []BubbleNode{
		{Geometry: geometry.NewCircle(geometry.NewPoint(0, 0), 5)},
	}

	// Exactly on top of circle 0.
	g.Expect(anyOverlap(geometry.NewPoint(0, 0), 5, placed, -1, -1)).To(BeTrue())
}

func TestAnyOverlap_SkipsAnchorIndices(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Two circles at the same position — would overlap, but both are skipped.
	placed := []BubbleNode{
		{Geometry: geometry.NewCircle(geometry.NewPoint(0, 0), 5)},
		{Geometry: geometry.NewCircle(geometry.NewPoint(0, 0), 5)},
	}

	g.Expect(anyOverlap(geometry.NewPoint(0, 0), 5, placed, 0, 1)).To(BeFalse())
}

func TestAnyOverlap_SkipsOneAnchor_OverlapsOther(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	placed := []BubbleNode{
		{Geometry: geometry.NewCircle(geometry.NewPoint(0, 0), 5)},
		{Geometry: geometry.NewCircle(geometry.NewPoint(0, 0), 5)},
	}

	g.Expect(anyOverlap(geometry.NewPoint(0, 0), 5, placed, 0, -1)).To(BeTrue())
}

// ---------------------------------------------------------------------------
// packCircles
// ---------------------------------------------------------------------------

func TestPackCircles_Empty_NoPanic(t *testing.T) {
	t.Parallel()

	circles := []BubbleNode{}
	packCircles(circles) // must not panic
}

func TestPackCircles_SingleCircle_AtOrigin(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	circles := []BubbleNode{{Geometry: geometry.NewCircle(geometry.OriginPoint, 10)}}
	packCircles(circles)

	g.Expect(circles[0].Geometry.Center.X).To(Equal(0.0))
	g.Expect(circles[0].Geometry.Center.Y).To(Equal(0.0))
}

func TestPackCircles_TwoCircles_AdjacentOnXAxis(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	circles := []BubbleNode{
		{Geometry: geometry.NewCircle(geometry.OriginPoint, 10)},
		{Geometry: geometry.NewCircle(geometry.OriginPoint, 10)},
	}
	packCircles(circles)

	g.Expect(circles[0].Geometry.Center.X).To(Equal(0.0))
	g.Expect(circles[0].Geometry.Center.Y).To(Equal(0.0))

	want := circles[0].Geometry.Radius + circles[1].Geometry.Radius + siblingPadding
	g.Expect(circles[1].Geometry.Center.X).To(BeNumerically("~", want, 1e-9))
	g.Expect(circles[1].Geometry.Center.Y).To(Equal(0.0))
}

func TestPackCircles_ThreeCircles_NoOverlap(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	circles := []BubbleNode{
		{Geometry: geometry.NewCircle(geometry.OriginPoint, 10)},
		{Geometry: geometry.NewCircle(geometry.OriginPoint, 10)},
		{Geometry: geometry.NewCircle(geometry.OriginPoint, 10)},
	}
	packCircles(circles)

	assertNoOverlaps(t, g, circles)
}

func TestPackCircles_ManyEqualCircles_NoOverlap(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	circles := make([]BubbleNode, 20)
	for i := range circles {
		circles[i].Geometry.Radius = 10
	}

	packCircles(circles)

	assertNoOverlaps(t, g, circles)
}

func TestPackCircles_VaryingRadii_NoOverlap(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	circles := make([]BubbleNode, 12)
	for i := range circles {
		circles[i].Geometry.Radius = float64(i+1) * 5
	}

	packCircles(circles)

	assertNoOverlaps(t, g, circles)
}

func TestPackCircles_LargeCircleFirst_NoOverlap(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// One very large circle followed by many small ones.
	circles := make([]BubbleNode, 10)

	circles[0].Geometry.Radius = 100
	for i := 1; i < len(circles); i++ {
		circles[i].Geometry.Radius = 5
	}

	packCircles(circles)

	assertNoOverlaps(t, g, circles)
}

// ---------------------------------------------------------------------------
// assertNoOverlaps verifies that no two circles in the slice overlap.
// ---------------------------------------------------------------------------.
func assertNoOverlaps(t *testing.T, g *GomegaWithT, circles []BubbleNode) {
	t.Helper()

	for i := range circles {
		for j := i + 1; j < len(circles); j++ {
			dist := circles[i].Geometry.Center.DistanceTo(circles[j].Geometry.Center)
			minDist := circles[i].Geometry.Radius + circles[j].Geometry.Radius + siblingPadding - 1e-6
			g.Expect(dist).To(
				BeNumerically(">=", minDist),
				"circles %d (r=%.1f at %.1f,%.1f) and %d (r=%.1f at %.1f,%.1f) overlap",
				i, circles[i].Geometry.Radius, circles[i].Geometry.Center.X, circles[i].Geometry.Center.Y,
				j, circles[j].Geometry.Radius, circles[j].Geometry.Center.X, circles[j].Geometry.Center.Y,
			)
		}
	}
}
