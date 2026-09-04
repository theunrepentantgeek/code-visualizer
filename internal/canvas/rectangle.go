package canvas

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
)

// Rectangle carries geometry and metric values for rectangular shapes.
type Rectangle struct {
	Spec       *RectangleSpec
	X, Y, W, H float64
	Fill       inks.MetricValue
	Border     inks.MetricValue
	Focus      model.GradientPoint
}

func (r *Rectangle) drawTo(b Backend) {
	fill := r.Spec.Fill.Fill(r.Fill, r.Focus)
	border := model.SolidFill{Color: r.Spec.Border.Dip(r.Border)}
	size := geometry.Size{Width: r.W, Height: r.H}

	b.DrawRectangle(
		geometry.NewPoint(r.X, r.Y),
		size,
		fill, border,
		r.Spec.BorderWidth,
	)
}
