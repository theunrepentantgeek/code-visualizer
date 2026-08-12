package canvas

import "github.com/theunrepentantgeek/code-visualizer/internal/inks"

// Line carries start and end positions for line segments.
type Line struct {
	Spec   *LineSpec
	X1, Y1 float64
	X2, Y2 float64
}

func (l *Line) drawTo(b Backend) {
	stroke := l.Spec.Stroke.Dip(inks.MetricValue{})

	b.DrawLine(
		Position{X: l.X1, Y: l.Y1},
		Position{X: l.X2, Y: l.Y2},
		stroke,
		l.Spec.StrokeWidth,
	)
}
