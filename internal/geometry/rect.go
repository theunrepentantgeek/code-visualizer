package geometry

import "math"

type Rect struct {
	Min Point
	Max Point
}

func RectFromPositionSize(position Point, size Size) Rect {
	return Rect{
		Min: position,
		Max: Point{X: position.X + size.Width, Y: position.Y + size.Height},
	}
}

func (r Rect) Valid() bool {
	return r.Min.Valid() && r.Max.Valid() &&
		r.Min.X <= r.Max.X && r.Min.Y <= r.Max.Y
}

func (r Rect) Empty() bool {
	return r.Valid() && (r.Min.X == r.Max.X || r.Min.Y == r.Max.Y)
}

func (r Rect) Width() float64  { return r.Max.X - r.Min.X }
func (r Rect) Height() float64 { return r.Max.Y - r.Min.Y }
func (r Rect) Size() Size      { return Size{Width: r.Width(), Height: r.Height()} }
func (r Rect) Center() Point   { return Midpoint(r.Min, r.Max) }

func (r Rect) Contains(point Point) bool {
	return r.Valid() && point.Valid() &&
		point.X >= r.Min.X && point.X <= r.Max.X &&
		point.Y >= r.Min.Y && point.Y <= r.Max.Y
}

func (r Rect) Translate(offset Vector) Rect {
	return Rect{Min: r.Min.Translate(offset), Max: r.Max.Translate(offset)}
}

func (r Rect) Inset(amount float64) (Rect, bool) {
	if !r.Valid() || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return Rect{}, false
	}

	result := Rect{
		Min: Point{X: r.Min.X + amount, Y: r.Min.Y + amount},
		Max: Point{X: r.Max.X - amount, Y: r.Max.Y - amount},
	}

	return result, result.Valid()
}

func (r Rect) ExpandToInclude(point Point) (Rect, bool) {
	if !r.Valid() || !point.Valid() {
		return Rect{}, false
	}

	return Rect{
		Min: Point{X: min(r.Min.X, point.X), Y: min(r.Min.Y, point.Y)},
		Max: Point{X: max(r.Max.X, point.X), Y: max(r.Max.Y, point.Y)},
	}, true
}

func (r Rect) Union(other Rect) (Rect, bool) {
	if !r.Valid() || !other.Valid() {
		return Rect{}, false
	}

	return Rect{
		Min: Point{X: min(r.Min.X, other.Min.X), Y: min(r.Min.Y, other.Min.Y)},
		Max: Point{X: max(r.Max.X, other.Max.X), Y: max(r.Max.Y, other.Max.Y)},
	}, true
}
