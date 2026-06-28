package inks_test

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

func TestLegendData_FixedInk_ReturnsNilSwatches(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := inks.FixedInk(color.RGBA{R: 255, G: 255, B: 255, A: 255})
	kind, swatches := inks.LegendData(ink)
	g.Expect(kind).To(Equal(model.LegendEntryNumeric))
	g.Expect(swatches).To(BeNil())
}

func TestLegendData_NumericInk_ReturnsBucketColours(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Temperature)
	ink := inks.NumericInk("file-size", []float64{10, 50, 100, 500, 1000}, pal)

	kind, swatches := inks.LegendData(ink)
	g.Expect(kind).To(Equal(model.LegendEntryNumeric))
	g.Expect(swatches).NotTo(BeNil())
	g.Expect(len(swatches)).To(BeNumerically(">", 0))

	if len(swatches) == 0 {
		return // unreachable; satisfies nilaway
	}

	for _, sw := range swatches {
		g.Expect(sw.Colour.A).To(Equal(uint8(255)))
	}

	last := swatches[len(swatches)-1]
	g.Expect(last.Label).To(BeEmpty())
}

func TestLegendData_CategoricalInk_ReturnsCategoryLabels(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Categorization)
	ink := inks.CategoricalInk("file-type", []string{"go", "py", "rs"}, pal)

	kind, swatches := inks.LegendData(ink)
	g.Expect(kind).To(Equal(model.LegendEntryCategorical))
	g.Expect(swatches).To(HaveLen(3))

	if len(swatches) < 3 {
		return // unreachable; satisfies nilaway
	}

	g.Expect(swatches[0].Label).To(Equal("go"))
	g.Expect(swatches[1].Label).To(Equal("py"))
	g.Expect(swatches[2].Label).To(Equal("rs"))
}

func TestLegendData_NumericInk_EmptyDataset_StillProducesOneSwatch(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Temperature)
	ink := inks.NumericInk("file-size", nil, pal)

	kind, swatches := inks.LegendData(ink)
	g.Expect(kind).To(Equal(model.LegendEntryNumeric))
	g.Expect(swatches).To(HaveLen(1))

	if len(swatches) < 1 {
		return // unreachable; satisfies nilaway
	}

	g.Expect(swatches[0].Label).To(BeEmpty())
	g.Expect(swatches[0].Colour.A).To(Equal(uint8(255)))
}
