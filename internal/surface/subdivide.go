package surface

import "math"

func SubdivideTriangle(triangle Triangle, breakpoints []float64) []Polygon {
	if !validTriangle(triangle) || !validBreakpoints(breakpoints) {
		return nil
	}

	if len(breakpoints) == 0 {
		return []Polygon{{
			Points: append([]Point(nil), triangle.Points[:]...),
			Value:  triangle.Value,
		}}
	}

	polygons := make([]Polygon, 0, len(breakpoints)+1)
	original := append([]Point(nil), triangle.Points[:]...)

	for bandIndex := range len(breakpoints) + 1 {
		points := append([]Point(nil), original...)
		if bandIndex > 0 {
			points = clipAtOrAbove(points, breakpoints[bandIndex-1])
		}
		if bandIndex < len(breakpoints) {
			points = clipBelow(points, breakpoints[bandIndex])
		}
		if !validPolygon(points) {
			continue
		}

		value, ok := representativeValue(points, breakpoints, bandIndex)
		if !ok {
			continue
		}

		polygons = append(polygons, Polygon{
			Points: append([]Point(nil), points...),
			Value:  value,
		})
	}

	return polygons
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
		startInside := inside(start.Value)
		endInside := inside(end.Value)

		switch {
		case startInside && endInside:
			clipped = append(clipped, end)
		case startInside && !endInside:
			if start.Value != end.Value {
				clipped = appendPoint(clipped, edgeIntersection(start, end, breakpoint))
			}
		case !startInside && endInside:
			if start.Value != end.Value {
				clipped = appendPoint(clipped, edgeIntersection(start, end, breakpoint))
			}
			clipped = appendPoint(clipped, end)
		}
	}

	return normalizePolygon(clipped)
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
