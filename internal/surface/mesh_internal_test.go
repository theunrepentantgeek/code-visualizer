package surface

import (
	"math"
	"testing"

	"github.com/onsi/gomega"
)

func TestTriangleInRegion_RejectsAnnulusHoleCrossingEdge(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	region := Annulus{InnerRadius: 6, OuterRadius: 12}
	triangle := Triangle{
		Points: [3]Point{
			{X: 6, Y: 0},
			{X: 3 * math.Sqrt(3), Y: 3},
			{X: 10, Y: 0},
		},
	}

	for _, point := range triangle.Points {
		g.Expect(region.Contains(point.X, point.Y)).To(gomega.BeTrue())
	}

	centroid := Point{
		X: (triangle.Points[0].X + triangle.Points[1].X + triangle.Points[2].X) / 3,
		Y: (triangle.Points[0].Y + triangle.Points[1].Y + triangle.Points[2].Y) / 3,
	}
	g.Expect(region.Contains(centroid.X, centroid.Y)).To(gomega.BeTrue())

	g.Expect(triangleInRegion(region, triangle)).To(gomega.BeFalse())
}

func TestTriangleInRegion_RejectsAnnulusCenterEnclosedAfterInnerBoundaryPruning(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	region := Annulus{InnerRadius: 1, OuterRadius: 8}
	originals := []Point{
		{X: 4, Y: 0},
		{X: 4 * math.Cos(2*math.Pi/3), Y: 4 * math.Sin(2*math.Pi/3)},
		{X: 4 * math.Cos(4*math.Pi/3), Y: 4 * math.Sin(4*math.Pi/3)},
	}

	triangle := Triangle{Points: [3]Point{originals[0], originals[1], originals[2]}}
	g.Expect(pointStrictlyInTriangle(Point{X: region.CX, Y: region.CY}, triangle)).To(gomega.BeTrue())

	for _, point := range triangle.Points {
		g.Expect(region.Contains(point.X, point.Y)).To(gomega.BeTrue())
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
	originals := []Point{{X: 19, Y: 0}}

	samples := boundarySamples(region, originals)

	g.Expect(samples).To(gomega.ContainElement(Point{X: 20, Y: 0}))
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

	g.Expect(loops).To(gomega.HaveLen(2))
	g.Expect(loops[0]).To(gomega.HaveLen(126))
	g.Expect(loops[1]).To(gomega.HaveLen(63))
	g.Expect(loops[0][0]).To(gomega.Equal(Point{X: 20, Y: 0}))
	g.Expect(loops[1][0]).To(gomega.Equal(Point{X: 10, Y: 0}))
	for _, loop := range loops {
		g.Expect(loop[len(loop)-1]).NotTo(gomega.Equal(loop[0]))
		g.Expect(loop[1].Y).To(gomega.BeNumerically(">", 0))
		for index, point := range loop {
			next := loop[(index+1)%len(loop)]
			g.Expect(Distance(point, next)).To(gomega.BeNumerically("<=", MaxBoundarySegmentLength))
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

	g.Expect(loops).To(gomega.Equal([][]Point{{
		{X: 1, Y: 2},
		{X: 2, Y: 2},
		{X: 3, Y: 2},
		{X: 3, Y: 3},
		{X: 2, Y: 3},
		{X: 1, Y: 3},
	}}))
	for index, point := range loops[0] {
		next := loops[0][(index+1)%len(loops[0])]
		g.Expect(Distance(point, next)).To(gomega.BeNumerically("<=", MaxBoundarySegmentLength))
	}
}

func TestBoundaryLoops_ReturnsNilForUnsupportedRegion(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	g.Expect(BoundaryLoops(unsupportedRegion{}, MaxBoundarySegmentLength)).To(gomega.BeNil())
}

type unsupportedRegion struct{}

func (unsupportedRegion) Bounds() Rect {
	return Rect{MaxX: 1, MaxY: 1}
}

func (unsupportedRegion) Contains(float64, float64) bool {
	return true
}
