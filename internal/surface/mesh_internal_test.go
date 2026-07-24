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

	for _, point := range boundarySamples(region, originals) {
		g.Expect(math.Hypot(point.X-region.CX, point.Y-region.CY)).NotTo(
			gomega.BeNumerically("~", region.InnerRadius),
		)
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
