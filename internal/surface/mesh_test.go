package surface_test

import (
	"math"
	"testing"

	"github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/surface"
)

func TestInterpolate_InterpolatesMidpointWithCompactKernel(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	originals := []surface.Sample{
		{Position: geometry.Point{X: 0, Y: 0}, Value: 0},
		{Position: geometry.Point{X: 4, Y: 0}, Value: 8},
	}

	g.Expect(surface.Interpolate(geometry.Point{X: 2, Y: 0}, originals)).To(
		gomega.Equal(4.0),
	)
}

func TestInterpolate_ReturnsObservedValueAtOriginalLocation(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	originals := []surface.Sample{
		{Position: geometry.Point{X: 0, Y: 0}, Value: 3},
		{Position: geometry.Point{X: 4, Y: 0}, Value: 8},
	}

	g.Expect(surface.Interpolate(geometry.Point{X: 0, Y: 0}, originals)).To(
		gomega.Equal(3.0),
	)
	g.Expect(surface.Interpolate(geometry.Point{X: 4, Y: 0}, originals)).To(gomega.Equal(8.0))
}

func TestBuild_ReturnsNoMeshWithFewerThanThreeOriginals(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	triangles := surface.Build(
		surface.Rect{MinX: 0, MinY: 0, MaxX: 10, MaxY: 10},
		[]surface.Sample{
			{Position: geometry.Point{X: 1, Y: 1}, Value: 2},
			{Position: geometry.Point{X: 9, Y: 9}, Value: 5},
		},
		42,
	)

	g.Expect(triangles).To(gomega.BeEmpty())
}

func TestBuildAndPoissonSamples_RejectTypedNilAnnulusRegion(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	var (
		annulus *surface.Annulus
		region  surface.Region = annulus
	)

	originals := []surface.Sample{
		{Position: geometry.Point{X: 0, Y: 0}, Value: 1},
		{Position: geometry.Point{X: 10, Y: 0}, Value: 2},
		{Position: geometry.Point{X: 0, Y: 10}, Value: 3},
	}

	var (
		triangles []surface.Triangle
		samples   []surface.Sample
	)

	g.Expect(func() {
		triangles = surface.Build(region, originals, 42)
		samples = surface.PoissonSamples(region, originals, surface.PoissonMinDistance, 42)
	}).NotTo(gomega.Panic())

	g.Expect(triangles).To(gomega.BeEmpty())
	g.Expect(samples).To(gomega.BeEmpty())
}

func TestBuild_IgnoresNonFiniteOriginalCoordinatesAndValues(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	originals := []surface.Sample{
		{Position: geometry.Point{X: 1, Y: 1}, Value: 1},
		{Position: geometry.Point{X: 9, Y: 1}, Value: 2},
		{Position: geometry.Point{X: 1, Y: 9}, Value: 3},
		{Position: geometry.Point{X: math.NaN(), Y: 5}, Value: 4},
		{Position: geometry.Point{X: 5, Y: math.Inf(1)}, Value: 5},
		{Position: geometry.Point{X: math.Inf(-1), Y: 5}, Value: 6},
		{Position: geometry.Point{X: 5, Y: 5}, Value: math.NaN()},
		{Position: geometry.Point{X: 6, Y: 6}, Value: math.Inf(1)},
		{Position: geometry.Point{X: 7, Y: 7}, Value: math.Inf(-1)},
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
		for _, sample := range triangle.Points {
			g.Expect(math.IsNaN(sample.Position.X) || math.IsInf(sample.Position.X, 0)).To(gomega.BeFalse())
			g.Expect(math.IsNaN(sample.Position.Y) || math.IsInf(sample.Position.Y, 0)).To(gomega.BeFalse())
			g.Expect(math.IsNaN(sample.Value) || math.IsInf(sample.Value, 0)).To(gomega.BeFalse())
		}
	}
}

func TestBuild_PreservesObservedVertexValues(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	originals := []surface.Sample{
		{Position: geometry.Point{X: 0, Y: 0}, Value: 1},
		{Position: geometry.Point{X: 5, Y: 0}, Value: 2},
		{Position: geometry.Point{X: 10, Y: 0}, Value: 3},
		{Position: geometry.Point{X: 15, Y: 0}, Value: 4},
		{Position: geometry.Point{X: 0, Y: 5}, Value: 5},
		{Position: geometry.Point{X: 5, Y: 5}, Value: 6},
		{Position: geometry.Point{X: 10, Y: 5}, Value: 7},
		{Position: geometry.Point{X: 15, Y: 5}, Value: 8},
		{Position: geometry.Point{X: 0, Y: 10}, Value: 9},
		{Position: geometry.Point{X: 5, Y: 10}, Value: 10},
		{Position: geometry.Point{X: 10, Y: 10}, Value: 11},
		{Position: geometry.Point{X: 15, Y: 10}, Value: 12},
		{Position: geometry.Point{X: 0, Y: 15}, Value: 13},
		{Position: geometry.Point{X: 5, Y: 15}, Value: 14},
		{Position: geometry.Point{X: 10, Y: 15}, Value: 15},
		{Position: geometry.Point{X: 15, Y: 15}, Value: 16},
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
		for _, sample := range triangle.Points {
			if sample.Original {
				foundOriginals[coordinate{x: sample.Position.X, y: sample.Position.Y}] = sample.Value
			}
		}
	}

	for _, original := range originals {
		g.Expect(foundOriginals).To(gomega.HaveKeyWithValue(
			coordinate{x: original.Position.X, y: original.Position.Y},
			original.Value,
		))
	}
}

func TestBuild_InterpolatesInfillFromOriginalsOnly(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	originals := []surface.Sample{
		{Position: geometry.Point{X: 0, Y: 0}, Value: 0},
		{Position: geometry.Point{X: 10, Y: 0}, Value: 8},
		{Position: geometry.Point{X: 0, Y: 10}, Value: 16},
		{Position: geometry.Point{X: 10, Y: 10}, Value: 24},
	}

	triangles := surface.Build(
		surface.Rect{MinX: 0, MinY: 0, MaxX: 10, MaxY: 10},
		originals,
		42,
	)

	foundInfill := false

	for _, triangle := range triangles {
		for _, sample := range triangle.Points {
			if sample.Original {
				continue
			}

			foundInfill = true

			g.Expect(sample.Value).To(gomega.BeNumerically("~", surface.Interpolate(sample.Position, originals)))
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
	originals := []surface.Sample{
		{Position: geometry.Point{X: 34, Y: 20}, Value: 1},
		{Position: geometry.Point{X: 32.1, Y: 27}, Value: 2},
		{Position: geometry.Point{X: 27, Y: 32.1}, Value: 3},
		{Position: geometry.Point{X: 20, Y: 34}, Value: 4},
		{Position: geometry.Point{X: 13, Y: 32.1}, Value: 5},
		{Position: geometry.Point{X: 7.9, Y: 27}, Value: 6},
		{Position: geometry.Point{X: 6, Y: 20}, Value: 7},
		{Position: geometry.Point{X: 7.9, Y: 13}, Value: 8},
		{Position: geometry.Point{X: 13, Y: 7.9}, Value: 9},
		{Position: geometry.Point{X: 20, Y: 6}, Value: 10},
		{Position: geometry.Point{X: 27, Y: 7.9}, Value: 11},
		{Position: geometry.Point{X: 32.1, Y: 13}, Value: 12},
		{Position: geometry.Point{X: 26, Y: 20}, Value: 13},
		{Position: geometry.Point{X: 25.2, Y: 23}, Value: 14},
		{Position: geometry.Point{X: 23, Y: 25.2}, Value: 15},
		{Position: geometry.Point{X: 20, Y: 26}, Value: 16},
		{Position: geometry.Point{X: 17, Y: 25.2}, Value: 17},
		{Position: geometry.Point{X: 14.8, Y: 23}, Value: 18},
		{Position: geometry.Point{X: 14, Y: 20}, Value: 19},
		{Position: geometry.Point{X: 14.8, Y: 17}, Value: 20},
		{Position: geometry.Point{X: 17, Y: 14.8}, Value: 21},
		{Position: geometry.Point{X: 20, Y: 14}, Value: 22},
		{Position: geometry.Point{X: 23, Y: 14.8}, Value: 23},
		{Position: geometry.Point{X: 25.2, Y: 17}, Value: 24},
	}

	first := surface.Build(region, originals, 42)
	second := surface.Build(region, originals, 42)

	g.Expect(first).NotTo(gomega.BeEmpty())
	g.Expect(first).To(gomega.Equal(second))

	// Rim triangles span chords of the boundary circles, so they may reach
	// inside the inner circle by up to the sagitta of a maximum-length chord.
	sagitta := region.InnerRadius -
		math.Sqrt(region.InnerRadius*region.InnerRadius-surface.MaxTriangleEdge*surface.MaxTriangleEdge/4)
	tolerantRegion := surface.Annulus{
		CX:          region.CX,
		CY:          region.CY,
		InnerRadius: region.InnerRadius - sagitta,
		OuterRadius: region.OuterRadius + sagitta,
	}

	for _, triangle := range first {
		for _, sample := range triangle.Points {
			g.Expect(tolerantRegion.Contains(sample.Position.X, sample.Position.Y)).To(gomega.BeTrue())
		}

		centroid := centroid(triangle)
		g.Expect(tolerantRegion.Contains(centroid.Position.X, centroid.Position.Y)).To(gomega.BeTrue())
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

	originals := []surface.Sample{
		{Position: geometry.Point{X: 30, Y: 20}, Value: 1},
		{Position: geometry.Point{X: 20, Y: 30}, Value: 2},
		{Position: geometry.Point{X: 10, Y: 20}, Value: 3},
		{Position: geometry.Point{X: 20, Y: 10}, Value: 4},
	}
	for _, original := range originals {
		radius := math.Hypot(original.Position.X-region.CX, original.Position.Y-region.CY)
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

		for _, sample := range triangle.Points {
			radius := math.Hypot(sample.Position.X-region.CX, sample.Position.Y-region.CY)
			if math.Abs(radius-region.InnerRadius) <= 1e-9 {
				hasInnerBoundaryVertex = true

				g.Expect(sample.Original).To(gomega.BeFalse())
				g.Expect(sample.Value).To(
					gomega.BeNumerically("~", surface.Interpolate(sample.Position, originals)),
				)
			}

			if math.Abs(radius-region.OuterRadius) <= 1e-9 {
				hasOuterBoundaryVertex = true

				g.Expect(sample.Original).To(gomega.BeFalse())
				g.Expect(sample.Value).To(
					gomega.BeNumerically("~", surface.Interpolate(sample.Position, originals)),
				)
			}
		}
	}

	g.Expect(hasInnerBoundaryVertex).To(gomega.BeTrue())
	g.Expect(hasOuterBoundaryVertex).To(gomega.BeTrue())
}

// Unsupported boundary samples are intentionally omitted from retained
// triangles; supported samples must still anchor the rendered rim.
func TestBuild_RetainsEverySupportedAnnulusBoundarySampleAsMeshVertex(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	region := surface.Annulus{
		CX:          500,
		CY:          500,
		InnerRadius: 100,
		OuterRadius: 200,
	}

	originals := make([]surface.Sample, 0, 120)

	for index := range 120 {
		angle := 2 * math.Pi * float64(index) / 120
		radius := 110 + 80*float64(index%7)/7

		originals = append(originals, surface.Sample{
			Position: geometry.Point{
				X: region.CX + radius*math.Cos(angle),
				Y: region.CY + radius*math.Sin(angle),
			},
			Value: radius,
		})
	}

	triangles := surface.Build(region, originals, 42)
	g.Expect(triangles).NotTo(gomega.BeEmpty())

	vertices := make(map[[2]float64]bool, len(triangles)*3)
	for _, triangle := range triangles {
		for _, sample := range triangle.Points {
			vertices[[2]float64{sample.Position.X, sample.Position.Y}] = true
		}
	}

	loops := surface.BoundaryLoops(region, surface.MaxTriangleEdge)
	g.Expect(loops).To(gomega.HaveLen(2))

	for _, loop := range loops {
		for _, sample := range loop {
			if surface.Interpolate(sample.Position, originals) == 0 {
				continue
			}

			g.Expect(vertices).To(gomega.HaveKey([2]float64{sample.Position.X, sample.Position.Y}))
		}
	}
}

func TestLongestEdge_ReturnsLengthOfLongestTriangleSide(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	triangle := surface.Triangle{
		Points: [3]surface.Sample{
			{Position: geometry.Point{X: 0, Y: 0}},
			{Position: geometry.Point{X: 3, Y: 0}},
			{Position: geometry.Point{X: 0, Y: 4}},
		},
	}

	g.Expect(surface.LongestEdge(triangle)).To(gomega.Equal(5.0))
}

func centroid(triangle surface.Triangle) surface.Sample {
	var centroid surface.Sample
	for _, sample := range triangle.Points {
		centroid.Position.X += sample.Position.X
		centroid.Position.Y += sample.Position.Y
	}

	centroid.Position.X /= float64(len(triangle.Points))
	centroid.Position.Y /= float64(len(triangle.Points))

	return centroid
}
