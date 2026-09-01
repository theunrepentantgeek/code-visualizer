package surface

import (
	"math"
	"testing"

	"github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

func TestTriangleInRegion_RejectsAnnulusHoleCrossingEdge(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	region := Annulus{InnerRadius: 100, OuterRadius: 200}
	triangle := Triangle{
		Points: [3]Sample{
			{Position: geometry.Point{X: 100, Y: 0}},
			{Position: geometry.Point{X: 0, Y: 100}},
			{Position: geometry.Point{X: 140, Y: 140}},
		},
	}

	for _, point := range triangle.Points {
		g.Expect(region.Contains(point.Position.X, point.Position.Y)).To(gomega.BeTrue())
	}

	centroid := Sample{
		Position: geometry.Point{
			X: (triangle.Points[0].Position.X + triangle.Points[1].Position.X + triangle.Points[2].Position.X) / 3,
			Y: (triangle.Points[0].Position.Y + triangle.Points[1].Position.Y + triangle.Points[2].Position.Y) / 3,
		},
	}
	g.Expect(region.Contains(centroid.Position.X, centroid.Position.Y)).To(gomega.BeTrue())

	g.Expect(triangleInRegion(region, triangle)).To(gomega.BeFalse())
}

func TestTriangleInRegion_RejectsAnnulusCenterEnclosedAfterInnerBoundaryPruning(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	region := Annulus{InnerRadius: 1, OuterRadius: 8}
	originals := []Sample{
		{Position: geometry.Point{X: 4, Y: 0}},
		{
			Position: geometry.Point{
				X: 4 * math.Cos(2*math.Pi/3),
				Y: 4 * math.Sin(2*math.Pi/3),
			},
		},
		{
			Position: geometry.Point{
				X: 4 * math.Cos(4*math.Pi/3),
				Y: 4 * math.Sin(4*math.Pi/3),
			},
		},
	}

	triangle := Triangle{Points: [3]Sample{originals[0], originals[1], originals[2]}}
	g.Expect(
		pointStrictlyInTriangle(Sample{Position: geometry.Point{X: region.CX, Y: region.CY}}, triangle),
	).To(gomega.BeTrue())

	for _, point := range triangle.Points {
		g.Expect(region.Contains(point.Position.X, point.Position.Y)).To(gomega.BeTrue())
	}

	for index, start := range triangle.Points {
		end := triangle.Points[(index+1)%len(triangle.Points)]
		g.Expect(squaredDistanceToSegment(region.CX, region.CY, start, end)).To(
			gomega.BeNumerically(">", region.InnerRadius*region.InnerRadius),
		)
	}

	g.Expect(triangleInRegion(region, triangle)).To(gomega.BeFalse())
}

func TestBoundarySamples_RetainsAnnulusBoundaryNearObservedPoint(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	region := Annulus{InnerRadius: 10, OuterRadius: 20}
	originals := []Sample{{Position: geometry.Point{X: 19, Y: 0}}}

	samples := boundarySamples(region, originals)

	g.Expect(samples).To(gomega.ContainElement(Sample{Position: geometry.Point{X: 20, Y: 0}}))
}

func TestBoundarySamples_SeedAnnulusAtTriangleResolution(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	region := Annulus{InnerRadius: 10, OuterRadius: 20}

	samples := boundarySamples(region, nil)

	g.Expect(samples).To(gomega.HaveLen(24))
}

func TestBoundaryLoops_ReturnsDenseOrderedAnnulusLoops(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	loops := BoundaryLoops(Annulus{InnerRadius: 10, OuterRadius: 20}, MaxBoundarySegmentLength)
	if len(loops) != 2 {
		t.Fatalf("expected 2 annulus loops, got %d", len(loops))
	}

	outerLoop := loops[0]
	if len(outerLoop) < 2 {
		t.Fatalf("expected outer loop to contain at least 2 points, got %d", len(outerLoop))
	}

	innerLoop := loops[1]
	if len(innerLoop) < 2 {
		t.Fatalf("expected inner loop to contain at least 2 points, got %d", len(innerLoop))
	}

	g.Expect(loops).To(gomega.HaveLen(2))
	g.Expect(outerLoop).To(gomega.HaveLen(126))
	g.Expect(innerLoop).To(gomega.HaveLen(63))
	g.Expect(outerLoop[0]).To(gomega.Equal(Sample{Position: geometry.Point{X: 20, Y: 0}}))
	g.Expect(innerLoop[0]).To(gomega.Equal(Sample{Position: geometry.Point{X: 10, Y: 0}}))

	for _, loop := range [][]Sample{outerLoop, innerLoop} {
		g.Expect(loop[len(loop)-1]).NotTo(gomega.Equal(loop[0]))
		g.Expect(loop[1].Position.Y).To(gomega.BeNumerically(">", 0))

		for index, point := range loop {
			next := loop[(index+1)%len(loop)]
			g.Expect(point.Position.DistanceTo(next.Position)).To(
				gomega.BeNumerically("<=", MaxBoundarySegmentLength),
			)
		}
	}
}

func TestBoundaryLoops_ReturnsDenseClosedRectPerimeter(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	loops := BoundaryLoops(
		Rect{MinX: 1, MinY: 2, MaxX: 3, MaxY: 3},
		MaxBoundarySegmentLength,
	)
	if len(loops) != 1 {
		t.Fatalf("expected 1 rectangle loop, got %d", len(loops))
	}

	rectLoop := loops[0]
	if len(rectLoop) < 2 {
		t.Fatalf("expected rectangle loop to contain at least 2 points, got %d", len(rectLoop))
	}

	g.Expect(loops).To(gomega.Equal([][]Sample{{
		{Position: geometry.Point{X: 1, Y: 2}},
		{Position: geometry.Point{X: 2, Y: 2}},
		{Position: geometry.Point{X: 3, Y: 2}},
		{Position: geometry.Point{X: 3, Y: 3}},
		{Position: geometry.Point{X: 2, Y: 3}},
		{Position: geometry.Point{X: 1, Y: 3}},
	}}))

	for index, point := range rectLoop {
		next := rectLoop[(index+1)%len(rectLoop)]
		g.Expect(point.Position.DistanceTo(next.Position)).To(
			gomega.BeNumerically("<=", MaxBoundarySegmentLength),
		)
	}
}

func TestBoundaryLoops_ReturnsNilForUnsupportedRegion(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	g.Expect(BoundaryLoops(unsupportedRegion{}, MaxBoundarySegmentLength)).To(gomega.BeNil())
}

func TestBoundaryLoops_ReturnsNilForTypedNilRegion(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	var region *Annulus

	g.Expect(func() {
		BoundaryLoops(region, MaxBoundarySegmentLength)
	}).NotTo(gomega.Panic())
	g.Expect(BoundaryLoops(region, MaxBoundarySegmentLength)).To(gomega.BeNil())
}

type unsupportedRegion struct{}

type typedNilBoundaryProvider struct{}

func (unsupportedRegion) Bounds() Rect {
	return Rect{MaxX: 1, MaxY: 1}
}

func (unsupportedRegion) Contains(float64, float64) bool {
	return true
}

func TestBoundaryLoops_ReturnsNilForTypedNilBoundaryProvider(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	var region Region = (*typedNilBoundaryProvider)(nil)

	g.Expect(func() {
		BoundaryLoops(region, MaxBoundarySegmentLength)
	}).NotTo(gomega.Panic())
	g.Expect(BoundaryLoops(region, MaxBoundarySegmentLength)).To(gomega.BeNil())
}

func TestRegionTriangles_OmitsTriangleWithUnsupportedVertex(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	region := Rect{MinX: -2, MinY: -2, MaxX: 2, MaxY: 2}
	points := []Sample{
		{Position: geometry.Point{X: 0, Y: 0}, Value: 1},
		{Position: geometry.Point{X: 1, Y: 0}, Value: 2, unsupported: true},
		{Position: geometry.Point{X: 0, Y: 1}, Value: 3},
	}

	triangles, complete := regionTriangles(region, points, []int{0, 1, 2})
	g.Expect(complete).To(gomega.BeTrue())
	g.Expect(triangles).To(gomega.BeEmpty())
}

func (*typedNilBoundaryProvider) Bounds() Rect {
	panic("typed-nil provider Bounds should not be called")
}

func (*typedNilBoundaryProvider) Contains(float64, float64) bool {
	panic("typed-nil provider Contains should not be called")
}

func (*typedNilBoundaryProvider) BoundaryLoops(float64) [][]Sample {
	panic("typed-nil provider BoundaryLoops should not be called")
}
