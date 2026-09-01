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
	a := BubbleNode{Position: geometry.Point{X: 0, Y: 0}, Radius: 5}
	b := BubbleNode{Position: geometry.Point{X: 20, Y: 0}, Radius: 5}

	p1, p2, ok := tangentPositions(5, a, b)

	g.Expect(ok).To(BeTrue())

	// Both candidate positions must be at the correct tangent distances from a and b.
	expectDist := func(p geometry.Point, center BubbleNode, r float64) {
		t.Helper()

		da := a.Radius + r + siblingPadding
		db := center.Radius + r + siblingPadding
		distA := p.DistanceTo(a.Position)
		distCenter := p.DistanceTo(center.Position)

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

	a := BubbleNode{Position: geometry.Point{X: 0, Y: 0}, Radius: 5}
	b := BubbleNode{Position: geometry.Point{X: 0, Y: 0}, Radius: 5}

	_, _, ok := tangentPositions(5, a, b)

	g.Expect(ok).To(BeFalse())
}

func TestTangentPositions_TooFarApart_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Circles with radius 1, 1000 units apart — far exceeds tangent reach.
	a := BubbleNode{Position: geometry.Point{X: 0, Y: 0}, Radius: 1}
	b := BubbleNode{Position: geometry.Point{X: 1000, Y: 0}, Radius: 1}

	_, _, ok := tangentPositions(1, a, b)

	g.Expect(ok).To(BeFalse())
}

func TestTangentPositions_OneInsideOther_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Circle b is entirely inside circle a; no external tangent circle fits.
	a := BubbleNode{Position: geometry.Point{X: 0, Y: 0}, Radius: 50}
	b := BubbleNode{Position: geometry.Point{X: 1, Y: 0}, Radius: 1}

	_, _, ok := tangentPositions(1, a, b)

	g.Expect(ok).To(BeFalse())
}

// ---------------------------------------------------------------------------
// anyOverlap
// ---------------------------------------------------------------------------

func TestAnyOverlap_NoCircles(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(anyOverlap(geometry.Point{X: 0, Y: 0}, 5, nil, -1, -1)).To(BeFalse())
}

func TestAnyOverlap_FarAway_NoOverlap(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	placed := []BubbleNode{
		{Position: geometry.Point{X: 0, Y: 0}, Radius: 5},
		{Position: geometry.Point{X: 100, Y: 0}, Radius: 5},
	}

	g.Expect(anyOverlap(geometry.Point{X: 0, Y: 50}, 5, placed, -1, -1)).To(BeFalse())
}

func TestAnyOverlap_DirectOverlap(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	placed := []BubbleNode{
		{Position: geometry.Point{X: 0, Y: 0}, Radius: 5},
	}

	// Exactly on top of circle 0.
	g.Expect(anyOverlap(geometry.Point{X: 0, Y: 0}, 5, placed, -1, -1)).To(BeTrue())
}

func TestAnyOverlap_SkipsAnchorIndices(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Two circles at the same position — would overlap, but both are skipped.
	placed := []BubbleNode{
		{Position: geometry.Point{X: 0, Y: 0}, Radius: 5},
		{Position: geometry.Point{X: 0, Y: 0}, Radius: 5},
	}

	g.Expect(anyOverlap(geometry.Point{X: 0, Y: 0}, 5, placed, 0, 1)).To(BeFalse())
}

func TestAnyOverlap_SkipsOneAnchor_OverlapsOther(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	placed := []BubbleNode{
		{Position: geometry.Point{X: 0, Y: 0}, Radius: 5}, // skipped (anchor 0)
		{Position: geometry.Point{X: 0, Y: 0}, Radius: 5}, // not skipped — overlaps
	}

	g.Expect(anyOverlap(geometry.Point{X: 0, Y: 0}, 5, placed, 0, -1)).To(BeTrue())
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

	g.Expect(circles[0].Position.X).To(Equal(0.0))
	g.Expect(circles[0].Position.Y).To(Equal(0.0))
}

func TestPackCircles_TwoCircles_AdjacentOnXAxis(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	circles := []BubbleNode{{Radius: 10}, {Radius: 10}}
	packCircles(circles)

	g.Expect(circles[0].Position.X).To(Equal(0.0))
	g.Expect(circles[0].Position.Y).To(Equal(0.0))

	want := circles[0].Radius + circles[1].Radius + siblingPadding
	g.Expect(circles[1].Position.X).To(BeNumerically("~", want, 1e-9))
	g.Expect(circles[1].Position.Y).To(Equal(0.0))
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
			dist := circles[i].Position.DistanceTo(circles[j].Position)
			minDist := circles[i].Radius + circles[j].Radius + siblingPadding - 1e-6
			g.Expect(dist).To(
				BeNumerically(">=", minDist),
				"circles %d (r=%.1f at %.1f,%.1f) and %d (r=%.1f at %.1f,%.1f) overlap",
				i, circles[i].Radius, circles[i].Position.X, circles[i].Position.Y,
				j, circles[j].Radius, circles[j].Position.X, circles[j].Position.Y,
			)
		}
	}
}
