package surface

import (
	"math"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

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
		Points: append([]Sample(nil), triangle.Points[:]...),
		Value:  triangle.Value,
	}
}

func subdivideTriangleBands(triangle Triangle, breakpoints []float64) []Polygon {
	polygons := make([]Polygon, 0, len(breakpoints)+1)

	original := append([]Sample(nil), triangle.Points[:]...)

	for bandIndex := range len(breakpoints) + 1 {
		polygon, ok := subdivideTriangleBand(original, breakpoints, bandIndex)
		if !ok {
			continue
		}

		polygons = append(polygons, polygon)
	}

	return polygons
}

func subdivideTriangleBand(original []Sample, breakpoints []float64, bandIndex int) (Polygon, bool) {
	points := clipPointsToBand(original, breakpoints, bandIndex)
	if !validPolygon(points) {
		return Polygon{}, false
	}

	value, ok := representativeValue(points, breakpoints, bandIndex)
	if !ok {
		return Polygon{}, false
	}

	return Polygon{
		Points: append([]Sample(nil), points...),
		Value:  value,
	}, true
}

func clipPointsToBand(original []Sample, breakpoints []float64, bandIndex int) []Sample {
	points := append([]Sample(nil), original...)

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
		if !isFiniteSample(point) || !isFinite(point.Value) {
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

func clipBelow(points []Sample, breakpoint float64) []Sample {
	return clipPolygon(points, breakpoint, func(value float64) bool {
		return value < breakpoint
	})
}

func clipAtOrAbove(points []Sample, breakpoint float64) []Sample {
	return clipPolygon(points, breakpoint, func(value float64) bool {
		return value >= breakpoint
	})
}

func clipPolygon(points []Sample, breakpoint float64, inside func(float64) bool) []Sample {
	if len(points) == 0 {
		return nil
	}

	clipped := make([]Sample, 0, len(points)+1)
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
	clipped []Sample,
	start, end Sample,
	breakpoint float64,
	inside func(float64) bool,
) []Sample {
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

func appendIntersectionPoint(points []Sample, start, end Sample, breakpoint float64) []Sample {
	if start.Value == end.Value {
		return points
	}

	return appendPoint(points, edgeIntersection(start, end, breakpoint))
}

func appendPoint(points []Sample, point Sample) []Sample {
	if len(points) > 0 && samePoint(points[len(points)-1], point) {
		if point.Original {
			points[len(points)-1].Original = true
		}

		return points
	}

	return append(points, point)
}

func normalizePolygon(points []Sample) []Sample {
	if len(points) == 0 {
		return nil
	}

	normalized := make([]Sample, 0, len(points))
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

func samePoint(a, b Sample) bool {
	return a.Position == b.Position && a.Value == b.Value
}

func edgeIntersection(start, end Sample, breakpoint float64) Sample {
	fraction := (breakpoint - start.Value) / (end.Value - start.Value)

	return Sample{
		Position: geometry.Lerp(start.Position, end.Position, fraction),
		Value:    breakpoint,
	}
}

func validPolygon(points []Sample) bool {
	if len(points) < 3 {
		return false
	}

	for _, point := range points {
		if !isFiniteSample(point) || !isFinite(point.Value) {
			return false
		}
	}

	area := polygonArea(points)

	return isFinite(area) && area > 0
}

func polygonArea(points []Sample) float64 {
	area := 0.0

	for index, point := range points {
		next := points[(index+1)%len(points)]
		area += point.Position.X*next.Position.Y - next.Position.X*point.Position.Y
	}

	return math.Abs(area) / 2
}

func representativeValue(points []Sample, breakpoints []float64, bandIndex int) (float64, bool) {
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
