package inks_test

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

var whiteColour = color.RGBA{R: 255, G: 255, B: 255, A: 255}

func TestNumericInk_Boundaries_ReturnsBucketValues(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	values := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90}
	pal := palette.GetPalette(palette.Neutral)
	ink := inks.NumericInk("test-metric", values, pal)

	g.Expect(ink.Boundaries()).NotTo(BeEmpty())
}

func TestFixedInk_Boundaries_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := inks.FixedInk(whiteColour)
	g.Expect(ink.Boundaries()).To(BeNil())
}

func TestCategoricalInk_Boundaries_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := inks.CategoricalInk("test-metric", []string{"go"}, palette.GetPalette(palette.Categorization))
	g.Expect(ink.Boundaries()).To(BeNil())
}

func TestNumericInk_Palette_ReturnsPalette(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Temperature)
	ink := inks.NumericInk("test-metric", []float64{1, 2}, pal)

	g.Expect(ink.Palette().Name).To(Equal(palette.Temperature))
}

func TestFixedInk_Palette_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := inks.FixedInk(whiteColour)
	g.Expect(ink.Palette().Colours).To(BeEmpty())
}

func TestCategoricalInk_Categories_ReturnsList(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cats := []string{"go", "rs", "py"}
	ink := inks.CategoricalInk("test-metric", cats, palette.GetPalette(palette.Categorization))

	g.Expect(ink.Categories()).To(Equal(cats))
}

func TestNumericInk_Categories_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := inks.NumericInk("test-metric", []float64{1, 2}, palette.GetPalette(palette.Neutral))
	g.Expect(ink.Categories()).To(BeNil())
}

func TestFixedInk_Categories_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := inks.FixedInk(whiteColour)
	g.Expect(ink.Categories()).To(BeNil())
}

func TestInkInfo_Fixed(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := inks.FixedInk(color.RGBA{R: 255, A: 255})
	info := ink.Info()
	g.Expect(info.Kind).To(Equal(inks.KindFixed))
	g.Expect(info.MetricName).To(Equal(metric.Name("")))
}

func TestInkInfo_Numeric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := inks.NumericInk("file-size", []float64{1, 2, 3}, palette.GetPalette(palette.Neutral))
	info := ink.Info()
	g.Expect(info.Kind).To(Equal(inks.KindNumeric))
	g.Expect(info.MetricName).To(Equal(metric.Name("file-size")))
}

func TestInkInfo_Categorical(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := inks.CategoricalInk("file-type", []string{"go", "rs"}, palette.GetPalette(palette.Categorization))
	info := ink.Info()
	g.Expect(info.Kind).To(Equal(inks.KindCategorical))
	g.Expect(info.MetricName).To(Equal(metric.Name("file-type")))
}
