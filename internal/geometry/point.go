package geometry

import "math"

type Point struct {
	X float64
	Y float64
}

func (p Point) Valid() bool {
	return !math.IsNaN(p.X) && !math.IsInf(p.X, 0) &&
		!math.IsNaN(p.Y) && !math.IsInf(p.Y, 0)
}

func (p Point) Translate(vector Vector) Point {
	return Point{X: p.X + vector.X, Y: p.Y + vector.Y}
}

func (p Point) VectorTo(other Point) Vector {
	return Vector{X: other.X - p.X, Y: other.Y - p.Y}
}

func (p Point) DistanceSquaredTo(other Point) float64 {
	if !p.Valid() || !other.Valid() {
		return math.NaN()
	}

	return p.VectorTo(other).LengthSquared()
}

func (p Point) DistanceTo(other Point) float64 {
	if !p.Valid() || !other.Valid() {
		return math.NaN()
	}

	return p.VectorTo(other).Length()
}

func Midpoint(a, b Point) Point {
	return Lerp(a, b, 0.5)
}

func Lerp(a, b Point, fraction float64) Point {
	if fraction == 0 {
		return a
	}
	if fraction == 1 {
		return b
	}

	return Point{
		X: (1-fraction)*a.X + fraction*b.X,
		Y: (1-fraction)*a.Y + fraction*b.Y,
	}
}
