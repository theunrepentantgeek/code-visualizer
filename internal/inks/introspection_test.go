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
