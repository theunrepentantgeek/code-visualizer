package inks_test

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

func TestFixedInk_Dip_ReturnsFixedColour(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	ink := inks.FixedInk(red)

	result := ink.Dip(inks.MeasureValue(99.9))
	g.Expect(result).To(Equal(red))
}

func TestFixedInk_Dip_IgnoresMetricValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	blue := color.RGBA{R: 0, G: 0, B: 255, A: 255}
	ink := inks.FixedInk(blue)

	g.Expect(ink.Dip(inks.QuantityValue(0))).To(Equal(blue))
	g.Expect(ink.Dip(inks.CategoryValue("anything"))).To(Equal(blue))
	g.Expect(ink.Dip(inks.MetricValue{})).To(Equal(blue))
}

func TestFixedInk_WithOpacity(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	base := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	ink := inks.FixedInk(base, inks.WithOpacity(0.5))

	result := ink.Dip(inks.MetricValue{})
	g.Expect(result.R).To(Equal(uint8(255)))
	g.Expect(result.G).To(Equal(uint8(255)))
	g.Expect(result.B).To(Equal(uint8(255)))
	g.Expect(result.A).To(BeNumerically("~", 128, 2))
}

func TestNumericInk_Dip_MapsToColour(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	values := []float64{10, 20, 30, 40, 50}
	pal := palette.GetPalette(palette.Neutral)
	ink := inks.NumericInk("test-metric", values, pal)

	lowResult := ink.Dip(inks.MeasureValue(10))
	highResult := ink.Dip(inks.MeasureValue(50))

	g.Expect(lowResult.A).To(Equal(uint8(255)))
	g.Expect(highResult.A).To(Equal(uint8(255)))
	g.Expect(lowResult).NotTo(Equal(highResult))
}

func TestNumericInk_Dip_UsesQuantity(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	values := []float64{1, 2, 3, 4, 5}
	pal := palette.GetPalette(palette.Neutral)
	ink := inks.NumericInk("test-metric", values, pal)

	lowResult := ink.Dip(inks.QuantityValue(1))
	highResult := ink.Dip(inks.QuantityValue(5))

	g.Expect(lowResult).NotTo(Equal(highResult))
}

func TestCategoricalInk_Dip_MapsCategories(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	categories := []string{"go", "rs", "py"}
	pal := palette.GetPalette(palette.Categorization)
	ink := inks.CategoricalInk("test-metric", categories, pal)

	goCol := ink.Dip(inks.CategoryValue("go"))
	rsCol := ink.Dip(inks.CategoryValue("rs"))
	pyCol := ink.Dip(inks.CategoryValue("py"))

	g.Expect(goCol.A).To(Equal(uint8(255)))
	g.Expect(rsCol.A).To(Equal(uint8(255)))
	g.Expect(pyCol.A).To(Equal(uint8(255)))

	colours := map[color.RGBA]bool{goCol: true, rsCol: true, pyCol: true}
	g.Expect(colours).To(HaveLen(3))
}

func TestCategoricalInk_Dip_UnknownCategory_ReturnsGrey(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	categories := []string{"go"}
	pal := palette.GetPalette(palette.Categorization)
	ink := inks.CategoricalInk("test-metric", categories, pal)

	result := ink.Dip(inks.CategoryValue("unknown"))
	g.Expect(result).To(Equal(color.RGBA{R: 128, G: 128, B: 128, A: 255}))
}

func TestNumericInk_WithOpacity(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	values := []float64{10, 50}
	pal := palette.GetPalette(palette.Neutral)
	ink := inks.NumericInk("test-metric", values, pal, inks.WithOpacity(0.18))

	result := ink.Dip(inks.MeasureValue(30))
	g.Expect(result.A).To(BeNumerically("~", 46, 2))
}

func TestNumericInk_EmptyValues_ReturnsMiddleColour(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Neutral)
	ink := inks.NumericInk("test-metric", nil, pal)

	result := ink.Dip(inks.MeasureValue(42))
	mid := len(pal.Colours) / 2
	g.Expect(result).To(Equal(pal.Colours[mid]))
}

func TestFixedInk_IsCopySafe(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	red := color.RGBA{R: 255, A: 255}
	ink1 := inks.FixedInk(red)
	ink2 := ink1

	r1 := ink1.Dip(inks.MetricValue{})
	r2 := ink2.Dip(inks.MetricValue{})
	g.Expect(r1).To(Equal(r2))
}
