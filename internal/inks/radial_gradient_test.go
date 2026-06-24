package inks_test

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

func TestRadialGradientInk_Dip_DelegatesToInner(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	red := color.RGBA{R: 255, A: 255}
	inner := inks.FixedInk(red)
	gradient := inks.NewRadialGradientInk(inner)

	result := gradient.Dip(inks.MetricValue{})
	g.Expect(result).To(Equal(red))
}

func TestRadialGradientInk_Fill_ReturnsRadialGradientFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	inner := inks.FixedInk(white)
	gradient := inks.NewRadialGradientInk(inner)

	focus := model.Point{X: 0.35, Y: 0.35}
	fill := gradient.Fill(inks.MetricValue{}, focus)

	rgf, ok := fill.(model.RadialGradientFill)
	g.Expect(ok).To(BeTrue())
	g.Expect(rgf.Center).To(Equal(white))
	g.Expect(rgf.Focus).To(Equal(focus))
	// Edge should be darker than centre
	g.Expect(rgf.Edge.R).To(BeNumerically("<", rgf.Center.R))
	g.Expect(rgf.Edge.G).To(BeNumerically("<", rgf.Center.G))
	g.Expect(rgf.Edge.B).To(BeNumerically("<", rgf.Center.B))
	g.Expect(rgf.Edge.A).To(Equal(uint8(255)))
}

func TestRadialGradientInk_Fill_DarkensBy40Percent(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	base := color.RGBA{R: 200, G: 100, B: 50, A: 255}
	inner := inks.FixedInk(base)
	gradient := inks.NewRadialGradientInk(inner)

	fill := gradient.Fill(inks.MetricValue{}, model.Point{X: 0.5, Y: 0.5})
	rgf, ok := fill.(model.RadialGradientFill)
	g.Expect(ok).To(BeTrue())

	// 40% darker: channel * 0.6
	g.Expect(rgf.Edge.R).To(Equal(uint8(120))) // 200 * 0.6 = 120
	g.Expect(rgf.Edge.G).To(Equal(uint8(60)))  // 100 * 0.6 = 60
	g.Expect(rgf.Edge.B).To(Equal(uint8(30)))  // 50 * 0.6 = 30
}

func TestRadialGradientInk_Fill_PreservesAlpha(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	base := color.RGBA{R: 200, G: 100, B: 50, A: 128}
	inner := inks.FixedInk(base)
	gradient := inks.NewRadialGradientInk(inner)

	fill := gradient.Fill(inks.MetricValue{}, model.Point{X: 0.5, Y: 0.5})
	rgf, ok := fill.(model.RadialGradientFill)
	g.Expect(ok).To(BeTrue())
	g.Expect(rgf.Center.A).To(Equal(base.A))
	g.Expect(rgf.Edge.A).To(Equal(base.A))
}

func TestRadialGradientInk_Info_DelegatesToInner(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	inner := inks.NumericInk("test-metric", []float64{1, 2}, palette.GetPalette(palette.Neutral))
	gradient := inks.NewRadialGradientInk(inner)

	g.Expect(gradient.Info()).To(Equal(inner.Info()))
}

func TestRadialGradientInk_LegendData_DelegatesToInner(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	inner := inks.CategoricalInk(
		"language",
		[]string{"go", "rs"},
		palette.GetPalette(palette.Categorization),
	)
	gradient := inks.NewRadialGradientInk(inner)

	innerKind, innerSwatches := inks.LegendData(inner)
	gradientKind, gradientSwatches := inks.LegendData(gradient)

	g.Expect(gradientKind).To(Equal(innerKind))
	g.Expect(gradientSwatches).To(Equal(innerSwatches))
}
