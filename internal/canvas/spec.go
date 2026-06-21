package canvas

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
)

// ShapeStyle bundles the visual properties shared by all closed-shape specs.
type ShapeStyle struct {
	Fill        inks.Ink
	Border      inks.Ink
	BorderWidth float64
}

// RectangleSpec defines the visual template for rectangles.
type RectangleSpec struct {
	ShapeStyle
}

// DiscSpec defines the visual template for circles/discs.
type DiscSpec struct {
	ShapeStyle
}

// LineSpec defines the visual template for lines.
type LineSpec struct {
	Stroke      inks.Ink
	StrokeWidth float64
}
