package surface

import (
	"math"

	"github.com/fogleman/delaunay"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

// Boundary samples are placed with trigonometry, so their recovered radius can
// differ from the region's radius by a few units in the last place of the
// floating-point representation. Comparing radii with this relative tolerance
// keeps the resulting rim triangles instead of discarding an arbitrary half of
// them and leaving a ragged edge.
const radiusTolerance = 1e-9

// Build creates an interpolated Delaunay triangle mesh restricted to region.
func Build(region Region, originals []Sample, seed uint64) []Triangle {
	if !isValidRegion(region) {
		return nil
	}

	model, ok := newInterpolationModel(originals)
	if !ok || len(model.observations) < 3 {
		return nil
	}

	samples, complete := meshSamples(region, model, seed)
	if !complete || len(samples) < 3 {
		return nil
	}

	triangulation, err := delaunay.Triangulate(delaunayPoints(samples))
	if err != nil {
		return nil
	}

	triangles, complete := regionTriangles(region, samples, triangulation.Triangles)
	if !complete {
		return nil
	}

	return triangles
}

func observedSamples(originals []Sample) []Sample {
	observed := make([]Sample, 0, len(originals))
	for _, original := range originals {
		if !isFiniteSample(original) || !isFinite(original.Value) {
			continue
		}

		original.Original = true
		observed = append(observed, original)
	}

	return observed
}

func meshSamples(region Region, model interpolationModel, seed uint64) ([]Sample, bool) {
	boundary := boundarySamples(region, model.observations)
	for index := range boundary {
		boundary[index] = model.assign(boundary[index])
	}

	samples := append([]Sample(nil), model.observations...)
	samples = append(samples, boundary...)
	samplingSources := append([]Sample(nil), samples...)

	infill := PoissonSamples(region, samplingSources, PoissonMinDistance, seed)
	for _, sample := range infill {
		samples = append(samples, model.assign(sample))
	}

	return refineMeshSamples(region, samples, model)
}

func refineMeshSamples(region Region, samples []Sample, model interpolationModel) ([]Sample, bool) {
	limit := refinementSampleLimit(region, len(samples))

	for {
		triangulation, err := delaunay.Triangulate(delaunayPoints(samples))
		if err != nil {
			return nil, false
		}

		candidates, oversized := refinementSamples(region, samples, triangulation.Triangles)
		if !oversized {
			return samples, true
		}

		if len(candidates) == 0 || len(samples)+len(candidates) > limit {
			return nil, false
		}

		for _, candidate := range candidates {
			samples = append(samples, model.assign(candidate))
		}
	}
}

// A grid at half the maximum edge has cell diagonals below the edge limit, so
// its vertices are a conservative refinement bound over the region's bounds.
// Existing samples do not consume this allowance, so it is added to sampleCount.
func refinementSampleLimit(region Region, sampleCount int) int {
	bounds := region.Bounds()
	spacing := MaxTriangleEdge / 2
	columns := math.Ceil((bounds.MaxX-bounds.MinX)/spacing) + 1
	rows := math.Ceil((bounds.MaxY-bounds.MinY)/spacing) + 1
	maxInt := int(^uint(0) >> 1)

	if sampleCount >= maxInt ||
		!isFinite(columns) ||
		!isFinite(rows) ||
		columns >= float64(maxInt) ||
		rows >= float64(maxInt) {
		return maxInt
	}

	columnCount := int(columns)
	rowCount := int(rows)

	remaining := maxInt - sampleCount
	if columnCount > remaining/rowCount {
		return maxInt
	}

	return sampleCount + columnCount*rowCount
}

func refinementSamples(
	region Region,
	samples []Sample,
	indexes []int,
) ([]Sample, bool) {
	candidates := make([]Sample, 0)

	var oversized bool

	for index := 0; index+2 < len(indexes); index += 3 {
		triangle, ok := triangleAt(samples, [3]int{
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

		candidate, found := refinementSample(region, triangle, samples, candidates)
		if found {
			candidates = append(candidates, candidate)
		}
	}

	return candidates, oversized
}

// Splitting the longest edge ensures the next triangulation cannot retain the offending edge.
func refinementSample(region Region, target Triangle, samples, candidates []Sample) (Sample, bool) {
	start, end, _ := longestTriangleEdge(target)
	candidate := Sample{Position: geometry.Midpoint(start.Position, end.Position)}

	if !isFiniteSample(candidate) ||
		!region.Contains(candidate.Position.X, candidate.Position.Y) ||
		isDuplicate(candidate, samples) ||
		isDuplicate(candidate, candidates) {
		return Sample{}, false
	}

	return candidate, true
}

func regionTriangles(region Region, samples []Sample, indexes []int) ([]Triangle, bool) {
	triangles := make([]Triangle, 0, len(indexes)/3)
	for index := 0; index+2 < len(indexes); index += 3 {
		triangle, ok := triangleAt(samples, [3]int{
			indexes[index],
			indexes[index+1],
			indexes[index+2],
		})
		if !ok ||
			isDegenerateTriangle(triangle) ||
			triangleIsUnsupported(triangle) ||
			!triangleInRegion(region, triangle) {
			continue
		}

		triangle.Value = (triangle.Points[0].Value + triangle.Points[1].Value + triangle.Points[2].Value) / 3
		if LongestEdge(triangle) > MaxTriangleEdge {
			return nil, false
		}

		triangles = append(triangles, triangle)
	}

	return triangles, true
}

func triangleIsUnsupported(triangle Triangle) bool {
	for _, sample := range triangle.Points {
		if sample.unsupported {
			return true
		}
	}

	return false
}

func triangleAt(samples []Sample, indexes [3]int) (Triangle, bool) {
	var triangle Triangle

	for sampleIndex, index := range indexes {
		if index < 0 || index >= len(samples) {
			return Triangle{}, false
		}

		triangle.Points[sampleIndex] = samples[index]
	}

	return triangle, true
}

func isDegenerateTriangle(triangle Triangle) bool {
	first, second, third := triangle.Points[0], triangle.Points[1], triangle.Points[2]
	start, end, _ := longestTriangleEdge(triangle)
	midpointSample := Sample{Position: geometry.Midpoint(start.Position, end.Position)}

	firstToSecond := first.Position.VectorTo(second.Position)
	firstToThird := first.Position.VectorTo(third.Position)

	return firstToSecond.X*firstToThird.Y == firstToSecond.Y*firstToThird.X ||
		isDuplicate(midpointSample, triangle.Points[:])
}

// LongestEdge returns the length of a triangle's longest side.
func LongestEdge(triangle Triangle) float64 {
	_, _, length := longestTriangleEdge(triangle)

	return length
}

func longestTriangleEdge(triangle Triangle) (start, end Sample, longest float64) {
	for index, sample := range triangle.Points {
		next := triangle.Points[(index+1)%len(triangle.Points)]
		if length := sample.Position.DistanceTo(next.Position); length > longest {
			start, end, longest = sample, next, length
		}
	}

	return start, end, longest
}

func delaunayPoints(samples []Sample) []delaunay.Point {
	result := make([]delaunay.Point, len(samples))
	for index, sample := range samples {
		result[index] = delaunay.Point{X: sample.Position.X, Y: sample.Position.Y}
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

	var center geometry.Point

	for _, sample := range triangle.Points {
		if !region.Contains(sample.Position.X, sample.Position.Y) {
			return false
		}

		center.X += sample.Position.X
		center.Y += sample.Position.Y
	}

	center.X /= float64(len(triangle.Points))
	center.Y /= float64(len(triangle.Points))

	return region.Contains(center.X, center.Y)
}

func boundarySamples(region Region, originals []Sample) []Sample {
	loops := BoundaryLoops(region, MaxTriangleEdge)
	if len(loops) == 0 {
		return nil
	}

	candidateCount := 0
	for _, loop := range loops {
		candidateCount += len(loop)
	}

	candidates := make([]Sample, 0, candidateCount)
	for _, loop := range loops {
		candidates = append(candidates, loop...)
	}

	samples := make([]Sample, 0, len(candidates))
	for _, candidate := range candidates {
		if !isFiniteSample(candidate) ||
			isDuplicate(candidate, originals) ||
			isDuplicate(candidate, samples) {
			continue
		}

		candidate.Original = false
		samples = append(samples, candidate)
	}

	return samples
}

func isDuplicate(sample Sample, samples []Sample) bool {
	for _, existing := range samples {
		if sample.Position == existing.Position {
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

	center := geometry.Point{X: annulus.CX, Y: annulus.CY}

	for _, sample := range triangle.Points {
		if !isFiniteSample(sample) {
			return false
		}

		if sample.Position.DistanceSquaredTo(center) > outerRadiusSquared {
			return false
		}
	}

	return true
}

func triangleAvoidsInnerRadius(annulus Annulus, triangle Triangle) bool {
	if annulus.InnerRadius == 0 {
		return true
	}

	if pointStrictlyInTriangle(
		geometry.Point{X: annulus.CX, Y: annulus.CY},
		triangle,
	) {
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

	center := geometry.Point{X: annulus.CX, Y: annulus.CY}
	for _, sample := range triangle.Points {
		if sample.Position.DistanceSquaredTo(center) < innerRadiusSquared {
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
	center := geometry.Point{X: annulus.CX, Y: annulus.CY}

	for index, start := range triangle.Points {
		end := triangle.Points[(index+1)%len(triangle.Points)]
		if squaredDistanceToSegment(center, start, end) < innerRadiusSquared {
			return false
		}
	}

	return true
}

func pointStrictlyInTriangle(point geometry.Point, triangle Triangle) bool {
	var hasPositive, hasNegative bool

	for index, start := range triangle.Points {
		end := triangle.Points[(index+1)%len(triangle.Points)]
		edge := start.Position.VectorTo(end.Position)
		toPoint := start.Position.VectorTo(point)
		crossProduct := edge.X*toPoint.Y - edge.Y*toPoint.X
		hasPositive = hasPositive || crossProduct > 0
		hasNegative = hasNegative || crossProduct < 0
	}

	return hasPositive != hasNegative
}

func squaredDistanceToSegment(point geometry.Point, start, end Sample) float64 {
	segment := start.Position.VectorTo(end.Position)
	lengthSquared := segment.LengthSquared()

	if lengthSquared == 0 {
		return point.DistanceSquaredTo(start.Position)
	}

	fraction := start.Position.VectorTo(point).Dot(segment) / lengthSquared
	fraction = math.Max(0, math.Min(1, fraction))
	nearest := start.Position.Translate(segment.Scale(fraction))

	return point.DistanceSquaredTo(nearest)
}
