package surface

import "math"

const (
	MaxTriangleEdge    = 8.0
	PoissonMinDistance = 4.0
	IDWPower           = 2.0
)

type Point struct {
	X        float64
	Y        float64
	Value    float64
	Original bool
}

type Rect struct {
	MinX float64
	MinY float64
	MaxX float64
	MaxY float64
}

func (r Rect) Bounds() Rect {
	return r
}

func (r Rect) Contains(x, y float64) bool {
	return x >= r.MinX && x <= r.MaxX && y >= r.MinY && y <= r.MaxY
}

type Region interface {
	Bounds() Rect
	Contains(x, y float64) bool
}

type boundaryPointProvider interface {
	boundaryPoints(maximumSegmentLength float64) []Point
}

type Annulus struct {
	CX          float64
	CY          float64
	InnerRadius float64
	OuterRadius float64
}

func (a Annulus) Bounds() Rect {
	return Rect{
		MinX: a.CX - a.OuterRadius,
		MinY: a.CY - a.OuterRadius,
		MaxX: a.CX + a.OuterRadius,
		MaxY: a.CY + a.OuterRadius,
	}
}

func (a Annulus) Contains(x, y float64) bool {
	if a.InnerRadius < 0 || a.OuterRadius < a.InnerRadius {
		return false
	}

	dx := x - a.CX
	dy := y - a.CY
	distanceSquared := dx*dx + dy*dy

	return distanceSquared >= a.InnerRadius*a.InnerRadius &&
		distanceSquared <= a.OuterRadius*a.OuterRadius
}

func (r Rect) boundaryPoints(maximumSegmentLength float64) []Point {
	if !isFinite(maximumSegmentLength) ||
		maximumSegmentLength <= 0 ||
		!isFinite(r.MinX) ||
		!isFinite(r.MinY) ||
		!isFinite(r.MaxX) ||
		!isFinite(r.MaxY) ||
		r.MinX >= r.MaxX ||
		r.MinY >= r.MaxY {
		return nil
	}

	corners := [4]Point{
		{X: r.MinX, Y: r.MinY},
		{X: r.MaxX, Y: r.MinY},
		{X: r.MaxX, Y: r.MaxY},
		{X: r.MinX, Y: r.MaxY},
	}
	points := make([]Point, 0, len(corners))

	for index, start := range corners {
		end := corners[(index+1)%len(corners)]
		points = append(points, segmentBoundaryPoints(start, end, maximumSegmentLength)...)
	}

	return points
}

func (a Annulus) boundaryPoints(maximumSegmentLength float64) []Point {
	if !isFinite(maximumSegmentLength) ||
		maximumSegmentLength <= 0 ||
		!isFinite(a.CX) ||
		!isFinite(a.CY) ||
		!isFinite(a.InnerRadius) ||
		!isFinite(a.OuterRadius) ||
		a.InnerRadius < 0 ||
		a.OuterRadius < a.InnerRadius {
		return nil
	}

	points := circularBoundaryPoints(a.CX, a.CY, a.OuterRadius, maximumSegmentLength)
	if a.InnerRadius > 0 {
		points = append(
			points,
			circularBoundaryPoints(a.CX, a.CY, a.InnerRadius, maximumSegmentLength)...,
		)
	}

	return points
}

func segmentBoundaryPoints(start, end Point, maximumSegmentLength float64) []Point {
	length := Distance(start, end)
	if !isFinite(length) {
		return nil
	}

	segments := max(1, int(math.Ceil(length/maximumSegmentLength)))

	points := make([]Point, 0, segments)
	for index := range segments {
		fraction := float64(index) / float64(segments)
		points = append(points, Point{
			X: start.X + (end.X-start.X)*fraction,
			Y: start.Y + (end.Y-start.Y)*fraction,
		})
	}

	return points
}

func circularBoundaryPoints(cx, cy, radius, maximumSegmentLength float64) []Point {
	if radius == 0 {
		return nil
	}

	segments := max(3, int(math.Ceil(2*math.Pi*radius/maximumSegmentLength)))

	points := make([]Point, 0, segments)
	for index := range segments {
		angle := 2 * math.Pi * float64(index) / float64(segments)
		points = append(points, Point{
			X: cx + radius*math.Cos(angle),
			Y: cy + radius*math.Sin(angle),
		})
	}

	return points
}

type Triangle struct {
	Points [3]Point
	Value  float64
}

func Distance(a, b Point) float64 {
	return math.Hypot(a.X-b.X, a.Y-b.Y)
}
