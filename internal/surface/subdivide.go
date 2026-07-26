package surface

import "math"

func SubdivideTriangle(triangle Triangle, breakpoints []float64) []Polygon {
	if !validTriangle(triangle) || !validBreakpoints(breakpoints) {
		return nil
	}

	if len(breakpoints) == 0 {
		return []Polygon{wholeTrianglePolygon(triangle)}
	}

	return subdivideTriangleBands(triangle, breakpoints)
}

func wholeTrianglePolygon(triangle Triangle) Polygon {
	return Polygon{
		Points: append([]Point(nil), triangle.Points[:]...),
		Value:  triangle.Value,
	}
}

func subdivideTriangleBands(triangle Triangle, breakpoints []float64) []Polygon {
	polygons := make([]Polygon, 0, len(breakpoints)+1)

	original := append([]Point(nil), triangle.Points[:]...)

	for bandIndex := range len(breakpoints) + 1 {
		polygon, ok := subdivideTriangleBand(original, breakpoints, bandIndex)
		if !ok {
			continue
		}

		polygons = append(polygons, polygon)
	}

	return polygons
}

func subdivideTriangleBand(original []Point, breakpoints []float64, bandIndex int) (Polygon, bool) {
	points := clipPointsToBand(original, breakpoints, bandIndex)
	if !validPolygon(points) {
		return Polygon{}, false
	}

	value, ok := representativeValue(points, breakpoints, bandIndex)
	if !ok {
		return Polygon{}, false
	}

	return Polygon{
		Points: append([]Point(nil), points...),
		Value:  value,
	}, true
}

func clipPointsToBand(original []Point, breakpoints []float64, bandIndex int) []Point {
	points := append([]Point(nil), original...)

	switch {
	case bandIndex == 0:
		return clipBelow(points, breakpoints[0])
	case bandIndex == len(breakpoints):
		return clipAtOrAbove(points, breakpoints[bandIndex-1])
	default:
		points = clipAtOrAbove(points, breakpoints[bandIndex-1])

		return clipBelow(points, breakpoints[bandIndex])
	}
}

func validTriangle(triangle Triangle) bool {
	if !isFinite(triangle.Value) {
		return false
	}

	for _, point := range triangle.Points {
		if !isFinitePoint(point) || !isFinite(point.Value) {
			return false
		}
	}

	return !isDegenerateTriangle(triangle)
}

func validBreakpoints(breakpoints []float64) bool {
	for index, breakpoint := range breakpoints {
		if !isFinite(breakpoint) {
			return false
		}

		if index > 0 && breakpoints[index-1] >= breakpoint {
			return false
		}
	}

	return true
}

func clipBelow(points []Point, breakpoint float64) []Point {
	return clipPolygon(points, breakpoint, func(value float64) bool {
		return value < breakpoint
	})
}

func clipAtOrAbove(points []Point, breakpoint float64) []Point {
	return clipPolygon(points, breakpoint, func(value float64) bool {
		return value >= breakpoint
	})
}

func clipPolygon(points []Point, breakpoint float64, inside func(float64) bool) []Point {
	if len(points) == 0 {
		return nil
	}

	clipped := make([]Point, 0, len(points)+1)
	for index, start := range points {
		end := points[(index+1)%len(points)]
		clipped = clipPolygonEdge(clipped, start, end, breakpoint, inside)
	}

	return normalizePolygon(clipped)
}

type clipTransition int

const (
	clipDrop clipTransition = iota
	clipKeepEnd
	clipKeepIntersection
	clipKeepIntersectionAndEnd
)

func clipPolygonEdge(
	clipped []Point,
	start, end Point,
	breakpoint float64,
	inside func(float64) bool,
) []Point {
	switch clipTransitionForEdge(inside(start.Value), inside(end.Value)) {
	case clipKeepEnd:
		return appendPoint(clipped, end)
	case clipKeepIntersection:
		return appendIntersectionPoint(clipped, start, end, breakpoint)
	case clipKeepIntersectionAndEnd:
		clipped = appendIntersectionPoint(clipped, start, end, breakpoint)

		return appendPoint(clipped, end)
	case clipDrop:
		return clipped
	default:
		panic("unhandled clip transition")
	}
}

func clipTransitionForEdge(startInside, endInside bool) clipTransition {
	switch {
	case startInside && endInside:
		return clipKeepEnd
	case startInside && !endInside:
		return clipKeepIntersection
	case !startInside && endInside:
		return clipKeepIntersectionAndEnd
	case !startInside && !endInside:
		return clipDrop
	default:
		return clipDrop
	}
}

func appendIntersectionPoint(points []Point, start, end Point, breakpoint float64) []Point {
	if start.Value == end.Value {
		return points
	}

	return appendPoint(points, edgeIntersection(start, end, breakpoint))
}

func appendPoint(points []Point, point Point) []Point {
	if len(points) > 0 && samePoint(points[len(points)-1], point) {
		if point.Original {
			points[len(points)-1].Original = true
		}

		return points
	}

	return append(points, point)
}

func normalizePolygon(points []Point) []Point {
	if len(points) == 0 {
		return nil
	}

	normalized := make([]Point, 0, len(points))
	for _, point := range points {
		normalized = appendPoint(normalized, point)
	}

	if len(normalized) > 1 && samePoint(normalized[0], normalized[len(normalized)-1]) {
		if normalized[len(normalized)-1].Original {
			normalized[0].Original = true
		}

		normalized = normalized[:len(normalized)-1]
	}

	return normalized
}

func samePoint(a, b Point) bool {
	return a.X == b.X && a.Y == b.Y && a.Value == b.Value
}

func edgeIntersection(start, end Point, breakpoint float64) Point {
	fraction := (breakpoint - start.Value) / (end.Value - start.Value)
	inverseFraction := 1 - fraction

	return Point{
		X:     inverseFraction*start.X + fraction*end.X,
		Y:     inverseFraction*start.Y + fraction*end.Y,
		Value: breakpoint,
	}
}

func validPolygon(points []Point) bool {
	if len(points) < 3 {
		return false
	}

	for _, point := range points {
		if !isFinitePoint(point) || !isFinite(point.Value) {
			return false
		}
	}

	area := polygonArea(points)

	return isFinite(area) && area > 0
}

func polygonArea(points []Point) float64 {
	area := 0.0

	for index, point := range points {
		next := points[(index+1)%len(points)]
		area += point.X*next.Y - next.X*point.Y
	}

	return math.Abs(area) / 2
}

func representativeValue(points []Point, breakpoints []float64, bandIndex int) (float64, bool) {
	// Reuse a fragment vertex value so the representative stays within the
	// triangle's realized scalar range and follows BucketIndex semantics exactly.
	for _, point := range points {
		if bucketIndex(breakpoints, point.Value) == bandIndex {
			return point.Value, true
		}
	}

	return 0, false
}

func bucketIndex(breakpoints []float64, value float64) int {
	for index, breakpoint := range breakpoints {
		if value < breakpoint {
			return index
		}
	}

	return len(breakpoints)
}
