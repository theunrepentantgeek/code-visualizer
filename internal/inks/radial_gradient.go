package inks

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
)

const defaultDarken = 0.4

// RadialGradientInk wraps another Ink to produce radial gradient fills.
// The inner ink provides the centre colour; edges are darkened by the
// configured fraction.
type RadialGradientInk struct {
	inner  Ink
	darken float64
}

// NewRadialGradientInk creates a RadialGradientInk that darkens edges by 40%.
func NewRadialGradientInk(inner Ink) Ink {
	return &RadialGradientInk{inner: inner, darken: defaultDarken}
}

func (g *RadialGradientInk) Dip(value MetricValue) color.RGBA {
	return g.inner.Dip(value)
}

func (g *RadialGradientInk) Fill(value MetricValue, focus model.Point) model.Fill {
	base := g.inner.Dip(value)

	return model.RadialGradientFill{
		Center: base,
		Edge:   darken(base, g.darken),
		Focus:  focus,
	}
}

func (g *RadialGradientInk) Info() Info {
	return g.inner.Info()
}

func (g *RadialGradientInk) LegendData() (model.LegendEntryKind, []model.LegendSwatch) {
	return g.inner.LegendData()
}

// darken reduces each RGB channel by the given fraction (0.4 = 40% darker).
func darken(c color.RGBA, fraction float64) color.RGBA {
	scale := 1.0 - fraction

	return color.RGBA{
		R: uint8(float64(c.R) * scale),
		G: uint8(float64(c.G) * scale),
		B: uint8(float64(c.B) * scale),
		A: c.A,
	}
}
