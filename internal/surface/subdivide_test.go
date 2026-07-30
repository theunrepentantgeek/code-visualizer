package surface_test

import (
	"math"
	"testing"

	"github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/surface"
)

func TestSubdivideTriangle_SplitsTwoLowOneHighVerticesAtBreakpoint(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	triangle := surface.Triangle{
		Points: [3]surface.Point{
			{X: 0, Y: 0, Value: 0},
			{X: 4, Y: 0, Value: 0},
			{X: 0, Y: 4, Value: 2},
		},
	}

	polygons := surface.SubdivideTriangle(triangle, []float64{1})

	g.Expect(polygons).To(gomega.HaveLen(2))

	if len(polygons) != 2 {
		t.Fatalf("expected 2 polygons, got %d", len(polygons))
	}

	g.Expect(polygons[0].Points).To(gomega.ConsistOf(
		surface.Point{X: 0, Y: 0, Value: 0},
		surface.Point{X: 4, Y: 0, Value: 0},
		surface.Point{X: 2, Y: 2, Value: 1},
		surface.Point{X: 0, Y: 2, Value: 1},
	))
	g.Expect(polygons[1].Points).To(gomega.ConsistOf(
		surface.Point{X: 0, Y: 2, Value: 1},
		surface.Point{X: 2, Y: 2, Value: 1},
		surface.Point{X: 0, Y: 4, Value: 2},
	))
	g.Expect(bucketIndex([]float64{1}, polygons[0].Value)).To(gomega.Equal(0))
	g.Expect(bucketIndex([]float64{1}, polygons[1].Value)).To(gomega.Equal(1))
}

func TestSubdivideTriangle_ReturnsEveryCrossedPaletteBand(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	triangle := surface.Triangle{
		Points: [3]surface.Point{
			{X: 0, Y: 0, Value: 0},
			{X: 6, Y: 0, Value: 0},
			{X: 0, Y: 6, Value: 3},
		},
	}

	polygons := surface.SubdivideTriangle(triangle, []float64{1, 2})

	g.Expect(polygons).To(gomega.HaveLen(3))

	if len(polygons) != 3 {
		t.Fatalf("expected 3 polygons, got %d", len(polygons))
	}

	g.Expect(bucketIndex([]float64{1, 2}, polygons[0].Value)).To(gomega.Equal(0))
	g.Expect(bucketIndex([]float64{1, 2}, polygons[1].Value)).To(gomega.Equal(1))
	g.Expect(bucketIndex([]float64{1, 2}, polygons[2].Value)).To(gomega.Equal(2))
	g.Expect(polygons[0].Points).To(gomega.ConsistOf(
		surface.Point{X: 0, Y: 0, Value: 0},
		surface.Point{X: 6, Y: 0, Value: 0},
		surface.Point{X: 4, Y: 2, Value: 1},
		surface.Point{X: 0, Y: 2, Value: 1},
	))
	g.Expect(polygons[1].Points).To(gomega.ConsistOf(
		surface.Point{X: 0, Y: 2, Value: 1},
		surface.Point{X: 4, Y: 2, Value: 1},
		surface.Point{X: 2, Y: 4, Value: 2},
		surface.Point{X: 0, Y: 4, Value: 2},
	))
	g.Expect(polygons[2].Points).To(gomega.ConsistOf(
		surface.Point{X: 0, Y: 4, Value: 2},
		surface.Point{X: 2, Y: 4, Value: 2},
		surface.Point{X: 0, Y: 6, Value: 3},
	))
}

func TestSubdivideTriangle_PreservesTinyPositiveAreaFragments(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	triangle := surface.Triangle{
		Points: [3]surface.Point{
			{X: 0, Y: 0, Value: 0},
			{X: 4e-7, Y: 0, Value: 0},
			{X: 0, Y: 4e-7, Value: 2},
		},
	}

	polygons := surface.SubdivideTriangle(triangle, []float64{1})

	g.Expect(polygons).To(gomega.HaveLen(2))

	if len(polygons) != 2 {
		t.Fatalf("expected 2 polygons, got %d", len(polygons))
	}

	g.Expect(polygons[0].Points).To(gomega.ConsistOf(
		surface.Point{X: 0, Y: 0, Value: 0},
		surface.Point{X: 4e-7, Y: 0, Value: 0},
		surface.Point{X: 2e-7, Y: 2e-7, Value: 1},
		surface.Point{X: 0, Y: 2e-7, Value: 1},
	))
	g.Expect(polygons[1].Points).To(gomega.ConsistOf(
		surface.Point{X: 0, Y: 2e-7, Value: 1},
		surface.Point{X: 2e-7, Y: 2e-7, Value: 1},
		surface.Point{X: 0, Y: 4e-7, Value: 2},
	))
}

func TestSubdivideTriangle_TreatsBreakpointValueAsUpperBand(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	triangle := surface.Triangle{
		Points: [3]surface.Point{
			{X: 0, Y: 0, Value: 1},
			{X: 4, Y: 0, Value: 1.5},
			{X: 0, Y: 4, Value: 1.75},
		},
	}

	polygons := surface.SubdivideTriangle(triangle, []float64{1})

	g.Expect(polygons).To(gomega.HaveLen(1))

	if len(polygons) != 1 {
		t.Fatalf("expected 1 polygon, got %d", len(polygons))
	}

	g.Expect(polygons[0].Points).To(gomega.ConsistOf(trianglePoints(triangle)))
	g.Expect(bucketIndex([]float64{1}, polygons[0].Value)).To(gomega.Equal(1))
}

func TestSubdivideTriangle_LeavesUncrossedTriangleWhole(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	triangle := surface.Triangle{
		Points: [3]surface.Point{
			{X: 0, Y: 0, Value: 2},
			{X: 2, Y: 0, Value: 2.5},
			{X: 0, Y: 2, Value: 3},
		},
		Value: 2.5,
	}

	polygons := surface.SubdivideTriangle(triangle, []float64{1})

	g.Expect(polygons).To(gomega.HaveLen(1))

	if len(polygons) != 1 {
		t.Fatalf("expected 1 polygon, got %d", len(polygons))
	}

	g.Expect(polygons[0].Points).To(gomega.ConsistOf(trianglePoints(triangle)))
	g.Expect(bucketIndex([]float64{1}, polygons[0].Value)).To(gomega.Equal(1))
	g.Expect(polygons[0].Value).To(gomega.And(
		gomega.BeNumerically(">=", 2.0),
		gomega.BeNumerically("<=", 3.0),
	))
}

func TestSubdivideTriangle_ReturnsWholeTriangleWhenBreakpointsEmpty(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	triangle := surface.Triangle{
		Points: [3]surface.Point{
			{X: 0, Y: 0, Value: 2},
			{X: 2, Y: 0, Value: 3},
			{X: 0, Y: 2, Value: 4},
		},
		Value: 3,
	}

	polygons := surface.SubdivideTriangle(triangle, nil)

	g.Expect(polygons).To(gomega.Equal([]surface.Polygon{{
		Points: trianglePoints(triangle),
		Value:  3,
	}}))

	if len(polygons) != 1 {
		t.Fatalf("expected 1 polygon, got %d", len(polygons))
	}

	polygons[0].Points[0].X = 99
	g.Expect(triangle.Points[0].X).To(gomega.Equal(0.0))
}

func TestSubdivideTriangle_ReturnsNilForInvalidBreakpointsOrGeometry(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	triangle := surface.Triangle{
		Points: [3]surface.Point{
			{X: 0, Y: 0, Value: 0},
			{X: 2, Y: 0, Value: 1},
			{X: 0, Y: 2, Value: 2},
		},
		Value: 1,
	}

	g.Expect(surface.SubdivideTriangle(triangle, []float64{2, 1})).To(gomega.BeNil())
	g.Expect(surface.SubdivideTriangle(triangle, []float64{1, 1})).To(gomega.BeNil())
	g.Expect(surface.SubdivideTriangle(triangle, []float64{1, math.NaN()})).To(gomega.BeNil())
	g.Expect(surface.SubdivideTriangle(triangle, []float64{1, math.Inf(1)})).To(gomega.BeNil())
	g.Expect(surface.SubdivideTriangle(surface.Triangle{
		Points: [3]surface.Point{
			{X: math.NaN(), Y: 0, Value: 0},
			{X: 2, Y: 0, Value: 1},
			{X: 0, Y: 2, Value: 2},
		},
		Value: 1,
	}, []float64{1})).To(gomega.BeNil())
	g.Expect(surface.SubdivideTriangle(surface.Triangle{
		Points: [3]surface.Point{
			{X: 0, Y: 0, Value: 0},
			{X: 2, Y: 0, Value: math.NaN()},
			{X: 0, Y: 2, Value: 2},
		},
		Value: 1,
	}, []float64{1})).To(gomega.BeNil())
	g.Expect(surface.SubdivideTriangle(surface.Triangle{
		Points: [3]surface.Point{
			{X: 0, Y: 0, Value: 0},
			{X: 2, Y: 0, Value: 1},
			{X: 0, Y: 2, Value: 2},
		},
		Value: math.Inf(1),
	}, []float64{1})).To(gomega.BeNil())
	g.Expect(surface.SubdivideTriangle(surface.Triangle{
		Points: [3]surface.Point{
			{X: 0, Y: 0, Value: 0},
			{X: 2, Y: 0, Value: 1},
			{X: 4, Y: 0, Value: 2},
		},
		Value: 1,
	}, []float64{1})).To(gomega.BeNil())

	overflowPolygons := surface.SubdivideTriangle(surface.Triangle{
		Points: [3]surface.Point{
			{X: 1e308, Y: 0, Value: 0},
			{X: -1e308, Y: 0, Value: 2},
			{X: 1e308, Y: 1, Value: 0},
		},
		Value: 2.0 / 3.0,
	}, []float64{1})
	g.Expect(overflowPolygons).To(gomega.HaveLen(2))

	for _, polygon := range overflowPolygons {
		for _, point := range polygon.Points {
			g.Expect(math.IsNaN(point.X) || math.IsInf(point.X, 0)).To(gomega.BeFalse())
			g.Expect(math.IsNaN(point.Y) || math.IsInf(point.Y, 0)).To(gomega.BeFalse())
			g.Expect(math.IsNaN(point.Value) || math.IsInf(point.Value, 0)).To(gomega.BeFalse())
		}
	}
}

func TestSubdivideTriangle_IsDeterministicAndDoesNotMutateInputs(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	triangle := surface.Triangle{
		Points: [3]surface.Point{
			{X: 0, Y: 0, Value: 0},
			{X: 4, Y: 0, Value: 0},
			{X: 0, Y: 4, Value: 2},
		},
		Value: 2.0 / 3.0,
	}
	breakpoints := []float64{1}
	originalTriangle := triangle

	originalBreakpoints := append([]float64(nil), breakpoints...)

	first := surface.SubdivideTriangle(triangle, breakpoints)
	second := surface.SubdivideTriangle(triangle, breakpoints)

	g.Expect(first).To(gomega.Equal(second))
	g.Expect(triangle).To(gomega.Equal(originalTriangle))
	g.Expect(breakpoints).To(gomega.Equal(originalBreakpoints))
}

func trianglePoints(triangle surface.Triangle) []surface.Point {
	return append([]surface.Point(nil), triangle.Points[:]...)
}

func bucketIndex(breakpoints []float64, value float64) int {
	for index, breakpoint := range breakpoints {
		if value < breakpoint {
			return index
		}
	}

	return len(breakpoints)
}
