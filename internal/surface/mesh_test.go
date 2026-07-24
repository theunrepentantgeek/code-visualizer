package surface_test

import (
	"math"
	"testing"

	"github.com/onsi/gomega"
	"github.com/theunrepentantgeek/code-visualizer/internal/surface"
)

func TestInterpolate_UsesInverseDistanceWeighting(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	originals := []surface.Point{
		{X: 0, Y: 0, Value: 0, Original: true},
		{X: 4, Y: 0, Value: 8, Original: true},
	}

	g.Expect(surface.Interpolate(surface.Point{X: 1, Y: 0}, originals)).To(
		gomega.BeNumerically("~", 0.8),
	)
}

func TestInterpolate_ReturnsObservedValueAtOriginalLocation(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	originals := []surface.Point{
		{X: 0, Y: 0, Value: 3, Original: true},
		{X: 4, Y: 0, Value: 8, Original: true},
	}

	g.Expect(surface.Interpolate(originals[0], originals)).To(
		gomega.Equal(3.0),
	)
	g.Expect(surface.Interpolate(surface.Point{X: 4, Y: 0}, originals)).To(gomega.Equal(8.0))
}

func TestBuild_ReturnsNoMeshWithFewerThanThreeOriginals(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	triangles := surface.Build(
		surface.Rect{MinX: 0, MinY: 0, MaxX: 10, MaxY: 10},
		[]surface.Point{{X: 1, Y: 1, Value: 2}, {X: 9, Y: 9, Value: 5}},
		42,
	)

	g.Expect(triangles).To(gomega.BeEmpty())
}

func TestBuildAndSample_RejectTypedNilAnnulusRegion(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	var annulus *surface.Annulus
	var region surface.Region = annulus
	originals := []surface.Point{
		{X: 0, Y: 0, Value: 1},
		{X: 10, Y: 0, Value: 2},
		{X: 0, Y: 10, Value: 3},
	}
	var triangles []surface.Triangle
	var samples []surface.Point

	g.Expect(func() {
		triangles = surface.Build(region, originals, 42)
		samples = surface.Sample(region, originals, surface.PoissonMinDistance, 42)
	}).NotTo(gomega.Panic())

	g.Expect(triangles).To(gomega.BeEmpty())
	g.Expect(samples).To(gomega.BeEmpty())
}

func TestBuild_IgnoresNonFiniteOriginalCoordinates(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	originals := []surface.Point{
		{X: 1, Y: 1, Value: 1},
		{X: 9, Y: 1, Value: 2},
		{X: 1, Y: 9, Value: 3},
		{X: math.NaN(), Y: 5, Value: 4},
		{X: 5, Y: math.Inf(1), Value: 5},
		{X: math.Inf(-1), Y: 5, Value: 6},
	}

	var triangles []surface.Triangle
	g.Expect(func() {
		triangles = surface.Build(
			surface.Rect{MinX: 0, MinY: 0, MaxX: 10, MaxY: 10},
			originals,
			42,
		)
	}).NotTo(gomega.Panic())

	g.Expect(triangles).NotTo(gomega.BeEmpty())
	for _, triangle := range triangles {
		for _, point := range triangle.Points {
			g.Expect(math.IsNaN(point.X) || math.IsInf(point.X, 0)).To(gomega.BeFalse())
			g.Expect(math.IsNaN(point.Y) || math.IsInf(point.Y, 0)).To(gomega.BeFalse())
		}
	}
}

func TestBuild_PreservesObservedVertexValues(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	originals := []surface.Point{
		{X: 0, Y: 0, Value: 1},
		{X: 5, Y: 0, Value: 2},
		{X: 10, Y: 0, Value: 3},
		{X: 15, Y: 0, Value: 4},
		{X: 0, Y: 5, Value: 5},
		{X: 5, Y: 5, Value: 6},
		{X: 10, Y: 5, Value: 7},
		{X: 15, Y: 5, Value: 8},
		{X: 0, Y: 10, Value: 9},
		{X: 5, Y: 10, Value: 10},
		{X: 10, Y: 10, Value: 11},
		{X: 15, Y: 10, Value: 12},
		{X: 0, Y: 15, Value: 13},
		{X: 5, Y: 15, Value: 14},
		{X: 10, Y: 15, Value: 15},
		{X: 15, Y: 15, Value: 16},
	}

	triangles := surface.Build(
		surface.Rect{MinX: 0, MinY: 0, MaxX: 15, MaxY: 15},
		originals,
		42,
	)

	g.Expect(triangles).NotTo(gomega.BeEmpty())
	type coordinate struct {
		x float64
		y float64
	}
	foundOriginals := make(map[coordinate]float64)
	for _, triangle := range triangles {
		for _, point := range triangle.Points {
			if point.Original {
				foundOriginals[coordinate{x: point.X, y: point.Y}] = point.Value
			}
		}
	}
	for _, original := range originals {
		g.Expect(foundOriginals).To(gomega.HaveKeyWithValue(
			coordinate{x: original.X, y: original.Y},
			original.Value,
		))
	}
}

func TestBuild_InterpolatesInfillFromOriginalsOnly(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	originals := []surface.Point{
		{X: 0, Y: 0, Value: 0},
		{X: 10, Y: 0, Value: 8},
		{X: 0, Y: 10, Value: 16},
		{X: 10, Y: 10, Value: 24},
	}

	triangles := surface.Build(
		surface.Rect{MinX: 0, MinY: 0, MaxX: 10, MaxY: 10},
		originals,
		42,
	)

	foundInfill := false
	for _, triangle := range triangles {
		for _, point := range triangle.Points {
			if point.Original {
				continue
			}

			foundInfill = true
			g.Expect(point.Value).To(gomega.BeNumerically("~", surface.Interpolate(point, originals)))
		}
	}
	g.Expect(foundInfill).To(gomega.BeTrue())
}

func TestBuild_RestrictsAnnularMeshToRegionAndMaximumEdge(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	region := surface.Annulus{
		CX:          20,
		CY:          20,
		InnerRadius: 6,
		OuterRadius: 14,
	}
	originals := []surface.Point{
		{X: 34, Y: 20, Value: 1},
		{X: 32.1, Y: 27, Value: 2},
		{X: 27, Y: 32.1, Value: 3},
		{X: 20, Y: 34, Value: 4},
		{X: 13, Y: 32.1, Value: 5},
		{X: 7.9, Y: 27, Value: 6},
		{X: 6, Y: 20, Value: 7},
		{X: 7.9, Y: 13, Value: 8},
		{X: 13, Y: 7.9, Value: 9},
		{X: 20, Y: 6, Value: 10},
		{X: 27, Y: 7.9, Value: 11},
		{X: 32.1, Y: 13, Value: 12},
		{X: 26, Y: 20, Value: 13},
		{X: 25.2, Y: 23, Value: 14},
		{X: 23, Y: 25.2, Value: 15},
		{X: 20, Y: 26, Value: 16},
		{X: 17, Y: 25.2, Value: 17},
		{X: 14.8, Y: 23, Value: 18},
		{X: 14, Y: 20, Value: 19},
		{X: 14.8, Y: 17, Value: 20},
		{X: 17, Y: 14.8, Value: 21},
		{X: 20, Y: 14, Value: 22},
		{X: 23, Y: 14.8, Value: 23},
		{X: 25.2, Y: 17, Value: 24},
	}

	first := surface.Build(region, originals, 42)
	second := surface.Build(region, originals, 42)

	g.Expect(first).NotTo(gomega.BeEmpty())
	g.Expect(first).To(gomega.Equal(second))
	for _, triangle := range first {
		for _, point := range triangle.Points {
			g.Expect(region.Contains(point.X, point.Y)).To(gomega.BeTrue())
		}
		centroid := centroid(triangle)
		g.Expect(region.Contains(centroid.X, centroid.Y)).To(gomega.BeTrue())
		g.Expect(surface.LongestEdge(triangle)).To(
			gomega.BeNumerically("<=", surface.MaxTriangleEdge),
		)
	}
}

func TestBuild_SeedsAnnulusBoundaries(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	region := surface.Annulus{
		CX:          20,
		CY:          20,
		InnerRadius: 6,
		OuterRadius: 14,
	}
	originals := []surface.Point{
		{X: 30, Y: 20, Value: 1},
		{X: 20, Y: 30, Value: 2},
		{X: 10, Y: 20, Value: 3},
		{X: 20, Y: 10, Value: 4},
	}
	for _, original := range originals {
		radius := math.Hypot(original.X-region.CX, original.Y-region.CY)
		g.Expect(radius).To(gomega.BeNumerically(">", region.InnerRadius))
		g.Expect(radius).To(gomega.BeNumerically("<", region.OuterRadius))
	}

	triangles := surface.Build(region, originals, 42)

	g.Expect(triangles).NotTo(gomega.BeEmpty())
	hasInnerBoundaryVertex := false
	hasOuterBoundaryVertex := false
	for _, triangle := range triangles {
		g.Expect(surface.LongestEdge(triangle)).To(
			gomega.BeNumerically("<=", surface.MaxTriangleEdge),
		)
		for _, point := range triangle.Points {
			radius := math.Hypot(point.X-region.CX, point.Y-region.CY)
			if math.Abs(radius-region.InnerRadius) <= 1e-9 {
				hasInnerBoundaryVertex = true
				g.Expect(point.Original).To(gomega.BeFalse())
				g.Expect(point.Value).To(
					gomega.BeNumerically("~", surface.Interpolate(point, originals)),
				)
			}
			if math.Abs(radius-region.OuterRadius) <= 1e-9 {
				hasOuterBoundaryVertex = true
				g.Expect(point.Original).To(gomega.BeFalse())
				g.Expect(point.Value).To(
					gomega.BeNumerically("~", surface.Interpolate(point, originals)),
				)
			}
		}
	}
	g.Expect(hasInnerBoundaryVertex).To(gomega.BeTrue())
	g.Expect(hasOuterBoundaryVertex).To(gomega.BeTrue())
}

func TestLongestEdge_ReturnsLengthOfLongestTriangleSide(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	triangle := surface.Triangle{
		Points: [3]surface.Point{{X: 0, Y: 0}, {X: 3, Y: 0}, {X: 0, Y: 4}},
	}

	g.Expect(surface.LongestEdge(triangle)).To(gomega.Equal(5.0))
}

func centroid(triangle surface.Triangle) surface.Point {
	var centroid surface.Point
	for _, point := range triangle.Points {
		centroid.X += point.X
		centroid.Y += point.Y
	}
	centroid.X /= float64(len(triangle.Points))
	centroid.Y /= float64(len(triangle.Points))

	return centroid
}
