package geometry

import "math"

type Vector struct {
	X float64
	Y float64
}

// ZeroVector is the additive identity, representing no displacement.
// It's a var, not a const, because Go structs cannot be declared const.
var ZeroVector = Vector{}

// NewVector constructs a Vector from Cartesian components.
func NewVector(x, y float64) Vector {
	return Vector{X: x, Y: y}
}

// NewRadialVector constructs a Vector from polar coordinates, converting an
// angle in radians and a length into Cartesian components.
func NewRadialVector(angle, length float64) Vector {
	return Vector{X: length * math.Cos(angle), Y: length * math.Sin(angle)}
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

// Unit returns v scaled to length 1, along with true, or ZeroVector and false
// if v is invalid or has zero length.
//
// Naively scaling v by 1/v.Length() breaks down at the extremes: for
// components near math.MaxFloat64, Length can itself overflow to +Inf (the
// true magnitude genuinely exceeds float64's range), so 1/Length underflows
// to 0 and the result silently collapses to the zero vector. For subnormal
// components, Length can underflow towards the smallest representable
// positive value, so 1/Length overflows to +Inf and the result becomes
// invalid. Pre-scaling both components by the largest absolute component
// first brings them into a range where Length is always safely
// representable, preserving direction at either extreme.
func (v Vector) Unit() (Vector, bool) {
	if !v.Valid() {
		return ZeroVector, false
	}

	scale := math.Max(math.Abs(v.X), math.Abs(v.Y))
	if scale == 0 {
		return ZeroVector, false
	}

	scaled := Vector{X: v.X / scale, Y: v.Y / scale}
	length := scaled.Length()

	return Vector{X: scaled.X / length, Y: scaled.Y / length}, true
}
