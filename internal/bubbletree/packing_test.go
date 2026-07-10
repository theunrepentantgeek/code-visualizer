package bubbletree

import (
	"math"
	"testing"

	. "github.com/onsi/gomega"
)

// ---------------------------------------------------------------------------
// tangentPositions
// ---------------------------------------------------------------------------

func TestTangentPositions_SymmetricCircles(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Two equal circles 20 apart (centre-to-centre), new circle radius 5.
	a := BubbleNode{X: 0, Y: 0, Radius: 5}
	b := BubbleNode{X: 20, Y: 0, Radius: 5}

	p1, p2, ok := tangentPositions(5, a, b)

	g.Expect(ok).To(BeTrue())

	// Both candidate positions must be at the correct tangent distances from a and b.
	expectDist := func(p point, center BubbleNode, r float64) {
		t.Helper()

		da := a.Radius + r + siblingPadding
		db := center.Radius + r + siblingPadding
		distA := math.Sqrt((p.x-a.X)*(p.x-a.X) + (p.y-a.Y)*(p.y-a.Y))
		distCenter := math.Sqrt((p.x-center.X)*(p.x-center.X) + (p.y-center.Y)*(p.y-center.Y))

		g.Expect(distA).To(BeNumerically("~", da, 1e-9))
		g.Expect(distCenter).To(BeNumerically("~", db, 1e-9))
	}

	expectDist(p1, b, 5)
	expectDist(p2, b, 5)

	// p1 and p2 are reflections across the x-axis.
	g.Expect(p1.x).To(BeNumerically("~", p2.x, 1e-9))
	g.Expect(p1.y).To(BeNumerically("~", -p2.y, 1e-9))
}

func TestTangentPositions_CoincidentCentres_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := BubbleNode{X: 0, Y: 0, Radius: 5}
	b := BubbleNode{X: 0, Y: 0, Radius: 5}

	_, _, ok := tangentPositions(5, a, b)

	g.Expect(ok).To(BeFalse())
}

func TestTangentPositions_TooFarApart_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Circles with radius 1, 1000 units apart — far exceeds tangent reach.
	a := BubbleNode{X: 0, Y: 0, Radius: 1}
	b := BubbleNode{X: 1000, Y: 0, Radius: 1}

	_, _, ok := tangentPositions(1, a, b)

	g.Expect(ok).To(BeFalse())
}

func TestTangentPositions_OneInsideOther_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Circle b is entirely inside circle a; no external tangent circle fits.
	a := BubbleNode{X: 0, Y: 0, Radius: 50}
	b := BubbleNode{X: 1, Y: 0, Radius: 1}

	_, _, ok := tangentPositions(1, a, b)

	g.Expect(ok).To(BeFalse())
}

// ---------------------------------------------------------------------------
// anyOverlap
// ---------------------------------------------------------------------------

func TestAnyOverlap_NoCircles(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(anyOverlap(point{0, 0}, 5, nil, -1, -1)).To(BeFalse())
}

func TestAnyOverlap_FarAway_NoOverlap(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	placed := []BubbleNode{
		{X: 0, Y: 0, Radius: 5},
		{X: 100, Y: 0, Radius: 5},
	}

	g.Expect(anyOverlap(point{0, 50}, 5, placed, -1, -1)).To(BeFalse())
}

func TestAnyOverlap_DirectOverlap(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	placed := []BubbleNode{
		{X: 0, Y: 0, Radius: 5},
	}

	// Exactly on top of circle 0.
	g.Expect(anyOverlap(point{0, 0}, 5, placed, -1, -1)).To(BeTrue())
}

func TestAnyOverlap_SkipsAnchorIndices(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Two circles at the same position — would overlap, but both are skipped.
	placed := []BubbleNode{
		{X: 0, Y: 0, Radius: 5},
		{X: 0, Y: 0, Radius: 5},
	}

	g.Expect(anyOverlap(point{0, 0}, 5, placed, 0, 1)).To(BeFalse())
}

func TestAnyOverlap_SkipsOneAnchor_OverlapsOther(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	placed := []BubbleNode{
		{X: 0, Y: 0, Radius: 5}, // skipped (anchor 0)
		{X: 0, Y: 0, Radius: 5}, // not skipped — overlaps
	}

	g.Expect(anyOverlap(point{0, 0}, 5, placed, 0, -1)).To(BeTrue())
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

	circles := []BubbleNode{{Radius: 10}}
	packCircles(circles)

	g.Expect(circles[0].X).To(Equal(0.0))
	g.Expect(circles[0].Y).To(Equal(0.0))
}

func TestPackCircles_TwoCircles_AdjacentOnXAxis(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	circles := []BubbleNode{{Radius: 10}, {Radius: 10}}
	packCircles(circles)

	g.Expect(circles[0].X).To(Equal(0.0))
	g.Expect(circles[0].Y).To(Equal(0.0))

	want := circles[0].Radius + circles[1].Radius + siblingPadding
	g.Expect(circles[1].X).To(BeNumerically("~", want, 1e-9))
	g.Expect(circles[1].Y).To(Equal(0.0))
}

func TestPackCircles_ThreeCircles_NoOverlap(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	circles := []BubbleNode{
		{Radius: 10},
		{Radius: 10},
		{Radius: 10},
	}
	packCircles(circles)

	assertNoOverlaps(t, g, circles)
}

func TestPackCircles_ManyEqualCircles_NoOverlap(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	circles := make([]BubbleNode, 20)
	for i := range circles {
		circles[i].Radius = 10
	}

	packCircles(circles)

	assertNoOverlaps(t, g, circles)
}

func TestPackCircles_VaryingRadii_NoOverlap(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	circles := make([]BubbleNode, 12)
	for i := range circles {
		circles[i].Radius = float64(i+1) * 5
	}

	packCircles(circles)

	assertNoOverlaps(t, g, circles)
}

func TestPackCircles_LargeCircleFirst_NoOverlap(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// One very large circle followed by many small ones.
	circles := make([]BubbleNode, 10)

	circles[0].Radius = 100
	for i := 1; i < len(circles); i++ {
		circles[i].Radius = 5
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
			dx := circles[i].X - circles[j].X
			dy := circles[i].Y - circles[j].Y
			dist := math.Sqrt(dx*dx + dy*dy)
			minDist := circles[i].Radius + circles[j].Radius + siblingPadding - 1e-6
			g.Expect(dist).To(
				BeNumerically(">=", minDist),
				"circles %d (r=%.1f at %.1f,%.1f) and %d (r=%.1f at %.1f,%.1f) overlap",
				i, circles[i].Radius, circles[i].X, circles[i].Y,
				j, circles[j].Radius, circles[j].X, circles[j].Y,
			)
		}
	}
}
