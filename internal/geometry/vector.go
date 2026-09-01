package geometry

import "math"

type Vector struct {
	X float64
	Y float64
}

func (v Vector) Valid() bool {
	return !math.IsNaN(v.X) && !math.IsInf(v.X, 0) &&
		!math.IsNaN(v.Y) && !math.IsInf(v.Y, 0)
}

func (v Vector) Add(other Vector) Vector {
	return Vector{X: v.X + other.X, Y: v.Y + other.Y}
}

func (v Vector) Subtract(other Vector) Vector {
	return Vector{X: v.X - other.X, Y: v.Y - other.Y}
}

func (v Vector) Scale(factor float64) Vector {
	return Vector{X: v.X * factor, Y: v.Y * factor}
}

func (v Vector) Dot(other Vector) float64 {
	return v.X*other.X + v.Y*other.Y
}

func (v Vector) LengthSquared() float64 { return v.Dot(v) }
func (v Vector) Length() float64        { return math.Hypot(v.X, v.Y) }

func (v Vector) Unit() (Vector, bool) {
	length := v.Length()
	if !v.Valid() || length == 0 {
		return Vector{}, false
	}
	return v.Scale(1 / length), true
}
