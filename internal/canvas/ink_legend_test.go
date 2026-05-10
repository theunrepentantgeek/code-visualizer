package canvas

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

func TestLegendSwatches_FixedInk_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := FixedInk(white)
	g.Expect(ink.legendSwatches()).To(BeNil())
}

func TestLegendSwatches_NumericInk_ReturnsBucketColours(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Temperature)
	ink := NumericInk("file-size", []float64{10, 50, 100, 500, 1000}, pal)

	swatches := ink.legendSwatches()
	g.Expect(swatches).NotTo(BeNil())
	g.Expect(len(swatches)).To(BeNumerically(">", 0))

	if len(swatches) == 0 {
		return // unreachable; satisfies nilaway
	}

	// Each swatch should have a non-zero colour
	for _, sw := range swatches {
		g.Expect(sw.Colour.A).To(Equal(uint8(255)))
	}

	// Last swatch should have empty label (no boundary after last bucket)
	last := swatches[len(swatches)-1]
	g.Expect(last.Label).To(BeEmpty())
}

func TestLegendSwatches_CategoricalInk_ReturnsCategoryLabels(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Categorization)
	ink := CategoricalInk("file-type", []string{"go", "py", "rs"}, pal)

	swatches := ink.legendSwatches()
	g.Expect(swatches).NotTo(BeNil())
	g.Expect(swatches).To(HaveLen(3))

	if len(swatches) < 3 {
		return // unreachable; satisfies nilaway
	}

	// Labels should be sorted and present
	g.Expect(swatches[0].Label).To(Equal("go"))
	g.Expect(swatches[1].Label).To(Equal("py"))
	g.Expect(swatches[2].Label).To(Equal("rs"))
}

func TestLegendEntryKind_FixedInk_ReturnsNumeric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	ink := FixedInk(white)
	g.Expect(ink.legendEntryKind()).To(Equal(model.LegendEntryNumeric))
}

func TestLegendEntryKind_CategoricalInk_ReturnsCategorical(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Categorization)
	ink := CategoricalInk("file-type", []string{"go"}, pal)
	g.Expect(ink.legendEntryKind()).To(Equal(model.LegendEntryCategorical))
}
