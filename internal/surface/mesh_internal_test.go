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
