package inks_test

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

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

func TestNumericBreakpoints_NumericInk(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	values := []float64{0, 10, 20, 30, 40, 50, 60, 70, 80}
	pal := palette.GetPalette(palette.Neutral)
	ink := inks.NumericInk("file-size", values, pal)

	g.Expect(inks.NumericBreakpoints(ink)).To(Equal(metric.ComputeBuckets(values, len(pal.Colours)).Boundaries))
}

func TestNumericBreakpoints_FixedAndCategoricalReturnNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(inks.NumericBreakpoints(inks.FixedInk(color.RGBA{R: 255, A: 255}))).To(BeNil())
	g.Expect(
		inks.NumericBreakpoints(
			inks.CategoricalInk("file-type", []string{"go", "rs"}, palette.GetPalette(palette.Categorization)),
		),
	).To(BeNil())
}

func TestNumericBreakpoints_ReturnsACopy(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	values := []float64{0, 10, 20, 30, 40, 50, 60, 70, 80}
	pal := palette.GetPalette(palette.Neutral)
	ink := inks.NumericInk("file-size", values, pal)

	first := inks.NumericBreakpoints(ink)
	g.Expect(first).NotTo(BeNil())
	g.Expect(first).NotTo(BeEmpty())

	if len(first) == 0 {
		t.Fatal("expected numeric breakpoints")
	}

	original := first[0]
	first[0] = original + 1

	second := inks.NumericBreakpoints(ink)
	g.Expect(second).To(Equal(metric.ComputeBuckets(values, len(pal.Colours)).Boundaries))

	if len(second) == 0 {
		t.Fatal("expected numeric breakpoints copy")
	}

	g.Expect(second[0]).To(Equal(original))
}
