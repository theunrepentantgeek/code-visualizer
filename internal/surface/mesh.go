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

	var (
		weightedValue float64
		totalWeight   float64
	)

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
	if !isValidRegion(region) {
		return nil
	}

	observed := observedPoints(originals)
	if len(observed) < 3 {
		return nil
	}

	points := meshPoints(region, observed, seed)

	triangulation, err := delaunay.Triangulate(delaunayPoints(points))
	if err != nil {
		return nil
	}

	return regionTriangles(region, points, triangulation.Triangles)
}

func observedPoints(originals []Point) []Point {
	observed := make([]Point, 0, len(originals))
	for _, original := range originals {
		if !isFinitePoint(original) {
			continue
		}

		original.Original = true
		observed = append(observed, original)
	}

	return observed
}

func meshPoints(region Region, observed []Point, seed uint64) []Point {
	boundary := boundarySamples(region, observed)
	for index := range boundary {
		boundary[index].Value = Interpolate(boundary[index], observed)
	}

	points := append([]Point(nil), observed...)
	points = append(points, boundary...)
	samplingSources := append([]Point(nil), points...)

	infill := Sample(region, samplingSources, PoissonMinDistance, seed)
	for _, point := range infill {
		point.Value = Interpolate(point, observed)
		points = append(points, point)
	}

	return points
}

func regionTriangles(region Region, points []Point, indexes []int) []Triangle {
	triangles := make([]Triangle, 0, len(indexes)/3)
	for index := 0; index+2 < len(indexes); index += 3 {
		triangle, ok := triangleAt(points, [3]int{
			indexes[index],
			indexes[index+1],
			indexes[index+2],
		})
		if !ok {
			continue
		}

		triangle.Value = (triangle.Points[0].Value + triangle.Points[1].Value + triangle.Points[2].Value) / 3
		if triangleInRegion(region, triangle) && LongestEdge(triangle) <= MaxTriangleEdge {
			triangles = append(triangles, triangle)
		}
	}

	return triangles
}

func triangleAt(points []Point, indexes [3]int) (Triangle, bool) {
	var triangle Triangle

	for pointIndex, index := range indexes {
		if index < 0 || index >= len(points) {
			return Triangle{}, false
		}

		triangle.Points[pointIndex] = points[index]
	}

	return triangle, true
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

	switch annulus := region.(type) {
	case Annulus:
		return triangleInAnnulus(annulus, triangle)
	case *Annulus:
		return annulus != nil && triangleInAnnulus(*annulus, triangle)
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

func boundarySamples(region Region, originals []Point) []Point {
	provider, ok := region.(boundaryPointProvider)
	if !ok {
		return nil
	}

	candidates := provider.boundaryPoints(PoissonMinDistance)

	samples := make([]Point, 0, len(candidates))
	for _, candidate := range candidates {
		if !isFinitePoint(candidate) || isNearOriginal(candidate, originals) || isDuplicate(candidate, samples) {
			continue
		}

		candidate.Original = false
		samples = append(samples, candidate)
	}

	return samples
}

func isNearOriginal(point Point, originals []Point) bool {
	for _, original := range originals {
		if Distance(point, original) < PoissonMinDistance {
			return true
		}
	}

	return false
}

func isDuplicate(point Point, points []Point) bool {
	for _, existing := range points {
		if point.X == existing.X && point.Y == existing.Y {
			return true
		}
	}

	return false
}

func triangleInAnnulus(annulus Annulus, triangle Triangle) bool {
	return validAnnulus(annulus) &&
		triangleWithinOuterRadius(annulus, triangle) &&
		triangleAvoidsInnerRadius(annulus, triangle)
}

func validAnnulus(annulus Annulus) bool {
	if !isFinite(annulus.CX) ||
		!isFinite(annulus.CY) ||
		!isFinite(annulus.InnerRadius) ||
		!isFinite(annulus.OuterRadius) ||
		annulus.InnerRadius < 0 ||
		annulus.OuterRadius < annulus.InnerRadius {
		return false
	}

	return true
}

func triangleWithinOuterRadius(annulus Annulus, triangle Triangle) bool {
	outerRadiusSquared := annulus.OuterRadius * annulus.OuterRadius

	for _, point := range triangle.Points {
		if !isFinitePoint(point) {
			return false
		}

		dx := point.X - annulus.CX

		dy := point.Y - annulus.CY
		if dx*dx+dy*dy > outerRadiusSquared {
			return false
		}
	}

	return true
}

func triangleAvoidsInnerRadius(annulus Annulus, triangle Triangle) bool {
	if annulus.InnerRadius == 0 {
		return true
	}

	innerRadiusSquared := annulus.InnerRadius * annulus.InnerRadius
	if pointStrictlyInTriangle(Point{X: annulus.CX, Y: annulus.CY}, triangle) {
		return false
	}

	return triangleEdgesAvoidInnerRadius(annulus, triangle, innerRadiusSquared)
}

func triangleEdgesAvoidInnerRadius(annulus Annulus, triangle Triangle, innerRadiusSquared float64) bool {
	for index, start := range triangle.Points {
		end := triangle.Points[(index+1)%len(triangle.Points)]
		if squaredDistanceToSegment(annulus.CX, annulus.CY, start, end) < innerRadiusSquared {
			return false
		}
	}

	return true
}

func pointStrictlyInTriangle(point Point, triangle Triangle) bool {
	var hasPositive, hasNegative bool

	for index, start := range triangle.Points {
		end := triangle.Points[(index+1)%len(triangle.Points)]
		crossProduct := (end.X-start.X)*(point.Y-start.Y) - (end.Y-start.Y)*(point.X-start.X)
		hasPositive = hasPositive || crossProduct > 0
		hasNegative = hasNegative || crossProduct < 0
	}

	return hasPositive != hasNegative
}

func squaredDistanceToSegment(x, y float64, start, end Point) float64 {
	dx := end.X - start.X
	dy := end.Y - start.Y

	lengthSquared := dx*dx + dy*dy
	if lengthSquared == 0 {
		dx = x - start.X
		dy = y - start.Y

		return dx*dx + dy*dy
	}

	fraction := ((x-start.X)*dx + (y-start.Y)*dy) / lengthSquared
	fraction = math.Max(0, math.Min(1, fraction))
	nearestX := start.X + fraction*dx
	nearestY := start.Y + fraction*dy
	dx = x - nearestX
	dy = y - nearestY

	return dx*dx + dy*dy
}
