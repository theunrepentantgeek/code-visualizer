package canvas

// Rectangle carries geometry and metric values for rectangular shapes.
type Rectangle struct {
	Spec       *RectangleSpec
	X, Y, W, H float64
	Fill       MetricValue
	Border     MetricValue
	Label      string
}

// Disc carries geometry and metric values for circular shapes.
type Disc struct {
	Spec   *DiscSpec
	X, Y   float64
	Radius float64
	Angle  float64 // angular position; used for radial/external label orientation
	Fill   MetricValue
	Border MetricValue
	Label  string
}

// Text carries position and content for standalone text.
type Text struct {
	Spec    *TextSpec
	X, Y    float64
	Content string
}

// Line carries start and end positions for line segments.
type Line struct {
	Spec   *LineSpec
	X1, Y1 float64
	X2, Y2 float64
}

// Path carries a sequence of positions for multi-point paths.
type Path struct {
	Spec   *LineSpec
	Points []Position
}
