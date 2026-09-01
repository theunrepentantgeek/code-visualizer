package geometry

import "math"

type Size struct {
	Width  float64
	Height float64
}

func (s Size) Valid() bool {
	return !math.IsNaN(s.Width) && !math.IsInf(s.Width, 0) &&
		!math.IsNaN(s.Height) && !math.IsInf(s.Height, 0) &&
		s.Width >= 0 && s.Height >= 0
}

func (s Size) Empty() bool {
	return s.Valid() && (s.Width == 0 || s.Height == 0)
}

func (s Size) Area() float64 { return s.Width * s.Height }

func (s Size) Scale(factor float64) Size {
	return Size{Width: s.Width * factor, Height: s.Height * factor}
}

func (s Size) AspectRatio() (float64, bool) {
	if !s.Valid() || s.Height == 0 {
		return 0, false
	}

	return s.Width / s.Height, true
}
