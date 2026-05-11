package canvas

// Rectangle carries geometry and metric values for rectangular shapes.
type Rectangle struct {
	Spec       *RectangleSpec
	X, Y, W, H float64
	Fill       MetricValue
	Border     MetricValue
	Label      string
}

func (r *Rectangle) drawTo(b Backend) {
	fill := r.Spec.Fill.Dip(r.Fill)
	border := r.Spec.Border.Dip(r.Border)

	b.DrawRectangle(
		Position{X: r.X, Y: r.Y},
		Size{Width: r.W, Height: r.H},
		fill, border,
		r.Spec.BorderWidth,
	)
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

func (d *Disc) drawTo(b Backend) {
	fill := d.Spec.Fill.Dip(d.Fill)
	border := d.Spec.Border.Dip(d.Border)

	b.DrawDisc(
		Position{X: d.X, Y: d.Y},
		d.Radius,
		fill, border,
		d.Spec.BorderWidth,
	)
}

// Text carries position and content for standalone text.
type Text struct {
	Spec    *TextSpec
	X, Y    float64
	Content string
}

func (t *Text) drawTo(b Backend) {
	ink := t.Spec.Ink.Dip(MetricValue{})

	b.DrawText(
		Position{X: t.X, Y: t.Y},
		t.Content, ink,
		t.Spec.FontSize,
		t.Spec.Anchor,
		t.Spec.Rotation,
	)
}

// Line carries start and end positions for line segments.
type Line struct {
	Spec   *LineSpec
	X1, Y1 float64
	X2, Y2 float64
}

func (l *Line) drawTo(b Backend) {
	stroke := l.Spec.Stroke.Dip(MetricValue{})

	b.DrawLine(
		Position{X: l.X1, Y: l.Y1},
		Position{X: l.X2, Y: l.Y2},
		stroke,
		l.Spec.StrokeWidth,
	)
}

// Path carries a sequence of positions for multi-point paths.
type Path struct {
	Spec   *LineSpec
	Points []Position
}

func (p *Path) drawTo(b Backend) {
	stroke := p.Spec.Stroke.Dip(MetricValue{})

	b.DrawPath(p.Points, stroke, p.Spec.StrokeWidth)
}
