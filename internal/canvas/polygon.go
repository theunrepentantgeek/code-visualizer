package canvas

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
)

// Polygon carries vertices and metric values for filled polygonal shapes.
type Polygon struct {
	Spec   *PolygonSpec
	Points []geometry.Point
	Fill   inks.MetricValue
	Border inks.MetricValue
}

func (p *Polygon) drawTo(b Backend) {
	fill := p.Spec.Fill.Fill(p.Fill, model.GradientPoint{X: 0.5, Y: 0.5})
	border := model.SolidFill{Color: p.Spec.Border.Dip(p.Border)}

	b.DrawPolygon(p.Points, fill, border, p.Spec.BorderWidth)
}
