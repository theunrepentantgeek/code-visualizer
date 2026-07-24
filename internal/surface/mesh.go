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
	if !isValidRegion(region) {
		return nil
	}

	observed := make([]Point, 0, len(originals))
	for _, original := range originals {
		if !isFinitePoint(original) {
			continue
		}

		original.Original = true
		observed = append(observed, original)
	}
	if len(observed) < 3 {
		return nil
	}

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
	if !isFinite(annulus.CX) ||
		!isFinite(annulus.CY) ||
		!isFinite(annulus.InnerRadius) ||
		!isFinite(annulus.OuterRadius) ||
		annulus.InnerRadius < 0 ||
		annulus.OuterRadius < annulus.InnerRadius {
		return false
	}

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

	innerRadiusSquared := annulus.InnerRadius * annulus.InnerRadius
	if annulus.InnerRadius > 0 &&
		pointStrictlyInTriangle(Point{X: annulus.CX, Y: annulus.CY}, triangle) {
		return false
	}

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
