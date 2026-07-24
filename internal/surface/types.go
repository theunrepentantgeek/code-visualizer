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

type Triangle struct {
	Points [3]Point
	Value  float64
}

func Distance(a, b Point) float64 {
	return math.Hypot(a.X-b.X, a.Y-b.Y)
}
