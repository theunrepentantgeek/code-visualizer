package canvas

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
)

// Line carries start and end positions for line segments.
type Line struct {
	Spec *LineSpec
	From geometry.Point
	To   geometry.Point
}

func (l *Line) drawTo(b Backend) {
	stroke := l.Spec.Stroke.Dip(inks.MetricValue{})

	b.DrawLine(l.From, l.To, stroke, l.Spec.StrokeWidth)
}
