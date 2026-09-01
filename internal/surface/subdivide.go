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
	samples := clipSamplesToBand(original, breakpoints, bandIndex)
	if !validPolygon(samples) {
		return Polygon{}, false
	}

	value, ok := representativeValue(samples, breakpoints, bandIndex)
	if !ok {
		return Polygon{}, false
	}

	return Polygon{
		Points: append([]Sample(nil), samples...),
		Value:  value,
	}, true
}

func clipSamplesToBand(original []Sample, breakpoints []float64, bandIndex int) []Sample {
	samples := append([]Sample(nil), original...)

	switch {
	case bandIndex == 0:
		return clipBelow(samples, breakpoints[0])
	case bandIndex == len(breakpoints):
		return clipAtOrAbove(samples, breakpoints[bandIndex-1])
	default:
		samples = clipAtOrAbove(samples, breakpoints[bandIndex-1])

		return clipBelow(samples, breakpoints[bandIndex])
	}
}

func validTriangle(triangle Triangle) bool {
	if !isFinite(triangle.Value) {
		return false
	}

	for _, sample := range triangle.Points {
		if !isFiniteSample(sample) || !isFinite(sample.Value) {
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

func clipBelow(samples []Sample, breakpoint float64) []Sample {
	return clipPolygon(samples, breakpoint, func(value float64) bool {
		return value < breakpoint
	})
}

func clipAtOrAbove(samples []Sample, breakpoint float64) []Sample {
	return clipPolygon(samples, breakpoint, func(value float64) bool {
		return value >= breakpoint
	})
}

func clipPolygon(samples []Sample, breakpoint float64, inside func(float64) bool) []Sample {
	if len(samples) == 0 {
		return nil
	}

	clipped := make([]Sample, 0, len(samples)+1)
	for index, start := range samples {
		end := samples[(index+1)%len(samples)]
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
		return appendSample(clipped, end)
	case clipKeepIntersection:
		return appendIntersectionSample(clipped, start, end, breakpoint)
	case clipKeepIntersectionAndEnd:
		clipped = appendIntersectionSample(clipped, start, end, breakpoint)

		return appendSample(clipped, end)
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

func appendIntersectionSample(samples []Sample, start, end Sample, breakpoint float64) []Sample {
	if start.Value == end.Value {
		return samples
	}

	return appendSample(samples, edgeIntersection(start, end, breakpoint))
}

func appendSample(samples []Sample, sample Sample) []Sample {
	if len(samples) > 0 && sameSample(samples[len(samples)-1], sample) {
		if sample.Original {
			samples[len(samples)-1].Original = true
		}

		return samples
	}

	return append(samples, sample)
}

func normalizePolygon(samples []Sample) []Sample {
	if len(samples) == 0 {
		return nil
	}

	normalized := make([]Sample, 0, len(samples))
	for _, sample := range samples {
		normalized = appendSample(normalized, sample)
	}

	if len(normalized) > 1 && sameSample(normalized[0], normalized[len(normalized)-1]) {
		if normalized[len(normalized)-1].Original {
			normalized[0].Original = true
		}

		normalized = normalized[:len(normalized)-1]
	}

	return normalized
}

func sameSample(a, b Sample) bool {
	return a.Position == b.Position && a.Value == b.Value
}

func edgeIntersection(start, end Sample, breakpoint float64) Sample {
	fraction := (breakpoint - start.Value) / (end.Value - start.Value)
	inverseFraction := 1 - fraction

	return Sample{
		Position: geometry.Point{
			X: inverseFraction*start.Position.X + fraction*end.Position.X,
			Y: inverseFraction*start.Position.Y + fraction*end.Position.Y,
		},
		Value: breakpoint,
	}
}

func validPolygon(samples []Sample) bool {
	if len(samples) < 3 {
		return false
	}

	for _, sample := range samples {
		if !isFiniteSample(sample) || !isFinite(sample.Value) {
			return false
		}
	}

	area := polygonArea(samples)

	return isFinite(area) && area > 0
}

func polygonArea(samples []Sample) float64 {
	area := 0.0

	for index, sample := range samples {
		next := samples[(index+1)%len(samples)]
		area += sample.Position.X*next.Position.Y - next.Position.X*sample.Position.Y
	}

	return math.Abs(area) / 2
}

func representativeValue(samples []Sample, breakpoints []float64, bandIndex int) (float64, bool) {
	// Reuse a fragment vertex value so the representative stays within the
	// triangle's realized scalar range and follows BucketIndex semantics exactly.
	for _, sample := range samples {
		if bucketIndex(breakpoints, sample.Value) == bandIndex {
			return sample.Value, true
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
