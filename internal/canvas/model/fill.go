package model

import "image/color"

// Point represents a 2D coordinate as fractions (may exceed [0,1]).
type Point struct {
	X, Y float64
}

// Fill is a sealed interface describing how a shape's interior is painted.
type Fill interface {
	isFill()
}

// SolidFill paints a uniform colour.
type SolidFill struct {
	Color color.RGBA
}

// RadialGradientFill paints a radial gradient from a centre colour
// (at the focus point) to an edge colour (at the shape boundary).
type RadialGradientFill struct {
	Center color.RGBA
	Edge   color.RGBA
	Focus  Point
}

func (SolidFill) isFill()          {}
func (RadialGradientFill) isFill() {}

// SolidColor extracts the primary colour from any Fill, falling back to opaque black.
// For SolidFill it returns the fill colour; for RadialGradientFill it returns
// the centre colour; for any unknown fill type it returns opaque black.
func SolidColor(f Fill) color.RGBA {
	switch v := f.(type) {
	case SolidFill:
		return v.Color
	case RadialGradientFill:
		return v.Center
	default:
		return color.RGBA{A: 255}
	}
}
