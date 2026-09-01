//revive:disable-next-line:max-public-structs Polygon must stay exported for the palette-band subdivision API.
package surface

import (
	"math"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

const (
	MaxTriangleEdge          = 8.0
	MaxBoundarySegmentLength = 1.0
	PoissonMinDistance       = 4.0
)

type Sample struct {
	Position    geometry.Point
	Value       float64
	unsupported bool
	Original    bool
}

type Region interface {
	Bounds() geometry.Rect
	Contains(point geometry.Point) bool
}

type Annulus struct {
	CX          float64
	CY          float64
	InnerRadius float64
	OuterRadius float64
}

func (a Annulus) Bounds() geometry.Rect {
	return geometry.Rect{
		Min: geometry.Point{X: a.CX - a.OuterRadius, Y: a.CY - a.OuterRadius},
		Max: geometry.Point{X: a.CX + a.OuterRadius, Y: a.CY + a.OuterRadius},
	}
}

func (a Annulus) Contains(point geometry.Point) bool {
	if a.InnerRadius < 0 || a.OuterRadius < a.InnerRadius {
		return false
	}

	dx := point.X - a.CX
	dy := point.Y - a.CY
	distanceSquared := dx*dx + dy*dy

	return distanceSquared >= a.InnerRadius*a.InnerRadius &&
		distanceSquared <= a.OuterRadius*a.OuterRadius
}

// rectBoundaryLoops returns the rectangle perimeter as one ordered loop.
func rectBoundaryLoops(r geometry.Rect, maximumSegmentLength float64) [][]Sample {
	if !isFinite(maximumSegmentLength) ||
		maximumSegmentLength <= 0 ||
		!r.Valid() ||
		r.Empty() {
		return nil
	}

	corners := [4]Sample{
		{Position: geometry.NewPoint(r.Min.X, r.Min.Y)},
		{Position: geometry.NewPoint(r.Max.X, r.Min.Y)},
		{Position: geometry.NewPoint(r.Max.X, r.Max.Y)},
		{Position: geometry.NewPoint(r.Min.X, r.Max.Y)},
	}
	loop := make([]Sample, 0, len(corners))

	for index, start := range corners {
		end := corners[(index+1)%len(corners)]
		loop = append(loop, segmentBoundaryPoints(start, end, maximumSegmentLength)...)
	}

	return [][]Sample{loop}
}

// BoundaryLoops returns the annulus outer boundary followed by its inner boundary.
func (a Annulus) BoundaryLoops(maximumSegmentLength float64) [][]Sample {
	if !validBoundaryLoopLength(maximumSegmentLength) || !validAnnulus(a) {
		return nil
	}

	return annulusBoundaryLoops(a, maximumSegmentLength)
}

// BoundaryLoops returns ordered boundary loops supplied by region.
func BoundaryLoops(region Region, maximumSegmentLength float64) [][]Sample {
	if isNilInterfaceValue(region) {
		return nil
	}

	switch value := region.(type) {
	case geometry.Rect:
		return rectBoundaryLoops(value, maximumSegmentLength)
	case Annulus:
		return value.BoundaryLoops(maximumSegmentLength)
	case *Annulus:
		return value.BoundaryLoops(maximumSegmentLength)
	default:
		return nil
	}
}

func annulusBoundaryLoops(a Annulus, maximumSegmentLength float64) [][]Sample {
	loops := make([][]Sample, 0, 2)

	loops = appendCircularBoundaryLoop(loops, a.CX, a.CY, a.OuterRadius, maximumSegmentLength)
	loops = appendCircularBoundaryLoop(loops, a.CX, a.CY, a.InnerRadius, maximumSegmentLength)

	return loops
}

func appendCircularBoundaryLoop(
	loops [][]Sample,
	cx, cy, radius, maximumSegmentLength float64,
) [][]Sample {
	if radius <= 0 {
		return loops
	}

	return append(
		loops,
		circularBoundaryPoints(cx, cy, radius, maximumSegmentLength),
	)
}

func validBoundaryLoopLength(maximumSegmentLength float64) bool {
	return isFinite(maximumSegmentLength) && maximumSegmentLength > 0
}

func segmentBoundaryPoints(start, end Sample, maximumSegmentLength float64) []Sample {
	length := start.Position.DistanceTo(end.Position)
	if !isFinite(length) {
		return nil
	}

	segments := max(1, int(math.Ceil(length/maximumSegmentLength)))
	segmentVector := start.Position.VectorTo(end.Position)

	points := make([]Sample, 0, segments)
	for index := range segments {
		fraction := float64(index) / float64(segments)
		points = append(points, Sample{
			Position: start.Position.Translate(segmentVector.Scale(fraction)),
		})
	}

	return points
}

func circularBoundaryPoints(cx, cy, radius, maximumSegmentLength float64) []Sample {
	if radius == 0 {
		return nil
	}

	segments := max(3, int(math.Ceil(2*math.Pi*radius/maximumSegmentLength)))

	points := make([]Sample, 0, segments)
	for index := range segments {
		angle := 2 * math.Pi * float64(index) / float64(segments)
		points = append(points, Sample{
			Position: geometry.NewPoint(cx+radius*math.Cos(angle), cy+radius*math.Sin(angle)),
		})
	}

	return points
}

type Triangle struct {
	Points [3]Sample
	Value  float64
}

type Polygon struct {
	Points []Sample
	Value  float64
}
