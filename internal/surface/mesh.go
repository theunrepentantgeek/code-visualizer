package surface

import (
	"math"

	"github.com/fogleman/delaunay"
)

// Boundary samples are placed with trigonometry, so their recovered radius can
// differ from the region's radius by a few units in the last place of the
// floating-point representation. Comparing radii with this relative tolerance
// keeps the resulting rim triangles instead of discarding an arbitrary half of
// them and leaving a ragged edge.
const radiusTolerance = 1e-9

// Build creates an interpolated Delaunay triangle mesh restricted to region.
func Build(region Region, originals []Point, seed uint64) []Triangle {
	if !isValidRegion(region) {
		return nil
	}

	model, ok := newInterpolationModel(originals)
	if !ok || len(model.observations) < 3 {
		return nil
	}

	points, complete := meshPoints(region, model, seed)
	if !complete || len(points) < 3 {
		return nil
	}

	triangulation, err := delaunay.Triangulate(delaunayPoints(points))
	if err != nil {
		return nil
	}

	triangles, complete := regionTriangles(region, points, triangulation.Triangles)
	if !complete {
		return nil
	}

	return triangles
}

func observedPoints(originals []Point) []Point {
	observed := make([]Point, 0, len(originals))
	for _, original := range originals {
		if !isFinitePoint(original) || !isFinite(original.Value) {
			continue
		}

		original.Original = true
		observed = append(observed, original)
	}

	return observed
}

func meshPoints(region Region, model interpolationModel, seed uint64) ([]Point, bool) {
	boundary := boundarySamples(region, model.observations)
	for index := range boundary {
		value, _ := model.interpolate(boundary[index])
		boundary[index].Value = value
	}

	points := append([]Point(nil), model.observations...)
	points = append(points, boundary...)
	samplingSources := append([]Point(nil), points...)

	infill := Sample(region, samplingSources, PoissonMinDistance, seed)
	for _, point := range infill {
		point.Value, _ = model.interpolate(point)
		points = append(points, point)
	}

	return refineMeshPoints(region, points, model)
}

func refineMeshPoints(region Region, points []Point, model interpolationModel) ([]Point, bool) {
	limit := refinementPointLimit(region, len(points))

	for {
		triangulation, err := delaunay.Triangulate(delaunayPoints(points))
		if err != nil {
			return nil, false
		}

		candidates, oversized := refinementPoints(region, points, triangulation.Triangles)
		if !oversized {
			return points, true
		}

		if len(candidates) == 0 || len(points)+len(candidates) > limit {
			return nil, false
		}

		for _, candidate := range candidates {
			candidate.Value, _ = model.interpolate(candidate)
			points = append(points, candidate)
		}
	}
}

// A grid at half the maximum edge has cell diagonals below the edge limit, so
// its vertices are a conservative refinement bound over the region's bounds.
// Existing points do not consume this allowance, so it is added to pointCount.
func refinementPointLimit(region Region, pointCount int) int {
	bounds := region.Bounds()
	spacing := MaxTriangleEdge / 2
	columns := math.Ceil((bounds.MaxX-bounds.MinX)/spacing) + 1
	rows := math.Ceil((bounds.MaxY-bounds.MinY)/spacing) + 1
	maxInt := int(^uint(0) >> 1)

	if pointCount >= maxInt ||
		!isFinite(columns) ||
		!isFinite(rows) ||
		columns >= float64(maxInt) ||
		rows >= float64(maxInt) {
		return maxInt
	}

	columnCount := int(columns)
	rowCount := int(rows)

	remaining := maxInt - pointCount
	if columnCount > remaining/rowCount {
		return maxInt
	}

	return pointCount + columnCount*rowCount
}

func refinementPoints(
	region Region,
	points []Point,
	indexes []int,
) ([]Point, bool) {
	candidates := make([]Point, 0)

	var oversized bool

	for index := 0; index+2 < len(indexes); index += 3 {
		triangle, ok := triangleAt(points, [3]int{
			indexes[index],
			indexes[index+1],
			indexes[index+2],
		})
		if !ok || isDegenerateTriangle(triangle) || !triangleInRegion(region, triangle) {
			continue
		}

		if LongestEdge(triangle) <= MaxTriangleEdge {
			continue
		}

		oversized = true

		candidate, found := refinementPoint(region, triangle, points, candidates)
		if found {
			candidates = append(candidates, candidate)
		}
	}

	return candidates, oversized
}

// Splitting the longest edge ensures the next triangulation cannot retain the offending edge.
func refinementPoint(region Region, target Triangle, points, candidates []Point) (Point, bool) {
	start, end, _ := longestTriangleEdge(target)
	candidate := Point{
		X: (start.X + end.X) / 2,
		Y: (start.Y + end.Y) / 2,
	}

	if !isFinitePoint(candidate) ||
		!region.Contains(candidate.X, candidate.Y) ||
		isDuplicate(candidate, points) ||
		isDuplicate(candidate, candidates) {
		return Point{}, false
	}

	return candidate, true
}

func regionTriangles(region Region, points []Point, indexes []int) ([]Triangle, bool) {
	triangles := make([]Triangle, 0, len(indexes)/3)
	for index := 0; index+2 < len(indexes); index += 3 {
		triangle, ok := triangleAt(points, [3]int{
			indexes[index],
			indexes[index+1],
			indexes[index+2],
		})
		if !ok || isDegenerateTriangle(triangle) {
			continue
		}

		triangle.Value = (triangle.Points[0].Value + triangle.Points[1].Value + triangle.Points[2].Value) / 3
		if triangleInRegion(region, triangle) {
			if LongestEdge(triangle) > MaxTriangleEdge {
				return nil, false
			}

			triangles = append(triangles, triangle)
		}
	}

	return triangles, true
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

func isDegenerateTriangle(triangle Triangle) bool {
	first, second, third := triangle.Points[0], triangle.Points[1], triangle.Points[2]
	start, end, _ := longestTriangleEdge(triangle)
	midpoint := Point{X: (start.X + end.X) / 2, Y: (start.Y + end.Y) / 2}

	return (second.X-first.X)*(third.Y-first.Y) == (second.Y-first.Y)*(third.X-first.X) ||
		isDuplicate(midpoint, triangle.Points[:])
}

// LongestEdge returns the length of a triangle's longest side.
func LongestEdge(triangle Triangle) float64 {
	_, _, length := longestTriangleEdge(triangle)

	return length
}

func longestTriangleEdge(triangle Triangle) (start, end Point, longest float64) {
	for index, point := range triangle.Points {
		next := triangle.Points[(index+1)%len(triangle.Points)]
		if length := Distance(point, next); length > longest {
			start, end, longest = point, next, length
		}
	}

	return start, end, longest
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
	loops := BoundaryLoops(region, MaxTriangleEdge)
	if len(loops) == 0 {
		return nil
	}

	candidateCount := 0
	for _, loop := range loops {
		candidateCount += len(loop)
	}

	candidates := make([]Point, 0, candidateCount)
	for _, loop := range loops {
		candidates = append(candidates, loop...)
	}

	samples := make([]Point, 0, len(candidates))
	for _, candidate := range candidates {
		if !isFinitePoint(candidate) ||
			isDuplicate(candidate, originals) ||
			isDuplicate(candidate, samples) {
			continue
		}

		candidate.Original = false
		samples = append(samples, candidate)
	}

	return samples
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
	outerRadius := annulus.OuterRadius * (1 + radiusTolerance)
	outerRadiusSquared := outerRadius * outerRadius

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

	if pointStrictlyInTriangle(Point{X: annulus.CX, Y: annulus.CY}, triangle) {
		return false
	}

	if !triangleOutsideInnerRadius(annulus, triangle) {
		return false
	}

	return triangleEdgesAvoidInnerRadius(annulus, triangle, innerChordLimitSquared(annulus))
}

func triangleOutsideInnerRadius(annulus Annulus, triangle Triangle) bool {
	innerRadius := annulus.InnerRadius * (1 - radiusTolerance)
	innerRadiusSquared := innerRadius * innerRadius

	for _, point := range triangle.Points {
		dx := point.X - annulus.CX

		dy := point.Y - annulus.CY
		if dx*dx+dy*dy < innerRadiusSquared {
			return false
		}
	}

	return true
}

// An edge joining two points on the inner circle passes closer to the centre
// than the inner radius by the chord's sagitta, so edges of the permitted
// length may approach this far without entering the empty core.
func innerChordLimitSquared(annulus Annulus) float64 {
	halfEdge := MaxTriangleEdge / 2

	return math.Max(0, annulus.InnerRadius*annulus.InnerRadius-halfEdge*halfEdge)
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
