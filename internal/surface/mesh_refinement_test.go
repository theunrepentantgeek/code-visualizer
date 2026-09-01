package surface

import (
	"math"
	"testing"

	"github.com/fogleman/delaunay"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

func TestRefinementSampleLimitReservesHalfEdgeGrid(t *testing.T) {
	t.Parallel()

	region := Rect{MinX: 0, MinY: 0, MaxX: 16, MaxY: 16}

	got := refinementSampleLimit(region, 3)

	want := 3 + 5*5 // Grid vertices at 0, 4, 8, 12, and 16 on both axes.
	if got != want {
		t.Fatalf("refinement sample limit = %d, want %d", got, want)
	}
}

func TestRefinementSamples_IgnoresDegenerateOversizedFace(t *testing.T) {
	t.Parallel()

	samples := []Sample{
		{Position: geometry.Point{X: 0, Y: 0}},
		{Position: geometry.Point{X: 12, Y: 0}},
		{Position: geometry.Point{X: 6, Y: 0}},
	}

	candidates, oversized := refinementSamples(
		Rect{MinX: 0, MinY: -1, MaxX: 12, MaxY: 1},
		samples,
		[]int{0, 1, 2},
	)

	if oversized {
		t.Fatal("degenerate face must not require refinement")
	}

	if len(candidates) != 0 {
		t.Fatalf("degenerate face generated %d refinement candidates", len(candidates))
	}
}

func TestIsDegenerateTriangle_RecognizesMidpointVertex(t *testing.T) {
	t.Parallel()

	start := Sample{Position: geometry.Point{X: 103.9590562495064, Y: 142.30031597003037}}
	end := Sample{Position: geometry.Point{X: 111.25205052331654, Y: 151.73148706176613}}
	midpointSample := Sample{Position: geometry.Midpoint(start.Position, end.Position)}

	if !isDegenerateTriangle(Triangle{Points: [3]Sample{start, end, midpointSample}}) {
		t.Fatal("triangle with a midpoint vertex must be degenerate")
	}
}

func TestMeshSamples_RefinesInRegionTrianglesToMaximumEdge(t *testing.T) {
	t.Parallel()

	region, originals := refinementTestMesh()

	model, ok := newInterpolationModel(originals)
	if !ok {
		t.Fatal("expected interpolation model")
	}

	samples, complete := meshSamples(region, model, 42)
	if !complete {
		t.Fatal("mesh refinement did not complete")
	}

	for _, triangle := range inRegionDelaunayTriangles(t, region, samples) {
		if LongestEdge(triangle) > MaxTriangleEdge {
			t.Fatalf(
				"generated oversized in-region triangle with edge %.2f",
				LongestEdge(triangle),
			)
		}
	}
}

func TestBuild_CoversEveryInRegionDelaunayFaceAfterRefinement(t *testing.T) {
	t.Parallel()

	region, originals := refinementTestMesh()

	model, ok := newInterpolationModel(originals)
	if !ok {
		t.Fatal("expected interpolation model")
	}

	samples, complete := meshSamples(region, model, 42)
	if !complete {
		t.Fatal("mesh refinement did not complete")
	}

	expected := inRegionDelaunayTriangles(t, region, samples)

	actual := Build(region, originals, 42)
	if len(actual) != len(expected) {
		t.Fatalf(
			"Build returned %d triangles, but the refined Delaunay mesh has %d in-region faces",
			len(actual),
			len(expected),
		)
	}

	for i, triangle := range actual {
		if triangle != expected[i] {
			t.Fatalf("Build omitted or reordered refined in-region face %d", i)
		}

		if LongestEdge(triangle) > MaxTriangleEdge {
			t.Fatalf("Build returned oversized in-region triangle with edge %.2f", LongestEdge(triangle))
		}
	}
}

func inRegionDelaunayTriangles(t *testing.T, region Region, samples []Sample) []Triangle {
	t.Helper()

	triangulation, err := delaunay.Triangulate(delaunayPoints(samples))
	if err != nil {
		t.Fatal(err)
	}

	triangles := make([]Triangle, 0, len(triangulation.Triangles)/3)
	for index := 0; index+2 < len(triangulation.Triangles); index += 3 {
		triangle, ok := triangleAt(samples, [3]int{
			triangulation.Triangles[index],
			triangulation.Triangles[index+1],
			triangulation.Triangles[index+2],
		})
		if !ok {
			t.Fatal("invalid Delaunay index")
		}

		if !triangleInRegion(region, triangle) {
			continue
		}

		if triangleIsUnsupported(triangle) {
			continue
		}

		triangle.Value = (triangle.Points[0].Value +
			triangle.Points[1].Value +
			triangle.Points[2].Value) / 3
		triangles = append(triangles, triangle)
	}

	return triangles
}

func refinementTestMesh() (Annulus, []Sample) {
	region := Annulus{
		CX:          180,
		CY:          180,
		InnerRadius: 60,
		OuterRadius: 160,
	}
	originals := make([]Sample, 0, 32)

	for i := range 32 {
		theta := float64(i) * 4 * math.Pi / 31
		radius := 60 + 100*float64(i)/31
		originals = append(originals, Sample{
			Position: geometry.Point{
				X: region.CX + radius*math.Sin(theta),
				Y: region.CY - radius*math.Cos(theta),
			},
		})
	}

	return region, originals
}
