package canvas

// ShapeStyle bundles the visual properties shared by all closed-shape specs.
type ShapeStyle struct {
	Fill        Ink
	Border      Ink
	BorderWidth float64
	ShowLabel   bool
	LabelInk    Ink
	LabelStyle  LabelStyle
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
	Stroke      Ink
	StrokeWidth float64
}
