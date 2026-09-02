package canvas

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
)

// Disc carries geometry and metric values for circular shapes.
type Disc struct {
	Spec   *DiscSpec
	X, Y   float64
	Radius float64
	Angle  float64 // angular position; used for radial/external label orientation
	Fill   inks.MetricValue
	Border inks.MetricValue
}

func (d *Disc) drawTo(b Backend) {
	fill := d.Spec.Fill.Fill(d.Fill, model.GradientPoint{X: 0.5, Y: 0.5})
	border := model.SolidFill{Color: d.Spec.Border.Dip(d.Border)}

	b.DrawDisc(
		geometry.NewPoint(d.X, d.Y),
		d.Radius,
		fill, border,
		d.Spec.BorderWidth,
	)
}
