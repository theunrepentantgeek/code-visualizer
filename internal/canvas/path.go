package canvas

import "github.com/theunrepentantgeek/code-visualizer/internal/inks"

// Path carries a sequence of positions for multi-point paths.
type Path struct {
	Spec   *LineSpec
	Points []Position
}

func (p *Path) drawTo(b Backend) {
	stroke := p.Spec.Stroke.Dip(inks.MetricValue{})

	b.DrawPath(p.Points, stroke, p.Spec.StrokeWidth)
}
