package surface

import (
	"math"

	"github.com/fogleman/delaunay"
)

// Interpolate estimates a point's value from observed points using inverse-distance weighting.
func Interpolate(point Point, originals []Point) float64 {
	if len(originals) == 0 {
		return 0
	}

	if point.Original {
		return point.Value
	}

	var weightedValue float64
	var totalWeight float64
	for _, original := range originals {
		distance := Distance(point, original)
		if distance == 0 {
			return original.Value
		}

		weight := 1 / math.Pow(distance, IDWPower)
		weightedValue += original.Value * weight
		totalWeight += weight
	}

	return weightedValue / totalWeight
}

// Build creates an interpolated Delaunay triangle mesh restricted to region.
func Build(region Region, originals []Point, seed uint64) []Triangle {
	if len(originals) < 3 {
		return nil
	}

	observed := append([]Point(nil), originals...)
	for index := range observed {
		observed[index].Original = true
	}
	points := append([]Point(nil), observed...)

	infill := Sample(region, observed, PoissonMinDistance, seed)
	for _, point := range infill {
		point.Value = Interpolate(point, observed)
		points = append(points, point)
	}

	triangulation, err := delaunay.Triangulate(delaunayPoints(points))
	if err != nil {
		return nil
	}

	triangles := make([]Triangle, 0, len(triangulation.Triangles)/3)
	for index := 0; index+2 < len(triangulation.Triangles); index += 3 {
		triangle := Triangle{
			Points: [3]Point{
				points[triangulation.Triangles[index]],
				points[triangulation.Triangles[index+1]],
				points[triangulation.Triangles[index+2]],
			},
		}
		triangle.Value = (triangle.Points[0].Value + triangle.Points[1].Value + triangle.Points[2].Value) / 3

		if triangleInRegion(region, triangle) && LongestEdge(triangle) <= MaxTriangleEdge {
			triangles = append(triangles, triangle)
		}
	}

	return triangles
}

// LongestEdge returns the length of a triangle's longest side.
func LongestEdge(triangle Triangle) float64 {
	return math.Max(
		Distance(triangle.Points[0], triangle.Points[1]),
		math.Max(
			Distance(triangle.Points[1], triangle.Points[2]),
			Distance(triangle.Points[2], triangle.Points[0]),
		),
	)
}

func delaunayPoints(points []Point) []delaunay.Point {
	result := make([]delaunay.Point, len(points))
	for index, point := range points {
		result[index] = delaunay.Point{X: point.X, Y: point.Y}
	}

	return result
}

func triangleInRegion(region Region, triangle Triangle) bool {
	if region == nil {
		return false
	}

	var center Point
	for _, point := range triangle.Points {
		if !region.Contains(point.X, point.Y) {
			return false
		}
		center.X += point.X
		center.Y += point.Y
	}
	center.X /= float64(len(triangle.Points))
	center.Y /= float64(len(triangle.Points))

	return region.Contains(center.X, center.Y)
}
