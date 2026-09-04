package geometry

import "math"

type Circle struct {
	Center Point
	Radius float64
}

func NewCircle(center Point, radius float64) Circle {
	return Circle{Center: center, Radius: radius}
}

func (c Circle) Valid() bool {
	return c.Center.Valid() &&
		!math.IsNaN(c.Radius) && !math.IsInf(c.Radius, 0) &&
		c.Radius >= 0
}

func (c Circle) Contains(point Point) bool {
	if !c.Valid() || !point.Valid() {
		return false
	}

	return c.Center.DistanceSquaredTo(point) <= c.Radius*c.Radius
}

func (c Circle) Encloses(other Circle) bool {
	if !c.Valid() || !other.Valid() || other.Radius > c.Radius {
		return false
	}

	distance := c.Center.DistanceTo(other.Center)

	return distance+other.Radius <= c.Radius
}

func (c Circle) Intersects(other Circle) bool {
	if !c.Valid() || !other.Valid() {
		return false
	}

	radii := c.Radius + other.Radius

	return c.Center.DistanceSquaredTo(other.Center) <= radii*radii
}

func (c Circle) Bounds() Rect {
	if !c.Valid() {
		return Rect{}
	}

	offset := Vector{X: c.Radius, Y: c.Radius}

	return Rect{
		Min: c.Center.Translate(offset.Scale(-1)),
		Max: c.Center.Translate(offset),
	}
}

func (c Circle) Translate(offset Vector) Circle {
	return NewCircle(c.Center.Translate(offset), c.Radius)
}
